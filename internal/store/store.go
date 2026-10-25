// Package store owns local SQLite security-state transitions.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001.sql
var migration string

//go:embed migrations/002.sql
var identityMigration string

//go:embed migrations/003.sql
var authenticationMigration string

//go:embed migrations/004.sql
var limitsMigration string

var (
	ErrInvalid = errors.New("invalid or inactive credential")
	ErrReplay  = errors.New("credential reuse; grant revoked")
)

// Store is a bounded, single-process SQLite store. It has no network dependencies.
type Store struct{ db *sql.DB }

// Request holds the persisted OP transaction and its one-use code state.
type Request struct {
	ID, UserID, ClientID string
	Payload              []byte
	Expires              time.Time
	Ready, Consumed      bool
}

// Token describes an active grant's access or refresh handle without storing it.
type Token struct {
	GrantID, UserID, ClientID, Kind string
	Payload                         []byte
	Expires                         time.Time
	Consumed                        bool
}

// Issued contains new random handles, returned only after a successful commit.
type Issued struct {
	Access, Refresh string
	Expires         time.Time
}

// Settings records actual engine and connection settings, not module assumptions.
type Settings struct {
	Version, Journal                      string
	ForeignKeys, Synchronous, BusyTimeout int
}

// Open applies a checksum-verified migration and verifies durable connection settings.
func Open(ctx context.Context, path string) (*Store, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	filePath := filepath.ToSlash(absolute)
	if filepath.VolumeName(absolute) != "" {
		filePath = "/" + filePath
	}
	dsn := &url.URL{Scheme: "file", Path: filePath}
	params := url.Values{"_foreign_keys": {"1"}, "_journal_mode": {"WAL"}, "_synchronous": {"FULL"}, "_busy_timeout": {"3000"}, "_txlock": {"immediate"}}
	dsn.RawQuery = params.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	storage := &Store{db: db}
	if err = storage.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	settings, err := storage.Settings(ctx)
	if err == nil && (settings.ForeignKeys != 1 || settings.Synchronous != 2 || settings.Journal != "wal" || settings.BusyTimeout != 3000 || !fixedEngine(settings.Version)) {
		err = fmt.Errorf("unsupported SQLite settings: %+v", settings)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return storage, nil
}

func fixedEngine(version string) bool {
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return false
	}
	return major == 3 && (minor > 51 || minor == 51 && patch >= 3 || minor == 50 && patch >= 7 || minor == 44 && patch >= 6)
}

func (storage *Store) migrate(ctx context.Context) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, digest TEXT NOT NULL)`); err != nil {
			return err
		}
		var latest int
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&latest); err != nil {
			return err
		}
		if latest > 4 {
			return errors.New("unsupported schema version")
		}
		for index, script := range []string{migration, identityMigration, authenticationMigration, limitsMigration} {
			version := index + 1
			if version <= latest {
				var digest string
				if err := tx.QueryRowContext(ctx, `SELECT digest FROM schema_migrations WHERE version=?`, version).Scan(&digest); err != nil || digest != Hash(script) {
					return errors.New("migration version/checksum mismatch")
				}
				continue
			}
			if _, err := tx.ExecContext(ctx, script); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(?,?)`, version, Hash(script)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Close releases the connection; callers must stop HTTP requests first.
func (storage *Store) Close() error { return storage.db.Close() }

// Health checks the live connection without returning configuration or credentials.
func (storage *Store) Health(ctx context.Context) error { return storage.db.PingContext(ctx) }

// Settings reads the actual runtime rather than trusting a dependency version.
func (storage *Store) Settings(ctx context.Context) (Settings, error) {
	var settings Settings
	for _, query := range []struct {
		sql    string
		target any
	}{
		{"SELECT sqlite_version()", &settings.Version}, {"PRAGMA journal_mode", &settings.Journal},
		{"PRAGMA foreign_keys", &settings.ForeignKeys}, {"PRAGMA synchronous", &settings.Synchronous}, {"PRAGMA busy_timeout", &settings.BusyTimeout},
	} {
		if err := storage.db.QueryRowContext(ctx, query.sql).Scan(query.target); err != nil {
			return Settings{}, err
		}
	}
	return settings, nil
}

