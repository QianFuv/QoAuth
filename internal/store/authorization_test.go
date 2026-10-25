package store

import (
	"testing"
	"time"
)

func authorizationFixture(t *testing.T) (*Store, Session, LoginRequest) {
	t.Helper()
	storage, _, _, admin, _ := ownerFixture(t)
	if err := storage.AddApplication(t.Context(), Application{ID: "rp", Name: "Example"}, admin.Digest); err != nil {
		t.Fatal(err)
	}
	if err := storage.Assign(t.Context(), admin.UserID, "rp", admin.Digest, true); err != nil {
		t.Fatal(err)
	}
	login := LoginRequest{ID: NewHandle(), ClientID: "rp", ClientGeneration: 1, Browser: admin.Digest, Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute).Unix()}
	if err := storage.CreateLogin(t.Context(), login); err != nil {
		t.Fatal(err)
	}
	return storage, admin, login
}

func TestAutomaticConsentCompletionCannotUndoWithdrawal(t *testing.T) {
	storage, session, login := authorizationFixture(t)
	approved := []byte(`{"Scopes":["openid"],"Audiences":["api"]}`)
	if err := storage.CompleteLogin(t.Context(), login, session, []byte(`{}`), approved, true); err != nil {
		t.Fatal(err)
	}
	login.ID = NewHandle()
	if err := storage.CreateLogin(t.Context(), login); err != nil {
		t.Fatal(err)
	}
	observed, err := storage.Consent(t.Context(), session.UserID, "rp")
	if err != nil || !consentCovers(observed, approved) {
		t.Fatal("missing initial consent")
	}
	if err := storage.WithdrawConsent(t.Context(), session.UserID, "rp", session.Digest); err != nil {
		t.Fatal(err)
	}
	if err := storage.CompleteLogin(t.Context(), login, session, []byte(`{}`), approved, false); err == nil {
		t.Fatal("automatic completion resurrected consent")
	}
	if _, err := storage.Consent(t.Context(), session.UserID, "rp"); err == nil {
		t.Fatal("consent restored")
	}
	if _, err := storage.RequestByID(t.Context(), login.ID); err == nil {
		t.Fatal("authorization survived")
	}
	current, err := storage.Login(t.Context(), login.ID)
	if err != nil || current.Completed {
		t.Fatal("failed completion was not atomic")
	}
	if err := storage.CompleteLogin(t.Context(), login, session, []byte(`{}`), approved, true); err != nil {
		t.Fatal("fresh explicit approval failed", err)
	}
}

func TestAuthorizationCommitFailureRollsBackConsentAndCompletion(t *testing.T) {
	storage, session, login := authorizationFixture(t)
	if _, err := storage.db.Exec(`CREATE TABLE completion_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER completion_fail AFTER INSERT ON auth_requests BEGIN INSERT INTO completion_fault VALUES('missing'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := storage.CompleteLogin(t.Context(), login, session, []byte(`{}`), []byte(`{"Scopes":["openid"]}`), true); err == nil {
		t.Fatal("commit unexpectedly succeeded")
	}
	if _, err := storage.Consent(t.Context(), session.UserID, "rp"); err == nil {
		t.Fatal("failed commit persisted consent")
	}
	if _, err := storage.RequestByID(t.Context(), login.ID); err == nil {
		t.Fatal("failed commit persisted authorization")
	}
	current, err := storage.Login(t.Context(), login.ID)
	if err != nil || current.Completed {
		t.Fatal("failed commit consumed transaction")
	}
	if _, err := storage.db.Exec(`DROP TRIGGER completion_fail`); err != nil {
		t.Fatal(err)
	}
	if err := storage.CompleteLogin(t.Context(), login, session, []byte(`{}`), []byte(`{"Scopes":["openid"]}`), true); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryCommitFailureRestoresAllAuthenticationState(t *testing.T) {
	storage, _, owner, prior, codes := ownerFixture(t)
	session := passwordSession(t, storage, owner)
	if _, err := storage.db.Exec(`CREATE TABLE recovery_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER recovery_fail AFTER INSERT ON browser_sessions BEGIN INSERT INTO recovery_fault VALUES('missing'); END;`); err != nil {
		t.Fatal(err)
	}
	if handle, err := storage.RecoverFactor(t.Context(), session, codes[0]); err == nil || handle != "" {
		t.Fatal("failed recovery returned authorization")
	}
	current, err := storage.Account(t.Context(), owner.ID)
	if err != nil || current.Version != owner.Version || string(current.Factor) != string(owner.Factor) {
		t.Fatal("failed recovery changed account")
	}
	var revoked, consumed int
	if err := storage.db.QueryRow(`SELECT revoked FROM browser_sessions WHERE digest=?`, prior.Digest).Scan(&revoked); err != nil || revoked != 0 {
		t.Fatal("prior session revoked")
	}
	if err := storage.db.QueryRow(`SELECT consumed FROM recovery_codes WHERE digest=?`, Hash(codes[0])).Scan(&consumed); err != nil || consumed != 0 {
		t.Fatal("recovery code consumed")
	}
	if _, err := storage.db.Exec(`DROP TRIGGER recovery_fail`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.RecoverFactor(t.Context(), session, codes[0]); err != nil {
		t.Fatal("recovery retry failed", err)
	}
}

func TestClientReplacementRejectsAuthorizationAlreadyBeingConstructed(t *testing.T) {
	storage, session, _ := authorizationFixture(t)
	observed, err := storage.Application(t.Context(), "rp")
	if err != nil {
		t.Fatal(err)
	}
	stale := LoginRequest{ID: NewHandle(), ClientID: observed.ID, ClientGeneration: observed.Generation, Browser: session.Digest, Payload: []byte(`{"old_permissions":true}`), Expires: time.Now().Add(time.Minute).Unix()}
	if err = storage.DisableApplication(t.Context(), observed.ID, session.Digest); err != nil {
		t.Fatal(err)
	}
	if err = storage.ReplaceDisabledApplication(t.Context(), Application{ID: "rp", Name: "Reviewed", Public: true, Scopes: []string{"openid"}}, session.Digest); err != nil {
		t.Fatal(err)
	}
	if err = storage.Assign(t.Context(), session.UserID, "rp", session.Digest, true); err != nil {
		t.Fatal(err)
	}
	if err = storage.CreateLogin(t.Context(), stale); err == nil {
		t.Fatal("in-flight old client configuration admitted after replacement")
	}
	if err = storage.CompleteLogin(t.Context(), stale, session, stale.Payload, []byte(`{}`), true); err == nil {
		t.Fatal("stale client request became issuable")
	}
	if _, err = storage.RequestByID(t.Context(), stale.ID); err == nil {
		t.Fatal("stale authorization survived replacement")
	}
	current, err := storage.Application(t.Context(), "rp")
	if err != nil || current.Generation <= observed.Generation {
		t.Fatal("replacement did not advance client generation", err)
	}
	fresh := stale
	fresh.ID = NewHandle()
	fresh.ClientGeneration = current.Generation
	fresh.Payload = []byte(`{"scope":"openid"}`)
	if err = storage.CreateLogin(t.Context(), fresh); err != nil {
		t.Fatal("current client configuration rejected", err)
	}
	if err = storage.CompleteLogin(t.Context(), fresh, session, fresh.Payload, []byte(`{"Scopes":["openid"]}`), true); err != nil {
		t.Fatal("current client could not complete", err)
	}
}
