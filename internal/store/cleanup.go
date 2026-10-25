package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CleanupResult contains row counts only, never credential or account contents.
type CleanupResult struct {
	Tokens, Grants, Requests, Logins, Sessions, RecoveryCodes, Activity int64
}

// Cleanup performs one bounded dependency-ordered maintenance transaction.
// Replay lineage is kept until the grant and every token in it have expired,
// even if the grant is revoked or the browser cookie has already expired.
func (storage *Store) Cleanup(ctx context.Context, limit int) (CleanupResult, error) {
	if limit < 1 || limit > 256 {
		return CleanupResult{}, errors.New("cleanup batch must be between 1 and 256 rows per table")
	}
	now := time.Now().Unix()
	cutoff := now - int64(30*24*time.Hour/time.Second)
	var counts CleanupResult
	err := storage.transaction(ctx, func(tx *sql.Tx) error {
		operations := []struct {
			statement string
			args      []any
			count     *int64
		}{
			{`DELETE FROM tokens WHERE digest IN (SELECT t.digest FROM grants g JOIN tokens t ON t.grant_id=g.id WHERE g.expires<=? AND NOT EXISTS(SELECT 1 FROM tokens future WHERE future.grant_id=g.id AND future.expires>?) LIMIT ?)`, []any{now, now, limit}, &counts.Tokens},
			{`DELETE FROM grants WHERE id IN (SELECT g.id FROM grants g WHERE g.expires<=? AND NOT EXISTS(SELECT 1 FROM tokens t WHERE t.grant_id=g.id) LIMIT ?)`, []any{now, limit}, &counts.Grants},
			{`DELETE FROM auth_requests WHERE id IN (SELECT r.id FROM auth_requests r WHERE r.expires<=? AND NOT EXISTS(SELECT 1 FROM grants g WHERE g.request_id=r.id) LIMIT ?)`, []any{now, limit}, &counts.Requests},
			{`DELETE FROM login_requests WHERE id IN (SELECT id FROM login_requests WHERE expires<=? LIMIT ?)`, []any{now, limit}, &counts.Logins},
			{`DELETE FROM browser_sessions WHERE digest IN (SELECT s.digest FROM browser_sessions s WHERE (s.expires<=? OR s.revoked=1) AND NOT EXISTS(SELECT 1 FROM auth_requests r WHERE r.session_digest=s.digest) AND NOT EXISTS(SELECT 1 FROM login_requests l WHERE l.browser_digest=s.digest) LIMIT ?)`, []any{now, limit}, &counts.Sessions},
			{`DELETE FROM recovery_codes WHERE digest IN (SELECT digest FROM recovery_codes WHERE consumed=1 LIMIT ?)`, []any{limit}, &counts.RecoveryCodes},
			{`DELETE FROM activity WHERE id IN (SELECT id FROM activity WHERE recorded_at<=? ORDER BY recorded_at,id LIMIT ?)`, []any{cutoff, limit}, &counts.Activity},
		}
		var total int64
		for _, operation := range operations {
			result, err := tx.ExecContext(ctx, operation.statement, operation.args...)
			if err != nil {
				return err
			}
			*operation.count, err = result.RowsAffected()
			if err != nil {
				return err
			}
			total += *operation.count
		}
		if total > 0 {
			return activity(ctx, tx, "maintenance-cleanup", "host")
		}
		return nil
	})
	if err != nil {
		return CleanupResult{}, err
	}
	return counts, nil
}
