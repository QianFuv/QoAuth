package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSigningRotationRetainsOnlyPublicOverlap(t *testing.T) {
	directory := t.TempDir()
	current, err := CreateKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	next, err := StageRotation(directory, current, now)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := StageRotation(directory, current, now.Add(time.Minute))
	if err != nil || next.ID != resumed.ID || next.Previous.Until != resumed.Previous.Until {
		t.Fatal("retry changed the key or retirement deadline")
	}
	if next.Encryption != current.Encryption || next.ID == current.ID || next.Previous.ID != current.ID {
		t.Fatal("rotation changed AES or lost public overlap")
	}
	if err = PublishRotation(directory); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadKeys(directory)
	if err != nil || reloaded.ID != next.ID || reloaded.Previous.Until != next.Previous.Until {
		t.Fatal("rotation did not persist")
	}
	if _, err = StageRotation(directory, reloaded, now.Add(time.Minute)); err == nil {
		t.Fatal("unbounded key overlap allowed")
	}
	if _, err = StageRotation(directory, reloaded, now.Add(VerificationWindow+time.Second)); err != nil {
		t.Fatal("retired public key blocked future rotation", err)
	}
	if err = os.WriteFile(filepath.Join(directory, "keys.next.json"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = StageRotation(directory, reloaded, now.Add(2*VerificationWindow)); err == nil {
		t.Fatal("partial key file silently replaced")
	}
	content, _ := os.ReadFile(filepath.Join(directory, "keys.next.json"))
	if string(content) != "partial" {
		t.Fatal("failed rotation changed partial evidence")
	}
}

func TestKeyPublicationFailurePreservesCandidate(t *testing.T) {
	directory := t.TempDir()
	current, err := CreateKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	next, err := StageRotation(directory, current, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "keys.json")
	if err = os.Rename(path, path+".saved"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err = PublishRotation(directory); err == nil {
		t.Fatal("publication overwrote a non-file")
	}
	staged, err := loadKeys(filepath.Join(directory, "keys.next.json"))
	if err != nil || staged.ID != next.ID {
		t.Fatal("failed publication lost the recoverable key")
	}
	if saved, err := loadKeys(path + ".saved"); err != nil || saved.ID != current.ID {
		t.Fatal("failed publication changed prior keys")
	}
}
