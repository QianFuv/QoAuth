package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"qoauth/internal/identity"
	"qoauth/internal/store"
	"qoauth/internal/webassets"
)

// Config requires a canonical origin; plain HTTP is an explicit loopback-only mode.
type Config struct {
	Issuer          string
	DevelopmentHTTP bool
}

func (config Config) Validate() error {
	parsed, err := url.Parse(config.Issuer)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("issuer must be a canonical origin without a path")
	}
	if parsed.Scheme == "https" && !config.DevelopmentHTTP {
		return nil
	}
	ip := net.ParseIP(parsed.Hostname())
	if config.DevelopmentHTTP && parsed.Scheme == "http" && ip != nil && ip.IsLoopback() {
		return nil
	}
	return errors.New("HTTPS required; development HTTP permits numeric loopback origins only")
}

// Server uses production account/session state and never imports test fixtures.
type Server struct {
	database      *store.Store
	keys          *identity.Keys
	config        Config
	provider      *op.Provider
	assets        http.Handler
	dummyPassword string
	limits        limiter
}

func New(ctx context.Context, database *store.Store, keys *identity.Keys, config Config) (*Server, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if keys == nil {
		return nil, errors.New("installation keys required")
	}
	if err := database.CheckInstallation(ctx, keys.Binding()); err != nil {
		return nil, err
	}
	dummy, err := identity.PasswordHash(store.NewHandle())
	if err != nil {
		return nil, err
	}
	server := &Server{database: database, keys: keys, config: config, assets: webassets.Handler(), dummyPassword: dummy, limits: limiter{entries: map[string]attempts{}}}
	options := []op.Option{op.WithCrypto(op.NewAES256GCMCrypto(keys.Encryption, keys.ID))}
	if config.DevelopmentHTTP {
		options = append(options, op.WithAllowInsecure())
	}
	server.provider, err = op.NewProvider(&op.Config{CryptoKey: keys.Encryption, CodeMethodS256: true, GrantTypeRefreshToken: true, DefaultLogoutRedirectURI: config.Issuer + "/", SupportedScopes: []string{"openid", "profile", "offline_access"}, SupportedClaims: []string{"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "azp", "amr", "at_hash", "c_hash", "name", "preferred_username"}}, server, op.StaticIssuer(config.Issuer), options...)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *Server) cookieName() string {
	if s.config.DevelopmentHTTP {
		return "qoauth_dev_session"
	}
	return "__Host-qoauth"
}
func (s *Server) setCookie(response http.ResponseWriter, handle string) {
	http.SetCookie(response, &http.Cookie{Name: s.cookieName(), Value: handle, Path: "/", Secure: !s.config.DevelopmentHTTP, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 43200})
}
func (s *Server) session(request *http.Request) (store.Session, error) {
	cookie, err := request.Cookie(s.cookieName())
	if err != nil {
		return store.Session{}, store.ErrInvalid
	}
	return s.database.Session(request.Context(), cookie.Value)
}
func (s *Server) browserContext(response http.ResponseWriter, request *http.Request, create bool) (*http.Request, error) {
	session, err := s.session(request)
	if err != nil && create {
		remote, _, _ := net.SplitHostPort(request.RemoteAddr)
		if !s.limits.allow("anonymous:"+remote, 60) {
			return request, store.ErrInvalid
		}
		handle, createErr := s.database.AnonymousSession(request.Context())
		if createErr != nil {
			return request, createErr
		}
		s.setCookie(response, handle)
		session, err = s.database.Session(request.Context(), handle)
	}
	if err != nil {
		return request, err
	}
	return request.WithContext(context.WithValue(request.Context(), sessionContext{}, session)), nil
}
func (s *Server) csrf(request *http.Request) bool {
	return request.Header.Get("Origin") == s.config.Issuer && subtle.ConstantTimeCompare([]byte(request.Header.Get("X-CSRF-Token")), []byte(currentSession(request.Context()).Csrf)) == 1 && currentSession(request.Context()).Csrf != ""
}
func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	request = request.WithContext(op.ContextWithIssuer(request.Context(), s.config.Issuer))
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	request.Body = http.MaxBytesReader(response, request.Body, 64<<10)
	origin, _ := url.Parse(s.config.Issuer)
	if request.Host != origin.Host {
		http.Error(response, "invalid host", 400)
		return
	}
	path := request.URL.Path
	if path == "/authorize/callback" && request.Method != http.MethodGet {
		http.Error(response, "method not allowed", 405)
		return
	}
	if path == "/authorize" || path == "/authorize/callback" || path == "/oauth/token" || path == "/revoke" || path == "/oauth/introspect" || path == "/end_session" {
		if _, err := url.ParseQuery(request.URL.RawQuery); err != nil || request.ParseForm() != nil || duplicateParameters(request.Form) {
			http.Error(response, "ambiguous protocol parameters", 400)
			return
		}
	}
	if path == "/.well-known/openid-configuration" {
		metadata := op.CreateDiscoveryConfig(op.ContextWithIssuer(request.Context(), s.config.Issuer), s.provider, s)
		metadata.ResponseTypesSupported = []string{"code"}
		metadata.GrantTypesSupported = []protocol.GrantType{protocol.GrantTypeCode, protocol.GrantTypeRefreshToken}
		metadata.CodeChallengeMethodsSupported = []protocol.CodeChallengeMethod{protocol.CodeChallengeMethodS256}
		metadata.IntrospectionEndpointAuthMethodsSupported = []protocol.AuthMethod{protocol.AuthMethodBasic}
		metadata.IntrospectionEndpointAuthSigningAlgValuesSupported = nil
		metadata.RevocationEndpointAuthMethodsSupported = []protocol.AuthMethod{protocol.AuthMethodNone, protocol.AuthMethodBasic}
		metadata.RevocationEndpointAuthSigningAlgValuesSupported = nil
		metadata.DeviceAuthorizationEndpoint = ""
		metadata.CheckSessionIframe = ""
		op.Discover(response, metadata)
		return
	}
	if path == "/authorize" || path == "/authorize/callback" || path == "/login" || strings.HasPrefix(path, "/api/") || path == "/end_session" {
		var err error
		request, err = s.browserContext(response, request, request.Method == http.MethodGet && (path == "/api/session" || path == "/authorize" || path == "/login"))
		if err != nil {
			http.Error(response, "session required", 401)
			return
		}
	}
	if strings.HasPrefix(path, "/api/") {
		s.api(response, request)
		return
	}
	if path == "/end_session" {
		if request.ParseForm() != nil {
			http.Error(response, "invalid logout", 400)
			return
		}
		parsed, err := op.ParseEndSessionRequest(request, s.provider.Decoder())
		if err != nil {
			http.Error(response, "invalid logout", 400)
			return
		}
		logout, err := op.ValidateEndSessionRequest(request.Context(), parsed, s.provider)
		if err != nil || logout.IDTokenHintClaims == nil || logout.UserID != currentSession(request.Context()).UserID {
			http.Error(response, "invalid logout context", 400)
			return
		}
		if request.Method == http.MethodGet {
			copy := request.Clone(request.Context())
			copy.URL.Path = "/"
			s.assets.ServeHTTP(response, copy)
			return
		}
		if request.Method != http.MethodPost || request.Header.Get("Origin") != s.config.Issuer || subtle.ConstantTimeCompare([]byte(request.PostForm.Get("csrf")), []byte(currentSession(request.Context()).Csrf)) != 1 {
			http.Error(response, "invalid logout proof", 403)
			return
		}
	}
	if path == "/login" {
		s.loginPage(response, request)
		return
	}
	if path == "/authorize/callback" {
		if len(request.URL.Query()) != 1 || len(request.URL.Query()["id"]) != 1 {
			http.Error(response, "invalid authorization transaction", 400)
			return
		}
		login, err := s.database.Login(request.Context(), request.URL.Query().Get("id"))
		if err != nil || !login.Completed || login.Browser != currentSession(request.Context()).Digest {
			http.Error(response, "invalid authorization transaction", 400)
			return
		}
	}
	if path == "/authorize" {
		if request.ParseForm() != nil {
			op.RequestError(response, request, protocol.ErrInvalidRequest(), nil)
			return
		}
		app, err := s.database.Application(request.Context(), request.Form.Get("client_id"))
		if err != nil {
			op.RequestError(response, request, protocol.ErrInvalidClient(), nil)
			return
		}
		for _, scope := range strings.Fields(request.Form.Get("scope")) {
			if !contains(app.Scopes, scope) {
				op.RequestError(response, request, protocol.ErrInvalidScope(), nil)
				return
			}
		}
		for _, unsupported := range []string{"request", "request_uri", "resource", "claims", "acr_values"} {
			if request.Form.Has(unsupported) {
				op.RequestError(response, request, protocol.ErrInvalidRequest(), nil)
				return
			}
		}
	}
	if path == "/oauth/token" || path == "/revoke" || path == "/oauth/introspect" {
		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", 405)
			return
		}
		if request.ParseForm() != nil || request.PostForm.Get("client_secret") != "" || request.PostForm.Get("client_assertion") != "" || request.Form.Has("resource") {
			op.RequestError(response, request, protocol.ErrInvalidRequest(), nil)
			return
		}
		if path == "/oauth/token" && request.PostForm.Get("grant_type") != "authorization_code" && request.PostForm.Get("grant_type") != "refresh_token" {
			op.RequestError(response, request, protocol.ErrUnsupportedGrantType(), nil)
			return
		}
	}
	if path == "/device_authorization" || path == "/fixture/login" {
		http.NotFound(response, request)
		return
	}
	if path == "/" || path == "/account" || path == "/admin" || strings.HasPrefix(path, "/assets/") {
		if !strings.HasPrefix(path, "/assets/") {
			copy := request.Clone(request.Context())
			copy.URL.Path = "/"
			request = copy
		}
		s.assets.ServeHTTP(response, request)
		return
	}
	s.provider.ServeHTTP(response, request)
}

func duplicateParameters(values url.Values) bool {
	for _, entries := range values {
		if len(entries) != 1 {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func writeJSON(response http.ResponseWriter, value any) {
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(value)
}

type attempts struct {
	Count int
	Until time.Time
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]attempts
}

func (l *limiter) allow(key string, maximum int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for entry, value := range l.entries {
		if now.After(value.Until) {
			delete(l.entries, entry)
		}
	}
	value, exists := l.entries[key]
	if !exists {
		if len(l.entries) >= 4096 {
			return false
		}
		value.Until = now.Add(time.Minute)
	}
	if value.Count >= maximum {
		return false
	}
	value.Count++
	l.entries[key] = value
	return true
}
