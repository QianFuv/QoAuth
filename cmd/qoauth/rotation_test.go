package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"qoauth/internal/identity"
	"qoauth/internal/store"
)

func TestSigningRotationResumesAtBothCommitBoundaries(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "staged", true: "binding-committed"}[committed], func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "state")
			var output bytes.Buffer
			if err := run([]string{"init", "--data-dir", directory, "--username", "owner"}, strings.NewReader("synthetic rotation password"), &output); err != nil {
				t.Fatal(err)
			}
			keys, err := identity.LoadKeys(directory)
			if err != nil {
				t.Fatal(err)
			}
			next, err := identity.StageRotation(directory, keys, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if committed {
				database, err := store.Open(t.Context(), filepath.Join(directory, "qoauth.db"))
				if err != nil {
					t.Fatal(err)
				}
				if err = database.RotateSigningBinding(t.Context(), keys.Binding(), next.Binding()); err != nil {
					t.Fatal(err)
				}
				database.Close()
			}
			if err = run([]string{"serve", "--data-dir", directory}, strings.NewReader(""), &output); err == nil || !strings.Contains(err.Error(), "unfinished") {
				t.Fatal("unfinished publication did not block serve")
			}
			if err = run([]string{"rotate-signing", "--data-dir", directory, "--confirm"}, strings.NewReader(""), &output); err != nil {
				t.Fatal(err)
			}
			loaded, err := identity.LoadKeys(directory)
			if err != nil || loaded.ID != next.ID || loaded.Previous.Until != next.Previous.Until || loaded.Encryption != keys.Encryption {
				t.Fatal("resumed rotation changed staged material")
			}
			if err = os.WriteFile(filepath.Join(directory, "probe.bin"), keys.Seal("process-user", "temporary process factor"), 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(os.Args[0], "-test.run=^TestKeyProcessHelper$")
			command.Env = append(os.Environ(), "QOAUTH_TEST_KEY_DIRECTORY="+directory)
			result, err := command.CombinedOutput()
			if err != nil || strings.TrimSpace(string(result)) != next.ID {
				t.Fatal("separate process failed rotated key/database/AES checks")
			}
		})
	}
}

func TestConcurrentSigningCommandsHaveOneSuccess(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state")
	if err := run([]string{"init", "--data-dir", directory, "--username", "owner"}, strings.NewReader("synthetic rotation password"), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var workers sync.WaitGroup
	for range 10 {
		workers.Go(func() {
			if err := run([]string{"rotate-signing", "--data-dir", directory, "--confirm"}, strings.NewReader(""), &bytes.Buffer{}); err == nil {
				successes.Add(1)
			}
		})
	}
	workers.Wait()
	if successes.Load() != 1 {
		t.Fatal("concurrent rotations succeeded", successes.Load())
	}
}
