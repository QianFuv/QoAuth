package oidc

import (
	"net/http/httptest"
	"testing"
)

func TestTrustedProxySeparatesClientLimitsAndRejectsForgedHeaders(t *testing.T) {
	test := newProduction(t)
	test.handler.config.TrustedProxy = "192.0.2.10"
	request := httptest.NewRequest("GET", test.server.URL+"/api/session", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("X-Real-IP", "198.51.100.1")
	for index := 0; index < 61; index++ {
		response := httptest.NewRecorder()
		test.handler.ServeHTTP(response, request)
		if index < 60 && response.Code != 200 {
			t.Fatal("allowance too small", response.Code)
		}
		if index == 60 && response.Code != 401 {
			t.Fatal("allowance not enforced", response.Code)
		}
	}
	request.Header.Set("X-Real-IP", "198.51.100.2")
	response := httptest.NewRecorder()
	test.handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatal("one proxy client blocked another")
	}
	request.RemoteAddr = "192.0.2.99:1234"
	if address, err := test.handler.clientAddress(request); err != nil || address != "192.0.2.99" {
		t.Fatal("untrusted header accepted")
	}
	request.RemoteAddr = "192.0.2.10:1234"
	for _, invalid := range []string{"", "198.51.100.1, 198.51.100.2", "not-an-address"} {
		request.Header.Set("X-Real-IP", invalid)
		if _, err := test.handler.clientAddress(request); err == nil {
			t.Fatal("unsanitized proxy header accepted")
		}
	}
}
