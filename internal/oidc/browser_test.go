package oidc

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"qoauth/internal/store"
)

// TestBrowserHarness serves only temporary synthetic state for a bounded manual check.
func TestBrowserHarness(t *testing.T) {
	statePath, binary := os.Getenv("QOAUTH_BROWSER_STATE"), os.Getenv("QOAUTH_BROWSER_RP")
	if statePath == "" || binary == "" {
		t.Skip("explicit local browser harness only")
	}
	test := newProduction(t)
	test.configure()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	origin := "http://" + address
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	app := store.Application{ID: "browser-rp", Name: "Standard Go RP", Redirects: []string{origin + "/callback"}, LogoutRedirects: []string{origin + "/signed-out"}, Scopes: []string{"openid", "profile", "offline_access"}}
	secret := test.api("/api/applications/create", map[string]any{"application": app}, 200)["client_secret"].(string)
	test.api("/api/assignments", map[string]any{"user": test.member, "client": app.ID, "allow": true}, 200)
	review := store.Application{ID: "new", Name: "Comma callbacks (synthetic)", Public: true, Redirects: []string{"https://rp.example.test/callback?next=one,https://other.example.test/callback"}, LogoutRedirects: []string{"https://rp.example.test/signed-out?next=one,https://other.example.test/signed-out"}, Scopes: []string{"openid"}, Audiences: []string{}}
	test.api("/api/applications/create", map[string]any{"application": review}, 200)
	test.api("/api/applications/disable", map[string]string{"client": review.ID}, 200)
	secretPath := filepath.Join(test.directory, "rp-secret")
	if err := os.WriteFile(secretPath, []byte(secret), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "--issuer", test.server.URL, "--client-id", app.ID, "--client-secret-file", secretPath, "--callback", origin+"/callback", "--listen", address, "--development-http")
	command.Stdout, command.Stderr = io.Discard, io.Discard
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	ready := false
	for range 50 {
		if response, err := http.Get(origin); err == nil {
			response.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("temporary Go RP did not start")
	}
	writeState := func() {
		code, err := totp.GenerateCode(test.ownerFactor, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(map[string]string{"issuer": test.server.URL, "rp": origin, "username": "member", "password": testPassword, "owner_code": code})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(statePath, payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeState()
	t.Cleanup(func() { _ = os.Remove(statePath); _ = os.Remove(statePath + ".stop") })
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(statePath + ".stop"); err == nil {
			return
		}
		writeState()
		time.Sleep(time.Second)
	}
	t.Fatal("browser harness reached its ten-minute bound")
}
