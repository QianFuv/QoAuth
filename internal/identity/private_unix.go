//go:build !windows

package identity

import (
	"errors"
	"os"
	"syscall"
)

func restrictFile(path string) error                            { return os.Chmod(path, 0600) }
func restrictDirectory(path string) error                       { return os.Chmod(path, 0700) }
func checkPrivateDirectory(path string, info os.FileInfo) error { return checkPrivateFile(path, info) }
func checkPrivateFile(_ string, info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (stat.Uid != uint32(os.Geteuid()) && stat.Uid != 0) {
		return errors.New("storage owner is not trusted")
	}
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("key file must be owner-only")
	}
	return nil
}

func checkDatabaseFile(path string, info os.FileInfo) error { return checkPrivateFile(path, info) }