// NewHandle generates a 256-bit random value using the standard CSPRNG.
func NewHandle() string {
	var value [32]byte
	_, _ = rand.Read(value[:])
	return base64.RawURLEncoding.EncodeToString(value[:])
}

// Hash derives a verifier for high-entropy handles, never for human passwords.
func Hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func (storage *Store) transaction(ctx context.Context, operation func(*sql.Tx) error) error {
	tx, err := storage.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = operation(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func activity(ctx context.Context, tx *sql.Tx, action, subject string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO activity(action,subject,recorded_at) VALUES(?,?,?)`, action, subject, time.Now().Unix())
	return err
}

// SeedFixture inserts only synthetic principals into a caller-owned temporary database.
// It never re-enables an existing disabled principal or restores a removed assignment.
func (storage *Store) SeedFixture(ctx context.Context, user string, clients []string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO users(id) VALUES(?)`, user)
		if err != nil {
			return err
		}
		inserted, _ := result.RowsAffected()
		for _, client := range clients {
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO clients(id) VALUES(?)`, client); err != nil {
				return err
			}
			if inserted == 1 {
				if _, err = tx.ExecContext(ctx, `INSERT INTO assignments VALUES(?,?)`, user, client); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func active(ctx context.Context, tx *sql.Tx, user, client string) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM assignments a JOIN users u ON u.id=a.user_id JOIN clients c ON c.id=a.client_id WHERE a.user_id=? AND a.client_id=? AND u.enabled=1 AND c.enabled=1`, user, client).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrInvalid
	}
	return nil
}

// CreateRequest stores the immutable validated protocol request before fixture login.
func (storage *Store) CreateRequest(ctx context.Context, request Request) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := active(ctx, tx, request.UserID, request.ClientID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO auth_requests(id,user_id,client_id,payload,expires) VALUES(?,?,?,?,?)`, request.ID, request.UserID, request.ClientID, request.Payload, request.Expires.Unix())
		return err
	})
}

// AuthenticateFixture completes only the synthetic login transaction once.
func (storage *Store) AuthenticateFixture(ctx context.Context, id string, payload []byte) error {
	result, err := storage.db.ExecContext(ctx, `UPDATE auth_requests SET ready=1,payload=? WHERE id=? AND ready=0 AND consumed=0 AND expires>?`, payload, id, time.Now().Unix())
	return one(result, err)
}

func one(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrInvalid
	}
	return nil
}

// SaveCode binds one opaque authorization code; the database retains only its hash.
func (storage *Store) SaveCode(ctx context.Context, id, code string) error {
	result, err := storage.db.ExecContext(ctx, `UPDATE auth_requests SET code_hash=? WHERE id=? AND ready=1 AND consumed=0 AND code_hash IS NULL AND expires>?`, Hash(code), id, time.Now().Unix())
	return one(result, err)
}

// RequestByID loads current account/assignment state as well as the request.
func (storage *Store) RequestByID(ctx context.Context, id string) (Request, error) {
	return storage.request(ctx, "r.id", id)
}

// RequestByCode preserves consumed lineage so validated replay can revoke its grant.
func (storage *Store) RequestByCode(ctx context.Context, code string) (Request, error) {
	return storage.request(ctx, "r.code_hash", Hash(code))
}

func (storage *Store) request(ctx context.Context, column, value string) (Request, error) {
	var request Request
	var expires int64
	query := `SELECT r.id,r.user_id,r.client_id,r.payload,r.expires,r.ready,r.consumed FROM auth_requests r JOIN users u ON u.id=r.user_id JOIN clients c ON c.id=r.client_id JOIN assignments a ON a.user_id=r.user_id AND a.client_id=r.client_id WHERE ` + column + `=? AND r.expires>? AND u.enabled=1 AND c.enabled=1 AND (r.session_digest='' OR EXISTS(SELECT 1 FROM browser_sessions s JOIN accounts ac ON ac.user_id=s.user_id WHERE s.digest=r.session_digest AND s.revoked=0 AND s.version=ac.version AND s.assurance=1 AND s.user_id=r.user_id AND (ac.admin=0 OR ac.factor IS NOT NULL)))`
	err := storage.db.QueryRowContext(ctx, query, value, time.Now().Unix()).Scan(&request.ID, &request.UserID, &request.ClientID, &request.Payload, &expires, &request.Ready, &request.Consumed)
	request.Expires = time.Unix(expires, 0)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrInvalid
	}
	return request, err
}

// ExchangeCode consumes the code and creates its entire grant in one transaction.
func (storage *Store) ExchangeCode(ctx context.Context, id, client string, payload []byte, refresh bool) (Issued, error) {
	var issued Issued
	replay := false
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		var user, storedClient, session string
		var ready, consumed bool
		var expires int64
		var code sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT user_id,client_id,ready,consumed,expires,code_hash,session_digest FROM auth_requests WHERE id=?`, id).Scan(&user, &storedClient, &ready, &consumed, &expires, &code, &session); err != nil {
			return ErrInvalid
		}
		if storedClient != client || !ready || !code.Valid || expires <= time.Now().Unix() {
			return ErrInvalid
		}
		if err := active(ctx, tx, user, client); err != nil {
			return err
		}
		if session != "" {
			if err := activeSession(ctx, tx, session, user, false); err != nil {
				return err
			}
		}
		if consumed {
			if _, err := tx.ExecContext(ctx, `UPDATE grants SET revoked=1 WHERE request_id=?`, id); err != nil {
				return err
			}
			replay = true
			return activity(ctx, tx, "code-reuse", user)
		}
		result, err := tx.ExecContext(ctx, `UPDATE auth_requests SET consumed=1 WHERE id=? AND consumed=0`, id)
		if err = one(result, err); err != nil {
			return err
		}
		grant := NewHandle()
		if _, err = tx.ExecContext(ctx, `INSERT INTO grants(id,request_id,user_id,client_id,expires) VALUES(?,?,?,?,?)`, grant, id, user, client, time.Now().Add(30*24*time.Hour).Unix()); err != nil {
			return err
		}
		issued, err = issue(ctx, tx, grant, payload, refresh)
		if err != nil {
			return err
		}
		return activity(ctx, tx, "code-exchange", user)
	})
	if err != nil {
		return Issued{}, err
	}
	if replay {
		return Issued{}, ErrReplay
	}
	return issued, nil
}

