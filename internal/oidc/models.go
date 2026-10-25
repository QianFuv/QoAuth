// Package oidc connects standard protocol handlers to the local identity store.
package oidc

import (
	"crypto/rsa"
	"net/url"
	"slices"
	"time"

	"github.com/go-jose/go-jose/v4"
	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"qoauth/internal/identity"
	"qoauth/internal/store"
)

type client struct {
	store.Application
	development bool
}

func (c *client) GetID() string                    { return c.ID }
func (c *client) RedirectURIs() []string           { return c.Redirects }
func (c *client) PostLogoutRedirectURIs() []string { return c.LogoutRedirects }
func (c *client) ApplicationType() op.ApplicationType {
	if c.Public {
		return op.ApplicationTypeUserAgent
	}
	return op.ApplicationTypeWeb
}
func (c *client) AuthMethod() protocol.AuthMethod {
	if c.Public {
		return protocol.AuthMethodNone
	}
	return protocol.AuthMethodBasic
}
func (c *client) ResponseTypes() []protocol.ResponseType {
	if c.Resource {
		return nil
	}
	return []protocol.ResponseType{protocol.ResponseTypeCode}
}
func (c *client) GrantTypes() []protocol.GrantType {
	if c.Resource {
		return nil
	}
	return []protocol.GrantType{protocol.GrantTypeCode, protocol.GrantTypeRefreshToken}
}
func (c *client) LoginURL(id string) string           { return "/login?request=" + url.QueryEscape(id) }
func (c *client) AccessTokenType() op.AccessTokenType { return op.AccessTokenTypeBearer }
func (c *client) IDTokenLifetime() time.Duration      { return identity.IDTokenLifetime }
func (c *client) DevMode() bool                       { return c.development }
func (c *client) RestrictAdditionalIdTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (c *client) RestrictAdditionalAccessTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (c *client) IsScopeAllowed(scope string) bool     { return slices.Contains(c.Scopes, scope) }
func (c *client) IDTokenUserinfoClaimsAssertion() bool { return false }
func (c *client) ClockSkew() time.Duration             { return 0 }

type tokenData struct {
	Request          protocol.AuthRequest
	User             string
	AuthTime         time.Time
	Mfa              bool
	Resources        []string
	InitialAuthEvent string
}

func (data *tokenData) GetSubject() string     { return data.User }
func (data *tokenData) GetClientID() string    { return data.Request.ClientID }
func (data *tokenData) GetScopes() []string    { return data.Request.Scopes }
func (data *tokenData) GetAudience() []string  { return []string{data.Request.ClientID} }
func (data *tokenData) GetAuthTime() time.Time { return data.AuthTime }
func (data *tokenData) GetAMR() []string {
	if data.Mfa {
		return []string{"pwd", "otp"}
	}
	return []string{"pwd"}
}

type authRequest struct {
	tokenData
	id       string
	complete bool
}

func (request *authRequest) GetID() string  { return request.id }
func (request *authRequest) GetACR() string { return "" }
func (request *authRequest) GetCodeChallenge() *protocol.CodeChallenge {
	return &protocol.CodeChallenge{Challenge: request.Request.CodeChallenge, Method: request.Request.CodeChallengeMethod}
}
func (request *authRequest) GetNonce() string       { return request.Request.Nonce }
func (request *authRequest) GetRedirectURI() string { return request.Request.RedirectURI }
func (request *authRequest) GetResponseType() protocol.ResponseType {
	return request.Request.ResponseType
}
func (request *authRequest) GetResponseMode() protocol.ResponseMode {
	return request.Request.ResponseMode
}
func (request *authRequest) GetState() string { return request.Request.State }
func (request *authRequest) Done() bool       { return request.complete }

type refreshRequest struct{ tokenData }

func (request *refreshRequest) SetCurrentScopes(scopes []string) {
	request.Request.Scopes = slices.Clone(scopes)
}

type signingKey struct {
	private *rsa.PrivateKey
	verify  *rsa.PublicKey
	id      string
	public  bool
}

func (key signingKey) ID() string                                  { return key.id }
func (key signingKey) SignatureAlgorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (key signingKey) Algorithm() jose.SignatureAlgorithm          { return jose.RS256 }
func (key signingKey) Use() string                                 { return "sig" }
func (key signingKey) Key() any {
	if key.verify != nil {
		return key.verify
	}
	if key.public {
		return &key.private.PublicKey
	}
	return key.private
}
