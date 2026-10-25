package oidc

import "testing"

func TestReadinessHasNoCookieOrSensitiveOutput(t *testing.T) {
	test := newProduction(t)
	status, body, headers := test.request("GET", "/readyz", nil, "")
	if status != 200 || body != "{\"ready\":true}\n" || headers.Get("Set-Cookie") != "" || headers.Get("Cache-Control") != "no-store" {
		t.Fatal("readiness success contract")
	}
	status, _, headers = test.request("POST", "/readyz", nil, "")
	if status != 405 || headers.Get("Allow") != "GET" || headers.Get("Set-Cookie") != "" {
		t.Fatal("readiness allowed mutation or created session")
	}
	test.database.Close()
	status, body, headers = test.request("GET", "/readyz", nil, "")
	if status != 503 || body != "{\"ready\":false}\n" || headers.Get("Set-Cookie") != "" {
		t.Fatal("readiness failure contract")
	}
}
