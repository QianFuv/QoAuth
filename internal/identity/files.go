package identity

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// CheckPath rejects symlink/reparse traversal through existing path components.
// Host-controlled parent directories are still required; hostile host races are out of scope.
func CheckPath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || checkReparse(current) != nil) {
			return errors.New("storage paths cannot traverse links or reparse points")
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}

// OpenPrivate verifies permissions, opened-file identity and absence of hard-link aliases.
func OpenPrivate(path string) (*os.File, error) { return openPrivate(path, os.O_RDONLY) }

func openPrivate(path string, flags int) (*os.File, error) {
	if err := CheckPath(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || checkDatabaseFile(path, info) != nil {
		return nil, errors.New("private regular file required")
	}
	file, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || singleLink(file, opened) != nil {
		file.Close()
		return nil, errors.New("storage file changed or has hard-link aliases")
	}
	return file, nil
}

// WritePrivate exclusively creates private output; failures remove only that new file.
func WritePrivate(path string, input io.Reader) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	isComplete := false
	defer func() {
		file.Close()
		if !isComplete {
			os.Remove(path)
		}
	}()
	if err = restrictFile(path); err != nil {
		return err
	}
	if _, err = io.Copy(file, input); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	isComplete = true
	return nil
}

// SyncPrivate flushes a completed SQLite snapshot before its manifest is published.
func SyncPrivate(path string) error {
	file, err := openPrivate(path, os.O_RDWR)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

// RemoveMarker removes only the caller's completed-operation marker durably.
func RemoveMarker(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

// CreateDirectory rejects even an empty existing destination instead of adopting it.
func CreateDirectory(path string) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return errors.New("destination must be a new directory with an existing trusted parent")
	}
	if err := restrictDirectory(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
