package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"time"
)

// InstallationState separates restoration quarantine from ordinary key binding.
type InstallationState struct {
	Schema      int
	Issuer      string
	Maintenance bool
	ReadyAfter  int64
}

func (storage *Store) InstallationState(ctx context.Context) (InstallationState, error) {
	state := InstallationState{Schema: len(migrationScripts())}
	err := storage.db.QueryRowContext(ctx, `SELECT issuer,maintenance,ready_after FROM installation WHERE id=1`).Scan(&state.Issuer, &state.Maintenance, &state.ReadyAfter)
	return state, err
}

// CheckServing enforces persisted issuer identity and restored-state readiness.
func (storage *Store) CheckServing(ctx context.Context, binding, issuer string, now time.Time) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var current, savedIssuer string
		var maintenance bool
		var readyAfter int64
		if err := tx.QueryRowContext(ctx, `SELECT key_id,issuer,maintenance,ready_after FROM installation WHERE id=1`).Scan(&current, &savedIssuer, &maintenance, &readyAfter); err != nil || current != binding {
			return errors.New("installation or matching keys unavailable")
		}
		if maintenance {
			return errors.New("restored installation requires recover-owner --confirm --restore-reviewed")
		}
		if now.Unix() < readyAfter {
			return errors.New("restored installation is in its signed-token expiry quarantine")
		}
		if savedIssuer != "" && savedIssuer != issuer {
			return errors.New("configured issuer does not match the installation")
		}
		if savedIssuer == "" {
			_, err := tx.ExecContext(ctx, `UPDATE installation SET issuer=? WHERE id=1`, issuer)
			return err
		}
		return nil
	})
}

// Snapshot uses SQLite's consistent snapshot operation while the caller owns the lock.
// The destination is a newly created private empty file; never a live database.
func (storage *Store) Snapshot(ctx context.Context, destination string) error {
	_, err := storage.db.ExecContext(ctx, `VACUUM INTO ?`, destination)
	return err
}

// InspectSnapshot opens existing bytes read-only, without migration or journal changes.
// The caller holds the directory lock; standalone snapshots must have no sidecars.
func InspectSnapshot(ctx context.Context, path, binding string) (InstallationState, error) {
	return inspectDatabase(ctx, path, binding, true)
}

// InspectInstallation includes committed WAL data under the caller's installation lock.
func InspectInstallation(ctx context.Context, path, binding string) (InstallationState, error) {
	return inspectDatabase(ctx, path, binding, false)
}

func inspectDatabase(ctx context.Context, path, binding string, standalone bool) (InstallationState, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return InstallationState{}, err
	}
	filePath := filepath.ToSlash(absolute)
	if filepath.VolumeName(absolute) != "" {
		filePath = "/" + filePath
	}
	dsn := &url.URL{Scheme: "file", Path: filePath, RawQuery: "mode=ro&_query_only=1&_trusted_schema=0"}
	if standalone {
		dsn.RawQuery += "&immutable=1"
	}
	database, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return InstallationState{}, err
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	inspected := &Store{db: database}
	if err = inspectSchema(ctx, database); err != nil {
		return InstallationState{}, err
	}
	var result string
	if err = database.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil || result != "ok" {
		return InstallationState{}, errors.New("database integrity check failed")
	}
	rows, err := database.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return InstallationState{}, err
	}
	hasViolation := rows.Next()
	err = rows.Err()
	rows.Close()
	if err != nil || hasViolation {
		return InstallationState{}, errors.New("database foreign-key check failed")
	}
	if err = inspected.CheckInstallation(ctx, binding); err != nil {
		return InstallationState{}, err
	}
	var owners, validOwners, installations int
	if err = database.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM accounts WHERE owner<>0),(SELECT count(*) FROM accounts WHERE owner=1 AND admin=1 AND require_mfa=1),(SELECT count(*) FROM installation)`).Scan(&owners, &validOwners, &installations); err != nil || owners != 1 || validOwners != 1 || installations != 1 {
		return InstallationState{}, errors.New("snapshot installation owner invariant failed")
	}
	return inspected.InstallationState(ctx)
}

func inspectSchema(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `SELECT version,digest FROM schema_migrations ORDER BY version`)
	if err != nil {
		return errors.New("snapshot schema ledger unavailable")
	}
	count := 0
	for rows.Next() {
		var version int
		var digest string
		if err = rows.Scan(&version, &digest); err != nil || count >= len(migrationScripts()) || version != count+1 || digest != Hash(migrationScripts()[count]) {
			rows.Close()
			return errors.New("unsupported snapshot schema version/checksum")
		}
		count++
	}
	err = rows.Err()
	rows.Close()
	if err != nil || count != len(migrationScripts()) {
		return errors.New("unsupported snapshot schema version/checksum")
	}
	expected, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer expected.Close()
	expected.SetMaxOpenConns(1)
	if err = (&Store{db: expected}).migrate(ctx); err != nil {
		return err
	}
	wanted, err := schemaObjects(ctx, expected)
	if err != nil {
		return err
	}
	actual, err := schemaObjects(ctx, database)
	if err != nil || !reflect.DeepEqual(actual, wanted) {
		return errors.New("snapshot schema objects differ from the supported schema")
	}
	return nil
}

func schemaObjects(ctx context.Context, database *sql.DB) ([]string, error) {
	rows, err := database.QueryContext(ctx, `SELECT type,name,tbl_name,COALESCE(sql,'') FROM sqlite_schema WHERE name NOT GLOB 'sqlite_*' ORDER BY type,name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects []string
	for rows.Next() {
		var kind, name, table, statement string
		if err = rows.Scan(&kind, &name, &table, &statement); err != nil {
			return nil, err
		}
		objects = append(objects, fmt.Sprintf("%s\x00%s\x00%s\x00%s", kind, name, table, statement))
		if len(objects) > 100 {
			return nil, errors.New("unexpected snapshot schema size")
		}
	}
	return objects, rows.Err()
}

