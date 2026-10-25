package store

import (
	"context"
	"testing"
	"time"
)

func TestReadinessIsReadOnlyAndChecksInstallation(t *testing.T) {
	storage, _, _, _, _ := ownerFixture(t)
	const issuer = "https://identity.example.test"
	const binding = "synthetic-key-id"
	if storage.Ready(t.Context(), binding, issuer) == nil {
		t.Fatal("unbound issuer reported ready")
	}
	state, _ := storage.InstallationState(t.Context())
	if state.Issuer != "" {
		t.Fatal("readiness adopted issuer")
	}
	if err := storage.CheckServing(t.Context(), binding, issuer, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := storage.Ready(t.Context(), binding, issuer); err != nil {
		t.Fatal(err)
	}
	if storage.Ready(t.Context(), "wrong", issuer) == nil || storage.Ready(t.Context(), binding, "https://wrong.example.test") == nil {
		t.Fatal("mismatched installation ready")
	}
	for _, statement := range []string{`UPDATE installation SET maintenance=1`, `UPDATE installation SET maintenance=0,ready_after=9223372036854775807`} {
		if _, err := storage.db.Exec(statement); err != nil {
			t.Fatal(err)
		}
		if storage.Ready(t.Context(), binding, issuer) == nil {
			t.Fatal("blocked restore ready")
		}
	}
	if _, err := storage.db.Exec(`UPDATE installation SET ready_after=0`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if storage.Ready(ctx, binding, issuer) == nil {
		t.Fatal("canceled read reported ready")
	}
	storage.db.SetMaxOpenConns(1)
	connection, err := storage.db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = storage.Ready(t.Context(), binding, issuer)
	connection.Close()
	if err == nil || time.Since(started) > 2*time.Second {
		t.Fatal("readiness did not bound pool wait")
	}
	storage.Close()
	if storage.Ready(t.Context(), binding, issuer) == nil {
		t.Fatal("closed storage ready")
	}
}
