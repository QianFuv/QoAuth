package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotInspectionRejectsSchemaAndOwnershipCorruption(t *testing.T) {
	for name, statement := range map[string]string{
		"extra-ledger":    `INSERT INTO schema_migrations VALUES(-1,'extra')`,
		"future-ledger":   `INSERT INTO schema_migrations VALUES(99,'future')`,
		"actual-schema":   `ALTER TABLE users ADD COLUMN unexpected TEXT`,
		"extra-trigger":   `CREATE TRIGGER unexpected AFTER INSERT ON activity BEGIN UPDATE users SET enabled=1; END`,
		"missing-owner":   `UPDATE accounts SET owner=0`,
		"duplicate-owner": `INSERT INTO users(id) VALUES('extra'); INSERT INTO accounts(user_id,username,password,admin,owner,require_mfa) VALUES('extra','other','invalid',1,1,1)`,
		"orphan":          `PRAGMA foreign_keys=OFF; INSERT INTO assignments VALUES('missing','missing')`,
	} {
		t.Run(name, func(t *testing.T) {
			storage, path, _, _, _ := ownerFixture(t)
			if _, err := InspectInstallation(t.Context(), path, "synthetic-key-id"); err != nil {
				t.Fatal("valid baseline rejected", err)
			}
			if _, err := storage.db.Exec(statement); err != nil {
				t.Fatal(err)
			}
			storage.Close()
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := InspectSnapshot(t.Context(), path, "synthetic-key-id"); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
			after, _ := os.ReadFile(path)
			if Hash(string(before)) != Hash(string(after)) {
				t.Fatal("read-only inspection changed input")
			}
		})
	}
}

func TestRestoreRevokesAllStateAndPreservesQuarantine(t *testing.T) {
	storage, path, owner, session, recovery := ownerFixture(t)
	member, err := storage.AddAccount(t.Context(), "member", "old-verifier", session.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err = storage.AddApplication(t.Context(), Application{ID: "rp", Name: "RP"}, session.Digest); err != nil {
		t.Fatal(err)
	}
	if err = storage.Assign(t.Context(), owner.ID, "rp", session.Digest, true); err != nil {
		t.Fatal(err)
	}
	request := Request{ID: "pending", UserID: owner.ID, ClientID: "rp", Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute)}
	if err = storage.CreateRequest(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if err = storage.AuthenticateFixture(t.Context(), request.ID, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err = storage.SaveCode(t.Context(), request.ID, "pre-backup-code"); err != nil {
		t.Fatal(err)
	}
	issued, err := storage.ExchangeCode(t.Context(), request.ID, "rp", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	if err = storage.RestoreState(t.Context(), "synthetic-key-id", "new-binding", deadline); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"tokens", "grants", "auth_requests", "login_requests", "browser_sessions", "recovery_codes", "consents", "assignments"} {
		var count int
		if err = storage.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("restore retained authentication", table)
		}
	}
	if _, err = storage.Lookup(t.Context(), issued.Access); err == nil {
		t.Fatal("restored access survived")
	}
	if _, err = storage.Rotate(t.Context(), issued.Refresh, "rp", nil); err == nil {
		t.Fatal("restored refresh survived")
	}
	if _, err = storage.RecoverFactor(t.Context(), session, recovery[0]); err == nil {
		t.Fatal("restored recovery code survived")
	}
	if err = storage.CheckServing(t.Context(), "new-binding", "https://auth.example.test", deadline.Add(time.Second)); err == nil {
		t.Fatal("unreviewed restore served")
	}
	if err = storage.RecoverRestoredOwner(t.Context(), "fresh-verifier"); err != nil {
		t.Fatal(err)
	}
	if err = storage.CheckServing(t.Context(), "new-binding", "https://auth.example.test", deadline.Add(-time.Second)); err == nil {
		t.Fatal("owner recovery bypassed signed-token quarantine")
	}
	storage.Close()
	storage, err = Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if err = storage.CheckServing(t.Context(), "new-binding", "https://auth.example.test", deadline); err != nil {
		t.Fatal(err)
	}
	if err = storage.CheckServing(t.Context(), "new-binding", "https://other.example.test", deadline); err == nil {
		t.Fatal("issuer drift accepted")
	}
	current, err := storage.Account(t.Context(), owner.ID)
	if err != nil || !current.Enabled || len(current.Factor) != 0 || !current.RequireMfa || current.Password != "fresh-verifier" {
		t.Fatal("owner was not restricted to new MFA")
	}
	other, _ := storage.Account(t.Context(), member)
	if other.Enabled {
		t.Fatal("unreviewed member enabled")
	}
	if _, err = storage.Application(t.Context(), "rp"); err == nil {
		t.Fatal("unreviewed client enabled")
	}
	if err = storage.RecoverDisabledUser(t.Context(), "member", "new-member-verifier"); err != nil {
		t.Fatal(err)
	}
	other, _ = storage.Account(t.Context(), member)
	if !other.Enabled || other.Admin || len(other.Factor) != 0 || !other.RequireMfa {
		t.Fatal("member recovery restored stale assurance or privilege")
	}
}

func TestRestoreCommitFailureLeavesNoPartialInvalidation(t *testing.T) {
	storage, _, _, _, _ := ownerFixture(t)
	if _, err := storage.db.Exec(`CREATE TABLE restore_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER restore_fail AFTER UPDATE ON installation BEGIN INSERT INTO restore_fault VALUES('missing'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := storage.RestoreState(t.Context(), "synthetic-key-id", "next-binding", time.Now().Add(time.Hour)); err == nil {
		t.Fatal("failed restore commit accepted")
	}
	if err := storage.CheckInstallation(t.Context(), "synthetic-key-id"); err != nil {
		t.Fatal("failed restore changed binding")
	}
	var sessions, codes int
	storage.db.QueryRow(`SELECT count(*) FROM browser_sessions`).Scan(&sessions)
	storage.db.QueryRow(`SELECT count(*) FROM recovery_codes`).Scan(&codes)
	if sessions == 0 || codes == 0 {
		t.Fatal("failed restore partially invalidated data")
	}
}

func TestSnapshotIncludesCommittedWal(t *testing.T) {
	storage, path, _, _, _ := ownerFixture(t)
	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Fatal("test did not create a WAL")
	}
	destination := filepath.Join(t.TempDir(), "snapshot.db")
	if err := storage.Snapshot(t.Context(), destination); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectSnapshot(t.Context(), destination, "synthetic-key-id"); err != nil {
		t.Fatal("snapshot omitted committed WAL owner/state", err)
	}
}
