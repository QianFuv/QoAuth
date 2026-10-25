package identity

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type interruptedCandidate struct{ wrote bool }

func (reader *interruptedCandidate) Read(buffer []byte) (int, error) {
	if reader.wrote {
		os.Exit(80)
	}
	reader.wrote = true
	return copy(buffer, []byte(`{"Version":2,`)), nil
}

func TestCandidateProcessHelper(t *testing.T) {
	directory := os.Getenv("QOAUTH_CANDIDATE_DIRECTORY")
	if directory == "" {
		t.Skip("subprocess helper")
	}
	if os.Getenv("QOAUTH_CANDIDATE_PHASE") == "writing" {
		_ = writeCandidate(directory, &interruptedCandidate{})
		os.Exit(2)
	}
	current, err := LoadKeys(directory)
	if err != nil {
		os.Exit(3)
	}
	if _, err = StageRotation(directory, current, time.Now()); err != nil {
		os.Exit(4)
	}
	os.Exit(80)
}

func TestInterruptedCandidateCreationKeepsCurrentInstallationRecoverable(t *testing.T) {
	for _, phase := range []string{"writing", "published"} {
		t.Run(phase, func(t *testing.T) {
			directory := t.TempDir()
			current, err := CreateKeys(directory)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(filepath.Join(directory, "keys.json"))
			command := exec.Command(os.Args[0], "-test.run=^TestCandidateProcessHelper$")
			command.Env = append(os.Environ(), "QOAUTH_CANDIDATE_DIRECTORY="+directory, "QOAUTH_CANDIDATE_PHASE="+phase)
			if err = command.Run(); err == nil || command.ProcessState.ExitCode() != 80 {
				t.Fatal("child did not terminate at requested publication boundary", err)
			}
			if phase == "writing" && HasPendingRotation(directory) {
				t.Fatal("partial write published an unusable rotation candidate")
			}
			published, _ := os.ReadFile(filepath.Join(directory, "keys.next.json"))
			next, err := StageRotation(directory, current, time.Now())
			if err != nil {
				t.Fatal("interrupted staging stranded intact installation", err)
			}
			again, err := StageRotation(directory, current, time.Now().Add(time.Minute))
			if err != nil || next.ID != again.ID || next.Previous.Until != again.Previous.Until {
				t.Fatal("completed candidate changed on retry", err)
			}
			resumed, _ := os.ReadFile(filepath.Join(directory, "keys.next.json"))
			if phase == "published" && !bytes.Equal(published, resumed) {
				t.Fatal("published candidate bytes changed after process exit")
			}
			if _, err = os.Stat(filepath.Join(directory, "keys.staging.json")); !os.IsNotExist(err) {
				t.Fatal("unpublished staging bytes were not reclaimed")
			}
			after, _ := os.ReadFile(filepath.Join(directory, "keys.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("staging modified current operational keys")
			}
		})
	}
}

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
