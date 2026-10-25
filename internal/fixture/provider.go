package fixture

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"qoauth/internal/store"
	"qoauth/internal/webassets"
)

// Keys are ephemeral fixture keys; they have no serialization or persistence API.
type Keys struct {
	private    *rsa.PrivateKey
	encryption [32]byte
}

// NewKeys uses standard crypto/rand and RSA solely for isolated synthetic fixtures.
func NewKeys() (*Keys, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	keys := &Keys{private: private}
	_, _ = rand.Read(keys.encryption[:])
	return keys, nil
}

// Config binds a synthetic fixture to loopback, temporary state and ephemeral keys.
type Config struct {
	Issuer, Callback string
	Keys             *Keys
	Secrets          map[string]string
}

// Secrets creates in-memory credentials for synthetic clients, not real bindings.
func Secrets() map[string]string {
	return map[string]string{"web-rp": store.NewHandle(), "other-rp": store.NewHandle(), "notes-api-validator": store.NewHandle(), "wrong-api-validator": store.NewHandle()}
}

type adapter struct {
	database *store.Store
	clients  map[string]*client
	keys     *Keys
}

// New constructs the real library OP with the SQLite adapter and embedded login UI.
// Only loopback fixture issuers/callbacks are accepted. The production binary does
// not import this package or offer a fixture flag.
func New(ctx context.Context, database *store.Store, config Config) (http.Handler, error) {
	for _, raw := range []string{config.Issuer, config.Callback} {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("invalid fixture URL")
		}
		ip := net.ParseIP(parsed.Hostname())
		if ip == nil || !ip.IsLoopback() {
			return nil, errors.New("fixture requires a loopback IP")
		}
	}
	if config.Keys == nil {
		return nil, errors.New("ephemeral keys required")
	}
	storage := &adapter{database: database, keys: config.Keys, clients: map[string]*client{}}
	ids := []string{"web-rp", "other-rp", "public-rp", "notes-api-validator", "wrong-api-validator"}
	for _, id := range ids {
		isPublic := id == "public-rp"
		isResource := strings.HasSuffix(id, "validator")
		if !isPublic && len(config.Secrets[id]) < 32 {
			return nil, errors.New("synthetic confidential client secret required")
		}
		storage.clients[id] = &client{id: id, secret: config.Secrets[id], callback: config.Callback, public: isPublic, resource: isResource}
	}
	if err := database.SeedFixture(ctx, Subject, ids); err != nil {
		return nil, err
	}
	provider, err := op.NewProvider(&op.Config{
		CryptoKey: config.Keys.encryption, CodeMethodS256: true, GrantTypeRefreshToken: true,
		SupportedScopes: []string{"openid", "profile", "email", "offline_access", "notes.read"},
		SupportedClaims: []string{"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "azp", "amr", "at_hash", "c_hash", "name", "preferred_username", "email", "email_verified"},
	}, storage, op.StaticIssuer(config.Issuer), op.WithAllowInsecure(), op.WithCrypto(op.NewAES256GCMCrypto(config.Keys.encryption, "m1-ephemeral")))
	if err != nil {
		return nil, err
	}
	assets := webassets.Handler()
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		request.Body = http.MaxBytesReader(response, request.Body, 64<<10)
		if request.URL.Path == provider.AuthorizationEndpoint().Relative() {
			if err := request.ParseForm(); err != nil {
				op.RequestError(response, request, oidc.ErrInvalidRequest(), nil)
				return
			}
			if configured := storage.clients[request.Form.Get("client_id")]; configured != nil {
				for _, scope := range strings.Fields(request.Form.Get("scope")) {
					if !configured.IsScopeAllowed(scope) {
						op.RequestError(response, request, oidc.ErrInvalidScope(), nil)
						return
					}
				}
			}
		}
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/assets/") {
			assets.ServeHTTP(response, request)
			return
		}
		if request.URL.Path == "/.well-known/openid-configuration" {
			metadata := op.CreateDiscoveryConfig(op.ContextWithIssuer(request.Context(), config.Issuer), provider, storage)
			metadata.ResponseTypesSupported = []string{"code"}
			metadata.GrantTypesSupported = []oidc.GrantType{oidc.GrantTypeCode, oidc.GrantTypeRefreshToken}
			metadata.CodeChallengeMethodsSupported = []oidc.CodeChallengeMethod{oidc.CodeChallengeMethodS256}
			metadata.IntrospectionEndpointAuthMethodsSupported = []oidc.AuthMethod{oidc.AuthMethodBasic}
			metadata.IntrospectionEndpointAuthSigningAlgValuesSupported = nil
			metadata.RevocationEndpointAuthMethodsSupported = []oidc.AuthMethod{oidc.AuthMethodNone, oidc.AuthMethodBasic}
			metadata.RevocationEndpointAuthSigningAlgValuesSupported = nil
			metadata.DeviceAuthorizationEndpoint = ""
			metadata.CheckSessionIframe = ""
			op.Discover(response, metadata)
			return
		}
		if request.URL.Path == "/fixture/login" {
			if request.Method == http.MethodGet {
				id := request.URL.Query().Get("request")
				if _, err := database.RequestByID(request.Context(), id); err != nil {
					http.Error(response, "invalid transaction", 400)
					return
				}
				http.SetCookie(response, &http.Cookie{Name: "m1_transaction", Value: id, Path: "/fixture/login", HttpOnly: true, Secure: strings.HasPrefix(config.Issuer, "https:"), SameSite: http.SameSiteStrictMode, MaxAge: 60})
				assets.ServeHTTP(response, request)
				return
			}
			if request.Method != http.MethodPost {
				http.Error(response, "method not allowed", 405)
				return
			}
			if err := request.ParseForm(); err != nil {
				http.Error(response, "invalid form", 400)
				return
			}
			id := request.PostForm.Get("request")
			cookie, err := request.Cookie("m1_transaction")
			if err != nil || cookie.Value != id || request.Header.Get("Origin") != config.Issuer {
				http.Error(response, "invalid fixture origin/transaction", 403)
				return
			}
			record, err := database.RequestByID(request.Context(), id)
			if err != nil {
				http.Error(response, "invalid transaction", 400)
				return
			}
			var data tokenData
			if json.Unmarshal(record.Payload, &data) != nil {
				http.Error(response, "invalid transaction", 400)
				return
			}
			data.AuthTime = time.Now().UTC()
			payload, err := json.Marshal(data)
			if err != nil || database.AuthenticateFixture(request.Context(), id, payload) != nil {
				http.Error(response, "invalid transaction", 400)
				return
			}
			callback := op.AuthCallbackURL(provider)(op.ContextWithIssuer(request.Context(), config.Issuer), id)
			http.Redirect(response, request, callback, http.StatusSeeOther)
			return
		}
		if request.URL.Path == "/oauth/token" {
			if request.Method != http.MethodPost {
				http.Error(response, "method not allowed", 405)
				return
			}
			if err := request.ParseForm(); err != nil {
				op.RequestError(response, request, oidc.ErrInvalidRequest(), nil)
				return
			}
			if grant := request.PostForm.Get("grant_type"); grant != "authorization_code" && grant != "refresh_token" {
				op.RequestError(response, request, oidc.ErrUnsupportedGrantType(), nil)
				return
			}
			if request.PostForm.Get("client_secret") != "" || request.PostForm.Get("client_assertion") != "" {
				op.RequestError(response, request, oidc.ErrInvalidClient(), nil)
				return
			}
		}
		if request.URL.Path == "/device_authorization" {
			http.NotFound(response, request)
			return
		}
		if request.URL.Path == provider.RevocationEndpoint().Relative() {
			if err := request.ParseForm(); err != nil || request.PostForm.Get("client_secret") != "" || request.PostForm.Get("client_assertion") != "" {
				op.RequestError(response, request, oidc.ErrInvalidClient(), nil)
				return
			}
		}
		provider.ServeHTTP(response, request)
	}), nil
}

