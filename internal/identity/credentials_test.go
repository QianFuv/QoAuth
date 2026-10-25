package identity

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestNewPasswordUnicodePolicy(t *testing.T) {
	for _, invalid := range []string{strings.Repeat("😀", 8), strings.Repeat("e\u0301", 14), strings.Repeat("a", 1025), strings.Repeat("e\u0301", 342), "PASSWORDPASSWORD", "invalid-utf8-credential\xff"} {
		if err := ValidateNewPassword(invalid); !errors.Is(err, ErrPasswordPolicy) || strings.Contains(err.Error(), invalid) {
			t.Fatal("invalid password accepted or echoed")
		}
	}
	for _, valid := range []string{strings.Repeat("😀", 15), strings.Repeat("e\u0301", 15), strings.Repeat("a", 1024), "  spaces are part of my password  "} {
		if err := ValidateNewPassword(valid); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := PasswordHash(strings.Repeat("e\u0301", 15))
	if err != nil || !CheckPassword(encoded, strings.Repeat("é", 15)) {
		t.Fatal("creation and verification normalize inconsistently")
	}
	spaced := "  spaces are part of my password  "
	encoded, err = PasswordHash(spaced)
	if err != nil || !CheckPassword(encoded, spaced) || CheckPassword(encoded, strings.TrimSpace(spaced)) {
		t.Fatal("password spaces were changed")
	}
}

func TestBoundedPasswordEncoding(t *testing.T) {
	password := "synthetic credential for tests"
	first, err := PasswordHash(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PasswordHash(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !CheckPassword(first, password) || CheckPassword(first, password+"wrong") {
		t.Fatal("password salt/verification")
	}
	for _, invalid := range []string{"", strings.Replace(first, "m=19456", "m=999999999", 1), first + "$extra", "$argon2id$v=19$m=0,t=0,p=0$AA$AA"} {
		if CheckPassword(invalid, password) {
			t.Fatal("unsafe encoded parameters accepted")
		}
	}
	for _, invalid := range []string{"short", "passwordpassword", strings.Repeat("a", 1025)} {
		if _, err = PasswordHash(invalid); err == nil {
			t.Fatal("password bounds ignored")
		}
	}
}

func TestTotpWindowReturnsActualStep(t *testing.T) {
	secret, _, err := NewTotp("synthetic")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1800000000, 0)
	for _, offset := range []int64{-2, -1, 0, 1, 2} {
		code, err := totp.GenerateCode(secret, now.Add(time.Duration(offset)*30*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		step, err := TotpStep(secret, code, now)
		if offset >= -1 && offset <= 1 {
			if err != nil || step != now.Unix()/30+offset {
				t.Fatalf("window %d: %d %v", offset, step, err)
			}
		} else if err == nil {
			t.Fatal("out-of-window code accepted")
		}
	}
}

func TestPersistentKeysAndContextBinding(t *testing.T) {
	directory := t.TempDir()
	keys, err := CreateKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	sealed := keys.Seal("first-user", "synthetic-factor")
	reloaded, err := LoadKeys(directory)
	if err != nil {
		t.Fatal(err)
	}
	if keys.ID != reloaded.ID || !keys.Signing.Equal(reloaded.Signing) {
		t.Fatal("keys changed on reload")
	}
	if value, err := reloaded.Open("first-user", sealed); err != nil || value != "synthetic-factor" {
		t.Fatal("encrypted factor did not survive restart")
	}
	if _, err := reloaded.Open("second-user", sealed); err == nil {
		t.Fatal("factor moved between accounts")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := reloaded.Open("first-user", sealed); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err = CreateKeys(directory); err == nil {
		t.Fatal("existing keys overwritten")
	}
	if err = os.WriteFile(filepath.Join(directory, "keys.json"), []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadKeys(directory); err == nil {
		t.Fatal("malformed keys accepted")
	}
	if _, err = LoadKeys(t.TempDir()); err == nil {
		t.Fatal("missing keys silently regenerated")
	}
}
