package identity

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallationLockSurvivesContentionAndProcessExit(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	lock, err := LockDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := LockDirectory(directory); err == nil {
		other.Close()
		t.Fatal("second local lock succeeded")
	}
	lock.Close()
	command := exec.Command(os.Args[0], "-test.run=^TestInstallationLockHelper$")
	command.Env = append(os.Environ(), "QOAUTH_TEST_LOCK_DIRECTORY="+directory)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { input.Close(); command.Process.Kill(); command.Wait() })
	if !bufio.NewScanner(output).Scan() {
		t.Fatal("child did not acquire installation lock")
	}
	if other, err := LockDirectory(directory); err == nil {
		other.Close()
		t.Fatal("cross-process lock bypassed")
	}
	input.Close()
	if err = command.Wait(); err != nil {
		t.Fatal("lock child failed")
	}
	lock, err = LockDirectory(directory)
	if err != nil {
		t.Fatal("process exit did not release lock", err)
	}
	lock.Close()
	if _, err = os.Stat(filepath.Join(directory, "installation.lock")); err != nil {
		t.Fatal("rendezvous file was removed")
	}
}

func TestInstallationLockRejectsNonFile(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	if err := PrepareDirectory(directory); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "installation.lock"), 0700); err != nil {
		t.Fatal(err)
	}
	if lock, err := LockDirectory(directory); err == nil {
		lock.Close()
		t.Fatal("directory accepted as installation lock")
	}
}

func TestInstallationLockHelper(t *testing.T) {
	directory := os.Getenv("QOAUTH_TEST_LOCK_DIRECTORY")
	if directory == "" {
		t.Skip("subprocess lock helper")
	}
	lock, err := LockDirectory(directory)
	if err != nil {
		os.Exit(2)
	}
	os.Stdout.WriteString("locked\n")
	var value [1]byte
	os.Stdin.Read(value[:])
	runtime.KeepAlive(lock)
	os.Exit(0)
}
