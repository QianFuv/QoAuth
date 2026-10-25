package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"qoauth/internal/identity"
)

func TestMaintenanceCommandsAndIncompleteDirectoryGuard(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	snapshot := filepath.Join(root, "snapshot")
	destination := filepath.Join(root, "restored")
	var output bytes.Buffer
	password := "synthetic maintenance credential"
	if err := run([]string{"init", "--data-dir", source, "--username", "owner"}, strings.NewReader(password), &output); err != nil {
		t.Fatal(err)
	}
	lock, err := identity.LockDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"backup", "--data-dir", source, "--snapshot", snapshot}, strings.NewReader(""), &output); err == nil {
		t.Fatal("backup bypassed running-installation lock")
	}
	lock.Close()
	if err = run([]string{"backup", "--data-dir", source, "--snapshot", snapshot}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"restore", "--data-dir", destination, "--snapshot", snapshot}, strings.NewReader(""), &output); err == nil {
		t.Fatal("restore did not require stopped-issuer confirmation")
	}
	if err = run([]string{"restore", "--data-dir", destination, "--snapshot", snapshot, "--confirm"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"recover-owner", "--data-dir", destination, "--confirm"}, strings.NewReader(password), &output); err == nil {
		t.Fatal("restored owner bypassed rollback review")
	}
	if err = run([]string{"recover-owner", "--data-dir", destination, "--confirm", "--restore-reviewed"}, strings.NewReader(password+" new"), &output); err != nil {
		t.Fatal(err)
	}
	if err = run([]string{"serve", "--data-dir", destination, "--issuer", "https://auth.example.test"}, strings.NewReader(""), &output); err == nil || !strings.Contains(err.Error(), "quarantine") {
		t.Fatal("CLI bypassed persistent quarantine", err)
	}
	if err = identity.WritePrivate(filepath.Join(source, "operation.incomplete"), strings.NewReader("partial")); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"init", "serve", "recover-owner", "rotate-signing", "backup"} {
		if err = run([]string{command, "--data-dir", source, "--username", "owner", "--confirm"}, strings.NewReader(password), &output); err == nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatal("command bypassed incomplete publication", command, err)
		}
	}
	if strings.Contains(output.String(), password) {
		t.Fatal("CLI leaked password")
	}
	if _, err = os.Stat(filepath.Join(destination, "keys.next.json")); !os.IsNotExist(err) {
		t.Fatal("restore inherited ordinary rotation overlap")
	}
}

func TestShutdownKeepsInstallationLockedUntilHandlersFinish(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state")
	lock, err := identity.LockDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	server := &http.Server{Addr: address, Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		close(entered)
		<-release
		response.WriteHeader(200)
	})}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { err := serveHTTP(ctx, server); lock.Close(); done <- err }()
	go func() {
		for range 100 {
			response, err := http.Get("http://" + address)
			if err == nil {
				response.Body.Close()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("test listener unavailable")
	}
	cancel()
	select {
	case <-done:
		close(release)
		t.Fatal("shutdown returned with a live handler")
	case <-time.After(30 * time.Millisecond):
	}
	if other, err := identity.LockDirectory(directory); err == nil {
		other.Close()
		close(release)
		t.Fatal("shutdown released lock early")
	}
	close(release)
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler drain did not finish")
	}
	other, err := identity.LockDirectory(directory)
	if err != nil {
		t.Fatal("drained server retained lock", err)
	}
	other.Close()
}
