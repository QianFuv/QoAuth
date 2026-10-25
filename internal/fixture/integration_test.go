package fixture

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

type handlerValue struct{ http.Handler }
type harness struct {
	t        *testing.T
	database *store.Store
	path     string
	config   Config
	server   *httptest.Server
	browser  *http.Client
	provider *rp.Provider
	current  atomic.Pointer[handlerValue]
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	test := &harness{t: t, path: filepath.Join(t.TempDir(), "oidc-fixture.db")}
	database, err := store.Open(t.Context(), test.path)
	if err != nil {
		t.Fatal(err)
	}
	test.database = database
	keys, err := NewKeys()
	if err != nil {
		t.Fatal(err)
	}
	test.server = httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		test.current.Load().ServeHTTP(response, request)
	}))
	issuer := "http://" + test.server.Listener.Addr().String()
	test.config = Config{Issuer: issuer, Callback: issuer + "/rp/callback", Keys: keys, Secrets: Secrets()}
	handler, err := New(t.Context(), database, test.config)
	if err != nil {
		t.Fatal(err)
	}
	test.current.Store(&handlerValue{handler})
	test.server.Start()
	jar, _ := cookiejar.New(nil)
	test.browser = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	t.Cleanup(func() { test.server.Close(); test.database.Close() })
	test.provider, err = rp.NewProvider(t.Context(), issuer)
	if err != nil {
		t.Fatal(err)
	}
	return test
}

func (test *harness) oauth(client string) oauth2.Config {
	endpoint := test.provider.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInHeader
	if client == "public-rp" {
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	}
	return oauth2.Config{ClientID: client, ClientSecret: test.config.Secrets[client], RedirectURL: test.config.Callback, Endpoint: endpoint, Scopes: []string{"openid", "profile", "email", "offline_access", "notes.read"}}
}

func (test *harness) request(method, address string, form url.Values, client string) (int, string, string) {
	test.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	request, err := http.NewRequestWithContext(test.t.Context(), method, address, body)
	if err != nil {
		test.t.Fatal(err)
	}
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", test.config.Issuer)
	}
	if client != "" {
		request.SetBasicAuth(client, test.config.Secrets[client])
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
	return response.StatusCode, string(content), response.Header.Get("Location")
}

func (test *harness) authorize(client string) (code, verifier, nonce string) {
	test.t.Helper()
	config := test.oauth(client)
	verifier = oauth2.GenerateVerifier()
	nonce = store.NewHandle()
	state := store.NewHandle()
	address := config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("prompt", "consent"))
	status, body, location := test.request("GET", address, nil, "")
	if status != 302 {
		test.t.Fatalf("authorize: %d %s", status, body)
	}
	login, err := url.Parse(location)
	if err != nil {
		test.t.Fatal(err)
	}
	status, body, _ = test.request("GET", test.config.Issuer+login.String(), nil, "")
	if status != 200 || !strings.Contains(body, "/assets/") {
		test.t.Fatalf("embedded login: %d %s", status, body)
	}
	status, body, location = test.request("POST", test.config.Issuer+"/fixture/login", url.Values{"request": {login.Query().Get("request")}}, "")
	if status != 303 {
		test.t.Fatalf("fixture login: %d %s", status, body)
	}
	status, body, location = test.request("GET", location, nil, "")
	if status != 302 {
		test.t.Fatalf("OP callback: %d %s", status, body)
	}
	callback, err := url.Parse(location)
	if err != nil {
		test.t.Fatal(err)
	}
	if callback.Query().Get("state") != state {
		test.t.Fatal("RP state mismatch")
	}
	code = callback.Query().Get("code")
	if code == "" {
		test.t.Fatalf("callback did not contain code: %s", location)
	}
	return code, verifier, nonce
}

func (test *harness) token(client string) *oauth2.Token {
	test.t.Helper()
	code, verifier, nonce := test.authorize(client)
	config := test.oauth(client)
	token, err := config.Exchange(test.t.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		test.t.Fatal(err)
	}
	identity, err := test.provider.Verifier(&rp.Config{ClientID: client, SupportedSigningAlgs: []string{"RS256"}}).Verify(test.t.Context(), token.Extra("id_token").(string))
	if err != nil || identity.Subject != Subject || identity.Nonce != nonce {
		test.t.Fatalf("independent RP ID Token validation: %+v %v", identity, err)
	}
	return token
}

