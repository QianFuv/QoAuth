package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"github.com/pquerna/otp/totp"
	"golang.org/x/oauth2"
	"qoauth/internal/identity"
	"qoauth/internal/store"
)

const testPassword = "isolated synthetic password canary"

type productionHarness struct {
	t            *testing.T
	directory    string
	database     *store.Store
	server       *httptest.Server
	handler      *Server
	browser      *http.Client
	provider     *rp.Provider
	csrf         string
	secrets      map[string]string
	member       string
	ownerFactor  string
	recovery     []string
	adminBrowser *http.Client
	adminCsrf    string
}

func newProduction(t *testing.T) *productionHarness {
	t.Helper()
	test := &productionHarness{t: t, directory: filepath.Join(t.TempDir(), "private"), secrets: map[string]string{}}
	if err := identity.PrepareDirectory(test.directory); err != nil {
		t.Fatal(err)
	}
	keys, err := identity.CreateKeys(test.directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = identity.PrepareDatabase(filepath.Join(test.directory, "qoauth.db"), true); err != nil {
		t.Fatal(err)
	}
	test.database, err = store.Open(t.Context(), filepath.Join(test.directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := identity.PasswordHash(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err = test.database.Bootstrap(t.Context(), "owner", encoded, keys.Binding()); err != nil {
		t.Fatal(err)
	}
	test.server = httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) { test.handler.ServeHTTP(response, request) }))
	issuer := "http://" + test.server.Listener.Addr().String()
	test.handler, err = New(t.Context(), test.database, keys, Config{Issuer: issuer, DevelopmentHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	test.server.Start()
	t.Cleanup(func() { test.server.Close(); test.database.Close() })
	test.newBrowser()
	test.provider, err = rp.NewProvider(t.Context(), issuer)
	if err != nil {
		t.Fatal(err)
	}
	return test
}
func (test *productionHarness) newBrowser() {
	jar, _ := cookiejar.New(nil)
	test.browser = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	test.state()
}
func (test *productionHarness) request(method, path string, body io.Reader, contentType string) (int, string, http.Header) {
	test.t.Helper()
	if strings.HasPrefix(path, "/") {
		path = test.server.URL + path
	}
	request, err := http.NewRequestWithContext(test.t.Context(), method, path, body)
	if err != nil {
		test.t.Fatal(err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
		request.Header.Set("Origin", test.server.URL)
		request.Header.Set("X-CSRF-Token", test.csrf)
	}
	response, err := test.browser.Do(request)
	if err != nil {
		test.t.Fatal(err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		test.t.Fatal(err)
	}
	return response.StatusCode, string(content), response.Header
}
func (test *productionHarness) api(path string, input any, wanted int) map[string]any {
	test.t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		test.t.Fatal(err)
	}
	status, body, _ := test.request(http.MethodPost, path, bytes.NewReader(payload), "application/json")
	if status != wanted {
		test.t.Fatalf("%s: status %d expected %d (response withheld)", path, status, wanted)
	}
	result := map[string]any{}
	if status == 200 && json.Unmarshal([]byte(body), &result) != nil {
		test.t.Fatal("invalid JSON")
	}
	return result
}
func (test *productionHarness) state() map[string]any {
	test.t.Helper()
	status, body, _ := test.request(http.MethodGet, "/api/session", nil, "")
	if status != 200 {
		test.t.Fatal("session state unavailable", status)
	}
	result := map[string]any{}
	if json.Unmarshal([]byte(body), &result) != nil {
		test.t.Fatal("bad state")
	}
	test.csrf = result["csrf"].(string)
	return result
}
func (test *productionHarness) login(username string) {
	test.t.Helper()
	test.state()
	test.api("/api/login", map[string]string{"username": username, "password": testPassword}, 200)
	test.state()
}
func (test *productionHarness) owner() {
	test.t.Helper()
	test.login("owner")
	if status, _, _ := test.request("GET", "/api/users", nil, ""); status != 403 {
		test.t.Fatal("unenrolled admin admitted")
	}
	setup := test.api("/api/mfa/start", map[string]any{}, 200)
	test.ownerFactor = setup["secret"].(string)
	code, err := totp.GenerateCode(test.ownerFactor, time.Now().Add(-30*time.Second))
	if err != nil {
		test.t.Fatal(err)
	}
	result := test.api("/api/mfa/confirm", map[string]string{"code": code}, 200)
	for _, value := range result["recovery_codes"].([]any) {
		test.recovery = append(test.recovery, value.(string))
	}
	test.state()
}
func (test *productionHarness) configure() {
	test.t.Helper()
	test.owner()
	created := test.api("/api/users/create", map[string]string{"username": "member", "password": testPassword}, 200)
	test.member = created["id"].(string)
	resource := store.Application{ID: "notes-api", Name: "Notes API", Resource: true, Audiences: []string{"notes-api"}, Scopes: []string{"notes.read"}}
	test.secrets[resource.ID] = test.api("/api/applications/create", map[string]any{"application": resource}, 200)["client_secret"].(string)
	for _, id := range []string{"web-rp", "second-rp"} {
		app := store.Application{ID: id, Name: id, Redirects: []string{test.server.URL + "/rp/callback"}, LogoutRedirects: []string{test.server.URL + "/rp/signed-out"}, Scopes: []string{"openid", "profile", "offline_access", "notes.read"}, Audiences: []string{"notes-api"}}
		test.secrets[id] = test.api("/api/applications/create", map[string]any{"application": app}, 200)["client_secret"].(string)
		test.api("/api/assignments", map[string]any{"user": test.member, "client": id, "allow": true}, 200)
	}
	test.adminBrowser = test.browser
	test.adminCsrf = test.csrf
	test.newBrowser()
	test.login("member")
}
func (test *productionHarness) oauth(client string) oauth2.Config {
	endpoint := test.provider.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInHeader
	return oauth2.Config{ClientID: client, ClientSecret: test.secrets[client], RedirectURL: test.server.URL + "/rp/callback", Endpoint: endpoint, Scopes: []string{"openid", "profile", "offline_access", "notes.read"}}
}
func (test *productionHarness) begin(client, prompt string) (string, string, string, string) {
	test.t.Helper()
	config := test.oauth(client)
	state, nonce, verifier := store.NewHandle(), store.NewHandle(), oauth2.GenerateVerifier()
	options := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce)}
	if prompt != "" {
		options = append(options, oauth2.SetAuthURLParam("prompt", prompt))
	}
	status, _, headers := test.request("GET", config.AuthCodeURL(state, options...), nil, "")
	if status != 302 {
		test.t.Fatal("authorization failed", status)
	}
	location := headers.Get("Location")
	parsed, _ := url.Parse(location)
	return parsed.Query().Get("request"), verifier, nonce, state
}
func (test *productionHarness) code(client string) (string, string, string) {
	test.t.Helper()
	id, verifier, nonce, state := test.begin(client, "consent")
	if status, body, _ := test.request("GET", "/login?request="+id, nil, ""); status != 200 || !strings.Contains(body, "/assets/") {
		test.t.Fatal("real embedded login missing")
	}
	result := test.api("/api/consent", map[string]any{"request": id, "approve": true}, 200)
	status, _, headers := test.request("GET", result["redirect"].(string), nil, "")
	if status != 302 {
		test.t.Fatal("OP callback failed", status)
	}
	callback, _ := url.Parse(headers.Get("Location"))
	if callback.Query().Get("state") != state || callback.Query().Get("code") == "" {
		test.t.Fatal("RP callback binding failed")
	}
	return callback.Query().Get("code"), verifier, nonce
}
func (test *productionHarness) token(client string) (*oauth2.Token, *rp.IDToken) {
	test.t.Helper()
	code, verifier, nonce := test.code(client)
	config := test.oauth(client)
	token, err := config.Exchange(test.t.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		test.t.Fatal("standard token exchange failed", err)
	}
	verified, err := test.provider.Verifier(&rp.Config{ClientID: client, SupportedSigningAlgs: []string{"RS256"}}).Verify(test.t.Context(), token.Extra("id_token").(string))
	if err != nil || verified.Nonce != nonce || verified.Subject != test.member {
		test.t.Fatal("ID token verification failed")
	}
	return token, verified
}
func (test *productionHarness) introspect(token, caller string) map[string]any {
	test.t.Helper()
	request, _ := http.NewRequestWithContext(test.t.Context(), "POST", test.server.URL+"/oauth/introspect", strings.NewReader(url.Values{"token": {token}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(caller, test.secrets[caller])
	response, err := test.browser.Do(request)
	if err != nil {
		test.t.Fatal(err)
	}
	defer response.Body.Close()
	var result map[string]any
	if response.StatusCode != 200 || json.NewDecoder(response.Body).Decode(&result) != nil {
		test.t.Fatal("introspection failed")
	}
	return result
}

func TestProductionStandardFlowAndPersistentKeys(t *testing.T) {
	test := newProduction(t)
	test.configure()
	token, verified := test.token("web-rp")
	info, err := test.provider.UserInfo(t.Context(), oauth2.StaticTokenSource(token))
	if err != nil || info.Subject != test.member {
		t.Fatal("UserInfo failed")
	}
	active := test.introspect(token.AccessToken, "notes-api")
	if active["active"] != true || active["aud"].([]any)[0] != "notes-api" {
		t.Fatal("resource audience failed")
	}
	if denied := test.introspect(token.AccessToken, "web-rp"); denied["active"] != false || len(denied) != 1 {
		t.Fatal("RP received resource introspection")
	}
	_, second := test.token("second-rp")
	var firstClaims, secondClaims struct {
		AuthTime int64 `json:"auth_time"`
	}
	verified.Claims(&firstClaims)
	second.Claims(&secondClaims)
	if firstClaims.AuthTime != secondClaims.AuthTime {
		t.Fatal("SSO consent changed authentication time")
	}
	test.database.Close()
	test.database, err = store.Open(t.Context(), filepath.Join(test.directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.LoadKeys(test.directory)
	if err != nil {
		t.Fatal(err)
	}
	test.handler, err = New(t.Context(), test.database, keys, test.handler.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = test.provider.Verifier(&rp.Config{ClientID: "web-rp"}).Verify(t.Context(), token.Extra("id_token").(string)); err != nil {
		t.Fatal("persisted key no longer verifies")
	}
	config := test.oauth("web-rp")
	rotated, err := config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token()
	if err != nil {
		t.Fatal("persisted refresh failed")
	}
	test.api("/api/logout", map[string]any{}, 200)
	if test.introspect(rotated.AccessToken, "notes-api")["active"] != false {
		t.Fatal("session logout retained grant")
	}
	if _, err = config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: rotated.RefreshToken}).Token(); err == nil {
		t.Fatal("logged-out refresh accepted")
	}
}

func TestProductionCsrfCookiesAndRecoveryRestrictions(t *testing.T) {
	test := newProduction(t)
	for _, origin := range []string{"", "null", "https://attacker.example"} {
		request, _ := http.NewRequest("POST", test.server.URL+"/api/login", strings.NewReader(`{"username":"owner","password":"`+testPassword+`"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", origin)
		request.Header.Set("X-CSRF-Token", test.csrf)
		response, err := test.browser.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 403 {
			t.Fatal("foreign/missing origin accepted")
		}
	}
	test.csrf = "wrong"
	test.api("/api/login", map[string]string{"username": "owner", "password": testPassword}, 403)
	test.state()
	server := &Server{config: Config{Issuer: "https://auth.example.test"}}
	recorder := httptest.NewRecorder()
	server.setCookie(recorder, "synthetic-handle")
	cookie := recorder.Result().Cookies()[0]
	if cookie.Name != "__Host-qoauth" || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" {
		t.Fatal("unsafe production cookie")
	}
	test.owner()
	oldCookie := test.browser.Jar.Cookies(mustURL(test.server.URL))[0].Value
	test.newBrowser()
	test.login("owner")
	test.api("/api/mfa/recover", map[string]string{"code": test.recovery[0]}, 200)
	state := test.state()
	if state["assurance"] != false || state["has_factor"] != false {
		t.Fatal("recovery bypassed replacement enrollment")
	}
	if status, _, _ := test.request("GET", "/api/users", nil, ""); status != 403 {
		t.Fatal("recovery administered installation")
	}
	if _, err := test.database.Session(t.Context(), oldCookie); err == nil {
		t.Fatal("factor recovery retained old session")
	}
	test.api("/api/mfa/recover", map[string]string{"code": test.recovery[0]}, 400)
}
func mustURL(raw string) *url.URL { parsed, _ := url.Parse(raw); return parsed }

func TestProductionPromptBindingReplayAndConcurrency(t *testing.T) {
	test := newProduction(t)
	test.configure()
	id, _, _, _ := test.begin("web-rp", "login")
	test.api("/api/consent", map[string]any{"request": id, "approve": true}, 403)
	test.login("member")
	test.api("/api/consent", map[string]any{"request": id, "approve": true}, 200)
	code, verifier, _ := test.code("web-rp")
	config := test.oauth("web-rp")
	wrong := test.oauth("second-rp")
	if _, err := wrong.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("cross-client code accepted")
	}
	if _, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(oauth2.GenerateVerifier())); err == nil {
		t.Fatal("wrong PKCE accepted")
	}
	token, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("code replay accepted")
	}
	if test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("code replay did not revoke")
	}
	token, _ = test.token("web-rp")
	var successes atomic.Int32
	var workers sync.WaitGroup
	start := make(chan struct{})
	for range 20 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			if _, err := config.TokenSource(context.Background(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token(); err == nil {
				successes.Add(1)
			}
		}()
	}
	close(start)
	workers.Wait()
	if successes.Load() != 1 || test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("refresh race or durable family revoke failed")
	}
}

func TestSafeProtocolLogs(t *testing.T) {
	var output bytes.Buffer
	logger := SafeLogger(&output)
	logger.Error("password-canary", slog.String("token", "token-canary"))
	logger.With("secret", "factor-canary").Warn("recovery-canary")
	for _, value := range []string{"password-canary", "token-canary", "factor-canary", "recovery-canary"} {
		if strings.Contains(output.String(), value) {
			t.Fatal("secret reached log")
		}
	}
	if !strings.Contains(output.String(), "protocol event") {
		t.Fatal("missing event record")
	}
}
