package oidc

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
	"qoauth/internal/identity"
)

func TestSigningRotationWithStandardRpAndOpaqueTokens(t *testing.T) {
	test := newProduction(t)
	test.configure()
	oldToken, _ := test.token("web-rp")
	oldKeys := test.handler.keys
	next, err := identity.StageRotation(test.directory, oldKeys, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = test.database.RotateSigningBinding(t.Context(), oldKeys.Binding(), next.Binding()); err != nil {
		t.Fatal(err)
	}
	if err = identity.PublishRotation(test.directory); err != nil {
		t.Fatal(err)
	}
	loaded, err := identity.LoadKeys(test.directory)
	if err != nil {
		t.Fatal(err)
	}
	test.handler, err = New(t.Context(), test.database, loaded, test.handler.config)
	if err != nil {
		t.Fatal(err)
	}
	verifier := test.provider.Verifier(&rp.Config{ClientID: "web-rp"})
	if _, err = verifier.Verify(t.Context(), oldToken.Extra("id_token").(string)); err != nil {
		t.Fatal("rotation broke unexpired old signed token")
	}
	if test.introspect(oldToken.AccessToken, "notes-api")["active"] != true {
		t.Fatal("RSA rotation broke opaque AES token")
	}
	config := test.oauth("web-rp")
	rotated, err := config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: oldToken.RefreshToken}).Token()
	if err != nil {
		t.Fatal("RSA rotation broke refresh")
	}
	newToken := rotated.Extra("id_token").(string)
	parsed, err := jose.ParseSigned(newToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || parsed.Signatures[0].Header.KeyID != next.ID {
		t.Fatal("old key still signing")
	}
	if _, err = verifier.Verify(t.Context(), newToken); err != nil {
		t.Fatal("standard RP did not fetch new signing key")
	}
	keys, err := test.handler.KeySet(t.Context())
	if err != nil || len(keys) != 2 {
		t.Fatal("overlap not published")
	}
	loaded.Previous.Until = time.Now().Add(-time.Second).Unix()
	status, body, _ := test.request(http.MethodGet, "/keys", nil, "")
	var jwks jose.JSONWebKeySet
	if status != 200 || json.Unmarshal([]byte(body), &jwks) != nil || len(jwks.Keys) != 1 || jwks.Keys[0].KeyID != next.ID {
		t.Fatal("running provider retained an expired key", status)
	}
}
