package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

func TestStandardClientCallbackRefreshLogoutAndClaimBinding(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var expectedNonce atomic.Value
	expectedNonce.Store("")
	var wrongNonce, wrongSubject atomic.Bool
	var refreshes atomic.Int32
	var issuer string
	op := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(response).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "userinfo_endpoint": issuer + "/userinfo", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(response).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			if request.ParseForm() != nil {
				http.Error(response, "invalid", 400)
				return
			}
			if request.Form.Get("grant_type") == "refresh_token" {
				refreshes.Add(1)
			}
			nonce := expectedNonce.Load().(string)
			if wrongNonce.Load() {
				nonce = "another-transaction"
			}
			encoded, signErr := jwt.Signed(signer).Claims(map[string]any{"iss": issuer, "sub": "member", "aud": "rp", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(), "nonce": nonce}).Serialize()
			if signErr != nil {
				http.Error(response, "signing failed", 500)
				return
			}
			_ = json.NewEncoder(response).Encode(map[string]any{"access_token": "synthetic-access", "refresh_token": "synthetic-refresh", "token_type": "Bearer", "expires_in": 60, "id_token": encoded})
		case "/userinfo":
			subject := "member"
			if wrongSubject.Load() {
				subject = "another-user"
			}
			_ = json.NewEncoder(response).Encode(map[string]string{"sub": subject})
		default:
			http.NotFound(response, request)
		}
	}))
	defer op.Close()
	issuer = op.URL
	provider, err := rp.NewProvider(t.Context(), issuer)
	if err != nil {
		t.Fatal(err)
	}
	app := &demo{provider: provider, config: oauth2.Config{ClientID: "rp", ClientSecret: "synthetic-secret", RedirectURL: "https://app.example.test/callback", Endpoint: provider.Endpoint(), Scopes: []string{"openid"}}, origin: "https://app.example.test", logout: issuer + "/end_session", secure: true, transactions: map[string]loginTransaction{}, sessions: map[string]appSession{}}
	login := func() *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest("GET", app.origin+"/login", nil))
		location, _ := url.Parse(response.Header().Get("Location"))
		expectedNonce.Store(location.Query().Get("nonce"))
		cookie := response.Result().Cookies()[0]
		if cookie.Name != "__Host-qoauth_rp" || !cookie.Secure || !cookie.HttpOnly || cookie.Domain != "" {
			t.Fatal("RP cookie security")
		}
		request := httptest.NewRequest("GET", app.origin+"/callback?state="+url.QueryEscape(location.Query().Get("state"))+"&code=synthetic-code", nil)
		request.AddCookie(cookie)
		response = httptest.NewRecorder()
		app.ServeHTTP(response, request)
		return response
	}
	wrongNonce.Store(true)
	if response := login(); response.Code != 401 || len(app.sessions) != 0 {
		t.Fatal("wrong nonce admitted")
	}
	wrongNonce.Store(false)
	response := login()
	if response.Code != 303 {
		t.Fatal("successful callback rejected", response.Code)
	}
	cookie := response.Result().Cookies()[0]
	session := app.sessions[cookie.Value]
	session.Token.Expiry = time.Now().Add(-time.Minute)
	app.sessions[cookie.Value] = session
	request := httptest.NewRequest("GET", app.origin+"/profile", nil)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != 200 || refreshes.Load() != 1 {
		t.Fatal("profile/refresh failed")
	}
	wrongSubject.Store(true)
	response = httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != 401 || len(app.sessions) != 0 {
		t.Fatal("mismatched UserInfo subject admitted")
	}
	wrongSubject.Store(false)
	response = login()
	cookie = response.Result().Cookies()[0]
	session = app.sessions[cookie.Value]
	request = httptest.NewRequest("POST", app.origin+"/logout", strings.NewReader(url.Values{"csrf": {session.Csrf}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", app.origin)
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != 303 || len(app.sessions) != 0 || !strings.HasPrefix(response.Header().Get("Location"), issuer+"/end_session?") {
		t.Fatal("confirmed logout failed")
	}
}
