package oidc

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	protocol "github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"qoauth/internal/identity"
	"qoauth/internal/store"
)

type apiInput struct {
	Username    string            `json:"username"`
	Password    string            `json:"password"`
	Code        string            `json:"code"`
	Request     string            `json:"request"`
	Approve     bool              `json:"approve"`
	User        string            `json:"user"`
	Client      string            `json:"client"`
	Session     string            `json:"session"`
	Allow       bool              `json:"allow"`
	Application store.Application `json:"application"`
}

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
var scopeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$`)

func (s *Server) api(response http.ResponseWriter, request *http.Request) {
	session := currentSession(request.Context())
	path := request.URL.Path
	var account store.Account
	if session.UserID != "" {
		var err error
		account, err = s.database.Account(request.Context(), session.UserID)
		if err != nil {
			http.Error(response, "session unavailable", 401)
			return
		}
	}
	if path == "/api/session" && request.Method == http.MethodGet {
		writeJSON(response, map[string]any{"csrf": session.Csrf, "account": account, "authenticated": session.UserID != "", "assurance": session.Assurance, "has_factor": len(account.Factor) > 0, "fresh": session.AuthTime > time.Now().Add(-5*time.Minute).Unix()})
		return
	}
	if path == "/api/interaction" && request.Method == http.MethodGet {
		s.interactionInfo(response, request)
		return
	}
	if request.Method == http.MethodGet {
		if !session.Assurance || (account.Admin && len(account.Factor) == 0) {
			http.Error(response, "authentication required", 403)
			return
		}
		switch path {
		case "/api/sessions":
			result, err := s.database.Sessions(request.Context(), account.ID)
			if err != nil {
				http.Error(response, "unavailable", 503)
				return
			}
			writeJSON(response, result)
			return
		case "/api/users", "/api/applications", "/api/activity":
			if !account.Admin {
				http.Error(response, "forbidden", 403)
				return
			}
			var result any
			var err error
			if path == "/api/users" {
				result, err = s.database.Accounts(request.Context())
			}
			if path == "/api/applications" {
				result, err = s.database.Applications(request.Context())
			}
			if path == "/api/activity" {
				result, err = s.database.Activity(request.Context())
			}
			if err != nil {
				http.Error(response, "unavailable", 503)
				return
			}
			writeJSON(response, result)
			return
		}
		http.NotFound(response, request)
		return
	}
	if request.Method != http.MethodPost {
		http.Error(response, "method not allowed", 405)
		return
	}
	if !s.csrf(request) {
		http.Error(response, "invalid origin or CSRF token", 403)
		return
	}
	var input apiInput
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(response, "invalid request", 400)
		return
	}
	if path == "/api/login" {
		remote, _ := s.clientAddress(request)
		if !s.limits.allow("network:"+remote, 30) || !s.limits.allow("login:"+strings.ToLower(input.Username), 10) {
			http.Error(response, "try again later", 429)
			return
		}
		candidate, err := s.database.AccountByName(request.Context(), input.Username)
		encoded := s.dummyPassword
		if err == nil {
			encoded = candidate.Password
		}
		valid := identity.CheckPassword(encoded, input.Password)
		if !valid || err != nil || !candidate.Enabled {
			http.Error(response, "invalid credentials", 401)
			return
		}
		handle, err := s.database.PasswordSession(request.Context(), session.Digest, candidate)
		if err != nil {
			http.Error(response, "authentication unavailable", 401)
			return
		}
		s.setCookie(response, handle)
		writeJSON(response, map[string]bool{"ok": true})
		return
	}
	if session.UserID == "" {
		http.Error(response, "authentication required", 401)
		return
	}
	var result any = map[string]bool{"ok": true}
	var err error
	switch path {
	case "/api/mfa/start":
		if session.AuthTime < time.Now().Add(-5*time.Minute).Unix() || len(account.Factor) > 0 {
			http.Error(response, "fresh login and unenrolled factor required", 403)
			return
		}
		secret, provisioning, createErr := identity.NewTotp(account.Username)
		if createErr != nil {
			http.Error(response, "unavailable", 503)
			return
		}
		err = s.database.StageFactor(request.Context(), session, s.keys.Seal(account.ID, secret))
		result = map[string]string{"secret": secret, "provisioning_uri": provisioning}
	case "/api/mfa/confirm", "/api/mfa/verify":
		if !s.limits.allow("mfa:"+account.ID, 5) {
			http.Error(response, "try again later", 429)
			return
		}
		sealed := account.Factor
		if path == "/api/mfa/confirm" {
			sealed = account.Pending
		}
		secret, openErr := s.keys.Open(account.ID, sealed)
		if openErr != nil {
			http.Error(response, "invalid proof", 401)
			return
		}
		step, proofErr := identity.TotpStep(secret, input.Code, time.Now())
		if proofErr != nil {
			http.Error(response, "invalid proof", 401)
			return
		}
		var handle string
		if path == "/api/mfa/confirm" {
			var codes []string
			handle, codes, err = s.database.ConfirmFactor(request.Context(), session, sealed, step)
			result = map[string]any{"recovery_codes": codes}
		} else {
			handle, err = s.database.ProveFactor(request.Context(), session, sealed, step)
		}
		if err == nil {
			s.setCookie(response, handle)
		}
	case "/api/mfa/recover":
		if session.AuthTime < time.Now().Add(-5*time.Minute).Unix() || !s.limits.allow("recovery:"+account.ID, 5) {
			http.Error(response, "fresh proof required", 403)
			return
		}
		var handle string
		handle, err = s.database.RecoverFactor(request.Context(), session, input.Code)
		if err == nil {
			s.setCookie(response, handle)
		}
	case "/api/logout":
		err = s.database.RevokeSession(request.Context(), session.Digest, account.ID)
	default:
		if !session.Assurance || (account.Admin && len(account.Factor) == 0) {
			http.Error(response, "complete MFA first", 403)
			return
		}
		switch path {
		case "/api/consent":
			s.consentAction(response, request, input)
			return
		case "/api/consent/withdraw":
			err = s.database.WithdrawConsent(request.Context(), account.ID, input.Client, session.Digest)
		case "/api/sessions/revoke":
			err = s.database.RevokeSession(request.Context(), input.Session, account.ID)
		case "/api/password/change":
			var encoded string
			encoded, err = identity.PasswordHash(input.Password)
			if err == nil {
				err = s.database.ResetAccount(request.Context(), account.ID, encoded, false, session.Digest)
			}
		default:
			if !account.Admin || session.AuthTime < time.Now().Add(-5*time.Minute).Unix() {
				http.Error(response, "fresh administrator MFA required", 403)
				return
			}
			switch path {
			case "/api/users/create":
				if !identifier.MatchString(input.Username) {
					http.Error(response, "invalid username", 400)
					return
				}
				var encoded string
				encoded, err = identity.PasswordHash(input.Password)
				if err == nil {
					var id string
					id, err = s.database.AddAccount(request.Context(), input.Username, encoded, session.Digest)
					result = map[string]string{"id": id}
				}
			case "/api/users/disable":
				err = s.database.DisableAccount(request.Context(), input.User, session.Digest)
			case "/api/users/password":
				var encoded string
				encoded, err = identity.PasswordHash(input.Password)
				if err == nil {
					err = s.database.ResetAccount(request.Context(), input.User, encoded, false, session.Digest)
				}
			case "/api/applications/create":
				app := input.Application
				if !s.validApplication(request, app) {
					http.Error(response, "invalid application configuration", 400)
					return
				}
				secret := ""
				app.SecretHash = ""
				if !app.Public {
					secret = store.NewHandle()
					app.SecretHash = store.Hash(secret)
				}
				err = s.database.AddApplication(request.Context(), app, session.Digest)
				result = map[string]string{"id": app.ID, "client_secret": secret}
			case "/api/applications/disable":
				err = s.database.DisableApplication(request.Context(), input.Client, session.Digest)
			case "/api/assignments":
				err = s.database.Assign(request.Context(), input.User, input.Client, session.Digest, input.Allow)
			default:
				http.NotFound(response, request)
				return
			}
		}
	}
	if err != nil {
		http.Error(response, "operation rejected", 400)
		return
	}
	writeJSON(response, result)
}

func (s *Server) validApplication(request *http.Request, app store.Application) bool {
	if !identifier.MatchString(app.ID) || len(app.Name) < 1 || len(app.Name) > 80 || len(app.Scopes) > 32 || len(app.Audiences) > 16 || len(app.Redirects) > 16 || len(app.LogoutRedirects) > 16 {
		return false
	}
	for _, scope := range app.Scopes {
		if !scopeName.MatchString(scope) || scope == "email" {
			return false
		}
	}
	for _, audience := range app.Audiences {
		if !identifier.MatchString(audience) {
			return false
		}
	}
	if app.Resource {
		return !app.Public && len(app.Audiences) == 1 && app.Audiences[0] == app.ID && len(app.Redirects) == 0 && len(app.LogoutRedirects) == 0
	}
	if len(app.Redirects) == 0 || !contains(app.Scopes, "openid") {
		return false
	}
	for _, raw := range append(append([]string{}, app.Redirects...), app.LogoutRedirects...) {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || strings.Contains(raw, "*") {
			return false
		}
		if parsed.Scheme != "https" {
			ip := net.ParseIP(parsed.Hostname())
			if !s.config.DevelopmentHTTP || parsed.Scheme != "http" || ip == nil || !ip.IsLoopback() {
				return false
			}
		}
	}
	for _, audience := range app.Audiences {
		resource, err := s.database.Application(request.Context(), audience)
		if err != nil || !resource.Resource {
			return false
		}
		for _, scope := range app.Scopes {
			if scope != "openid" && scope != "profile" && scope != "offline_access" && !contains(resource.Scopes, scope) {
				return false
			}
		}
	}
	return true
}

func (s *Server) loadInteraction(request *http.Request) (store.LoginRequest, tokenData, error) {
	id := request.URL.Query().Get("request")
	login, err := s.database.Login(request.Context(), id)
	var data tokenData
	if err != nil || login.Browser != currentSession(request.Context()).Digest || json.Unmarshal(login.Payload, &data) != nil {
		return login, data, store.ErrInvalid
	}
	return login, data, nil
}
func (s *Server) interactionInfo(response http.ResponseWriter, request *http.Request) {
	login, data, err := s.loadInteraction(request)
	if err != nil || login.Completed {
		http.Error(response, "invalid authorization transaction", 400)
		return
	}
	session := currentSession(request.Context())
	app, err := s.database.Application(request.Context(), login.ClientID)
	if err != nil {
		http.Error(response, "application unavailable", 400)
		return
	}
	writeJSON(response, map[string]any{"request": login.ID, "application": app.Name, "scopes": data.Request.Scopes, "audiences": data.Resources, "needs_login": !s.eligible(request.Context(), session, data), "needs_consent": contains(data.Request.Prompt, "consent") || !s.hasConsent(request.Context(), session.UserID, data)})
}
func (s *Server) complete(request *http.Request, login store.LoginRequest, data tokenData, explicitApproval bool) error {
	session := currentSession(request.Context())
	if !s.eligible(request.Context(), session, data) {
		return store.ErrInvalid
	}
	account, err := s.database.Account(request.Context(), session.UserID)
	if err != nil {
		return err
	}
	data.User = session.UserID
	data.AuthTime = time.Unix(session.AuthTime, 0)
	data.Mfa = len(account.Factor) > 0
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	approved, _ := json.Marshal(consent{Scopes: data.Request.Scopes, Audiences: data.Resources})
	return s.database.CompleteLogin(request.Context(), login, session, payload, approved, explicitApproval)
}
func (s *Server) callback(id string) string {
	return op.AuthCallbackURL(s.provider)(op.ContextWithIssuer(context.Background(), s.config.Issuer), id)
}
func (s *Server) loginPage(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(response, "method not allowed", 405)
		return
	}
	if request.URL.Query().Get("request") == "" {
		copy := request.Clone(request.Context())
		copy.URL.Path = "/"
		s.assets.ServeHTTP(response, copy)
		return
	}
	login, data, err := s.loadInteraction(request)
	if err != nil || login.Completed {
		http.Error(response, "invalid authorization transaction", 400)
		return
	}
	session := currentSession(request.Context())
	if s.eligible(request.Context(), session, data) && !contains(data.Request.Prompt, "consent") && s.hasConsent(request.Context(), session.UserID, data) {
		if err = s.complete(request, login, data, false); err != nil {
			op.RequestError(response, request, protocol.ErrAccessDenied(), nil)
			return
		}
		http.Redirect(response, request, s.callback(login.ID), 303)
		return
	}
	if contains(data.Request.Prompt, "none") {
		op.AuthRequestError(response, request, &data.Request, protocol.ErrLoginRequired(), s.provider)
		return
	}
	copy := request.Clone(request.Context())
	copy.URL.Path = "/"
	s.assets.ServeHTTP(response, copy)
}
func (s *Server) consentAction(response http.ResponseWriter, request *http.Request, input apiInput) {
	copy := request.Clone(request.Context())
	query := copy.URL.Query()
	query.Set("request", input.Request)
	copy.URL.RawQuery = query.Encode()
	login, data, err := s.loadInteraction(copy)
	if err != nil || login.Completed {
		http.Error(response, "invalid authorization transaction", 400)
		return
	}
	if !input.Approve {
		if err = s.database.CancelLogin(request.Context(), login.ID, currentSession(request.Context()).Digest); err != nil {
			http.Error(response, "invalid authorization transaction", 400)
			return
		}
		redirect, _ := url.Parse(data.Request.RedirectURI)
		query := redirect.Query()
		query.Set("error", "access_denied")
		query.Set("state", data.Request.State)
		redirect.RawQuery = query.Encode()
		writeJSON(response, map[string]string{"redirect": redirect.String()})
		return
	}
	if err = s.complete(request, login, data, true); err != nil {
		http.Error(response, "authorization rejected", 403)
		return
	}
	writeJSON(response, map[string]string{"redirect": s.callback(login.ID)})
}
