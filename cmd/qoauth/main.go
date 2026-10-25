// Command qoauth provides explicit local initialization and the authentication server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"qoauth/internal/identity"
	auth "qoauth/internal/oidc"
	"qoauth/internal/store"
)

func main() {
	slog.SetDefault(auth.SafeLogger(os.Stderr))
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "QoAuth:", err)
		os.Exit(1)
	}
}

func run(arguments []string, input io.Reader, output io.Writer) error {
	command := "serve"
	if len(arguments) > 0 && !strings.HasPrefix(arguments[0], "-") {
		command = arguments[0]
		arguments = arguments[1:]
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(output)
	directory := flags.String("data-dir", "", "Private installation data directory")
	username := flags.String("username", "", "Initial owner username (init only)")
	issuer := flags.String("issuer", "", "Canonical public HTTPS origin")
	listen := flags.String("listen", "127.0.0.1:8080", "HTTP listen address behind a trusted TLS proxy")
	development := flags.Bool("development-http", false, "Permit explicitly local HTTP development only")
	trustedProxy := flags.String("trusted-proxy", "", "One proxy IP that overwrites X-Real-IP with the client IP")
	confirm := flags.Bool("confirm", false, "Confirm host owner recovery and invalidate existing access")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if *directory == "" {
		return errors.New("--data-dir is required; no automatic initialization")
	}
	if command != "init" && command != "recover-owner" && command != "serve" {
		return errors.New("expected init, recover-owner or serve")
	}
	ctx := context.Background()
	if command == "init" {
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`).MatchString(*username) {
			return errors.New("valid --username required")
		}
		password, err := readPassword(input)
		if err != nil {
			return err
		}
		encoded, err := identity.PasswordHash(password)
		if err != nil {
			return err
		}
		if err = identity.PrepareDirectory(*directory); err != nil {
			return err
		}
		path := filepath.Join(*directory, "qoauth.db")
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			if _, keyErr := os.Lstat(filepath.Join(*directory, "keys.json")); !os.IsNotExist(keyErr) {
				return errors.New("database missing beside existing keys; restore the installation instead of initializing")
			}
		}
		if err = identity.PrepareDatabase(path, true); err != nil {
			return err
		}
		database, err := store.Open(ctx, filepath.Join(*directory, "qoauth.db"))
		if err != nil {
			return errors.New("database unavailable")
		}
		defer database.Close()
		if !database.CanBootstrap(ctx) {
			return errors.New("installation is already initialized or contains fixture data")
		}
		keys, err := identity.LoadKeys(*directory)
		if err != nil {
			if _, statErr := os.Lstat(filepath.Join(*directory, "keys.json")); !os.IsNotExist(statErr) {
				return err
			}
			keys, err = identity.CreateKeys(*directory)
			if err != nil {
				return err
			}
		}
		if err = database.Bootstrap(ctx, *username, encoded, keys.Binding()); err != nil {
			return err
		}
		fmt.Fprintln(output, "Owner initialized. Sign in and confirm TOTP before administration.")
		return nil
	}
	if err := identity.PrepareDirectory(*directory); err != nil {
		return err
	}
	keys, err := identity.LoadKeys(*directory)
	if err != nil {
		return err
	}
	if err = identity.PrepareDatabase(filepath.Join(*directory, "qoauth.db"), false); err != nil {
		return err
	}
	database, err := store.Open(ctx, filepath.Join(*directory, "qoauth.db"))
	if err != nil {
		return errors.New("database unavailable")
	}
	defer database.Close()
	if err = database.CheckInstallation(ctx, keys.Binding()); err != nil {
		return err
	}
	if command == "recover-owner" {
		if !*confirm {
			return errors.New("owner recovery requires --confirm")
		}
		password, err := readPassword(input)
		if err != nil {
			return err
		}
		encoded, err := identity.PasswordHash(password)
		if err != nil {
			return err
		}
		if err = database.RecoverOwner(ctx, encoded); err != nil {
			return errors.New("owner recovery failed")
		}
		fmt.Fprintln(output, "Owner recovered with restricted access. Confirm a new TOTP before administration.")
		return nil
	}
	config := auth.Config{Issuer: *issuer, DevelopmentHTTP: *development, TrustedProxy: *trustedProxy}
	if err = config.Validate(); err != nil {
		return err
	}
	if *development {
		host, _, err := net.SplitHostPort(*listen)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return errors.New("development HTTP must listen on numeric loopback")
		}
	}
	handler, err := auth.New(ctx, database, keys, config)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute, MaxHeaderBytes: 32 << 10}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Fprintln(output, "QoAuth serving the configured issuer behind its trusted proxy.")
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.New("HTTP server stopped unexpectedly")
	}
	return nil
}

func readPassword(input io.Reader) (string, error) {
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", errors.New("supply the password through redirected stdin; terminal echo and password arguments are not supported")
		}
	}
	value, err := io.ReadAll(io.LimitReader(input, 1027))
	if err != nil {
		return "", errors.New("cannot read password")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(value), "\n"), "\r")
	if len(password) > 1024 {
		return "", errors.New("password exceeds input limit")
	}
	return password, nil
}
