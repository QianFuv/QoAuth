package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestProductionAuthorizationPolicy(t *testing.T) {
	test := newProduction(t)
	test.configure()
	config := test.oauth("web-rp")
	for _, variant := range []string{"redirect", "scope", "resource", "plain", "missing-pkce", "missing-nonce", "missing-state", "implicit", "request-object"} {
		t.Run(variant, func(t *testing.T) {
			address, _ := url.Parse(config.AuthCodeURL("test-state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "test-nonce")))
			query := address.Query()
			switch variant {
			case "redirect":
				query.Set("redirect_uri", config.RedirectURL+"?changed=true")
			case "scope":
				query.Set("scope", "openid admin")
			case "resource":
				query.Set("resource", "another-api")
			case "plain":
				query.Set("code_challenge_method", "plain")
			case "missing-pkce":
				query.Del("code_challenge")
			case "missing-nonce":
				query.Del("nonce")
			case "missing-state":
				query.Del("state")
			case "implicit":
				query.Set("response_type", "id_token token")
			case "request-object":
				query.Set("request", "unsupported")
			}
			address.RawQuery = query.Encode()
			status, _, headers := test.request("GET", address.String(), nil, "")
			location := headers.Get("Location")
			if status == 200 || strings.Contains(location, "/login?") {
				t.Fatal("invalid authorization admitted")
			}
			if variant == "redirect" && location != "" {
				t.Fatal("unregistered redirect used")
			}
		})
	}
	address := config.AuthCodeURL("none-state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "none-nonce"), oauth2.SetAuthURLParam("prompt", "none"))
	_, _, headers := test.request("GET", address, nil, "")
	if !strings.Contains(headers.Get("Location"), "consent_required") {
		t.Fatal("silent flow did not require consent")
	}
	test.token("web-rp")
	id, _, _, _ := test.begin("web-rp", "none")
	status, _, headers := test.request("GET", "/login?request="+id, nil, "")
	if status != 303 || !strings.Contains(headers.Get("Location"), "/authorize/callback") {
		t.Fatal("eligible silent SSO did not complete")
	}
	test.newBrowser()
	_, _, headers = test.request("GET", address, nil, "")
	if !strings.Contains(headers.Get("Location"), "login_required") {
		t.Fatal("silent anonymous login admitted")
	}
}

func TestProductionCrossBrowserConsentAndLogout(t *testing.T) {
	test := newProduction(t)
	test.configure()
	id, _, _, _ := test.begin("web-rp", "consent")
	originalBrowser, originalCsrf := test.browser, test.csrf
	test.newBrowser()
	test.login("member")
	test.api("/api/consent", map[string]any{"request": id, "approve": true}, 400)
	test.browser, test.csrf = originalBrowser, originalCsrf
	denied := test.api("/api/consent", map[string]any{"request": id, "approve": false}, 200)
	if !strings.Contains(denied["redirect"].(string), "access_denied") {
		t.Fatal("denial did not return standard error")
	}
	test.api("/api/consent", map[string]any{"request": id, "approve": true}, 400)
	token, _ := test.token("web-rp")
	query := url.Values{"id_token_hint": {token.Extra("id_token").(string)}, "post_logout_redirect_uri": {test.server.URL + "/rp/signed-out"}, "state": {"logout-state"}}
	if status, _, _ := test.request("GET", "/end_session?"+query.Encode(), nil, ""); status != 200 {
		t.Fatal("logout confirmation missing")
	}
	if test.introspect(token.AccessToken, "notes-api")["active"] != true {
		t.Fatal("GET logout mutated session")
	}
	if status, _, _ := test.request("POST", "/end_session", strings.NewReader(query.Encode()), "application/x-www-form-urlencoded"); status != 403 {
		t.Fatal("logout missing CSRF accepted")
	}
	query.Set("csrf", test.csrf)
	status, _, headers := test.request("POST", "/end_session", strings.NewReader(query.Encode()), "application/x-www-form-urlencoded")
	if status != 302 || !strings.Contains(headers.Get("Location"), "state=logout-state") {
		t.Fatal("confirmed standard logout failed", status)
	}
	if test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("confirmed logout retained grant")
	}
}

func TestProductionSecurityChangesInvalidateAuthorization(t *testing.T) {
	for _, change := range []string{"disable", "password", "assignment", "consent"} {
		t.Run(change, func(t *testing.T) {
			test := newProduction(t)
			test.configure()
			token, _ := test.token("web-rp")
			code, verifier, _ := test.code("web-rp")
			if change == "consent" {
				test.api("/api/consent/withdraw", map[string]string{"client": "web-rp"}, 200)
			} else {
				memberBrowser, memberCsrf := test.browser, test.csrf
				test.browser, test.csrf = test.adminBrowser, test.adminCsrf
				switch change {
				case "disable":
					test.api("/api/users/disable", map[string]string{"user": test.member}, 200)
				case "password":
					test.api("/api/users/password", map[string]string{"user": test.member, "password": testPassword + " updated"}, 200)
				case "assignment":
					test.api("/api/assignments", map[string]any{"user": test.member, "client": "web-rp", "allow": false}, 200)
					test.api("/api/assignments", map[string]any{"user": test.member, "client": "web-rp", "allow": true}, 200)
				}
				test.browser, test.csrf = memberBrowser, memberCsrf
			}
			config := test.oauth("web-rp")
			if _, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
				t.Fatal("security change retained pending code")
			}
			if _, err := config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token(); err == nil {
				t.Fatal("security change retained refresh")
			}
			if test.introspect(token.AccessToken, "notes-api")["active"] != false {
				t.Fatal("security change retained access")
			}
			if change == "password" || change == "disable" {
				if test.state()["authenticated"] != false {
					t.Fatal("security change retained browser session")
				}
			}
		})
	}
}

func TestProductionErrorsAndActivityDoNotLeakCredentials(t *testing.T) {
	var output bytes.Buffer
	prior := slog.Default()
	slog.SetDefault(SafeLogger(&output))
	t.Cleanup(func() { slog.SetDefault(prior) })
	test := newProduction(t)
	test.configure()
	test.api("/api/login", map[string]string{"username": "member", "password": "wrong-password-secret-canary"}, 401)
	request, _ := http.NewRequestWithContext(context.Background(), "POST", test.server.URL+"/oauth/token", strings.NewReader(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"refresh-secret-canary"}}.Encode()))
	request.SetBasicAuth("web-rp", test.secrets["web-rp"])
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := test.browser.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	events, err := test.database.Activity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	activity, _ := json.Marshal(events)
	for _, canary := range []string{testPassword, "wrong-password-secret-canary", "refresh-secret-canary", test.secrets["web-rp"], test.ownerFactor, test.recovery[0]} {
		if strings.Contains(output.String(), canary) || strings.Contains(string(body), canary) || strings.Contains(string(activity), canary) {
			t.Fatal("credential reached logs/error/activity")
		}
	}
}
