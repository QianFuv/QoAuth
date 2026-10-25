package identity

import (
	"errors"
	"os"
)

// PrepareDatabase creates only a missing main file, then checks all SQLite files.
// Existing permissions are never silently repaired. Call before opening SQLite.
func PrepareDatabase(path string, create bool) error {
	if create {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			defer file.Close()
			if err = restrictFile(path); err != nil {
				return err
			}
		} else if !os.IsExist(err) {
			return err
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, err := os.Lstat(path + suffix)
		if suffix != "" && os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("database files must be regular private files")
		}
		if err = checkDatabaseFile(path+suffix, info); err != nil {
			return err
		}
		file, err := OpenPrivate(path + suffix)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}
