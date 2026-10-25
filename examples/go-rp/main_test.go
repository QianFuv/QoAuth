package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCallbackBindingAndLogoutCsrf(t *testing.T) {
	app := &demo{origin: "https://app.example.test", secure: true, transactions: map[string]loginTransaction{"browser": {State: "expected-state", Expires: time.Now().Add(time.Minute)}}, sessions: map[string]appSession{"signed-in": {Subject: "synthetic-user", Csrf: "expected-proof", Expires: time.Now().Add(time.Minute)}}}
	request := httptest.NewRequest("GET", "https://app.example.test/callback?state=wrong&code=unused", nil)
	request.AddCookie(&http.Cookie{Name: app.cookieName(), Value: "browser"})
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != 400 || len(app.transactions) != 0 {
		t.Fatal("wrong state retained a reusable login transaction")
	}
	for _, origin := range []string{"", "https://attacker.example"} {
		request = httptest.NewRequest("POST", "https://app.example.test/logout", strings.NewReader(url.Values{"csrf": {"expected-proof"}}.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", origin)
		request.AddCookie(&http.Cookie{Name: app.cookieName(), Value: "signed-in"})
		response = httptest.NewRecorder()
		app.ServeHTTP(response, request)
		if response.Code != 403 || len(app.sessions) != 1 {
			t.Fatal("cross-origin logout mutated app session")
		}
	}
}
