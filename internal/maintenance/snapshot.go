// Package maintenance owns stopped-installation snapshots and isolated restores.
package maintenance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qoauth/internal/identity"
	"qoauth/internal/store"
)

const markerName = "operation.incomplete"

type manifest struct {
	Version  int    `json:"version"`
	Schema   int    `json:"schema"`
	Created  int64  `json:"created"`
	Database string `json:"database_sha256"`
	Keys     string `json:"keys_sha256"`
	Issuer   string `json:"issuer"`
}

// CheckOperationalDirectory prevents partial copies or snapshots becoming services.
func CheckOperationalDirectory(directory string) error {
	for _, name := range []string{markerName, "manifest.json"} {
		if _, err := os.Lstat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			return errors.New("directory is an incomplete operation or backup snapshot; restore into a new directory")
		}
	}
	return nil
}

func disjointPaths(source, destination string) error {
	first, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	second, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	for _, pair := range [][2]string{{first, second}, {second, first}} {
		relative, err := filepath.Rel(pair[0], pair[1])
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			return errors.New("source and destination must be separate non-nested directories")
		}
	}
	return nil
}

func newDestination(directory string) (*identity.InstallationLock, error) {
	if err := identity.CreateDirectory(directory); err != nil {
		return nil, err
	}
	lock, err := identity.LockDirectory(directory)
	if err != nil {
		return nil, err
	}
	if err = identity.WritePrivate(filepath.Join(directory, markerName), strings.NewReader("Incomplete maintenance operation. Do not serve or initialize. Retry in a new directory.\n")); err != nil {
		lock.Close()
		return nil, err
	}
	return lock, nil
}

func fileDigest(path string) (string, error) {
	file, err := identity.OpenPrivate(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func copyPrivate(source, destination string) error {
	file, err := identity.OpenPrivate(source)
	if err != nil {
		return err
	}
	defer file.Close()
	return identity.WritePrivate(destination, file)
}

// Backup requires the caller's installation lock for the entire DB/key operation.
// Completion is a validated manifest, never the existence of partially copied files.
func Backup(ctx context.Context, directory, destination string) error {
	if err := disjointPaths(directory, destination); err != nil {
		return err
	}
	if err := CheckOperationalDirectory(directory); err != nil {
		return err
	}
	if identity.HasPendingRotation(directory) {
		return errors.New("finish pending rotation before backup")
	}
	keys, err := identity.LoadKeys(directory)
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "qoauth.db")
	if err = identity.PrepareDatabase(path, false); err != nil {
		return err
	}
	state, err := store.InspectInstallation(ctx, path, keys.Binding())
	if err != nil {
		return err
	}
	if state.Maintenance || state.ReadyAfter > time.Now().Unix() {
		return errors.New("finish restore review and quarantine before backup")
	}
	lock, err := newDestination(destination)
	if err != nil {
		return err
	}
	defer lock.Close()
	database, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	defer database.Close()
	snapshotPath := filepath.Join(destination, "qoauth.db")
	if err = identity.PrepareDatabase(snapshotPath, true); err != nil {
		return err
	}
	if err = database.Snapshot(ctx, snapshotPath); err != nil {
		return errors.New("snapshot incomplete; source preserved, use a new destination")
	}
	if err = identity.SyncPrivate(snapshotPath); err != nil {
		return err
	}
	if err = copyPrivate(filepath.Join(directory, "keys.json"), filepath.Join(destination, "keys.json")); err != nil {
		return err
	}
	if _, err = store.InspectSnapshot(ctx, snapshotPath, keys.Binding()); err != nil {
		return err
	}
	metadata := manifest{Version: 1, Schema: state.Schema, Created: time.Now().Unix(), Issuer: state.Issuer}
	metadata.Database, err = fileDigest(snapshotPath)
	if err != nil {
		return err
	}
	metadata.Keys, err = fileDigest(filepath.Join(destination, "keys.json"))
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if err = identity.WritePrivate(filepath.Join(destination, "manifest.json"), bytes.NewReader(encoded)); err != nil {
		return err
	}
	return identity.RemoveMarker(filepath.Join(destination, markerName))
}

func validateSnapshot(ctx context.Context, directory string) (*identity.Keys, error) {
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 4 {
		return nil, errors.New("snapshot must contain only its database, keys, manifest and lock")
	}
	for _, entry := range entries {
		if entry.IsDir() || (entry.Name() != "qoauth.db" && entry.Name() != "keys.json" && entry.Name() != "manifest.json" && entry.Name() != "installation.lock") {
			return nil, errors.New("snapshot contains unexpected members or SQLite sidecars")
		}
	}
	file, err := identity.OpenPrivate(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(io.LimitReader(file, 16385))
	file.Close()
	if err != nil || len(content) > 16384 {
		return nil, errors.New("snapshot manifest exceeds bounds")
	}
	var metadata manifest
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&metadata) != nil || decoder.Decode(&struct{}{}) != io.EOF || metadata.Version != 1 || metadata.Created <= 0 {
		return nil, errors.New("unsupported snapshot manifest")
	}
	for name, expected := range map[string]string{"qoauth.db": metadata.Database, "keys.json": metadata.Keys} {
		actual, err := fileDigest(filepath.Join(directory, name))
		if err != nil || actual != expected {
			return nil, errors.New("snapshot file integrity mismatch")
		}
	}
	keys, err := identity.LoadKeys(directory)
	if err != nil {
		return nil, err
	}
	state, err := store.InspectSnapshot(ctx, filepath.Join(directory, "qoauth.db"), keys.Binding())
	if err != nil {
		return nil, err
	}
	if state.Schema != metadata.Schema || state.Issuer != metadata.Issuer || state.Maintenance || state.ReadyAfter > time.Now().Unix() {
		return nil, errors.New("snapshot installation metadata mismatch or quarantine")
	}
	return keys, nil
}

// Restore holds the snapshot lock and publishes only a sanitized new installation.
// The operator must keep all old issuers stopped through cutover; a local lock
// cannot stop a copied installation elsewhere or revoke independent RP sessions.
func Restore(ctx context.Context, snapshot, destination string) error {
	if err := disjointPaths(snapshot, destination); err != nil {
		return err
	}
	if _, err := os.Stat(snapshot); err != nil {
		return errors.New("snapshot directory unavailable")
	}
	sourceLock, err := identity.LockDirectory(snapshot)
	if err != nil {
		return err
	}
	defer sourceLock.Close()
	oldKeys, err := validateSnapshot(ctx, snapshot)
	if err != nil {
		return err
	}
	lock, err := newDestination(destination)
	if err != nil {
		return err
	}
	defer lock.Close()
	path := filepath.Join(destination, "qoauth.db")
	if err = copyPrivate(filepath.Join(snapshot, "qoauth.db"), path); err != nil {
		return err
	}
	keys, err := identity.FreshRestoreKeys(oldKeys)
	if err != nil {
		return err
	}
	if err = identity.WriteKeys(destination, keys); err != nil {
		return err
	}
	database, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	err = database.RestoreState(ctx, oldKeys.Binding(), keys.Binding(), time.Now().Add(identity.VerificationWindow))
	closeErr := database.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if _, err = store.InspectSnapshot(ctx, path, keys.Binding()); err != nil {
		return err
	}
	if err = identity.SyncPrivate(path); err != nil {
		return err
	}
	return identity.RemoveMarker(filepath.Join(destination, markerName))
}
