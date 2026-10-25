// Command go-rp demonstrates a standard server-side OIDC client with local sessions.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	rp "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type loginTransaction struct {
	State, Nonce, Verifier string
	Expires                time.Time
}
type appSession struct {
	Token                  *oauth2.Token
	Subject, IDToken, Csrf string
	Expires                time.Time
}
type demo struct {
	mu             sync.Mutex
	provider       *rp.Provider
	config         oauth2.Config
	origin, logout string
	secure         bool
	transactions   map[string]loginTransaction
	sessions       map[string]appSession
}

var page = template.Must(template.New("page").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>QoAuth Go RP</title><h1>Standard Go RP</h1>{{if .Subject}}<p>Signed in as {{.Subject}}</p><p><a href="/profile">Read UserInfo / refresh if expired</a></p><form method="post" action="/logout"><input type="hidden" name="csrf" value="{{.Csrf}}"><button>Sign out of this app and QoAuth</button></form>{{else}}<a href="/login">Sign in with QoAuth</a>{{end}}</html>`))

func randomValue() string {
	var value [32]byte
	_, _ = rand.Read(value[:])
	return base64.RawURLEncoding.EncodeToString(value[:])
}
func main() {
	issuer := flag.String("issuer", "", "QoAuth issuer")
	id := flag.String("client-id", "", "Registered confidential client ID")
	secretFile := flag.String("client-secret-file", "", "Private client secret file, not a command-line secret")
	callback := flag.String("callback", "http://127.0.0.1:8081/callback", "Exact registered callback")
	listen := flag.String("listen", "127.0.0.1:8081", "Local HTTP address behind the application's TLS proxy")
	development := flag.Bool("development-http", false, "Explicit local HTTP example")
	flag.Parse()
	parsed, err := url.Parse(*callback)
	if err != nil || parsed.Host == "" || parsed.Path != "/callback" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		log.Fatal("invalid callback")
	}
	if parsed.Scheme != "https" {
		host, _, listenErr := net.SplitHostPort(*listen)
		ip := net.ParseIP(parsed.Hostname())
		listenIP := net.ParseIP(host)
		if !*development || parsed.Scheme != "http" || ip == nil || !ip.IsLoopback() || listenErr != nil || listenIP == nil || !listenIP.IsLoopback() {
			log.Fatal("HTTPS required except explicit loopback example")
		}
	}
	secret, err := os.ReadFile(*secretFile)
	if err != nil || len(secret) > 1024 {
		log.Fatal("client secret file unavailable")
	}
	provider, err := rp.NewProvider(context.Background(), *issuer)
	if err != nil {
		log.Fatal("provider discovery failed")
	}
	var metadata struct {
		EndSession string `json:"end_session_endpoint"`
	}
	if provider.Claims(&metadata) != nil {
		log.Fatal("invalid discovery")
	}
	endpoint := provider.Endpoint()
	endpoint.AuthStyle = oauth2.AuthStyleInHeader
	app := &demo{provider: provider, config: oauth2.Config{ClientID: *id, ClientSecret: strings.TrimSpace(string(secret)), RedirectURL: *callback, Endpoint: endpoint, Scopes: []string{"openid", "profile", "offline_access"}}, origin: parsed.Scheme + "://" + parsed.Host, logout: metadata.EndSession, secure: parsed.Scheme == "https", transactions: map[string]loginTransaction{}, sessions: map[string]appSession{}}
	server := http.Server{Addr: *listen, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute}
	if err = server.ListenAndServe(); err != nil {
		log.Fatal("RP server stopped")
	}
}
func (app *demo) cookie(response http.ResponseWriter, value string) {
	http.SetCookie(response, &http.Cookie{Name: "qoauth_rp", Value: value, Path: "/", HttpOnly: true, Secure: app.secure, SameSite: http.SameSiteLaxMode, MaxAge: 900})
}
func (app *demo) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	app.mu.Lock()
	defer app.mu.Unlock()
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	request.Body = http.MaxBytesReader(response, request.Body, 8192)
	now := time.Now()
	for key, transaction := range app.transactions {
		if now.After(transaction.Expires) {
			delete(app.transactions, key)
		}
	}
	for key, session := range app.sessions {
		if now.After(session.Expires) {
			delete(app.sessions, key)
		}
	}
	cookie, _ := request.Cookie("qoauth_rp")
	handle := ""
	if cookie != nil {
		handle = cookie.Value
	}
	session, authenticated := app.sessions[handle]
	switch request.URL.Path {
	case "/login":
		if request.Method != "GET" || len(app.transactions) >= 64 {
			http.Error(response, "login unavailable", 400)
			return
		}
		handle = randomValue()
		transaction := loginTransaction{State: randomValue(), Nonce: randomValue(), Verifier: oauth2.GenerateVerifier(), Expires: now.Add(5 * time.Minute)}
		app.transactions[handle] = transaction
		app.cookie(response, handle)
		http.Redirect(response, request, app.config.AuthCodeURL(transaction.State, oauth2.S256ChallengeOption(transaction.Verifier), oauth2.SetAuthURLParam("nonce", transaction.Nonce), oauth2.SetAuthURLParam("prompt", "consent")), 302)
	case "/callback":
		transaction, exists := app.transactions[handle]
		delete(app.transactions, handle)
		if request.Method != "GET" || !exists || request.URL.Query().Get("state") != transaction.State || request.URL.Query().Get("error") != "" {
			http.Error(response, "invalid login response", 400)
			return
		}
		token, err := app.config.Exchange(request.Context(), request.URL.Query().Get("code"), oauth2.VerifierOption(transaction.Verifier))
		if err != nil {
			http.Error(response, "login exchange failed", 401)
			return
		}
		identity, ok := token.Extra("id_token").(string)
		if !ok {
			http.Error(response, "missing identity", 401)
			return
		}
		verified, err := app.provider.Verifier(&rp.Config{ClientID: app.config.ClientID, SupportedSigningAlgs: []string{"RS256"}}).Verify(request.Context(), identity)
		if err != nil || verified.Nonce != transaction.Nonce {
			http.Error(response, "invalid identity", 401)
			return
		}
		handle = randomValue()
		app.sessions[handle] = appSession{Token: token, Subject: verified.Subject, IDToken: identity, Csrf: randomValue(), Expires: now.Add(15 * time.Minute)}
		app.cookie(response, handle)
		http.Redirect(response, request, "/", 303)
	case "/profile":
		if !authenticated {
			http.Error(response, "sign in first", 401)
			return
		}
		token, err := app.config.TokenSource(request.Context(), session.Token).Token()
		if err != nil {
			delete(app.sessions, handle)
			http.Error(response, "session expired; sign in again", 401)
			return
		}
		session.Token = token
		app.sessions[handle] = session
		info, err := app.provider.UserInfo(request.Context(), oauth2.StaticTokenSource(token))
		if err != nil {
			delete(app.sessions, handle)
			http.Error(response, "provider no longer accepts this session", 401)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(response, "Verified subject:", info.Subject)
	case "/logout":
		if !authenticated || request.Method != "POST" || request.Header.Get("Origin") != app.origin || request.ParseForm() != nil || request.PostForm.Get("csrf") != session.Csrf {
			http.Error(response, "invalid logout", 403)
			return
		}
		delete(app.sessions, handle)
		app.cookie(response, "")
		target, err := url.Parse(app.logout)
		if err != nil {
			http.Error(response, "logout unavailable", 503)
			return
		}
		query := target.Query()
		query.Set("id_token_hint", session.IDToken)
		query.Set("post_logout_redirect_uri", app.origin+"/signed-out")
		target.RawQuery = query.Encode()
		http.Redirect(response, request, target.String(), 303)
	case "/signed-out":
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(response, "Signed out of this application.")
	case "/":
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(response, session)
	default:
		http.NotFound(response, request)
	}
}