// RestoreState invalidates all authentication and fences old configuration in one commit.
func (storage *Store) RestoreState(ctx context.Context, oldBinding, newBinding string, readyAfter time.Time) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE installation SET key_id=?,maintenance=1,ready_after=? WHERE id=1 AND key_id=?`, newBinding, readyAfter.Unix(), oldBinding)
		if err = one(result, err); err != nil {
			return err
		}
		for _, statement := range []string{
			`DELETE FROM tokens`, `DELETE FROM grants`, `DELETE FROM auth_requests`,
			`DELETE FROM login_requests`, `DELETE FROM browser_sessions`, `DELETE FROM recovery_codes`,
			`DELETE FROM consents`, `DELETE FROM assignments`, `UPDATE clients SET enabled=0`,
			`UPDATE users SET enabled=0`,
			`UPDATE accounts SET version=version+1,pending_factor=NULL,pending_until=0`,
		} {
			if _, err = tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return activity(ctx, tx, "restore-quarantine", "host")
	})
}

// RecoverRestoredOwner requires host review, a replacement password and new MFA.
// Quarantine is retained even after configuration review and recovery succeed.
func (storage *Store) RecoverRestoredOwner(ctx context.Context, password string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var owner string
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM accounts WHERE owner=1 AND (SELECT maintenance FROM installation WHERE id=1)=1`).Scan(&owner); err != nil {
			return ErrInvalid
		}
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET password=?,version=version+1,factor=NULL,pending_factor=NULL,pending_until=0,last_step=-1,require_mfa=1 WHERE user_id=?`, password, owner); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE users SET enabled=1 WHERE id=?`, owner); err != nil {
			return err
		}
		if err := invalidateAccount(ctx, tx, owner); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id=?; UPDATE installation SET maintenance=0 WHERE id=1`, owner); err != nil {
			return err
		}
		return activity(ctx, tx, "restore-owner-reviewed", "host")
	})
}

// RecoverDisabledUser is the explicit local path for reviewed restored accounts.
// It replaces credentials/factors and drops roles and assignments before re-enabling.
func (storage *Store) RecoverDisabledUser(ctx context.Context, username, password string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var user string
		if err := tx.QueryRowContext(ctx, `SELECT a.user_id FROM accounts a JOIN users u ON u.id=a.user_id WHERE a.username=? AND a.owner=0 AND u.enabled=0 AND (SELECT maintenance FROM installation WHERE id=1)=0`, username).Scan(&user); err != nil {
			return ErrInvalid
		}
		if _, err := tx.ExecContext(ctx, `UPDATE accounts SET password=?,version=version+1,admin=0,factor=NULL,pending_factor=NULL,pending_until=0,last_step=-1,require_mfa=1 WHERE user_id=?`, password, user); err != nil {
			return err
		}
		if err := invalidateAccount(ctx, tx, user); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM assignments WHERE user_id=?; DELETE FROM consents WHERE user_id=?; DELETE FROM recovery_codes WHERE user_id=?; UPDATE users SET enabled=1 WHERE id=?`, user, user, user, user); err != nil {
			return err
		}
		return activity(ctx, tx, "host-user-recovery", user)
	})
}
