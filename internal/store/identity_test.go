package store

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func ownerFixture(t *testing.T) (*Store, string, Account, Session, []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "identity.db")
	storage, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { storage.Close() })
	if err = storage.Bootstrap(t.Context(), "owner", "synthetic-verifier", "synthetic-key-id"); err != nil {
		t.Fatal("bootstrap", err)
	}
	owner, err := storage.AccountByName(t.Context(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	session := passwordSession(t, storage, owner)
	sealed := []byte("synthetic-sealed-factor")
	if err = storage.StageFactor(t.Context(), session, sealed); err != nil {
		t.Fatal("stage factor", err)
	}
	handle, codes, err := storage.ConfirmFactor(t.Context(), session, sealed, 100)
	if err != nil {
		t.Fatal("confirm factor", err)
	}
	session, err = storage.Session(t.Context(), handle)
	if err != nil {
		t.Fatal(err)
	}
	owner, err = storage.Account(t.Context(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	return storage, path, owner, session, codes
}

func TestProductionInvalidationReachesCodesAndFamilies(t *testing.T) {
	storage, _, _, admin, _ := ownerFixture(t)
	id, err := storage.AddAccount(t.Context(), "member", "synthetic-verifier", admin.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err = storage.AddApplication(t.Context(), Application{ID: "rp", Name: "Example"}, admin.Digest); err != nil {
		t.Fatal(err)
	}
	if err = storage.Assign(t.Context(), id, "rp", admin.Digest, true); err != nil {
		t.Fatal(err)
	}
	account, err := storage.Account(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	session := passwordSession(t, storage, account)
	prepare := func() string {
		request := LoginRequest{ID: NewHandle(), ClientID: "rp", Browser: session.Digest, Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute).Unix()}
		if err := storage.CreateLogin(t.Context(), request); err != nil {
			t.Fatal(err)
		}
		if err := storage.CompleteLogin(t.Context(), request, session, []byte(`{}`), []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
		if err := storage.SaveCode(t.Context(), request.ID, NewHandle()); err != nil {
			t.Fatal(err)
		}
		return request.ID
	}
	issued, err := storage.ExchangeCode(t.Context(), prepare(), "rp", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	pending := prepare()
	if err = storage.ResetAccount(t.Context(), id, "replacement-verifier", false, admin.Digest); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.Lookup(t.Context(), issued.Access); err == nil {
		t.Fatal("password reset retained access")
	}
	if _, err = storage.Rotate(t.Context(), issued.Refresh, "rp", []byte(`{}`)); err == nil {
		t.Fatal("password reset retained refresh")
	}
	if _, err = storage.ExchangeCode(t.Context(), pending, "rp", []byte(`{}`), true); err == nil {
		t.Fatal("password reset retained pending code")
	}
	if err = storage.CompleteLogin(t.Context(), LoginRequest{ID: NewHandle(), ClientID: "rp"}, session, []byte(`{}`), []byte(`{}`)); err == nil {
		t.Fatal("stale session completed authorization")
	}
}
func passwordSession(t *testing.T, storage *Store, account Account) Session {
	t.Helper()
	anonymous, err := storage.AnonymousSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pre, err := storage.Session(t.Context(), anonymous)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := storage.PasswordSession(t.Context(), pre.Digest, account)
	if err != nil {
		t.Fatal(err)
	}
	session, err := storage.Session(t.Context(), handle)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestOwnerClaimAndCredentialFencing(t *testing.T) {
	storage, _, owner, session, _ := ownerFixture(t)
	if err := storage.Bootstrap(t.Context(), "other", "verifier", "other-key"); err == nil {
		t.Fatal("ownership reopened")
	}
	if err := storage.CheckInstallation(t.Context(), "different-key"); err == nil {
		t.Fatal("wrong key accepted")
	}
	if err := storage.DisableAccount(t.Context(), owner.ID, session.Digest); err == nil {
		t.Fatal("last owner disabled")
	}
	partial := passwordSession(t, storage, owner)
	if _, err := storage.AddAccount(t.Context(), "forbidden", "verifier", partial.Digest); err == nil {
		t.Fatal("password-only admin admitted")
	}
	if err := storage.RecoverOwner(t.Context(), "changed-verifier"); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AddAccount(t.Context(), "stale", "verifier", session.Digest); err == nil {
		t.Fatal("stale admin session authorized")
	}
	anonymous, err := storage.AnonymousSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pre, err := storage.Session(t.Context(), anonymous)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = storage.PasswordSession(t.Context(), pre.Digest, owner); err == nil {
		t.Fatal("verified old password raced through recovery")
	}
	current, err := storage.Account(t.Context(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Factor) != 0 || !current.RequireMfa {
		t.Fatal("owner recovery did not require enrollment")
	}
}

func TestTotpAndRecoveryAtomicSessions(t *testing.T) {
	storage, path, owner, _, codes := ownerFixture(t)
	other, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	sessions := make([]Session, 20)
	for index := range sessions {
		sessions[index] = passwordSession(t, storage, owner)
	}
	var successes atomic.Int32
	runConcurrent(t, func(index int) error {
		selected := storage
		if index%2 == 1 {
			selected = other
		}
		_, err := selected.ProveFactor(t.Context(), sessions[index], owner.Factor, 101)
		if err == nil {
			successes.Add(1)
		}
		return err
	})
	if successes.Load() != 1 {
		t.Fatalf("TOTP successes=%d", successes.Load())
	}
	if _, err = storage.ProveFactor(t.Context(), passwordSession(t, storage, owner), owner.Factor, 101); err == nil {
		t.Fatal("TOTP replay accepted")
	}
	for index := range sessions {
		sessions[index] = passwordSession(t, storage, owner)
	}
	successes.Store(0)
	runConcurrent(t, func(index int) error {
		selected := storage
		if index%2 == 1 {
			selected = other
		}
		handle, err := selected.RecoverFactor(t.Context(), sessions[index], codes[0])
		if err == nil {
			successes.Add(1)
			restricted, readErr := selected.Session(t.Context(), handle)
			if readErr != nil || restricted.Assurance {
				t.Error("recovery granted full assurance")
			}
		}
		return err
	})
	if successes.Load() != 1 {
		t.Fatalf("recovery successes=%d", successes.Load())
	}
	current, err := storage.Account(t.Context(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = storage.RecoverFactor(t.Context(), passwordSession(t, storage, current), codes[1]); err == nil {
		t.Fatal("old recovery set remained active")
	}
}

func TestTotpCommitFailureDoesNotConsumeStep(t *testing.T) {
	storage, _, owner, _, _ := ownerFixture(t)
	session := passwordSession(t, storage, owner)
	if _, err := storage.db.Exec(`CREATE TABLE identity_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER identity_fail AFTER INSERT ON browser_sessions BEGIN INSERT INTO identity_fault VALUES('missing'); END;`); err != nil {
		t.Fatal(err)
	}
	if handle, err := storage.ProveFactor(t.Context(), session, owner.Factor, 101); err == nil || handle != "" {
		t.Fatal("failed commit returned session")
	}
	if _, err := storage.db.Exec(`DROP TRIGGER identity_fail`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.ProveFactor(t.Context(), session, owner.Factor, 101); err != nil {
		t.Fatal("failed commit consumed TOTP step", err)
	}
}