func (test *harness) introspect(token, caller string) map[string]any {
	test.t.Helper()
	status, body, _ := test.request("POST", test.config.Issuer+"/oauth/introspect", url.Values{"token": {token}}, caller)
	if status != 200 {
		test.t.Fatalf("introspection: %d %s", status, body)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		test.t.Fatal(err)
	}
	return result
}

func (test *harness) restart() {
	test.t.Helper()
	test.database.Close()
	database, err := store.Open(test.t.Context(), test.path)
	if err != nil {
		test.t.Fatal(err)
	}
	test.database = database
	handler, err := New(test.t.Context(), database, test.config)
	if err != nil {
		test.t.Fatal(err)
	}
	test.current.Store(&handlerValue{handler})
}

func TestStandardRpLoginRefreshRevokeAndRestart(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	if token.RefreshToken == "" {
		t.Fatal("offline consent did not issue refresh token")
	}
	info, err := test.provider.UserInfo(t.Context(), oauth2.StaticTokenSource(token))
	if err != nil || info.Subject != Subject {
		t.Fatalf("UserInfo: %+v %v", info, err)
	}
	active := test.introspect(token.AccessToken, "notes-api-validator")
	if active["active"] != true || active["client_id"] != "web-rp" || !strings.Contains(active["scope"].(string), "notes.read") {
		t.Fatal(active)
	}
	for _, caller := range []string{"wrong-api-validator", "web-rp"} {
		if denied := test.introspect(token.AccessToken, caller); denied["active"] != false || len(denied) != 1 {
			t.Fatalf("wrong resource disclosed token: %v", denied)
		}
	}
	if status, _, _ := test.request("POST", test.config.Issuer+"/oauth/introspect", url.Values{"token": {token.AccessToken}}, ""); status != 401 {
		t.Fatalf("unauthenticated introspection: %d", status)
	}
	test.restart()
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != true {
		t.Fatal("valid grant lost on same-key restart")
	}
	config := test.oauth("web-rp")
	rotated, err := config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token()
	if err != nil || rotated.RefreshToken == token.RefreshToken {
		t.Fatalf("standard client refresh: %v", err)
	}
	if status, body, _ := test.request("POST", test.config.Issuer+"/revoke", url.Values{"token": {rotated.RefreshToken}, "token_type_hint": {"refresh_token"}}, "web-rp"); status != 200 {
		t.Fatalf("revoke: %d %s", status, body)
	}
	test.restart()
	if test.introspect(rotated.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("revocation resurrected on restart")
	}
	if _, err = config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: rotated.RefreshToken}).Token(); err == nil {
		t.Fatal("revoked refresh accepted")
	}
	if _, err = test.provider.UserInfo(t.Context(), oauth2.StaticTokenSource(rotated)); err == nil {
		t.Fatal("revoked UserInfo accepted")
	}
}

