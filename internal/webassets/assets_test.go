package webassets

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtocolPathsNeverFallBackToHtml(t *testing.T) {
	for _, path := range []string{"/api/missing", "/oauth/token", "/.well-known/openid-configuration", "/assets/../api/missing"} {
		response := httptest.NewRecorder()
		Handler().ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 404 || strings.Contains(response.Body.String(), "<html") {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "/assets/") || response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing embedded bundle or CSP")
	}
}

func TestHtmlPreservesValidatedLogoutFormTarget(t *testing.T) {
	response := httptest.NewRecorder()
	policy := "default-src 'none'; form-action 'self' https://rp.example.test; frame-ancestors 'none'"
	response.Header().Set("Content-Security-Policy", policy)
	Handler().ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	if response.Header().Get("Content-Security-Policy") != policy || response.Header().Get("Referrer-Policy") != "strict-origin" {
		t.Fatal("native logout form loses its validated target or canonical Origin")
	}
}

func TestFontsLoadOnlyFromOwnOrigin(t *testing.T) {
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	policy := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "font-src 'self';") || strings.Contains(policy, "font-src 'self' ") || !strings.HasPrefix(policy, "default-src 'none';") {
		t.Fatalf("self-hosted fonts must be allowed without external or data sources: %s", policy)
	}
}