func (storage *adapter) CreateAuthRequest(ctx context.Context, request *oidc.AuthRequest, _ string) (op.AuthRequest, error) {
	configured := storage.clients[request.ClientID]
	if configured == nil || request.CodeChallenge == "" || request.CodeChallengeMethod != oidc.CodeChallengeMethodS256 || request.Nonce == "" || request.State == "" {
		return nil, oidc.ErrInvalidRequest()
	}
	for _, scope := range request.Scopes {
		if !configured.IsScopeAllowed(scope) {
			return nil, oidc.ErrInvalidScope()
		}
	}
	if slices.Contains(request.Prompt, oidc.PromptNone) {
		return nil, oidc.ErrLoginRequired()
	}
	data := tokenData{Request: *request, User: Subject, Resources: []string{Audience}}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	record := store.Request{ID: store.NewHandle(), UserID: Subject, ClientID: request.ClientID, Payload: payload, Expires: time.Now().Add(time.Minute)}
	if err = storage.database.CreateRequest(ctx, record); err != nil {
		return nil, err
	}
	return &authRequest{tokenData: data, id: record.ID}, nil
}

func requestFromRecord(record store.Request, err error) (op.AuthRequest, error) {
	if err != nil {
		return nil, err
	}
	var data tokenData
	if err = json.Unmarshal(record.Payload, &data); err != nil {
		return nil, err
	}
	return &authRequest{tokenData: data, id: record.ID, complete: record.Ready}, nil
}
func (storage *adapter) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	return requestFromRecord(storage.database.RequestByID(ctx, id))
}
func (storage *adapter) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	return requestFromRecord(storage.database.RequestByCode(ctx, code))
}
func (storage *adapter) SaveAuthCode(ctx context.Context, id, code string) error {
	return storage.database.SaveCode(ctx, id, code)
}
func (storage *adapter) DeleteAuthRequest(context.Context, string) error { return nil }

