//go:build !windows

package identity

import (
	"errors"
	"os"
)

func restrictFile(path string) error                            { return os.Chmod(path, 0600) }
func restrictDirectory(path string) error                       { return os.Chmod(path, 0700) }
func checkPrivateDirectory(path string, info os.FileInfo) error { return checkPrivateFile(path, info) }
func checkPrivateFile(_ string, info os.FileInfo) error {
	if info.Mode().Perm()&0077 != 0 {
		return errors.New("key file must be owner-only")
	}
	return nil
}
