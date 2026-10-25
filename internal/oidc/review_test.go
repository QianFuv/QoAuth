package oidc

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

func TestDuplicateProtocolParametersAreRejected(t *testing.T) {
	test := newProduction(t)
	test.configure()
	memberBrowser, memberCsrf := test.browser, test.csrf
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	app := store.Application{ID: "limited", Name: "Limited", Public: true, Redirects: []string{test.server.URL + "/rp/callback"}, Scopes: []string{"openid"}}
	test.api("/api/applications/create", map[string]any{"application": app}, 200)
	test.browser, test.csrf = memberBrowser, memberCsrf
	config := test.oauth("limited")
	config.Scopes = []string{"openid"}
	target := config.AuthCodeURL("state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "nonce"))
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			duplicate := "scope=openid%20profile%20offline_access"
			requestURL, body := target+"&"+duplicate, ""
			if method == http.MethodPost {
				requestURL, body = target, duplicate
			}
			status, _, _ := test.request(method, requestURL, strings.NewReader(body), "application/x-www-form-urlencoded")
			if status != 400 {
				t.Fatalf("duplicate scope accepted: %d", status)
			}
		})
	}
}

func TestMaxAgeRequiresAuthenticationEventNotCookieRotation(t *testing.T) {
	test := newProduction(t)
	test.configure()
	account, err := test.database.Account(t.Context(), test.member)
	if err != nil {
		t.Fatal(err)
	}
	zero, positive := uint(0), uint(60)
	now := time.Now().Unix()
	session := store.Session{UserID: account.ID, Assurance: true, Digest: "rotated-cookie", AuthEvent: "old-authentication", AuthTime: now}
	data := tokenData{Request: protocol.AuthRequest{MaxAge: &zero}, InitialAuthEvent: session.AuthEvent}
	if test.handler.eligible(t.Context(), session, data) {
		t.Fatal("cookie rotation satisfied max_age=0")
	}
	session.AuthEvent = "fresh-password-in-same-second"
	if !test.handler.eligible(t.Context(), session, data) {
		t.Fatal("same-second fresh authentication rejected")
	}
	data.Request.MaxAge = &positive
	session.AuthTime = now - 120
	if test.handler.eligible(t.Context(), session, data) {
		t.Fatal("expired positive max_age accepted")
	}
	session.AuthTime = now
	if !test.handler.eligible(t.Context(), session, data) {
		t.Fatal("fresh positive max_age rejected")
	}
	config := test.oauth("web-rp")
	target := config.AuthCodeURL("state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "nonce"), oauth2.SetAuthURLParam("prompt", "none"), oauth2.SetAuthURLParam("max_age", "0"))
	status, _, headers := test.request("GET", target, nil, "")
	parsed, _ := url.Parse(headers.Get("Location"))
	if status != 302 || parsed.Query().Get("error") != "login_required" {
		t.Fatal("silent max_age=0 failed to require authentication")
	}
}

func TestCallbackRejectsMixedBrowserTransactions(t *testing.T) {
	test := newProduction(t)
	test.configure()
	first, _, _, _ := test.begin("web-rp", "consent")
	test.api("/api/consent", map[string]any{"request": first, "approve": true}, 200)
	firstBrowser, firstCsrf := test.browser, test.csrf
	test.newBrowser()
	test.login("member")
	second, _, _, _ := test.begin("web-rp", "consent")
	test.api("/api/consent", map[string]any{"request": second, "approve": true}, 200)
	secondBrowser, secondCsrf := test.browser, test.csrf
	test.browser, test.csrf = firstBrowser, firstCsrf
	status, _, _ := test.request("POST", "/authorize/callback?id="+first, strings.NewReader(url.Values{"id": {second}}.Encode()), "application/x-www-form-urlencoded")
	if status != 405 {
		t.Fatalf("mixed callback accepted: %d", status)
	}
	test.browser, test.csrf = secondBrowser, secondCsrf
	status, _, headers := test.request("GET", "/authorize/callback?id="+second, nil, "")
	if status != 302 || !strings.Contains(headers.Get("Location"), "code=") {
		t.Fatal("legitimate callback unusable")
	}
}
