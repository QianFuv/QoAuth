package oidc

import (
	"bytes"
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

func TestAdministratorResetPreservesMfaAndRevokesAuthentication(t *testing.T) {
	test := newProduction(t)
	test.configure()
	setup := test.api("/api/mfa/start", map[string]any{}, 200)
	factor := setup["secret"].(string)
	proof, _ := totp.GenerateCode(factor, time.Now().Add(-30*time.Second))
	test.api("/api/mfa/confirm", map[string]string{"code": proof}, 200)
	test.state()
	before, _ := test.database.Account(t.Context(), test.member)
	token, _ := test.token("web-rp")
	code, verifier, _ := test.code("web-rp")
	memberBrowser, memberCsrf := test.browser, test.csrf
	input := map[string]string{"user": test.member, "password": testPassword + " replacement"}
	test.api("/api/users/password", input, 403)
	test.browser, test.csrf = test.adminBrowser, "wrong-csrf"
	test.api("/api/users/password", input, 403)
	test.csrf = test.adminCsrf
	adminRequest := httptest.NewRequest("POST", test.server.URL+"/api/users/password", nil)
	adminURL := adminRequest.URL
	for _, cookie := range test.adminBrowser.Jar.Cookies(adminURL) {
		adminRequest.AddCookie(cookie)
	}
	admin, err := test.handler.session(adminRequest)
	if err != nil {
		t.Fatal(err)
	}
	for _, incomplete := range []bool{false, true} {
		blocked := admin
		if incomplete {
			blocked.Assurance = false
		} else {
			blocked.AuthTime = time.Now().Add(-6 * time.Minute).Unix()
		}
		request := httptest.NewRequest("POST", test.server.URL+"/api/users/password", strings.NewReader(`{"user":"`+test.member+`","password":"`+input["password"]+`"}`))
		request.Header.Set("Origin", test.server.URL)
		request.Header.Set("X-CSRF-Token", admin.Csrf)
		request = request.WithContext(context.WithValue(request.Context(), sessionContext{}, blocked))
		response := httptest.NewRecorder()
		test.handler.api(response, request)
		if response.Code != 403 {
			t.Fatal("stale or incomplete MFA accepted")
		}
	}
	test.api("/api/users/password", input, 200)
	after, _ := test.database.Account(t.Context(), test.member)
	if !bytes.Equal(before.Factor, after.Factor) || before.LastStep != after.LastStep || !after.RequireMfa || after.Version != before.Version+1 {
		t.Fatal("reset modified MFA or omitted credential revision")
	}
	test.database.Close()
	test.database, err = store.Open(t.Context(), filepath.Join(test.directory, "qoauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	test.handler.database = test.database
	test.browser, test.csrf = memberBrowser, memberCsrf
	if test.state()["authenticated"] != false {
		t.Fatal("reset retained browser session across restart")
	}
	config := test.oauth("web-rp")
	if _, err = config.Exchange(t.Context(), code, oauth2.VerifierOption(verifier)); err == nil {
		t.Fatal("reset retained code")
	}
	if _, err = config.TokenSource(t.Context(), &oauth2.Token{RefreshToken: token.RefreshToken}).Token(); err == nil {
		t.Fatal("reset retained refresh")
	}
	if test.introspect(token.AccessToken, "notes-api")["active"] != false {
		t.Fatal("reset retained access")
	}
	test.api("/api/login", map[string]string{"username": "member", "password": testPassword}, 401)
	test.api("/api/login", map[string]string{"username": "member", "password": input["password"]}, 200)
	if test.state()["assurance"] != false {
		t.Fatal("reset bypassed enrolled MFA")
	}
	proof, _ = totp.GenerateCode(factor, time.Now())
	test.api("/api/mfa/verify", map[string]string{"code": proof}, 200)
	if test.state()["assurance"] != true {
		t.Fatal("preserved factor cannot complete sign-in")
	}
}
