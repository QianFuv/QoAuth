package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Account exposes current authorization state; credential fields are never JSON.
type Account struct {
	ID                     string `json:"id"`
	Username               string `json:"username"`
	Admin                  bool   `json:"admin"`
	Owner                  bool   `json:"owner"`
	Enabled                bool   `json:"enabled"`
	RequireMfa             bool   `json:"require_mfa"`
	Version                int    `json:"-"`
	Password               string `json:"-"`
	Factor, Pending        []byte `json:"-"`
	PendingUntil, LastStep int64  `json:"-"`
}

const accountColumns = `a.user_id,a.username,a.password,a.admin,a.owner,a.version,a.require_mfa,a.factor,a.pending_factor,a.pending_until,a.last_step,u.enabled`

func scanAccount(row interface{ Scan(...any) error }) (Account, error) {
	var account Account
	err := row.Scan(&account.ID, &account.Username, &account.Password, &account.Admin, &account.Owner, &account.Version, &account.RequireMfa, &account.Factor, &account.Pending, &account.PendingUntil, &account.LastStep, &account.Enabled)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrInvalid
	}
	return account, err
}

// Bootstrap claims exactly one new installation; it never reopens initialized state.
func (storage *Store) Bootstrap(ctx context.Context, username, password, keyID string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM installation)+(SELECT count(*) FROM users)`).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("installation is already initialized or contains fixture data")
		}
		id := NewHandle()
		if _, err := tx.ExecContext(ctx, `INSERT INTO installation VALUES(1,?)`, keyID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(id) VALUES(?)`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO accounts(user_id,username,password,admin,owner,require_mfa) VALUES(?,?,?,1,1,1)`, id, username, password); err != nil {
			return err
		}
		return activity(ctx, tx, "initialize", id)
	})
}

// CanBootstrap prevents even key creation when an installation already owns state.
func (storage *Store) CanBootstrap(ctx context.Context) bool {
	var count int
	return storage.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM installation)+(SELECT count(*) FROM users)`).Scan(&count) == nil && count == 0
}

// CheckInstallation binds a database to its original signing-key identity.
func (storage *Store) CheckInstallation(ctx context.Context, keyID string) error {
	var stored string
	if err := storage.db.QueryRowContext(ctx, `SELECT key_id FROM installation WHERE id=1`).Scan(&stored); err != nil || stored != keyID {
		return errors.New("installation or matching keys unavailable")
	}
	return nil
}
func (storage *Store) Account(ctx context.Context, id string) (Account, error) {
	return scanAccount(storage.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts a JOIN users u ON u.id=a.user_id WHERE a.user_id=?`, id))
}
func (storage *Store) AccountByName(ctx context.Context, name string) (Account, error) {
	return scanAccount(storage.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts a JOIN users u ON u.id=a.user_id WHERE a.username=?`, name))
}
func (storage *Store) Accounts(ctx context.Context) ([]Account, error) {
	rows, err := storage.db.QueryContext(ctx, `SELECT `+accountColumns+` FROM accounts a JOIN users u ON u.id=a.user_id ORDER BY a.username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Account{}
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	return result, rows.Err()
}
func (storage *Store) AddAccount(ctx context.Context, name, password, actor string) (string, error) {
	id := NewHandle()
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		actor, err = mutationActor(ctx, tx, actor, "")
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(id) VALUES(?)`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO accounts(user_id,username,password) VALUES(?,?,?)`, id, name, password); err != nil {
			return err
		}
		return activity(ctx, tx, "user-create", actor)
	})
	return id, err
}

// Session is a revocable browser record; its raw cookie handle is never stored.
type Session struct {
	Digest, UserID, Csrf string
	Version              int
	Assurance            bool
	AuthTime, Expires    int64
}

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var session Session
	err := row.Scan(&session.Digest, &session.UserID, &session.Csrf, &session.Version, &session.Assurance, &session.AuthTime, &session.Expires)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrInvalid
	}
	return session, err
}
func (storage *Store) Session(ctx context.Context, handle string) (Session, error) {
	return scanSession(storage.db.QueryRowContext(ctx, `SELECT s.digest,COALESCE(s.user_id,''),s.csrf,s.version,s.assurance,s.auth_time,s.expires FROM browser_sessions s LEFT JOIN accounts a ON a.user_id=s.user_id LEFT JOIN users u ON u.id=s.user_id WHERE s.digest=? AND s.revoked=0 AND s.expires>? AND (s.user_id IS NULL OR (u.enabled=1 AND a.version=s.version))`, Hash(handle), time.Now().Unix()))
}
func (storage *Store) AnonymousSession(ctx context.Context) (string, error) {
	handle := NewHandle()
	_, err := storage.db.ExecContext(ctx, `INSERT INTO browser_sessions(digest,csrf,expires) VALUES(?,?,?)`, Hash(handle), NewHandle(), time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		return "", err
	}
	return handle, nil
}
func activeSession(ctx context.Context, tx *sql.Tx, digest, user string, allowPartial bool) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM browser_sessions s JOIN accounts a ON a.user_id=s.user_id JOIN users u ON u.id=s.user_id WHERE s.digest=? AND s.user_id=? AND s.revoked=0 AND s.version=a.version AND u.enabled=1 AND (? OR (s.assurance=1 AND (a.admin=0 OR a.factor IS NOT NULL)))`, digest, user, allowPartial).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrInvalid
	}
	return nil
}

