package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/go-jose/go-jose/v4"
	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"qoauth/internal/store"
)

type sessionContext struct{}

func currentSession(ctx context.Context) store.Session {
	session, _ := ctx.Value(sessionContext{}).(store.Session)
	return session
}

type consent struct{ Scopes, Audiences []string }

func (s *Server) hasConsent(ctx context.Context, user string, data tokenData) bool {
	payload, err := s.database.Consent(ctx, user, data.Request.ClientID)
	if err != nil {
		return false
	}
	var approved consent
	if json.Unmarshal(payload, &approved) != nil {
		return false
	}
	for _, scope := range data.Request.Scopes {
		if !slices.Contains(approved.Scopes, scope) {
			return false
		}
	}
	for _, audience := range data.Resources {
		if !slices.Contains(approved.Audiences, audience) {
			return false
		}
	}
	return true
}
func (s *Server) eligible(ctx context.Context, session store.Session, data tokenData) bool {
	if session.UserID == "" || !session.Assurance {
		return false
	}
	account, err := s.database.Account(ctx, session.UserID)
	if err != nil || !account.Enabled || (account.Admin && len(account.Factor) == 0) {
		return false
	}
	if data.Request.MaxAge != nil {
		if *data.Request.MaxAge == 0 {
			return session.Digest != data.InitialSession
		}
		if uint64(time.Now().Unix()-session.AuthTime) > uint64(*data.Request.MaxAge) {
			return false
		}
	}
	return true
}
func (s *Server) CreateAuthRequest(ctx context.Context, request *protocol.AuthRequest, _ string) (op.AuthRequest, error) {
	if request.CodeChallenge == "" || request.CodeChallengeMethod != protocol.CodeChallengeMethodS256 || request.Nonce == "" || request.State == "" {
		return nil, protocol.ErrInvalidRequest()
	}
	app, err := s.database.Application(ctx, request.ClientID)
	if err != nil {
		return nil, protocol.ErrInvalidClient()
	}
	session := currentSession(ctx)
	if session.Digest == "" {
		return nil, protocol.ErrServerError()
	}
	data := tokenData{Request: *request, Resources: app.Audiences, InitialSession: session.Digest}
	if slices.Contains(request.Prompt, protocol.PromptNone) {
		if !s.eligible(ctx, session, data) {
			return nil, protocol.ErrLoginRequired()
		}
		if !s.hasConsent(ctx, session.UserID, data) {
			return nil, &protocol.Error{ErrorType: "consent_required"}
		}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	id := store.NewHandle()
	err = s.database.CreateLogin(ctx, store.LoginRequest{ID: id, ClientID: app.ID, Browser: session.Digest, Payload: payload, Expires: time.Now().Add(5 * time.Minute).Unix()})
	return &authRequest{tokenData: data, id: id}, err
}
func requestFromRecord(record store.Request, err error) (op.AuthRequest, error) {
	if err != nil {
		return nil, protocol.ErrInvalidGrant()
	}
	var data tokenData
	if json.Unmarshal(record.Payload, &data) != nil {
		return nil, protocol.ErrServerError()
	}
	return &authRequest{tokenData: data, id: record.ID, complete: record.Ready}, nil
}
func (s *Server) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	return requestFromRecord(s.database.RequestByID(ctx, id))
}
func (s *Server) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return requestFromRecord(s.database.RequestByCode(ctx, code))
}
func (s *Server) SaveAuthCode(ctx context.Context, id, code string) error {
	return s.database.SaveCode(ctx, id, code)
}
func (s *Server) DeleteAuthRequest(context.Context, string) error { return nil }
func (s *Server) issue(ctx context.Context, request op.TokenRequest, current string, refresh bool) (store.Issued, error) {
	var issued store.Issued
	var err error
	switch typed := request.(type) {
	case *authRequest:
		payload, marshalErr := json.Marshal(typed.tokenData)
		if marshalErr != nil {
			return issued, marshalErr
		}
		issued, err = s.database.ExchangeCode(ctx, typed.id, typed.GetClientID(), payload, refresh)
	case *refreshRequest:
		payload, marshalErr := json.Marshal(typed.tokenData)
		if marshalErr != nil {
			return issued, marshalErr
		}
		issued, err = s.database.Rotate(ctx, current, typed.GetClientID(), payload)
	default:
		err = store.ErrInvalid
	}
	if err != nil {
		if errors.Is(err, store.ErrInvalid) || errors.Is(err, store.ErrReplay) {
			return store.Issued{}, protocol.ErrInvalidGrant()
		}
		return store.Issued{}, protocol.ErrServerError()
	}
	return issued, nil
}
func (s *Server) CreateAccessToken(ctx context.Context, request op.TokenRequest) (string, time.Time, error) {
	issued, err := s.issue(ctx, request, "", false)
	return issued.Access, issued.Expires, err
}
func (s *Server) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, current string) (string, string, time.Time, error) {
	issued, err := s.issue(ctx, request, current, true)
	return issued.Access, issued.Refresh, issued.Expires, err
}
func (s *Server) TokenRequestByRefreshToken(ctx context.Context, handle string) (op.RefreshTokenRequest, error) {
	token, err := s.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "refresh" {
		return nil, op.ErrInvalidRefreshToken
	}
	var data tokenData
	if json.Unmarshal(token.Payload, &data) != nil {
		return nil, op.ErrInvalidRefreshToken
	}
	return &refreshRequest{tokenData: data}, nil
}
func (s *Server) TerminateSession(context.Context, string, string) error {
	return protocol.ErrInvalidRequest()
}
func (s *Server) TerminateSessionFromRequest(ctx context.Context, request *op.EndSessionRequest) (string, error) {
	session := currentSession(ctx)
	if session.UserID == "" || request.UserID != session.UserID || request.IDTokenHintClaims == nil {
		return "", protocol.ErrInvalidRequest()
	}
	if err := s.database.RevokeSession(ctx, session.Digest, session.UserID); err != nil {
		return "", protocol.ErrServerError()
	}
	return request.RedirectURI, nil
}
func (s *Server) RevokeToken(ctx context.Context, handle, _ string, client string) *protocol.Error {
	if err := s.database.Revoke(ctx, handle, client); err != nil {
		return protocol.ErrServerError()
	}
	return nil
}
func (s *Server) GetRefreshTokenInfo(ctx context.Context, client, handle string) (string, string, error) {
	token, err := s.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "refresh" || token.ClientID != client {
		return "", "", op.ErrInvalidRefreshToken
	}
	return token.UserID, handle, nil
}
func (s *Server) SigningKey(context.Context) (op.SigningKey, error) {
	return signingKey{private: s.keys.Signing, id: s.keys.ID}, nil
}
func (s *Server) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}
func (s *Server) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{signingKey{private: s.keys.Signing, id: s.keys.ID, public: true}}, nil
}
func (s *Server) Health(ctx context.Context) error { return s.database.Health(ctx) }
func (s *Server) GetClientByClientID(ctx context.Context, id string) (op.Client, error) {
	app, err := s.database.Application(ctx, id)
	if err != nil {
		return nil, protocol.ErrInvalidClient()
	}
	return &client{Application: app, development: s.config.DevelopmentHTTP}, nil
}
func (s *Server) AuthorizeClientIDSecret(ctx context.Context, id, secret string) error {
	app, err := s.database.Application(ctx, id)
	if err != nil || app.Public || subtle.ConstantTimeCompare([]byte(store.Hash(secret)), []byte(app.SecretHash)) != 1 {
		return protocol.ErrInvalidClient()
	}
	return nil
}
func (s *Server) SetUserinfoFromScopes(context.Context, *protocol.UserInfo, string, string, []string) error {
	return nil
}
func (s *Server) fillUserinfo(ctx context.Context, info *protocol.UserInfo, user string, scopes []string) error {
	account, err := s.database.Account(ctx, user)
	if err != nil || !account.Enabled {
		return store.ErrInvalid
	}
	info.Subject = user
	if slices.Contains(scopes, "profile") {
		info.PreferredUsername = account.Username
		info.Name = account.Username
	}
	return nil
}
func (s *Server) SetUserinfoFromRequest(ctx context.Context, info *protocol.UserInfo, request op.IDTokenRequest, scopes []string) error {
	return s.fillUserinfo(ctx, info, request.GetSubject(), scopes)
}
func (s *Server) SetUserinfoFromToken(ctx context.Context, info *protocol.UserInfo, handle, user, _ string) error {
	token, err := s.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "access" || token.UserID != user {
		return store.ErrInvalid
	}
	var data tokenData
	if json.Unmarshal(token.Payload, &data) != nil || !slices.Contains(data.Request.Scopes, "openid") {
		return store.ErrInvalid
	}
	return s.fillUserinfo(ctx, info, user, data.Request.Scopes)
}
func (s *Server) SetIntrospectionFromToken(ctx context.Context, info *protocol.IntrospectionResponse, handle, user, caller string) error {
	token, err := s.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "access" || token.UserID != user {
		return store.ErrInvalid
	}
	validator, err := s.database.Application(ctx, caller)
	if err != nil || !validator.Resource {
		return store.ErrInvalid
	}
	var data tokenData
	if json.Unmarshal(token.Payload, &data) != nil || len(data.Resources) == 0 {
		return store.ErrInvalid
	}
	for _, audience := range data.Resources {
		if !slices.Contains(validator.Audiences, audience) {
			return store.ErrInvalid
		}
	}
	info.Subject = user
	info.ClientID = token.ClientID
	info.Audience = data.Resources
	info.Scope = data.Request.Scopes
	info.Expiration = protocol.FromTime(token.Expires)
	info.TokenType = "Bearer"
	return nil
}
func (s *Server) GetPrivateClaimsFromScopes(context.Context, string, string, []string) (map[string]any, error) {
	return nil, nil
}
func (s *Server) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, protocol.ErrInvalidClient()
}
func (s *Server) ValidateJWTProfileScopes(context.Context, string, []string) ([]string, error) {
	return nil, protocol.ErrUnsupportedGrantType()
}

var _ op.Storage = (*Server)(nil)
var _ op.CanTerminateSessionFromRequest = (*Server)(nil)
