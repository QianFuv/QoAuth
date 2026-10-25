package oidc

import (
	"encoding/json"
	"strings"
	"testing"

	"qoauth/internal/identity"
)

func TestProductionPasswordPolicyMetadataAndCreationErrors(t *testing.T) {
	test := newProduction(t)
	policy := test.state()["password_policy"].(map[string]any)
	if policy["min_characters"] != float64(identity.MinPasswordCharacters) || policy["max_bytes"] != float64(identity.MaxPasswordBytes) {
		t.Fatal("UI policy differs from credential policy")
	}
	test.owner()
	var previous string
	for _, path := range []string{"/api/users/create", "/api/users/password", "/api/password/change"} {
		payload, _ := json.Marshal(map[string]string{"username": "policy-member", "user": "unmodified-target", "password": "short-canary"})
		status, body, _ := test.request("POST", path, strings.NewReader(string(payload)), "application/json")
		if status != 400 || !strings.Contains(body, "password policy:") || strings.Contains(body, "short-canary") || (previous != "" && body != previous) {
			t.Fatal("creation policy differs or echoes credentials", path)
		}
		previous = body
	}
}