func mutationActor(ctx context.Context, tx *sql.Tx, digest, target string) (string, error) {
	var user string
	var admin bool
	err := tx.QueryRowContext(ctx, `SELECT s.user_id,a.admin FROM browser_sessions s JOIN accounts a ON a.user_id=s.user_id JOIN users u ON u.id=s.user_id WHERE s.digest=? AND s.revoked=0 AND s.version=a.version AND u.enabled=1 AND s.assurance=1 AND s.expires>? AND s.auth_time>? AND (a.admin=0 OR a.factor IS NOT NULL)`, digest, time.Now().Unix(), time.Now().Add(-5*time.Minute).Unix()).Scan(&user, &admin)
	if err != nil || (!admin && (target == "" || user != target)) {
		return "", ErrInvalid
	}
	return user, nil
}
func revokeSession(ctx context.Context, tx *sql.Tx, digest string) error {
	_, err := tx.ExecContext(ctx, `UPDATE browser_sessions SET revoked=1 WHERE digest=?; UPDATE grants SET revoked=1 WHERE request_id IN (SELECT id FROM auth_requests WHERE session_digest=?); UPDATE auth_requests SET consumed=1 WHERE session_digest=?`, digest, digest, digest)
	return err
}
func installSession(ctx context.Context, tx *sql.Tx, old, user string, version int, assurance bool, authTime int64) (string, error) {
	var enabled, liveVersion int
	if err := tx.QueryRowContext(ctx, `SELECT u.enabled,a.version FROM accounts a JOIN users u ON u.id=a.user_id WHERE user_id=?`, user).Scan(&enabled, &liveVersion); err != nil || enabled != 1 || liveVersion != version {
		return "", ErrInvalid
	}
	var live int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM browser_sessions WHERE digest=? AND revoked=0 AND expires>?`, old, time.Now().Unix()).Scan(&live); err != nil || live != 1 {
		return "", ErrInvalid
	}
	handle := NewHandle()
	lifetime := 5 * time.Minute
	if assurance {
		lifetime = 12 * time.Hour
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO browser_sessions(digest,user_id,csrf,version,assurance,auth_time,expires) VALUES(?,?,?,?,?,?,?)`, Hash(handle), user, NewHandle(), version, assurance, authTime, time.Now().Add(lifetime).Unix()); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE login_requests SET browser_digest=? WHERE browser_digest=? AND completed=0`, Hash(handle), old); err != nil {
		return "", err
	}
	if err := revokeSession(ctx, tx, old); err != nil {
		return "", err
	}
	return handle, nil
}

// PasswordSession checks the observed password version again inside the transaction.
func (storage *Store) PasswordSession(ctx context.Context, old string, account Account) (string, error) {
	var handle string
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		handle, err = installSession(ctx, tx, old, account.ID, account.Version, !account.Admin && !account.RequireMfa && len(account.Factor) == 0, time.Now().Unix())
		if err != nil {
			return err
		}
		return activity(ctx, tx, "password-login", account.ID)
	})
	if err != nil {
		return "", err
	}
	return handle, nil
}
func (storage *Store) RevokeSession(ctx context.Context, digest, user string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, digest, user, true); err != nil {
			return err
		}
		if err := revokeSession(ctx, tx, digest); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM login_requests WHERE browser_digest=?`, digest); err != nil {
			return err
		}
		return activity(ctx, tx, "session-revoke", user)
	})
}
func (storage *Store) Sessions(ctx context.Context, user string) ([]Session, error) {
	rows, err := storage.db.QueryContext(ctx, `SELECT digest,user_id,csrf,version,assurance,auth_time,expires FROM browser_sessions WHERE user_id=? AND revoked=0 AND expires>?`, user, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Session{}
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		session.Csrf = ""
		result = append(result, session)
	}
	return result, rows.Err()
}
func invalidateAccount(ctx context.Context, tx *sql.Tx, user string) error {
	_, err := tx.ExecContext(ctx, `UPDATE browser_sessions SET revoked=1 WHERE user_id=?; DELETE FROM login_requests WHERE browser_digest IN (SELECT digest FROM browser_sessions WHERE user_id=?); UPDATE auth_requests SET consumed=1 WHERE user_id=?; UPDATE grants SET revoked=1 WHERE user_id=?; UPDATE accounts SET pending_factor=NULL,pending_until=0 WHERE user_id=?`, user, user, user, user, user)
	return err
}

