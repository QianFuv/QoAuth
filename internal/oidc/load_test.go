package oidc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

// TestModestLocalLoad is opt-in, bounded to ten minutes and uses synthetic temporary state.
func TestModestLocalLoad(t *testing.T) {
	raw := os.Getenv("QOAUTH_LOAD_SECONDS")
	if raw == "" {
		t.Skip("explicit local modest-load run only")
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 60 || seconds > 600 {
		t.Fatal("duration must be 60..600 seconds")
	}
	test := newProduction(t)
	test.configure()
	owner, err := test.database.AccountByName(t.Context(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := test.database.Sessions(t.Context(), owner.ID)
	if err != nil || len(sessions) != 1 {
		t.Fatal("synthetic owner session unavailable")
	}
	for index := 0; index < 98; index++ {
		if _, err = test.database.AddAccount(t.Context(), fmt.Sprintf("load-user-%03d", index), owner.Password, sessions[0].Digest); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 7; index++ {
		app := store.Application{ID: fmt.Sprintf("load-client-%02d", index), Name: "Synthetic load client", Public: true, Redirects: []string{test.server.URL + "/rp/callback"}, Scopes: []string{"openid"}}
		if err = test.database.AddApplication(t.Context(), app, sessions[0].Digest); err != nil {
			t.Fatal(err)
		}
	}
	token, _ := test.token("web-rp")
	config := test.oauth("web-rp")
	nextRefresh := time.Now().Add(4 * time.Minute)
	durations := make([]float64, 0, seconds*10)
	client := &http.Client{Timeout: 2 * time.Second}
	started := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for index := 0; index < seconds*10; index++ {
		<-ticker.C
		if time.Now().After(nextRefresh) {
			token, err = config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token()
			if err != nil {
				t.Fatal("load grant refresh failed")
			}
			nextRefresh = time.Now().Add(4 * time.Minute)
		}
		method, path := "GET", "/readyz"
		var body io.Reader
		if index%3 == 1 {
			path = "/.well-known/openid-configuration"
		}
		if index%3 == 2 {
			method, path = "POST", "/oauth/introspect"
			body = strings.NewReader(url.Values{"token": {token.AccessToken}}.Encode())
		}
		request, _ := http.NewRequestWithContext(t.Context(), method, test.server.URL+path, body)
		if method == "POST" {
			request.SetBasicAuth("notes-api", test.secrets["notes-api"])
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		before := time.Now()
		response, err := client.Do(request)
		if err != nil {
			t.Fatal("load transport failed")
		}
		var result map[string]any
		err = json.NewDecoder(response.Body).Decode(&result)
		response.Body.Close()
		if response.StatusCode != 200 || err != nil || (method == "POST" && result["active"] != true) {
			t.Fatal("load response failed")
		}
		durations = append(durations, float64(time.Since(before).Microseconds())/1000)
	}
	sort.Float64s(durations)
	statistics, err := os.Stat(filepath.Join(test.directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]any{"requests": len(durations), "seconds": time.Since(started).Seconds(), "target_rps": 10, "users": 100, "applications": 10, "concurrency": 1, "p50_ms": durations[len(durations)/2], "p95_ms": durations[len(durations)*95/100], "max_ms": durations[len(durations)-1], "database_bytes": statistics.Size(), "mix": "equal readiness/discovery/authenticated introspection; refresh every four minutes; not password-login capacity"}
	encoded, _ := json.Marshal(result)
	t.Log(string(encoded))
}
