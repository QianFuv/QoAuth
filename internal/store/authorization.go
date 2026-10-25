package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"
)

// LoginRequest keeps validated authorization input unbound until a real login.
type LoginRequest struct {
	ID, ClientID, Browser string
	ClientGeneration      int64
	Payload               []byte
	Expires               int64
	Completed             bool
}

func (storage *Store) CreateLogin(ctx context.Context, request LoginRequest) error {
	admitted := false
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := pruneAnonymous(ctx, tx); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO login_requests(id,client_id,browser_digest,payload,expires) SELECT ?,?,?,?,? WHERE EXISTS(SELECT 1 FROM clients WHERE id=? AND enabled=1 AND generation=?) AND EXISTS(SELECT 1 FROM browser_sessions WHERE digest=? AND revoked=0 AND expires>?) AND (SELECT count(*) FROM login_requests WHERE browser_digest=?)<16 AND (SELECT count(*) FROM login_requests)<2048`, request.ID, request.ClientID, request.Browser, request.Payload, request.Expires, request.ClientID, request.ClientGeneration, request.Browser, time.Now().Unix(), request.Browser)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		admitted = count == 1
		return err
	})
	if err == nil && !admitted {
		return ErrInvalid
	}
	return err
}

func pruneAnonymous(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM login_requests WHERE id IN (SELECT id FROM login_requests WHERE expires<=? LIMIT 64)`, time.Now().Unix()); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM browser_sessions WHERE digest IN (SELECT digest FROM browser_sessions WHERE user_id IS NULL AND (revoked=1 OR expires<=?) LIMIT 64)`, time.Now().Unix())
	return err
}
func (storage *Store) Login(ctx context.Context, id string) (LoginRequest, error) {
	var request LoginRequest
	err := storage.db.QueryRowContext(ctx, `SELECT id,client_id,browser_digest,payload,expires,completed FROM login_requests WHERE id=? AND expires>?`, id, time.Now().Unix()).Scan(&request.ID, &request.ClientID, &request.Browser, &request.Payload, &request.Expires, &request.Completed)
	if err == sql.ErrNoRows {
		err = ErrInvalid
	}
	return request, err
}
func (storage *Store) CancelLogin(ctx context.Context, id, browser string) error {
	result, err := storage.db.ExecContext(ctx, `DELETE FROM login_requests WHERE id=? AND browser_digest=? AND completed=0`, id, browser)
	return one(result, err)
}
func (storage *Store) Consent(ctx context.Context, user, client string) ([]byte, error) {
	var payload []byte
	err := storage.db.QueryRowContext(ctx, `SELECT payload FROM consents WHERE user_id=? AND client_id=?`, user, client).Scan(&payload)
	return payload, err
}

// CompleteLogin commits session binding, consent and the issuable request together.
func (storage *Store) CompleteLogin(ctx context.Context, login LoginRequest, session Session, payload, consent []byte, explicitApproval bool) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		if err := activeSession(ctx, tx, session.Digest, session.UserID, false); err != nil {
			return err
		}
		if session.Expires <= time.Now().Unix() {
			return ErrInvalid
		}
		if err := active(ctx, tx, session.UserID, login.ClientID); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE login_requests SET completed=1 WHERE id=? AND browser_digest=? AND completed=0 AND expires>?`, login.ID, session.Digest, time.Now().Unix())
		if err = one(result, err); err != nil {
			return err
		}
		if explicitApproval {
			if _, err = tx.ExecContext(ctx, `INSERT INTO consents(user_id,client_id,payload) VALUES(?,?,?) ON CONFLICT(user_id,client_id) DO UPDATE SET payload=excluded.payload`, session.UserID, login.ClientID, consent); err != nil {
				return err
			}
		} else {
			var existing []byte
			if err = tx.QueryRowContext(ctx, `SELECT payload FROM consents WHERE user_id=? AND client_id=?`, session.UserID, login.ClientID).Scan(&existing); err != nil || !consentCovers(existing, consent) {
				return ErrInvalid
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO auth_requests(id,user_id,client_id,payload,expires,ready,session_digest) VALUES(?,?,?,?,?,1,?)`, login.ID, session.UserID, login.ClientID, payload, time.Now().Add(time.Minute).Unix(), session.Digest); err != nil {
			return err
		}
		return activity(ctx, tx, "authorize", session.UserID)
	})
}

// Consent is the exact set approved for one application and its resources.
type Consent struct{ Scopes, Audiences []string }

func consentCovers(existing, requested []byte) bool {
	var approved, required Consent
	if json.Unmarshal(existing, &approved) != nil || json.Unmarshal(requested, &required) != nil {
		return false
	}
	for _, scope := range required.Scopes {
		if !slices.Contains(approved.Scopes, scope) {
			return false
		}
	}
	for _, audience := range required.Audiences {
		if !slices.Contains(approved.Audiences, audience) {
			return false
		}
	}
	return true
}
