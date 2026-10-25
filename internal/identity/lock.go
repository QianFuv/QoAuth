package identity

import (
	"errors"
	"os"
	"path/filepath"
)

// InstallationLock serializes all local commands, including the serving process.
// The lock file is permanent: unlinking it would let another process lock a new inode.
type InstallationLock struct{ file *os.File }

// LockDirectory requires private storage and obtains a nonblocking OS lock.
// The operating system releases ownership when the process exits or crashes.
func LockDirectory(directory string) (*InstallationLock, error) {
	if err := PrepareDirectory(directory); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, "installation.lock")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		if err = restrictFile(path); err != nil {
			file.Close()
			return nil, err
		}
	} else if os.IsExist(err) {
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() || checkPrivateFile(path, info) != nil {
			return nil, errors.New("installation lock must be a private regular file")
		}
		file, err = os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			opened, statErr := file.Stat()
			if statErr != nil || !os.SameFile(info, opened) {
				file.Close()
				return nil, errors.New("installation lock changed while opening")
			}
		}
	}
	if err != nil {
		return nil, errors.New("installation lock unavailable")
	}
	if err = lockFile(file); err != nil {
		file.Close()
		return nil, errors.New("installation is busy; stop its server and other local commands first")
	}
	return &InstallationLock{file: file}, nil
}

// Close releases the OS lock without deleting its persistent rendezvous file.
func (lock *InstallationLock) Close() error { return lock.file.Close() }
