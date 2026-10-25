package store

import (
	"context"
	"database/sql"
)

// RotateSigningBinding commits one key transition; retrying a committed one is safe.
// Its caller holds the installation lock and has already synced the pending key file.
func (storage *Store) RotateSigningBinding(ctx context.Context, previous, next string) error {
	return storage.transaction(ctx, func(tx *sql.Tx) error {
		var current string
		if err := tx.QueryRowContext(ctx, `SELECT key_id FROM installation WHERE id=1`).Scan(&current); err != nil {
			return err
		}
		if current == next {
			return nil
		}
		if current != previous {
			return ErrInvalid
		}
		if _, err := tx.ExecContext(ctx, `UPDATE installation SET key_id=? WHERE id=1`, next); err != nil {
			return err
		}
		return activity(ctx, tx, "signing-rotate", "host")
	})
}
