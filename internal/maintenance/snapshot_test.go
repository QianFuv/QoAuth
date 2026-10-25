package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"qoauth/internal/identity"
	"qoauth/internal/store"
)

func snapshotFixture(t *testing.T) (string, string, string, *identity.Keys) {
	t.Helper()
	root := t.TempDir()
	directory := filepath.Join(root, "source")
	lock, err := identity.LockDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	keys, err := identity.CreateKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = identity.PrepareDatabase(filepath.Join(directory, "qoauth.db"), true); err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(t.Context(), filepath.Join(directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Bootstrap(t.Context(), "owner", "synthetic-verifier", keys.Binding()); err != nil {
		t.Fatal(err)
	}
	if err = database.CheckServing(t.Context(), keys.Binding(), "https://auth.example.test", time.Now()); err != nil {
		t.Fatal(err)
	}
	database.Close()
	snapshot := filepath.Join(root, "snapshot")
	if err = Backup(t.Context(), directory, snapshot); err != nil {
		t.Fatal(err)
	}
	return directory, snapshot, filepath.Join(root, "restored"), keys
}

func TestSnapshotRestoreChangesSigningTrustAndPreservesSource(t *testing.T) {
	directory, snapshot, destination, oldKeys := snapshotFixture(t)
	before, err := fileDigest(filepath.Join(directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err = Restore(t.Context(), snapshot, destination); err != nil {
		t.Fatal(err)
	}
	keys, err := identity.LoadKeys(destination)
	if err != nil || keys.ID == oldKeys.ID || keys.Previous != nil || keys.Encryption != oldKeys.Encryption {
		t.Fatal("restore retained old signing trust or changed AES", err)
	}
	database, err := store.Open(t.Context(), filepath.Join(destination, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	state, err := database.InstallationState(t.Context())
	if err != nil || !state.Maintenance || state.ReadyAfter < time.Now().Add(9*time.Minute).Unix() || state.Issuer != "https://auth.example.test" {
		t.Fatal("restore did not persist review/quarantine/issuer")
	}
	if err = database.CheckInstallation(t.Context(), keys.Binding()); err != nil {
		t.Fatal("restored key binding failed")
	}
	after, err := fileDigest(filepath.Join(directory, "qoauth.db"))
	if err != nil || before != after {
		t.Fatal("restore modified original installation")
	}
	if err = CheckOperationalDirectory(snapshot); err == nil {
		t.Fatal("snapshot allowed as live installation")
	}
	if err = Restore(t.Context(), snapshot, destination); err == nil {
		t.Fatal("existing destination overwritten")
	}
}

func TestSnapshotRejectsTamperingSidecarsAndMissingMembers(t *testing.T) {
	for _, mode := range []string{"modified-database", "modified-keys", "missing-key", "partial-manifest", "sidecar", "unknown-member"} {
		t.Run(mode, func(t *testing.T) {
			_, snapshot, destination, _ := snapshotFixture(t)
			var err error
			switch mode {
			case "modified-database":
				err = os.WriteFile(filepath.Join(snapshot, "qoauth.db"), []byte("corrupt"), 0600)
			case "modified-keys":
				err = os.WriteFile(filepath.Join(snapshot, "keys.json"), []byte("corrupt"), 0600)
			case "missing-key":
				err = os.Remove(filepath.Join(snapshot, "keys.json"))
			case "partial-manifest":
				err = os.WriteFile(filepath.Join(snapshot, "manifest.json"), []byte(`{"version":`), 0600)
			case "sidecar":
				err = identity.WritePrivate(filepath.Join(snapshot, "qoauth.db-wal"), bytes.NewReader(nil))
			case "unknown-member":
				err = identity.WritePrivate(filepath.Join(snapshot, "extra"), bytes.NewReader(nil))
			}
			if err != nil {
				t.Fatal(err)
			}
			if err = Restore(t.Context(), snapshot, destination); err == nil {
				t.Fatal("unsafe snapshot accepted")
			}
			if _, err = os.Stat(destination); !os.IsNotExist(err) {
				t.Fatal("invalid snapshot created destination state")
			}
		})
	}
}

func TestRehashedMismatchedKeysStillFailInstallationBinding(t *testing.T) {
	_, snapshot, destination, keys := snapshotFixture(t)
	other, err := identity.FreshRestoreKeys(keys)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(snapshot, "keys.json")); err != nil {
		t.Fatal(err)
	}
	if err = identity.WriteKeys(snapshot, other); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(snapshot, "manifest.json"))
	var metadata manifest
	if err = json.Unmarshal(content, &metadata); err != nil {
		t.Fatal(err)
	}
	metadata.Keys, err = fileDigest(filepath.Join(snapshot, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	content, _ = json.Marshal(metadata)
	if err = os.WriteFile(filepath.Join(snapshot, "manifest.json"), content, 0600); err != nil {
		t.Fatal(err)
	}
	if err = Restore(t.Context(), snapshot, destination); err == nil {
		t.Fatal("hash-valid but mismatched key snapshot accepted")
	}
}

func TestIncompleteOperationNeverBecomesOperational(t *testing.T) {
	directory, snapshot, destination, _ := snapshotFixture(t)
	lock, err := newDestination(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	for _, name := range []string{"qoauth.db", "keys.json"} {
		if err = copyPrivate(filepath.Join(directory, name), filepath.Join(destination, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err = CheckOperationalDirectory(destination); err == nil {
		t.Fatal("matching copied pair bypassed incomplete marker")
	}
	if err = Restore(t.Context(), snapshot, destination); err == nil {
		t.Fatal("partial destination reused")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = Backup(ctx, directory, filepath.Join(t.TempDir(), "failed")); err == nil {
		t.Fatal("cancelled snapshot succeeded")
	}
}