func TestPublicClientAndProtocolPolicy(t *testing.T) {
	test := newHarness(t)
	test.token("public-rp")
	status, body, _ := test.request("GET", test.config.Issuer+"/.well-known/openid-configuration", nil, "")
	if status != 200 {
		t.Fatal(body)
	}
	var metadata struct {
		GrantTypes    []string `json:"grant_types_supported"`
		ResponseTypes []string `json:"response_types_supported"`
		Methods       []string `json:"code_challenge_methods_supported"`
	}
	if err := json.Unmarshal([]byte(body), &metadata); err != nil {
		t.Fatal(err)
	}
	if strings.Join(metadata.GrantTypes, ",") != "authorization_code,refresh_token" || strings.Join(metadata.ResponseTypes, ",") != "code" || strings.Join(metadata.Methods, ",") != "S256" {
		t.Fatalf("overadvertised features: %+v", metadata)
	}
	for _, change := range []string{"missing", "plain", "redirect", "scope", "implicit", "none"} {
		t.Run(change, func(t *testing.T) {
			config := test.oauth("web-rp")
			address, _ := url.Parse(config.AuthCodeURL("state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "nonce")))
			query := address.Query()
			switch change {
			case "missing":
				query.Del("code_challenge")
				query.Del("code_challenge_method")
			case "plain":
				query.Set("code_challenge_method", "plain")
			case "redirect":
				query.Set("redirect_uri", "https://invalid.example/callback")
			case "scope":
				query.Set("scope", "openid admin")
			case "implicit":
				query.Set("response_type", "id_token token")
			case "none":
				query.Set("prompt", "none")
			}
			address.RawQuery = query.Encode()
			status, body, location := test.request("GET", address.String(), nil, "")
			if strings.Contains(location, "/fixture/login") || status == 200 {
				t.Fatalf("invalid request accepted: %d %s %s", status, body, location)
			}
			if change == "redirect" && strings.HasPrefix(location, "https://invalid.example") {
				t.Fatal("unregistered redirect followed")
			}
		})
	}
}

func TestCodeBindingPkceAndReplay(t *testing.T) {
	test := newHarness(t)
	code, verifier, _ := test.authorize("web-rp")
	base := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {test.config.Callback}, "code_verifier": {verifier}}
	for _, variant := range []string{"wrong-verifier", "missing-verifier", "wrong-redirect", "wrong-client"} {
		t.Run(variant, func(t *testing.T) {
			form := url.Values{}
			for key, values := range base {
				form[key] = append([]string(nil), values...)
			}
			client := "web-rp"
			switch variant {
			case "wrong-verifier":
				form.Set("code_verifier", oauth2.GenerateVerifier())
			case "missing-verifier":
				form.Del("code_verifier")
			case "wrong-redirect":
				form.Set("redirect_uri", test.config.Callback+"/wrong")
			case "wrong-client":
				client = "other-rp"
			}
			if status, body, _ := test.request("POST", test.config.Issuer+"/oauth/token", form, client); status < 400 || !strings.Contains(body, "error") {
				t.Fatalf("bad exchange: %d %s", status, body)
			}
		})
	}
	config := test.oauth("web-rp")
	token, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("code replay accepted")
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("code replay did not revoke grant")
	}
}

func TestActualOpConcurrentRefreshAndFailureRollback(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	var successes atomic.Int32
	if status, _, _ := test.request("POST", test.config.Issuer+"/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token.RefreshToken}}, "other-rp"); status < 400 {
		t.Fatal("another client refreshed the grant")
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != true {
		t.Fatal("another client's refresh attempt revoked the legitimate grant")
	}
	var workers sync.WaitGroup
	start := make(chan struct{})
	for range 20 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			status, _, _ := test.request("POST", test.config.Issuer+"/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token.RefreshToken}}, "web-rp")
			if status == 200 {
				successes.Add(1)
			}
		}()
	}
	close(start)
	workers.Wait()
	if successes.Load() != 1 {
		t.Fatalf("refresh successes=%d", successes.Load())
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("family revocation rolled back with OAuth error")
	}
	code, verifier, _ := test.authorize("web-rp")
	fault, err := sql.Open("sqlite", test.path)
	if err != nil {
		t.Fatal(err)
	}
	defer fault.Close()
	if _, err = fault.Exec(`CREATE TABLE fixture_deferred (user_id TEXT REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED);
	CREATE TRIGGER fixture_fail AFTER INSERT ON tokens BEGIN INSERT INTO fixture_deferred VALUES('missing-user'); END;`); err != nil {
		t.Fatal(err)
	}
	config := test.oauth("web-rp")
	if result, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil || result != nil {
		t.Fatal("failed transaction issued tokens")
	}
	if _, err = fault.Exec(`DROP TRIGGER fixture_fail`); err != nil {
		t.Fatal(err)
	}
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err != nil {
		t.Fatal("real OP consumed code before failed transaction", err)
	}
}

