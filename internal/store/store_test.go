package store

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.db")
	storage, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { storage.Close() })
	if err = storage.SeedFixture(t.Context(), "synthetic", []string{"rp", "other"}); err != nil {
		t.Fatal(err)
	}
	return storage, path
}

func code(t *testing.T, storage *Store) string {
	t.Helper()
	id := NewHandle()
	if err := storage.CreateRequest(t.Context(), Request{ID: id, UserID: "synthetic", ClientID: "rp", Payload: []byte(`{"scope":"openid"}`), Expires: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := storage.AuthenticateFixture(t.Context(), id, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveCode(t.Context(), id, NewHandle()); err != nil {
		t.Fatal(err)
	}
	return id
}

func issued(t *testing.T, storage *Store) Issued {
	t.Helper()
	result, err := storage.ExchangeCode(t.Context(), code(t, storage), "rp", []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestRuntimeMigrationAndRestart(t *testing.T) {
	storage, path := fixture(t)
	settings, err := storage.Settings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual SQLite: %+v", settings)
	if !fixedEngine(settings.Version) || settings.Journal != "wal" || settings.Synchronous != 2 || settings.ForeignKeys != 1 || settings.BusyTimeout != 3000 {
		t.Fatal(settings)
	}
	token := issued(t, storage)
	if err = storage.Revoke(t.Context(), token.Refresh, "rp"); err != nil {
		t.Fatal(err)
	}
	storage.Close()
	reopened, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = reopened.Lookup(t.Context(), token.Access); !errors.Is(err, ErrInvalid) {
		t.Fatalf("revocation lost on restart: %v", err)
	}
	if _, err = reopened.db.Exec(`UPDATE schema_migrations SET digest='changed'`); err != nil {
		t.Fatal(err)
	}
	reopened.Close()
	if invalid, err := Open(t.Context(), path); err == nil {
		invalid.Close()
		t.Fatal("modified migration accepted")
	}
}

func TestCodeAndRefreshConcurrencyAcrossConnections(t *testing.T) {
	storage, path := fixture(t)
	other, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	request := code(t, storage)
	var count atomic.Int32
	runConcurrent(t, func(index int) error {
		selected := storage
		if index%2 == 1 {
			selected = other
		}
		_, err := selected.ExchangeCode(t.Context(), request, "rp", []byte(`{}`), true)
		if err == nil {
			count.Add(1)
		}
		return err
	})
	if count.Load() != 1 {
		t.Fatalf("code successors=%d", count.Load())
	}
	token := issued(t, storage)
	count.Store(0)
	runConcurrent(t, func(index int) error {
		selected := storage
		if index%2 == 1 {
			selected = other
		}
		_, err := selected.Rotate(t.Context(), token.Refresh, "rp", []byte(`{}`))
		if err == nil {
			count.Add(1)
		}
		return err
	})
	if count.Load() != 1 {
		t.Fatalf("refresh successors=%d", count.Load())
	}
	if _, err = storage.Lookup(t.Context(), token.Access); !errors.Is(err, ErrInvalid) {
		t.Fatalf("replay did not durably revoke family: %v", err)
	}
}

func runConcurrent(t *testing.T, operation func(int) error) {
	t.Helper()
	var workers sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 20)
	for index := range 20 {
		workers.Add(1)
		go func() { defer workers.Done(); <-start; results <- operation(index) }()
	}
	close(start)
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, ErrInvalid) && !errors.Is(err, ErrReplay) {
			t.Fatalf("unexpected concurrency error: %v", err)
		}
	}
}

func TestTransactionFailureLeavesNoPartialIssuance(t *testing.T) {
	storage, _ := fixture(t)
	request := code(t, storage)
	if _, err := storage.db.Exec(`CREATE TRIGGER fail_token BEFORE INSERT ON tokens BEGIN SELECT RAISE(ABORT,'injected insertion failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if result, err := storage.ExchangeCode(t.Context(), request, "rp", []byte(`{}`), true); err == nil || result.Access != "" {
		t.Fatal("failed commit returned a usable token")
	}
	var grants int
	if err := storage.db.QueryRow(`SELECT count(*) FROM grants`).Scan(&grants); err != nil || grants != 0 {
		t.Fatalf("partial grant: %d %v", grants, err)
	}
	if _, err := storage.db.Exec(`DROP TRIGGER fail_token`); err != nil {
		t.Fatal(err)
	}
	result, err := storage.ExchangeCode(t.Context(), request, "rp", []byte(`{}`), true)
	if err != nil {
		t.Fatal("code was consumed despite rollback", err)
	}
	if _, err = storage.db.Exec(`CREATE TRIGGER fail_refresh BEFORE INSERT ON tokens BEGIN SELECT RAISE(ABORT,'injected rotation failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if rotated, err := storage.Rotate(t.Context(), result.Refresh, "rp", []byte(`{}`)); err == nil || rotated.Access != "" {
		t.Fatal("partial refresh returned")
	}
	if _, err = storage.db.Exec(`DROP TRIGGER fail_refresh`); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.Rotate(t.Context(), result.Refresh, "rp", []byte(`{}`)); err != nil {
		t.Fatal("rollback consumed refresh", err)
	}
}

func TestRecoveryConsumptionIsAtomic(t *testing.T) {
	storage, _ := fixture(t)
	recovery := NewHandle()
	if err := storage.AddRecoveryFixture(t.Context(), "synthetic", recovery); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.db.Exec(`CREATE TRIGGER fail_activity BEFORE INSERT ON activity BEGIN SELECT RAISE(ABORT,'injected recovery failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := storage.ConsumeRecovery(t.Context(), "synthetic", recovery); err == nil {
		t.Fatal("injected failure ignored")
	}
	if _, err := storage.db.Exec(`DROP TRIGGER fail_activity`); err != nil {
		t.Fatal(err)
	}
	var count atomic.Int32
	runConcurrent(t, func(_ int) error {
		err := storage.ConsumeRecovery(t.Context(), "synthetic", recovery)
		if err == nil {
			count.Add(1)
		}
		return err
	})
	if count.Load() != 1 {
		t.Fatalf("recovery successes=%d", count.Load())
	}
}

func TestClientBindingAndChangesInvalidatePendingState(t *testing.T) {
	storage, _ := fixture(t)
	pending := code(t, storage)
	token := issued(t, storage)
	if _, err := storage.ExchangeCode(t.Context(), pending, "other", []byte(`{}`), true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-client code: %v", err)
	}
	if _, err := storage.Rotate(t.Context(), token.Refresh, "other", []byte(`{}`)); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-client refresh: %v", err)
	}
	if _, err := storage.Lookup(t.Context(), token.Access); err != nil {
		t.Fatal("wrong client revoked another client's grant", err)
	}
	if err := storage.Invalidate(t.Context(), "synthetic", "rp"); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.ExchangeCode(t.Context(), pending, "rp", []byte(`{}`), true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("removed assignment redeemed pending code: %v", err)
	}
	if _, err := storage.Lookup(t.Context(), token.Access); !errors.Is(err, ErrInvalid) {
		t.Fatal("grant survived assignment removal")
	}
}

func TestProcessExitRollsBackUncommittedConsumption(t *testing.T) {
	storage, path := fixture(t)
	id := code(t, storage)
	child := exec.Command(os.Args[0], "-test.run=^TestCrashHelper$")
	child.Env = append(os.Environ(), "QOAUTH_CRASH_DB="+path, "QOAUTH_CRASH_REQUEST="+id)
	err := child.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("crash helper: %v", err)
	}
	if _, err = storage.ExchangeCode(t.Context(), id, "rp", []byte(`{}`), true); err != nil {
		t.Fatal("uncommitted consumption survived process exit", err)
	}
}

func TestCrashHelper(t *testing.T) {
	path := os.Getenv("QOAUTH_CRASH_DB")
	if path == "" {
		t.Skip("subprocess only")
	}
	storage, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := storage.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE auth_requests SET consumed=1 WHERE id=?`, os.Getenv("QOAUTH_CRASH_REQUEST")); err != nil {
		t.Fatal(err)
	}
	os.Exit(23)
}
