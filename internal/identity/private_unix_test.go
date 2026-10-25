//go:build !windows

package identity

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

type ownerFixtureInfo struct {
	os.FileInfo
	owner uint32
}

func (info ownerFixtureInfo) Sys() any { return &syscall.Stat_t{Uid: info.owner} }

func TestPrivateStorageOwnershipAndDatabaseModes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database.db")
	if err := PrepareDatabase(path, true); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkPrivateFile(path, ownerFixtureInfo{FileInfo: info, owner: uint32(os.Geteuid()) + 12345}); err == nil {
		t.Fatal("untrusted owner accepted")
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		if err := os.WriteFile(path+suffix, nil, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path+suffix, 0644); err != nil {
			t.Fatal(err)
		}
		if err := PrepareDatabase(path, false); err == nil {
			t.Fatal("unsafe mode accepted")
		}
		if err := os.Chmod(path+suffix, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