// ResetAccount is called only after server authorization or explicit host recovery.
func (storage *Store) ResetAccount(ctx context.Context, user, password string, resetFactor bool, actor string) error {
	return storage.resetAccount(ctx, user, password, resetFactor, actor, false)
}

// RecoverOwner is an explicit host-only command, never routed through HTTP.
func (storage *Store) RecoverOwner(ctx context.Context, password string) error {
	var owner string
	if err := storage.db.QueryRowContext(ctx, `SELECT user_id FROM accounts WHERE owner=1`).Scan(&owner); err != nil {
		return err
	}
	return storage.resetAccount(ctx, owner, password, true, "host-recovery", true)
}
func (storage *Store) resetAccount(ctx context.Context, user, password string, resetFactor bool, actor string, isHost bool) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if !isHost {
			var err error
			actor, err = mutationActor(ctx, tx, actor, user)
			if err != nil {
				return err
			}
		}
		result, err := tx.ExecContext(ctx, `UPDATE accounts SET password=?,version=version+1,pending_factor=NULL,pending_until=0 WHERE user_id=?`, password, user)
		if err = one(result, err); err != nil {
			return err
		}
		if resetFactor {
			if _, err = tx.ExecContext(ctx, `UPDATE accounts SET factor=NULL,last_step=-1,require_mfa=1 WHERE user_id=?; DELETE FROM recovery_codes WHERE user_id=?`, user, user); err != nil {
				return err
			}
		}
		if err = invalidateAccount(ctx, tx, user); err != nil {
			return err
		}
		return activity(ctx, tx, "credential-reset", actor)
	})
}
func (storage *Store) DisableAccount(ctx context.Context, user, actor string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		actor, err = mutationActor(ctx, tx, actor, "")
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE users SET enabled=0 WHERE id=? AND EXISTS(SELECT 1 FROM accounts WHERE user_id=? AND owner=0)`, user, user)
		if err = one(result, err); err != nil {
			return err
		}
		if err = invalidateAccount(ctx, tx, user); err != nil {
			return err
		}
		return activity(ctx, tx, "user-disable", actor)
	})
}
func (storage *Store) StageFactor(ctx context.Context, session Session, sealed []byte) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, session.Digest, session.UserID, true); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE accounts SET pending_factor=?,pending_until=? WHERE user_id=? AND version=? AND factor IS NULL`, sealed, time.Now().Add(5*time.Minute).Unix(), session.UserID, session.Version)
		return one(result, err)
	})
}

