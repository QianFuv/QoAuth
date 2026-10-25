package store

import "testing"

func TestSigningBindingCommitFailureAndIdempotentResume(t *testing.T) {
	storage, _, _, _, _ := ownerFixture(t)
	if _, err := storage.db.Exec(`CREATE TABLE key_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER key_fail AFTER UPDATE ON installation BEGIN INSERT INTO key_fault VALUES('missing'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := storage.RotateSigningBinding(t.Context(), "synthetic-key-id", "next-binding"); err == nil {
		t.Fatal("failed commit accepted")
	}
	if err := storage.CheckInstallation(t.Context(), "synthetic-key-id"); err != nil {
		t.Fatal("failed commit changed active binding")
	}
	if _, err := storage.db.Exec(`DROP TRIGGER key_fail`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := storage.RotateSigningBinding(t.Context(), "synthetic-key-id", "next-binding"); err != nil {
			t.Fatal("retry failed", err)
		}
	}
	if err := storage.RotateSigningBinding(t.Context(), "unrelated", "replacement"); err == nil {
		t.Fatal("unrelated binding accepted")
	}
	var count int
	if err := storage.db.QueryRow(`SELECT count(*) FROM activity WHERE action='signing-rotate'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("rotation activity was lost or duplicated")
	}
}
