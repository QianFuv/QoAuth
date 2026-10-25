package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"qoauth/internal/identity"
	"qoauth/internal/store"
)

func TestCommandInitializationRecoveryAndNewProcess(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state")
	password := "synthetic CLI credential canary"
	var output bytes.Buffer
	if err := run([]string{"init", "--data-dir", directory, "--username", "owner"}, strings.NewReader(password), &output); err != nil {
		t.Fatal(err)
	}
	keys, err := identity.LoadKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"init", "--data-dir", directory, "--username", "attacker"}, strings.NewReader(password), &output); err == nil {
		t.Fatal("ownership reopened")
	}
	lock, err := identity.LockDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"serve", "recover-owner", "init"} {
		if err = run([]string{command, "--data-dir", directory, "--confirm", "--username", "owner"}, strings.NewReader(password), &output); err == nil || !strings.Contains(err.Error(), "busy") {
			t.Fatal("command bypassed installation lock", command, err)
		}
	}
	lock.Close()
	if err = run([]string{"recover-owner", "--data-dir", directory}, strings.NewReader(password), &output); err == nil {
		t.Fatal("recovery did not require explicit confirmation")
	}
	if err = run([]string{"recover-owner", "--data-dir", directory, "--confirm"}, strings.NewReader(password+" changed"), &output); err != nil {
		t.Fatal(err)
	}
	sealed := keys.Seal("process-user", "temporary process factor")
	if err = os.WriteFile(filepath.Join(directory, "probe.bin"), sealed, 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=TestKeyProcessHelper")
	command.Env = append(os.Environ(), "QOAUTH_TEST_KEY_DIRECTORY="+directory)
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatal("key restart subprocess failed")
	}
	if strings.TrimSpace(string(result)) != keys.ID {
		t.Fatal("new process did not retain signing identity")
	}
	if strings.Contains(output.String(), password) || strings.Contains(string(result), "temporary process factor") {
		t.Fatal("credential exposed by command")
	}
	databasePath := filepath.Join(directory, "qoauth.db")
	if err = os.Rename(databasePath, databasePath+".saved"); err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"init", "--data-dir", directory, "--username", "replacement"}, strings.NewReader(password), &output); err == nil {
		t.Fatal("missing database reinitialized under old keys")
	}
	if _, err = os.Stat(databasePath); !os.IsNotExist(err) {
		t.Fatal("replacement database created")
	}
	if err = os.Rename(databasePath+".saved", databasePath); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(directory, "keys.json")); err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"init", "--data-dir", directory, "--username", "owner"}, strings.NewReader(password), &output); err == nil {
		t.Fatal("missing initialized keys regenerated")
	}
	if _, err = os.Stat(filepath.Join(directory, "keys.json")); !os.IsNotExist(err) {
		t.Fatal("key file recreated despite initialized marker")
	}
}

func TestKeyProcessHelper(t *testing.T) {
	directory := os.Getenv("QOAUTH_TEST_KEY_DIRECTORY")
	if directory == "" {
		t.Skip("subprocess helper")
	}
	keys, err := identity.LoadKeys(directory)
	if err != nil {
		os.Exit(2)
	}
	sealed, err := os.ReadFile(filepath.Join(directory, "probe.bin"))
	if err != nil {
		os.Exit(3)
	}
	if plain, err := keys.Open("process-user", sealed); err != nil || plain != "temporary process factor" {
		os.Exit(4)
	}
	database, err := store.Open(context.Background(), filepath.Join(directory, "qoauth.db"))
	if err != nil {
		os.Exit(5)
	}
	if err = database.CheckInstallation(context.Background(), keys.Binding()); err != nil {
		os.Exit(6)
	}
	database.Close()
	os.Stdout.WriteString(keys.ID)
	os.Exit(0)
}
