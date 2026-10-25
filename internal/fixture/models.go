// Package fixture is a synthetic M1 feasibility harness, never a production login.
// Its identities and secrets must only be used with isolated local test state.
package fixture

import (
	"crypto/rsa"
	"net/url"
	"slices"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const Subject = "synthetic-m1-user"
const Audience = "notes-api"

type client struct {
	id, secret, callback string
	public, resource     bool
}

func (client *client) GetID() string                    { return client.id }
func (client *client) RedirectURIs() []string           { return []string{client.callback} }
func (client *client) PostLogoutRedirectURIs() []string { return []string{client.callback} }
func (client *client) ApplicationType() op.ApplicationType {
	if client.public {
		return op.ApplicationTypeUserAgent
	}
	return op.ApplicationTypeWeb
}
func (client *client) AuthMethod() oidc.AuthMethod {
	if client.public {
		return oidc.AuthMethodNone
	}
	return oidc.AuthMethodBasic
}
func (client *client) ResponseTypes() []oidc.ResponseType {
	if client.resource {
		return nil
	}
	return []oidc.ResponseType{oidc.ResponseTypeCode}
}
func (client *client) GrantTypes() []oidc.GrantType {
	if client.resource {
		return nil
	}
	return []oidc.GrantType{oidc.GrantTypeCode, oidc.GrantTypeRefreshToken}
}
func (client *client) LoginURL(id string) string {
	return "/fixture/login?request=" + url.QueryEscape(id)
}
func (client *client) AccessTokenType() op.AccessTokenType { return op.AccessTokenTypeBearer }
func (client *client) IDTokenLifetime() time.Duration      { return 5 * time.Minute }
func (client *client) DevMode() bool                       { return true }
func (client *client) RestrictAdditionalIdTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (client *client) RestrictAdditionalAccessTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (client *client) IsScopeAllowed(scope string) bool {
	return slices.Contains([]string{"openid", "profile", "email", "offline_access", "notes.read"}, scope)
}
func (client *client) IDTokenUserinfoClaimsAssertion() bool { return false }
func (client *client) ClockSkew() time.Duration             { return 0 }

type tokenData struct {
	Request   oidc.AuthRequest
	User      string
	AuthTime  time.Time
	Resources []string
}

func (data *tokenData) GetSubject() string     { return data.User }
func (data *tokenData) GetClientID() string    { return data.Request.ClientID }
func (data *tokenData) GetScopes() []string    { return data.Request.Scopes }
func (data *tokenData) GetAudience() []string  { return []string{data.Request.ClientID} }
func (data *tokenData) GetAuthTime() time.Time { return data.AuthTime }
func (data *tokenData) GetAMR() []string       { return []string{"m1_fixture"} }

type authRequest struct {
	tokenData
	id       string
	complete bool
}

func (request *authRequest) GetID() string  { return request.id }
func (request *authRequest) GetACR() string { return "" }
func (request *authRequest) GetCodeChallenge() *oidc.CodeChallenge {
	return &oidc.CodeChallenge{Challenge: request.Request.CodeChallenge, Method: request.Request.CodeChallengeMethod}
}
func (request *authRequest) GetNonce() string                   { return request.Request.Nonce }
func (request *authRequest) GetRedirectURI() string             { return request.Request.RedirectURI }
func (request *authRequest) GetResponseType() oidc.ResponseType { return request.Request.ResponseType }
func (request *authRequest) GetResponseMode() oidc.ResponseMode { return request.Request.ResponseMode }
func (request *authRequest) GetState() string                   { return request.Request.State }
func (request *authRequest) Done() bool                         { return request.complete }

type refreshRequest struct{ tokenData }

func (request *refreshRequest) SetCurrentScopes(scopes []string) {
	request.Request.Scopes = slices.Clone(scopes)
}

type signingKey struct {
	private *rsa.PrivateKey
	public  bool
}

func (key signingKey) ID() string                                  { return "m1-ephemeral" }
func (key signingKey) SignatureAlgorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (key signingKey) Algorithm() jose.SignatureAlgorithm          { return jose.RS256 }
func (key signingKey) Use() string                                 { return "sig" }
func (key signingKey) Key() any {
	if key.public {
		return &key.private.PublicKey
	}
	return key.private
}
