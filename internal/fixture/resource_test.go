package fixture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

// resourceHandler is a test-only RFC 7662 consumer, separate from the OP adapter.
// It has no positive cache and denies access when introspection is unavailable.
func resourceHandler(endpoint, secret, audience, scope string) http.Handler {
	client := &http.Client{Timeout: 200 * time.Millisecond}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		token, hasBearer := strings.CutPrefix(request.Header.Get("Authorization"), "Bearer ")
		if !hasBearer || token == "" {
			http.Error(response, "unauthorized", http.StatusUnauthorized)
			return
		}
		form := url.Values{"token": {token}}
		lookup, err := http.NewRequestWithContext(request.Context(), http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			http.Error(response, "unavailable", http.StatusServiceUnavailable)
			return
		}
		lookup.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		lookup.SetBasicAuth("notes-api-validator", secret)
		result, err := client.Do(lookup)
		if err != nil {
			http.Error(response, "unavailable", http.StatusServiceUnavailable)
			return
		}
		defer result.Body.Close()
		var claims struct {
			Active   bool     `json:"active"`
			Audience []string `json:"aud"`
			Scope    string   `json:"scope"`
		}
		if result.StatusCode != http.StatusOK || json.NewDecoder(result.Body).Decode(&claims) != nil || !claims.Active || !slices.Contains(claims.Audience, audience) || !slices.Contains(strings.Fields(claims.Scope), scope) {
			http.Error(response, "forbidden", http.StatusForbidden)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
}

func TestIndependentResourceAccessAndFailClosed(t *testing.T) {
	test := newHarness(t)
	token := test.token("web-rp")
	endpoint := test.config.Issuer + "/oauth/introspect"
	secret := test.config.Secrets["notes-api-validator"]
	resource := httptest.NewServer(resourceHandler(endpoint, secret, Audience, "notes.read"))
	defer resource.Close()
	access := func(address string) int {
		t.Helper()
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+token.AccessToken)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		return response.StatusCode
	}
	if status := access(resource.URL); status != http.StatusNoContent {
		t.Fatalf("valid RP-to-API request: %d", status)
	}
	for _, policy := range []struct{ audience, scope string }{{"other-api", "notes.read"}, {Audience, "notes.write"}} {
		denied := httptest.NewServer(resourceHandler(endpoint, secret, policy.audience, policy.scope))
		status := access(denied.URL)
		denied.Close()
		if status != http.StatusForbidden {
			t.Fatalf("resource ignored audience/scope: %d", status)
		}
	}
	if status, body, _ := test.request(http.MethodPost, test.config.Issuer+"/revoke", url.Values{"token": {token.RefreshToken}}, "web-rp"); status != http.StatusOK {
		t.Fatalf("revoke: %d %s", status, body)
	}
	if status := access(resource.URL); status != http.StatusForbidden {
		t.Fatalf("revoked resource request: %d", status)
	}
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer slow.Close()
	defer close(release)
	unavailable := httptest.NewServer(resourceHandler(slow.URL, secret, Audience, "notes.read"))
	defer unavailable.Close()
	if status := access(unavailable.URL); status != http.StatusServiceUnavailable {
		t.Fatalf("introspection timeout failed open: %d", status)
	}
}
