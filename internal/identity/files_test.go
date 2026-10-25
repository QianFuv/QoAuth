package identity

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("injected disk-copy failure") }

func TestPrivateCopyFailureAndHardLinkRejection(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	if err := PrepareDirectory(directory); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "private.json")
	if err := WritePrivate(path, failedReader{}); err == nil {
		t.Fatal("failed copy accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed new output remained published")
	}
	if err := WritePrivate(path, strings.NewReader("synthetic")); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivate(path, strings.NewReader("replacement")); err == nil {
		t.Fatal("existing file overwritten")
	}
	if err := os.Link(path, filepath.Join(directory, "alias.json")); err != nil {
		t.Fatal(err)
	}
	if file, err := OpenPrivate(path); err == nil {
		file.Close()
		t.Fatal("hard-linked secret accepted")
	}
}
