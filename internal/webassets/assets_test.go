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
