package oidc

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	zrp "github.com/zitadel/oidc/v3/pkg/client/rp"
	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"golang.org/x/oauth2"
	"qoauth/internal/store"
)

type expectedNonceContext struct{}

// TestZitadelRelyingPartyLifecycle uses the second RP implementation's discovery and verifier.
// It deliberately retains the independent coreos tests; both stacks share oauth2/JOSE dependencies.
func TestZitadelRelyingPartyLifecycle(t *testing.T) {
	test := newProduction(t)
	test.configure()
	party, err := zrp.NewRelyingPartyOIDC(t.Context(), test.server.URL, "second-rp", test.secrets["second-rp"], test.server.URL+"/rp/callback", []string{"openid", "profile", "offline_access", "notes.read"}, zrp.WithHTTPClient(test.server.Client()), zrp.WithAuthStyle(oauth2.AuthStyleInHeader), zrp.WithVerifierOpts(zrp.WithSupportedSigningAlgorithms("RS256"), zrp.WithNonce(func(ctx context.Context) string { nonce, _ := ctx.Value(expectedNonceContext{}).(string); return nonce })))
	if err != nil {
		t.Fatal(err)
	}
	state, nonce, verifier := store.NewHandle(), store.NewHandle(), oauth2.GenerateVerifier()
	authContext := context.WithValue(t.Context(), expectedNonceContext{}, nonce)
	address := zrp.AuthURL(state, party, zrp.WithCodeChallenge(protocol.NewSHACodeChallenge(verifier)), zrp.AuthURLOpt(zrp.WithURLParam("nonce", nonce)), zrp.WithPrompt("consent"))
	status, _, headers := test.request("GET", address, nil, "")
	if status != 302 {
		t.Fatal("second RP authorization rejected")
	}
	login, err := url.Parse(headers.Get("Location"))
	if err != nil || login.Query().Get("request") == "" {
		t.Fatal("login transaction missing")
	}
	if status, body, _ := test.request("GET", login.String(), nil, ""); status != 200 || !strings.Contains(body, "/assets/") {
		t.Fatal("embedded login unavailable")
	}
	completion := test.api("/api/consent", map[string]any{"request": login.Query().Get("request"), "approve": true}, 200)
	status, _, headers = test.request("GET", completion["redirect"].(string), nil, "")
	callback, err := url.Parse(headers.Get("Location"))
	if err != nil || status != 302 || callback.Scheme+"://"+callback.Host+callback.Path != test.server.URL+"/rp/callback" || callback.Query().Get("state") != state || callback.Query().Get("error") != "" || callback.Query().Get("code") == "" {
		t.Fatal("callback/state contract failed")
	}
	code := callback.Query().Get("code")
	_, err = zrp.CodeExchange[*protocol.IDTokenClaims](authContext, code, party, zrp.WithCodeVerifier(oauth2.GenerateVerifier()))
	var oauthError *oauth2.RetrieveError
	if !errors.As(err, &oauthError) || oauthError.ErrorCode != "invalid_grant" {
		t.Fatal("wrong PKCE did not produce invalid_grant")
	}
	tokens, err := zrp.CodeExchange[*protocol.IDTokenClaims](authContext, code, party, zrp.WithCodeVerifier(verifier))
	if err != nil {
		t.Fatal("second RP exchange failed", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.IDToken == "" || tokens.IDTokenClaims.Subject != test.member || tokens.IDTokenClaims.Nonce != nonce || tokens.IDTokenClaims.AccessTokenHash == "" {
		t.Fatal("second RP token claims incomplete")
	}
	wrongNonce := context.WithValue(t.Context(), expectedNonceContext{}, "different-nonce")
	if _, err = zrp.VerifyIDToken[*protocol.IDTokenClaims](wrongNonce, tokens.IDToken, party.IDTokenVerifier()); !errors.Is(err, protocol.ErrNonceInvalid) {
		t.Fatal("second RP accepted wrong nonce")
	}
	if _, err = zrp.VerifyTokens[*protocol.IDTokenClaims](authContext, "substituted-access", tokens.IDToken, party.IDTokenVerifier()); !errors.Is(err, protocol.ErrAtHash) {
		t.Fatal("second RP accepted substituted access token")
	}
	info, err := zrp.Userinfo[*protocol.UserInfo](t.Context(), tokens.AccessToken, tokens.Type(), test.member, party)
	if err != nil || info.Subject != test.member || info.PreferredUsername != "member" {
		t.Fatal("second RP UserInfo failed")
	}
	if _, err = zrp.Userinfo[*protocol.UserInfo](t.Context(), tokens.AccessToken, tokens.Type(), "other-subject", party); !errors.Is(err, zrp.ErrUserInfoSubNotMatching) {
		t.Fatal("second RP accepted substituted UserInfo subject")
	}
	refreshed, err := zrp.RefreshTokens[*protocol.IDTokenClaims](t.Context(), party, tokens.RefreshToken, "", "")
	if err != nil {
		t.Fatal("second RP refresh failed", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" || refreshed.AccessToken == tokens.AccessToken || refreshed.RefreshToken == tokens.RefreshToken || refreshed.IDTokenClaims == nil || refreshed.IDTokenClaims.Subject != test.member || refreshed.IDTokenClaims.Nonce != "" {
		t.Fatal("second RP refresh contract failed")
	}
	if _, err = zrp.Userinfo[*protocol.UserInfo](t.Context(), refreshed.AccessToken, refreshed.Type(), test.member, party); err != nil {
		t.Fatal("refreshed UserInfo failed")
	}
	query := url.Values{"id_token_hint": {refreshed.IDToken}, "post_logout_redirect_uri": {test.server.URL + "/rp/signed-out"}, "state": {"second-rp-logout"}}
	endpoint := party.GetEndSessionEndpoint()
	if status, _, _ = test.request("GET", endpoint+"?"+query.Encode(), nil, ""); status != 200 {
		t.Fatal("discovered logout confirmation missing")
	}
	if test.introspect(refreshed.AccessToken, "notes-api")["active"] != true {
		t.Fatal("GET logout ended grant")
	}
	if status, _, _ = test.request("POST", endpoint, strings.NewReader(query.Encode()), "application/x-www-form-urlencoded"); status != 403 {
		t.Fatal("logout accepted missing CSRF")
	}
	if test.introspect(refreshed.AccessToken, "notes-api")["active"] != true {
		t.Fatal("rejected logout ended grant")
	}
	query.Set("csrf", test.csrf)
	status, _, headers = test.request("POST", endpoint, strings.NewReader(query.Encode()), "application/x-www-form-urlencoded")
	logout, err := url.Parse(headers.Get("Location"))
	if err != nil || status != 302 || logout.Scheme+"://"+logout.Host+logout.Path != test.server.URL+"/rp/signed-out" || logout.Query().Get("state") != "second-rp-logout" {
		t.Fatal("browser-confirmed logout failed")
	}
	if test.introspect(refreshed.AccessToken, "notes-api")["active"] != false {
		t.Fatal("logout retained access")
	}
	if _, err = zrp.Userinfo[*protocol.UserInfo](t.Context(), refreshed.AccessToken, refreshed.Type(), test.member, party); err == nil {
		t.Fatal("logout retained UserInfo")
	}
	if _, err = zrp.RefreshTokens[*protocol.IDTokenClaims](t.Context(), party, refreshed.RefreshToken, "", ""); err == nil {
		t.Fatal("logout retained refresh")
	}
}
