package identity

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPrivateStorageRejectsUntrustedOwner(t *testing.T) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	if !trustedOwner(user.User.Sid) {
		t.Fatal("current owner rejected")
	}
	fixture, err := windows.SecurityDescriptorFromString("O:S-1-5-21-111111111-222222222-333333333-1001D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;FA;;;SY)(A;;FA;;;BA)")
	if err != nil {
		t.Fatal(err)
	}
	owner, _, err := fixture.Owner()
	if err != nil || trustedOwner(owner) {
		t.Fatal("untrusted owner accepted despite private DACL")
	}
}

func TestDatabaseAndSidecarPermissionsFailClosed(t *testing.T) {
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		t.Run(suffix, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "private")
			if err := PrepareDirectory(directory); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "database.db")
			if err := PrepareDatabase(path, true); err != nil {
				t.Fatal(err)
			}
			if suffix != "" {
				if err := os.WriteFile(path+suffix, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := PrepareDatabase(path, false); err != nil {
				t.Fatal("safe inherited sidecar rejected", err)
			}
			unsafe, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
			if err != nil {
				t.Fatal(err)
			}
			acl, _, err := unsafe.DACL()
			if err != nil {
				t.Fatal(err)
			}
			if err := windows.SetNamedSecurityInfo(path+suffix, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
				t.Fatal(err)
			}
			if err := PrepareDatabase(path, false); err == nil {
				t.Fatal("permissive database file accepted")
			}
		})
	}
}