func issue(ctx context.Context, tx *sql.Tx, grant string, payload []byte, refresh bool) (Issued, error) {
	issued := Issued{Access: NewHandle(), Expires: time.Now().Add(5 * time.Minute)}
	_, err := tx.ExecContext(ctx, `INSERT INTO tokens(digest,grant_id,kind,payload,expires) VALUES(?,?,'access',?,?)`, Hash(issued.Access), grant, payload, issued.Expires.Unix())
	if err != nil {
		return Issued{}, err
	}
	if refresh {
		issued.Refresh = NewHandle()
		_, err = tx.ExecContext(ctx, `INSERT INTO tokens(digest,grant_id,kind,payload,expires) VALUES(?,?,'refresh',?,?)`, Hash(issued.Refresh), grant, payload, time.Now().Add(7*24*time.Hour).Unix())
	}
	return issued, err
}

const tokenQuery = `SELECT t.grant_id,g.user_id,g.client_id,t.kind,t.payload,t.expires,t.consumed FROM tokens t JOIN grants g ON g.id=t.grant_id JOIN auth_requests r ON r.id=g.request_id JOIN users u ON u.id=g.user_id JOIN clients c ON c.id=g.client_id JOIN assignments a ON a.user_id=g.user_id AND a.client_id=g.client_id WHERE t.digest=? AND g.revoked=0 AND g.expires>? AND t.expires>? AND u.enabled=1 AND c.enabled=1 AND (r.session_digest='' OR EXISTS(SELECT 1 FROM browser_sessions s JOIN accounts ac ON ac.user_id=s.user_id WHERE s.digest=r.session_digest AND s.revoked=0 AND s.version=ac.version AND s.assurance=1 AND s.user_id=g.user_id AND (ac.admin=0 OR ac.factor IS NOT NULL)))`

func scanToken(row *sql.Row) (Token, error) {
	var token Token
	var expires int64
	err := row.Scan(&token.GrantID, &token.UserID, &token.ClientID, &token.Kind, &token.Payload, &expires, &token.Consumed)
	token.Expires = time.Unix(expires, 0)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrInvalid
	}
	return token, err
}

