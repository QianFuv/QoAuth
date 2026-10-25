// Command qoauth provides explicit local initialization and the authentication server.
package main

import (
	"context"
	"encoding/json"
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
	"qoauth/internal/maintenance"
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
	username := flags.String("username", "", "Owner name for init, or disabled account for recover-user")
	issuer := flags.String("issuer", "", "Canonical public HTTPS origin")
	listen := flags.String("listen", "127.0.0.1:8080", "HTTP listen address behind a trusted TLS proxy")
	development := flags.Bool("development-http", false, "Permit explicitly local HTTP development only")
	trustedProxy := flags.String("trusted-proxy", "", "One proxy IP that overwrites X-Real-IP with the client IP")
	confirm := flags.Bool("confirm", false, "Confirm recovery/rotation; restore requires the old issuer to remain stopped")
	snapshot := flags.String("snapshot", "", "New snapshot directory for backup, existing snapshot for restore")
	restoreReviewed := flags.Bool("restore-reviewed", false, "Confirm stopped old issuer and review of rolled-back users, clients and RP sessions")
	batch := flags.Int("batch-size", 128, "Maximum cleanup rows per table (1-256), one transaction")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if *directory == "" {
		return errors.New("--data-dir is required; no automatic initialization")
	}
	if command != "init" && command != "recover-owner" && command != "recover-user" && command != "serve" && command != "rotate-signing" && command != "backup" && command != "restore" && command != "cleanup" {
		return errors.New("expected init, serve, recover-owner, recover-user, rotate-signing, backup, restore or cleanup")
	}
	ctx := context.Background()
	if command == "restore" {
		if *snapshot == "" || !*confirm {
			return errors.New("restore requires --snapshot and --confirm that the old issuer is stopped and stays stopped")
		}
		if err := maintenance.Restore(ctx, *snapshot, *directory); err != nil {
			return err
		}
		fmt.Fprintln(output, "Restored into a new directory with fresh signing trust. Old issuer must stay stopped. All access is invalidated; recover-owner --confirm --restore-reviewed and a ten-minute token-expiry quarantine are required. Other users and clients remain disabled.")
		return nil
	}
	lock, err := identity.LockDirectory(*directory)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = maintenance.CheckOperationalDirectory(*directory); err != nil {
		return err
	}
	if command != "rotate-signing" && identity.HasPendingRotation(*directory) {
		return errors.New("unfinished key publication; resume rotate-signing before other commands")
	}
	if command == "backup" {
		if *snapshot == "" {
			return errors.New("backup requires a new --snapshot directory")
		}
		if err = maintenance.Backup(ctx, *directory, *snapshot); err != nil {
			return err
		}
		fmt.Fprintln(output, "Consistent database and plaintext key snapshot completed in private storage. Protect the whole snapshot as installation credentials.")
		return nil
	}
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
	if command == "rotate-signing" {
		if !*confirm {
			return errors.New("signing rotation requires --confirm")
		}
		return rotateSigning(ctx, database, keys, *directory, output)
	}
	if err = database.CheckInstallation(ctx, keys.Binding()); err != nil {
		return err
	}
	if command == "cleanup" {
		counts, err := database.Cleanup(ctx, *batch)
		if err != nil {
			return errors.New("cleanup failed; authentication data remains transactionally protected")
		}
		return json.NewEncoder(output).Encode(counts)
	}
	if command == "recover-owner" || command == "recover-user" {
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
		state, err := database.InstallationState(ctx)
		if err != nil {
			return err
		}
		if state.Maintenance {
			if command != "recover-owner" || !*restoreReviewed {
				return errors.New("restored owner recovery requires --restore-reviewed; other identities stay disabled")
			}
			err = database.RecoverRestoredOwner(ctx, encoded)
		} else if command == "recover-user" {
			err = database.RecoverDisabledUser(ctx, *username, encoded)
		} else {
			err = database.RecoverOwner(ctx, encoded)
		}
		if err != nil {
			return errors.New("local account recovery failed")
		}
		fmt.Fprintln(output, "Account recovered with restricted access. Confirm a new TOTP; restored issuer quarantine and other disabled identities remain unchanged.")
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
	fmt.Fprintln(output, "QoAuth serving the configured issuer behind its trusted proxy.")
	return serveHTTP(ctx, server)
}

func rotateSigning(ctx context.Context, database *store.Store, keys *identity.Keys, directory string, output io.Writer) error {
	if !identity.HasPendingRotation(directory) {
		if err := database.CheckInstallation(ctx, keys.Binding()); err != nil {
			return err
		}
	}
	next, err := identity.StageRotation(directory, keys, time.Now())
	if err != nil {
		return err
	}
	if err = database.RotateSigningBinding(ctx, keys.Binding(), next.Binding()); err != nil {
		return errors.New("rotation binding did not commit; keep files and retry the same command")
	}
	if err = identity.PublishRotation(directory); err != nil {
		return errors.New("rotation publication incomplete; keep files and retry the same command")
	}
	fmt.Fprintln(output, "Signing key rotated. AES unchanged; previous public key expires after its bounded verification window.")
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
