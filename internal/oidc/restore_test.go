package oidc

import (
	"crypto"
	"path/filepath"
	"strings"
	"testing"
	"time"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"qoauth/internal/identity"
	"qoauth/internal/maintenance"
	"qoauth/internal/store"
)

func TestRestoredSigningTrustAndCachedRpExpiryBoundary(t *testing.T) {
	test := newProduction(t)
	test.configure()
	token, _ := test.token("web-rp")
	pendingCode, _, _ := test.code("web-rp")
	if _, err := test.database.RequestByCode(t.Context(), pendingCode); err != nil {
		t.Fatal("test code was not live before snapshot")
	}
	plainAccess, err := test.handler.provider.Crypto().Decrypt(token.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	accessHandle, _, ok := strings.Cut(plainAccess, ":")
	if !ok {
		t.Fatal("unexpected opaque access format")
	}
	for _, handle := range []string{accessHandle, token.RefreshToken} {
		if _, err := test.database.Lookup(t.Context(), handle); err != nil {
			t.Fatal("test token was not live before snapshot")
		}
	}
	oldJwt := token.Extra("id_token").(string)
	warm := test.provider.Verifier(&rp.Config{ClientID: "web-rp"})
	if _, err := warm.Verify(t.Context(), oldJwt); err != nil {
		t.Fatal(err)
	}
	config := test.handler.config
	test.server.Close()
	test.database.Close()
	lock, err := identity.LockDirectory(test.directory)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	snapshot := filepath.Join(root, "snapshot")
	if err = maintenance.Backup(t.Context(), test.directory, snapshot); err != nil {
		t.Fatal(err)
	}
	lock.Close()
	destination := filepath.Join(root, "restored")
	if err = maintenance.Restore(t.Context(), snapshot, destination); err != nil {
		t.Fatal(err)
	}
	keys, err := identity.LoadKeys(destination)
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(t.Context(), filepath.Join(destination, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err = New(t.Context(), database, keys, config); err == nil {
		t.Fatal("OP served before restore review")
	}
	if err = database.RecoverRestoredOwner(t.Context(), "new-synthetic-verifier"); err != nil {
		t.Fatal(err)
	}
	if _, err = New(t.Context(), database, keys, config); err == nil {
		t.Fatal("OP served before signed-token quarantine expired")
	}
	if _, err = database.RequestByCode(t.Context(), pendingCode); err == nil {
		t.Fatal("old authorization code survived restore")
	}
	for _, handle := range []string{accessHandle, token.RefreshToken} {
		if _, err = database.Lookup(t.Context(), handle); err == nil {
			t.Fatal("old token state survived restore")
		}
	}
	public := &rp.StaticKeySet{PublicKeys: []crypto.PublicKey{&keys.Signing.PublicKey}}
	cold := rp.NewVerifier(config.Issuer, public, &rp.Config{ClientID: "web-rp", SupportedSigningAlgs: []string{"RS256"}})
	if _, err = cold.Verify(t.Context(), oldJwt); err == nil {
		t.Fatal("restored key accepted old signed token")
	}
	if _, err = warm.Verify(t.Context(), oldJwt); err != nil {
		t.Fatal("test failed to demonstrate cached RP trust")
	}
	afterWindow := test.provider.Verifier(&rp.Config{ClientID: "web-rp", Now: func() time.Time { return time.Now().Add(identity.VerificationWindow) }})
	if _, err = afterWindow.Verify(t.Context(), oldJwt); err == nil {
		t.Fatal("cached RP accepted expired old token after quarantine")
	}
}

func TestDisabledClientReplacementNeedsNewCredentialsAndAssignments(t *testing.T) {
	test := newProduction(t)
	test.configure()
	token, _ := test.token("web-rp")
	memberBrowser, memberCsrf := test.browser, test.csrf
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	app := store.Application{ID: "web-rp", Name: "Reviewed", Redirects: []string{test.server.URL + "/rp/callback"}, LogoutRedirects: []string{test.server.URL + "/rp/signed-out"}, Scopes: []string{"openid", "profile", "offline_access", "notes.read"}, Audiences: []string{"notes-api"}}
	test.api("/api/applications/replace-disabled", map[string]any{"application": app}, 400)
	test.api("/api/applications/disable", map[string]any{"client": app.ID}, 200)
	result := test.api("/api/applications/replace-disabled", map[string]any{"application": app}, 200)
	secret := result["client_secret"].(string)
	if secret == "" || secret == test.secrets[app.ID] {
		t.Fatal("replacement reused old secret")
	}
	if err := test.handler.AuthorizeClientIDSecret(t.Context(), app.ID, test.secrets[app.ID]); err == nil {
		t.Fatal("old client credential survived replacement")
	}
	test.api("/api/assignments", map[string]any{"user": test.member, "client": app.ID, "allow": true}, 200)
	test.secrets[app.ID] = secret
	test.browser, test.csrf = memberBrowser, memberCsrf
	if test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("replacement and regrant resurrected old access")
	}
	test.token(app.ID)
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	resource := store.Application{ID: "notes-api", Name: "Reviewed API", Resource: true, Audiences: []string{"notes-api"}, Scopes: []string{"notes.new"}}
	test.api("/api/applications/disable", map[string]any{"client": resource.ID}, 200)
	test.api("/api/applications/replace-disabled", map[string]any{"application": resource}, 400)
	for _, id := range []string{"web-rp", "second-rp"} {
		test.api("/api/applications/disable", map[string]any{"client": id}, 200)
	}
	test.api("/api/applications/replace-disabled", map[string]any{"application": resource}, 200)
	test.api("/api/applications/replace-disabled", map[string]any{"application": app}, 400)
}