// Lookup includes consumed refresh lineage; only Rotate may accept or revoke it.
func (storage *Store) Lookup(ctx context.Context, handle string) (Token, error) {
	now := time.Now().Unix()
	return scanToken(storage.db.QueryRowContext(ctx, tokenQuery, Hash(handle), now, now))
}

// Rotate serializes consumption, successor issuance and durable replay revocation.
func (storage *Store) Rotate(ctx context.Context, handle, client string, payload []byte) (Issued, error) {
	var issued Issued
	replay := false
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		now := time.Now().Unix()
		token, err := scanToken(tx.QueryRowContext(ctx, tokenQuery, Hash(handle), now, now))
		if err != nil {
			return err
		}
		if token.Kind != "refresh" || token.ClientID != client {
			return ErrInvalid
		}
		if token.Consumed {
			if _, err = tx.ExecContext(ctx, `UPDATE grants SET revoked=1 WHERE id=?`, token.GrantID); err != nil {
				return err
			}
			replay = true
			return activity(ctx, tx, "refresh-reuse", token.UserID)
		}
		result, err := tx.ExecContext(ctx, `UPDATE tokens SET consumed=1 WHERE digest=? AND consumed=0`, Hash(handle))
		if err = one(result, err); err != nil {
			return err
		}
		issued, err = issue(ctx, tx, token.GrantID, payload, true)
		if err != nil {
			return err
		}
		return activity(ctx, tx, "refresh", token.UserID)
	})
	if err != nil {
		return Issued{}, err
	}
	if replay {
		return Issued{}, ErrReplay
	}
	return issued, nil
}

// Revoke invalidates the whole associated grant, without disclosing unknown tokens.
func (storage *Store) Revoke(ctx context.Context, handle, client string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE grants SET revoked=1 WHERE client_id=? AND id IN (SELECT grant_id FROM tokens WHERE digest=?)`, client, Hash(handle))
		if err != nil {
			return err
		}
		return activity(ctx, tx, "revoke", client)
	})
}

// Invalidate revokes user/client state and pending codes. An empty client disables the user.
func (storage *Store) Invalidate(ctx context.Context, user, client string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if client == "" {
			if _, err := tx.ExecContext(ctx, `UPDATE users SET enabled=0 WHERE id=?`, user); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `DELETE FROM assignments WHERE user_id=? AND client_id=?`, user, client); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE grants SET revoked=1 WHERE user_id=? AND (?='' OR client_id=?)`, user, client, client); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE auth_requests SET consumed=1 WHERE user_id=? AND (?='' OR client_id=?)`, user, client, client); err != nil {
			return err
		}
		return activity(ctx, tx, "invalidate", user)
	})
}

// EndSession revokes matching grants and pending codes without removing assignments.
func (storage *Store) EndSession(ctx context.Context, user, client string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE grants SET revoked=1 WHERE user_id=? AND client_id=?`, user, client); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE auth_requests SET consumed=1 WHERE user_id=? AND client_id=?`, user, client)
		return err
	})
}

// AddRecoveryFixture saves a verifier for an explicitly synthetic recovery code.
func (storage *Store) AddRecoveryFixture(ctx context.Context, user, code string) error {
	_, err := storage.db.ExecContext(ctx, `INSERT INTO recovery_codes(digest,user_id) VALUES(?,?)`, Hash(code), user)
	return err
}

// ConsumeRecovery atomically consumes one code and records the resulting event.
func (storage *Store) ConsumeRecovery(ctx context.Context, user, code string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE recovery_codes SET consumed=1 WHERE digest=? AND user_id=? AND consumed=0 AND EXISTS(SELECT 1 FROM users WHERE id=? AND enabled=1)`, Hash(code), user, user)
		if err = one(result, err); err != nil {
			return err
		}
		return activity(ctx, tx, "recovery-code", user)
	})
}

// ClientActive guards configured clients even before a grant exists.
func (storage *Store) ClientActive(ctx context.Context, id string) bool {
	var enabled int
	return storage.db.QueryRowContext(ctx, `SELECT enabled FROM clients WHERE id=?`, id).Scan(&enabled) == nil && enabled == 1
}
