package identity

import (
	"errors"
	"os"
)

// PrepareDirectory creates private installation storage, or validates existing storage.
func PrepareDirectory(path string) error {
	err := os.Mkdir(path, 0700)
	if err == nil {
		return restrictDirectory(path)
	}
	if !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("data directory must be a real private directory")
	}
	return checkPrivateDirectory(path, info)
}