func TestDisableInvalidatesPendingCodeAndAccess(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	code, verifier, _ := test.authorize("web-rp")
	if err := test.database.Invalidate(t.Context(), Subject, ""); err != nil {
		t.Fatal(err)
	}
	config := test.oauth("web-rp")
	if _, err := config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("disabled user redeemed pending code")
	}
	if _, err := config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token(); err == nil {
		t.Fatal("disabled user refreshed")
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("disabled user token active")
	}
	test.restart()
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("fixture seed re-enabled user")
	}
}

func TestOpaqueTamperingAndIndependentRpRejection(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	bytes := []byte(token.AccessToken)
	index := len(bytes) / 2
	if bytes[index] == 'A' {
		bytes[index] = 'B'
	} else {
		bytes[index] = 'A'
	}
	if test.introspect(string(bytes), "notes-api-validator")["active"] != false {
		t.Fatal("tampered opaque token accepted")
	}
	plain, err := op.NewAES256GCMCrypto(test.config.Keys.encryption, "m1-ephemeral").Decrypt(token.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := op.NewAESCrypto(test.config.Keys.encryption).Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if test.introspect(legacy, "notes-api-validator")["active"] != false {
		t.Fatal("legacy unauthenticated encryption accepted")
	}
	identity := token.Extra("id_token").(string)
	if _, err = test.provider.Verifier(&rp.Config{ClientID: "other-rp"}).Verify(t.Context(), identity); err == nil {
		t.Fatal("wrong audience accepted")
	}
	if _, err = test.provider.Verifier(&rp.Config{ClientID: "web-rp", SupportedSigningAlgs: []string{"ES256"}}).Verify(t.Context(), identity); err == nil {
		t.Fatal("wrong signing algorithm accepted")
	}
	if _, err = test.provider.Verifier(&rp.Config{ClientID: "web-rp"}).Verify(t.Context(), identity+"tampered"); err == nil {
		t.Fatal("bad signature accepted")
	}
	wrongIssuer := rp.NewVerifier("https://wrong.example", rp.NewRemoteKeySet(context.Background(), test.config.Issuer+"/keys"), &rp.Config{ClientID: "web-rp"})
	if _, err = wrongIssuer.Verify(t.Context(), identity); err == nil {
		t.Fatal("wrong issuer accepted")
	}
	request, _ := http.NewRequest("GET", test.config.Issuer+"/.well-known/openid-configuration", nil)
	request.Host = "forged.example"
	request.Header.Set("Forwarded", "host=forged.example;proto=https")
	response, err := test.browser.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var metadata map[string]any
	json.NewDecoder(response.Body).Decode(&metadata)
	if metadata["issuer"] != test.config.Issuer {
		t.Fatal("issuer derived from untrusted headers")
	}
}

func TestRpLogoutBindsRedirectAndRevokesGrant(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	query := url.Values{"id_token_hint": {token.Extra("id_token").(string)}, "post_logout_redirect_uri": {"https://wrong.example/logout"}, "state": {"logout-state"}}
	if status, _, location := test.request("GET", test.config.Issuer+"/end_session?"+query.Encode(), nil, ""); status < 400 || strings.HasPrefix(location, "https://wrong.example") {
		t.Fatal("invalid logout redirect accepted")
	}
	query.Set("post_logout_redirect_uri", test.config.Callback)
	status, body, location := test.request("GET", test.config.Issuer+"/end_session?"+query.Encode(), nil, "")
	if status != 302 || location != test.config.Callback+"?state=logout-state" {
		t.Fatalf("logout: %d %s %s", status, body, location)
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("logout retained grant")
	}
}

func TestExpiredCodeAndTokenAreRejected(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	code, verifier, _ := test.authorize("web-rp")
	database, err := sql.Open("sqlite", test.path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err = database.Exec(`UPDATE auth_requests SET expires=0; UPDATE tokens SET expires=0;`); err != nil {
		t.Fatal(err)
	}
	config := test.oauth("web-rp")
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("expired code accepted")
	}
	if test.introspect(token.AccessToken, "notes-api-validator")["active"] != false {
		t.Fatal("expired access token active")
	}
	if _, err = test.provider.UserInfo(t.Context(), oauth2.StaticTokenSource(token)); err == nil {
		t.Fatal("expired UserInfo accepted")
	}
}
