package oidc

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

func TestDisabledApplicationReviewIsCredentialFreeAndDoesNotRestoreGrants(t *testing.T) {
	test := newProduction(t)
	test.configure()
	token, _ := test.token("web-rp")
	code, verifier, _ := test.code("web-rp")
	if status, _, _ := test.request("GET", "/api/applications", nil, ""); status != 403 {
		t.Fatal("non-admin can list configuration")
	}
	original, err := test.database.Application(t.Context(), "web-rp")
	if err != nil {
		t.Fatal(err)
	}
	memberBrowser, memberCsrf := test.browser, test.csrf
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	test.api("/api/applications/disable", map[string]string{"client": "web-rp"}, 200)
	status, body, _ := test.request("GET", "/api/applications", nil, "")
	if status != 200 {
		t.Fatal("settings unavailable")
	}
	for _, forbidden := range []string{"secret", "generation", test.secrets["web-rp"], original.SecretHash} {
		if strings.Contains(body, forbidden) {
			t.Fatal("sensitive registration field leaked")
		}
	}
	var views []store.ApplicationView
	if err = json.Unmarshal([]byte(body), &views); err != nil {
		t.Fatal(err)
	}
	var reviewed store.ApplicationView
	for _, app := range views {
		if app.ID == "web-rp" {
			reviewed = app
		}
	}
	if reviewed.ID == "" || reviewed.Enabled || reviewed.Name != original.Name || len(reviewed.LogoutRedirects) != 1 || reviewed.LogoutRedirects[0] != original.LogoutRedirects[0] || len(reviewed.Redirects) != 1 || reviewed.Redirects[0] != original.Redirects[0] {
		t.Fatal("disabled settings cannot be reviewed")
	}
	if _, err = test.database.Application(t.Context(), reviewed.ID); err == nil {
		t.Fatal("listing enabled authentication")
	}
	replacement := store.Application{ID: reviewed.ID, Name: reviewed.Name, Redirects: reviewed.Redirects, LogoutRedirects: reviewed.LogoutRedirects, Scopes: []string{"openid"}, Audiences: []string{}, Public: reviewed.Public, Resource: reviewed.Resource}
	result := test.api("/api/applications/replace-disabled", map[string]any{"application": replacement}, 200)
	newSecret := result["client_secret"].(string)
	if newSecret == "" || newSecret == test.secrets["web-rp"] {
		t.Fatal("client secret was restored")
	}
	test.browser, test.csrf = memberBrowser, memberCsrf
	configWithoutAssignment := test.oauth("web-rp")
	configWithoutAssignment.Scopes = []string{"openid"}
	status, _, headers := test.request("GET", configWithoutAssignment.AuthCodeURL("review-state", oauth2.S256ChallengeOption(oauth2.GenerateVerifier()), oauth2.SetAuthURLParam("nonce", "review-nonce")), nil, "")
	if status != 302 {
		t.Fatal("new authorization rejected before assignment check")
	}
	location, _ := url.Parse(headers.Get("Location"))
	test.api("/api/consent", map[string]any{"request": location.Query().Get("request"), "approve": true}, 403)
	test.browser, test.csrf = test.adminBrowser, test.adminCsrf
	test.api("/api/assignments", map[string]any{"user": test.member, "client": "web-rp", "allow": true}, 200)
	test.secrets["web-rp"] = newSecret
	config := test.oauth("web-rp")
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("old code revived after reassignment")
	}
	if _, err = config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token(); err == nil {
		t.Fatal("old refresh revived after reassignment")
	}
	if test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("old access revived")
	}
}