// ConfirmFactor consumes enrollment proof and replaces all older security state.
func (storage *Store) ConfirmFactor(ctx context.Context, session Session, sealed []byte, step int64) (string, []string, error) {
	var handle string
	codes := []string{}
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, session.Digest, session.UserID, true); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE accounts SET factor=pending_factor,pending_factor=NULL,pending_until=0,last_step=?,require_mfa=1,version=version+1 WHERE user_id=? AND version=? AND factor IS NULL AND pending_factor=? AND pending_until>?`, step, session.UserID, session.Version, sealed, time.Now().Unix())
		if err = one(result, err); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id=?`, session.UserID); err != nil {
			return err
		}
		for range 8 {
			code := NewHandle()
			codes = append(codes, code)
			if _, err = tx.ExecContext(ctx, `INSERT INTO recovery_codes(digest,user_id) VALUES(?,?)`, Hash(code), session.UserID); err != nil {
				return err
			}
		}
		if err = invalidateAccount(ctx, tx, session.UserID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE browser_sessions SET revoked=0 WHERE digest=?`, session.Digest); err != nil {
			return err
		}
		handle, err = installSession(ctx, tx, session.Digest, session.UserID, session.Version+1, true, session.AuthTime)
		if err != nil {
			return err
		}
		return activity(ctx, tx, "mfa-enroll", session.UserID)
	})
	if err != nil {
		return "", nil, err
	}
	return handle, codes, nil
}
func (storage *Store) ProveFactor(ctx context.Context, session Session, sealed []byte, step int64) (string, error) {
	var handle string
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, session.Digest, session.UserID, true); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE accounts SET last_step=? WHERE user_id=? AND version=? AND factor=? AND last_step<?`, step, session.UserID, session.Version, sealed, step)
		if err = one(result, err); err != nil {
			return err
		}
		handle, err = installSession(ctx, tx, session.Digest, session.UserID, session.Version, true, session.AuthTime)
		if err != nil {
			return err
		}
		return activity(ctx, tx, "mfa-verify", session.UserID)
	})
	if err != nil {
		return "", err
	}
	return handle, nil
}