func (storage *adapter) issue(ctx context.Context, request op.TokenRequest, current string, refresh bool) (store.Issued, error) {
	var result store.Issued
	var err error
	switch typed := request.(type) {
	case *authRequest:
		payload, marshalErr := json.Marshal(typed.tokenData)
		if marshalErr != nil {
			return result, marshalErr
		}
		result, err = storage.database.ExchangeCode(ctx, typed.id, typed.GetClientID(), payload, refresh)
	case *refreshRequest:
		payload, marshalErr := json.Marshal(typed.tokenData)
		if marshalErr != nil {
			return result, marshalErr
		}
		result, err = storage.database.Rotate(ctx, current, typed.GetClientID(), payload)
	default:
		err = store.ErrInvalid
	}
	if errors.Is(err, store.ErrInvalid) || errors.Is(err, store.ErrReplay) {
		return store.Issued{}, oidc.ErrInvalidGrant()
	}
	return result, err
}
func (storage *adapter) CreateAccessToken(ctx context.Context, request op.TokenRequest) (string, time.Time, error) {
	result, err := storage.issue(ctx, request, "", false)
	return result.Access, result.Expires, err
}
func (storage *adapter) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, current string) (string, string, time.Time, error) {
	result, err := storage.issue(ctx, request, current, true)
	return result.Access, result.Refresh, result.Expires, err
}
func (storage *adapter) TokenRequestByRefreshToken(ctx context.Context, handle string) (op.RefreshTokenRequest, error) {
	token, err := storage.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "refresh" {
		return nil, op.ErrInvalidRefreshToken
	}
	var data tokenData
	if err = json.Unmarshal(token.Payload, &data); err != nil {
		return nil, err
	}
	return &refreshRequest{tokenData: data}, nil
}
func (storage *adapter) TerminateSession(ctx context.Context, user, client string) error {
	return storage.database.EndSession(ctx, user, client)
}
func (storage *adapter) RevokeToken(ctx context.Context, handle, _ string, client string) *oidc.Error {
	if err := storage.database.Revoke(ctx, handle, client); err != nil {
		return oidc.ErrServerError()
	}
	return nil
}
func (storage *adapter) GetRefreshTokenInfo(ctx context.Context, client, handle string) (string, string, error) {
	token, err := storage.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "refresh" || token.ClientID != client {
		return "", "", op.ErrInvalidRefreshToken
	}
	return token.UserID, handle, nil
}
func (storage *adapter) SigningKey(context.Context) (op.SigningKey, error) {
	return signingKey{private: storage.keys.private}, nil
}
func (storage *adapter) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}
func (storage *adapter) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{signingKey{private: storage.keys.private, public: true}}, nil
}
func (storage *adapter) Health(ctx context.Context) error { return storage.database.Health(ctx) }
func (storage *adapter) GetClientByClientID(ctx context.Context, id string) (op.Client, error) {
	configured := storage.clients[id]
	if configured == nil || !storage.database.ClientActive(ctx, id) {
		return nil, oidc.ErrInvalidClient()
	}
	return configured, nil
}
func (storage *adapter) AuthorizeClientIDSecret(ctx context.Context, id, secret string) error {
	configured := storage.clients[id]
	if configured == nil || configured.public || !storage.database.ClientActive(ctx, id) || subtle.ConstantTimeCompare([]byte(store.Hash(secret)), []byte(store.Hash(configured.secret))) != 1 {
		return oidc.ErrInvalidClient()
	}
	return nil
}
func (storage *adapter) SetUserinfoFromScopes(context.Context, *oidc.UserInfo, string, string, []string) error {
	return nil
}
func fillUserinfo(info *oidc.UserInfo, subject string, scopes []string) {
	info.Subject = subject
	if slices.Contains(scopes, "profile") {
		info.PreferredUsername = "synthetic-user"
		info.Name = "M1 synthetic user"
	}
	if slices.Contains(scopes, "email") {
		info.Email = "fixture@example.invalid"
		info.EmailVerified = false
	}
}
func (storage *adapter) SetUserinfoFromRequest(_ context.Context, info *oidc.UserInfo, request op.IDTokenRequest, scopes []string) error {
	fillUserinfo(info, request.GetSubject(), scopes)
	return nil
}
func (storage *adapter) SetUserinfoFromToken(ctx context.Context, info *oidc.UserInfo, handle, subject, _ string) error {
	token, err := storage.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "access" || token.UserID != subject {
		return store.ErrInvalid
	}
	var data tokenData
	if err = json.Unmarshal(token.Payload, &data); err != nil {
		return err
	}
	if !slices.Contains(data.Request.Scopes, "openid") {
		return store.ErrInvalid
	}
	fillUserinfo(info, subject, data.Request.Scopes)
	return nil
}
func (storage *adapter) SetIntrospectionFromToken(ctx context.Context, info *oidc.IntrospectionResponse, handle, subject, caller string) error {
	token, err := storage.database.Lookup(ctx, handle)
	if err != nil || token.Kind != "access" || token.UserID != subject || caller != "notes-api-validator" {
		return store.ErrInvalid
	}
	var data tokenData
	if err = json.Unmarshal(token.Payload, &data); err != nil {
		return err
	}
	if !slices.Contains(data.Resources, Audience) {
		return store.ErrInvalid
	}
	info.Subject = subject
	info.ClientID = token.ClientID
	info.Audience = data.Resources
	info.Scope = data.Request.Scopes
	info.Expiration = oidc.FromTime(token.Expires)
	info.TokenType = "Bearer"
	return nil
}
func (storage *adapter) GetPrivateClaimsFromScopes(context.Context, string, string, []string) (map[string]any, error) {
	return nil, nil
}
func (storage *adapter) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, oidc.ErrInvalidClient()
}
func (storage *adapter) ValidateJWTProfileScopes(context.Context, string, []string) ([]string, error) {
	return nil, oidc.ErrUnsupportedGrantType()
}

var _ op.Storage = (*adapter)(nil)
