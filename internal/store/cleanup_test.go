package store

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"modernc.org/sqlite"
)

func TestCleanupPreservesExpiredSessionAndReplayFamily(t *testing.T) {
	storage, path, owner, session, _ := ownerFixture(t)
	if err := storage.AddApplication(t.Context(), Application{ID: "rp", Name: "RP"}, session.Digest); err != nil {
		t.Fatal(err)
	}
	if err := storage.Assign(t.Context(), owner.ID, "rp", session.Digest, true); err != nil {
		t.Fatal(err)
	}
	request := Request{ID: "retained", UserID: owner.ID, ClientID: "rp", Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute)}
	if err := storage.CreateRequest(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if err := storage.AuthenticateFixture(t.Context(), request.ID, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveCode(t.Context(), request.ID, "retained-code"); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.db.Exec(`UPDATE auth_requests SET session_digest=? WHERE id='retained'`, session.Digest); err != nil {
		t.Fatal(err)
	}
	first, err := storage.ExchangeCode(t.Context(), request.ID, "rp", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := storage.Rotate(t.Context(), first.Refresh, "rp", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = storage.db.Exec(`UPDATE browser_sessions SET expires=0; UPDATE auth_requests SET expires=0`); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.Cleanup(t.Context(), 64); err != nil {
		t.Fatal(err)
	}
	storage.Close()
	storage, err = Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if _, err = storage.Lookup(t.Context(), second.Access); err != nil {
		t.Fatal("expired cookie cleanup destroyed live offline grant", err)
	}
	if _, err = storage.Rotate(t.Context(), first.Refresh, "rp", nil); err != ErrReplay {
		t.Fatal("cleanup lost unexpired consumed ancestor", err)
	}
	if _, err = storage.Cleanup(t.Context(), 64); err != nil {
		t.Fatal(err)
	}
	storage.Close()
	storage, err = Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if _, err = storage.Lookup(t.Context(), second.Access); err == nil {
		t.Fatal("retained replay did not revoke family")
	}
	var count int
	if err = storage.db.QueryRow(`SELECT count(*) FROM tokens`).Scan(&count); err != nil || count != 4 {
		t.Fatal("revoked lineage removed before its expiry")
	}
}

func TestCleanupBoundsActualChildrenAndEventuallyRemovesDeadFamilies(t *testing.T) {
	storage, _ := fixture(t)
	issued(t, storage)
	if _, err := storage.db.Exec(`UPDATE grants SET expires=0; UPDATE auth_requests SET expires=0; UPDATE tokens SET expires=0; WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<500) INSERT INTO tokens(digest,grant_id,kind,payload,expires,consumed) SELECT 'ancestor-'||n,(SELECT id FROM grants LIMIT 1),'refresh','{}',0,1 FROM counter; WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<300) INSERT INTO activity(action,subject,recorded_at) SELECT 'old','synthetic',0 FROM counter`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.db.Exec(`UPDATE tokens SET expires=? WHERE digest='ancestor-1'`, time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	counts, err := storage.Cleanup(t.Context(), 31)
	if err != nil || counts.Tokens != 0 || counts.Grants != 0 || counts.Requests != 0 || counts.Activity != 31 {
		t.Fatal("future-expiring family lost lineage or activity ignored bound", counts, err)
	}
	if _, err = storage.db.Exec(`UPDATE tokens SET expires=0 WHERE digest='ancestor-1'`); err != nil {
		t.Fatal(err)
	}
	for range 18 {
		counts, err = storage.Cleanup(t.Context(), 31)
		if err != nil {
			t.Fatal(err)
		}
		for _, count := range []int64{counts.Tokens, counts.Grants, counts.Requests, counts.Logins, counts.Sessions, counts.RecoveryCodes, counts.Activity} {
			if count > 31 {
				t.Fatal("cleanup exceeded per-table deletion limit")
			}
		}
	}
	for _, table := range []string{"tokens", "grants", "auth_requests"} {
		var count int
		if err = storage.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("dead family did not become reclaimable", table)
		}
	}
	for _, limit := range []int{0, 257} {
		if _, err = storage.Cleanup(t.Context(), limit); err == nil {
			t.Fatal("unbounded cleanup input accepted")
		}
	}
}

func TestCleanupRollbackOnCommitAndSqliteFull(t *testing.T) {
	for _, mode := range []string{"commit", "full"} {
		t.Run(mode, func(t *testing.T) {
			storage, _ := fixture(t)
			issued(t, storage)
			if _, err := storage.db.Exec(`UPDATE grants SET expires=0; UPDATE tokens SET expires=0`); err != nil {
				t.Fatal(err)
			}
			if mode == "commit" {
				if _, err := storage.db.Exec(`CREATE TABLE cleanup_fault(user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER cleanup_fail AFTER DELETE ON tokens BEGIN INSERT INTO cleanup_fault VALUES('missing'); END;`); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := storage.db.Exec(`CREATE TRIGGER cleanup_fail AFTER DELETE ON tokens BEGIN INSERT INTO activity(action,subject,recorded_at) VALUES('fault',zeroblob(8388608),0); END;`); err != nil {
					t.Fatal(err)
				}
				var pages int
				if err := storage.db.QueryRow(`PRAGMA page_count`).Scan(&pages); err != nil {
					t.Fatal(err)
				}
				if _, err := storage.db.Exec(fmt.Sprintf("PRAGMA max_page_count=%d", pages)); err != nil {
					t.Fatal(err)
				}
			}
			counts, err := storage.Cleanup(t.Context(), 64)
			if err == nil || counts != (CleanupResult{}) {
				t.Fatal("failed cleanup reported committed work")
			}
			var sqliteError *sqlite.Error
			if mode == "full" && (!errors.As(err, &sqliteError) || sqliteError.Code() != 13) {
				t.Fatal("test did not exercise SQLite capacity failure", err)
			}
			var count int
			if err = storage.db.QueryRow(`SELECT count(*) FROM tokens`).Scan(&count); err != nil || count != 2 {
				t.Fatal("failed cleanup partially removed lineage")
			}
			if _, err = storage.db.Exec(`DROP TRIGGER cleanup_fail`); err != nil {
				t.Fatal(err)
			}
			if _, err = storage.Cleanup(t.Context(), 64); err != nil {
				t.Fatal("cleanup did not recover after injected failure", err)
			}
		})
	}
}

func TestExpiredReplayRemainsRejectedWithoutClaimingFamilyRevocation(t *testing.T) {
	storage, _ := fixture(t)
	first := issued(t, storage)
	second, err := storage.Rotate(t.Context(), first.Refresh, "rp", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = storage.db.Exec(`UPDATE tokens SET expires=0 WHERE digest=?`, Hash(first.Refresh)); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.Cleanup(t.Context(), 64); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.Rotate(t.Context(), first.Refresh, "rp", nil); err == nil {
		t.Fatal("expired ancestor accepted")
	}
	if _, err = storage.Lookup(t.Context(), second.Access); err != nil {
		t.Fatal("baseline expired-replay semantics unexpectedly changed", err)
	}
}