// RecoverFactor is password-session plus recovery proof, never an MFA-complete login.
func (storage *Store) RecoverFactor(ctx context.Context, session Session, code string) (string, error) {
	var handle string
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, session.Digest, session.UserID, true); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE recovery_codes SET consumed=1 WHERE digest=? AND user_id=? AND consumed=0`, Hash(code), session.UserID)
		if err = one(result, err); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE accounts SET factor=NULL,pending_factor=NULL,pending_until=0,last_step=-1,require_mfa=1,version=version+1 WHERE user_id=?; UPDATE recovery_codes SET consumed=1 WHERE user_id=?`, session.UserID, session.UserID); err != nil {
			return err
		}
		if err = invalidateAccount(ctx, tx, session.UserID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE browser_sessions SET revoked=0 WHERE digest=?`, session.Digest); err != nil {
			return err
		}
		handle, err = installSession(ctx, tx, session.Digest, session.UserID, session.Version+1, false, session.AuthTime)
		if err != nil {
			return err
		}
		return activity(ctx, tx, "mfa-recovery", session.UserID)
	})
	if err != nil {
		return "", err
	}
	return handle, nil
}

// Application is the small registration/authorization model, not an RBAC engine.
type Application struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Redirects       []string `json:"redirects"`
	LogoutRedirects []string `json:"logout_redirects"`
	Scopes          []string `json:"scopes"`
	Audiences       []string `json:"audiences"`
	Public          bool     `json:"public"`
	Resource        bool     `json:"resource"`
	SecretHash      string   `json:"secret_hash,omitempty"`
}

func (storage *Store) Application(ctx context.Context, id string) (Application, error) {
	var result Application
	var payload []byte
	err := storage.db.QueryRowContext(ctx, `SELECT a.payload FROM client_config a JOIN clients c ON c.id=a.client_id WHERE client_id=? AND c.enabled=1`, id).Scan(&payload)
	if err != nil {
		return result, ErrInvalid
	}
	err = json.Unmarshal(payload, &result)
	return result, err
}
func (storage *Store) Applications(ctx context.Context) ([]Application, error) {
	rows, err := storage.db.QueryContext(ctx, `SELECT a.payload FROM client_config a JOIN clients c ON c.id=a.client_id WHERE c.enabled=1 ORDER BY a.client_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Application{}
	for rows.Next() {
		var payload []byte
		var app Application
		if err = rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &app); err != nil {
			return nil, err
		}
		app.SecretHash = ""
		result = append(result, app)
	}
	return result, rows.Err()
}
func (storage *Store) AddApplication(ctx context.Context, app Application, actor string) error {
	payload, err := json.Marshal(app)
	if err != nil {
		return err
	}
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		actor, err = mutationActor(ctx, tx, actor, "")
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO clients(id) VALUES(?)`, app.ID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO client_config(client_id,payload) VALUES(?,?)`, app.ID, payload); err != nil {
			return err
		}
		return activity(ctx, tx, "application-create", actor)
	})
}
func (storage *Store) DisableApplication(ctx context.Context, id, actor string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		actor, err = mutationActor(ctx, tx, actor, "")
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE clients SET enabled=0 WHERE id=?; UPDATE grants SET revoked=1 WHERE client_id=?; UPDATE auth_requests SET consumed=1 WHERE client_id=?; DELETE FROM login_requests WHERE client_id=?; DELETE FROM consents WHERE client_id=?`, id, id, id, id, id); err != nil {
			return err
		}
		return activity(ctx, tx, "application-disable", actor)
	})
}
func (storage *Store) Assign(ctx context.Context, user, client, actor string, allow bool) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var err error
		actor, err = mutationActor(ctx, tx, actor, "")
		if err != nil {
			return err
		}
		if allow {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO assignments VALUES(?,?)`, user, client); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, `DELETE FROM assignments WHERE user_id=? AND client_id=?; DELETE FROM consents WHERE user_id=? AND client_id=?; UPDATE grants SET revoked=1 WHERE user_id=? AND client_id=?; UPDATE auth_requests SET consumed=1 WHERE user_id=? AND client_id=?`, user, client, user, client, user, client, user, client); err != nil {
				return err
			}
		}
		return activity(ctx, tx, "assignment-change", actor)
	})
}

// WithdrawConsent permanently invalidates the prior grant generation.
func (storage *Store) WithdrawConsent(ctx context.Context, user, client, actor string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, actor, user, false); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM consents WHERE user_id=? AND client_id=?; UPDATE grants SET revoked=1 WHERE user_id=? AND client_id=?; UPDATE auth_requests SET consumed=1 WHERE user_id=? AND client_id=?`, user, client, user, client, user, client); err != nil {
			return err
		}
		return activity(ctx, tx, "consent-withdraw", user)
	})
}

type Activity struct {
	Action  string `json:"action"`
	Subject string `json:"subject"`
	Time    int64  `json:"time"`
}

func (storage *Store) Activity(ctx context.Context) ([]Activity, error) {
	rows, err := storage.db.QueryContext(ctx, `SELECT action,subject,recorded_at FROM activity ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Activity{}
	for rows.Next() {
		var event Activity
		if err = rows.Scan(&event.Action, &event.Subject, &event.Time); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
