package store

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestAnonymousStateHasAtomicQuotasAndBoundedReclamation(t *testing.T) {
	storage, path := fixture(t)
	other, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	handle, err := storage.AnonymousSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	runConcurrent(t, func(index int) error {
		selected := storage
		if index%2 == 1 {
			selected = other
		}
		err := selected.CreateLogin(t.Context(), LoginRequest{ID: NewHandle(), ClientID: "rp", ClientGeneration: 1, Browser: Hash(handle), Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute).Unix()})
		if err == nil {
			successes.Add(1)
		}
		return err
	})
	if successes.Load() != 16 {
		t.Fatalf("pending quota: %d", successes.Load())
	}
	if _, err := storage.db.Exec(`UPDATE login_requests SET expires=0; UPDATE browser_sessions SET expires=0 WHERE user_id IS NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AnonymousSession(t.Context()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := storage.db.QueryRow(`SELECT count(*) FROM login_requests`).Scan(&count); err != nil || count != 0 {
		t.Fatal("expired requests retained")
	}
	if _, err := storage.db.Exec(`WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<4095) INSERT INTO browser_sessions(digest,csrf,expires) SELECT 'quota-'||n,'synthetic',9999999999 FROM counter`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AnonymousSession(t.Context()); err == nil {
		t.Fatal("anonymous capacity exceeded")
	}
	if _, err := storage.db.Exec(`UPDATE browser_sessions SET expires=0 WHERE digest LIKE 'quota-%'`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AnonymousSession(t.Context()); err != nil {
		t.Fatal("expired state not reclaimable", err)
	}
	if err := storage.db.QueryRow(`SELECT count(*) FROM browser_sessions WHERE user_id IS NULL`).Scan(&count); err != nil || count != 4033 {
		t.Fatal("cleanup batch unbounded or incorrect", count)
	}
}

func TestRejectedAdmissionStillCommitsBacklogCleanup(t *testing.T) {
	storage, _ := fixture(t)
	if _, err := storage.db.Exec(`WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<4300) INSERT INTO browser_sessions(digest,csrf,expires) SELECT 'old-'||n,'synthetic',0 FROM counter`); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.AnonymousSession(t.Context()); err == nil {
		t.Fatal("admission exceeded capacity")
	}
	var count int
	if err := storage.db.QueryRow(`SELECT count(*) FROM browser_sessions`).Scan(&count); err != nil || count != 4236 {
		t.Fatal("quota denial rolled back cleanup", count)
	}
	for range 3 {
		_, _ = storage.AnonymousSession(t.Context())
	}
	handle, err := storage.AnonymousSession(t.Context())
	if err != nil {
		t.Fatal("anonymous backlog never recovered", err)
	}
	if _, err := storage.db.Exec(`WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<2200) INSERT INTO login_requests(id,client_id,browser_digest,payload,expires) SELECT 'old-'||n,'rp','old-browser','{}',0 FROM counter`); err != nil {
		t.Fatal(err)
	}
	request := LoginRequest{ID: NewHandle(), ClientID: "rp", ClientGeneration: 1, Browser: Hash(handle), Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute).Unix()}
	if err := storage.CreateLogin(t.Context(), request); err == nil {
		t.Fatal("request quota exceeded")
	}
	if err := storage.db.QueryRow(`SELECT count(*) FROM login_requests`).Scan(&count); err != nil || count != 2136 {
		t.Fatal("request denial rolled back cleanup", count)
	}
	for range 3 {
		request.ID = NewHandle()
		_ = storage.CreateLogin(t.Context(), request)
	}
	request.ID = NewHandle()
	if err := storage.CreateLogin(t.Context(), request); err != nil {
		t.Fatal("request backlog never recovered", err)
	}
}

func TestExpiredBrowserQuotaBehindAnotherCleanupBatch(t *testing.T) {
	storage, _ := fixture(t)
	handle, err := storage.AnonymousSession(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.db.Exec(`WITH RECURSIVE counter(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM counter WHERE n<100) INSERT INTO login_requests(id,client_id,browser_digest,payload,expires) SELECT 'older-'||n,'rp','another-browser','{}',0 FROM counter`); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if _, err := storage.db.Exec(`INSERT INTO login_requests(id,client_id,browser_digest,payload,expires) VALUES(?,'rp',?,'{}',1)`, NewHandle(), Hash(handle)); err != nil {
			t.Fatal(err)
		}
	}
	request := LoginRequest{ID: NewHandle(), ClientID: "rp", ClientGeneration: 1, Browser: Hash(handle), Payload: []byte(`{}`), Expires: time.Now().Add(time.Minute).Unix()}
	_ = storage.CreateLogin(t.Context(), request)
	request.ID = NewHandle()
	if err := storage.CreateLogin(t.Context(), request); err != nil {
		t.Fatal("expired per-browser quota never cleared", err)
	}
}
