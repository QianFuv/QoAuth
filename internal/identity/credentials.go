// Package identity implements bounded credential handling using established primitives.
package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

var ErrCredential = errors.New("invalid credential")
var hashSlots = make(chan struct{}, 4)

const passwordPrefix = "$argon2id$v=19$m=19456,t=2,p=1$"

// PasswordHash accepts long passwords and uses a fixed bounded Argon2id profile.
func PasswordHash(password string) (string, error) {
	if len(password) > 1024 {
		return "", errors.New("password exceeds 1024 UTF-8 bytes")
	}
	password = norm.NFC.String(password)
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 15 || len(password) > 1024 {
		return "", errors.New("password must contain at least 15 characters and at most 1024 UTF-8 bytes")
	}
	blocked := []string{"passwordpassword", "123456789012345", "qwertyuiopasdfgh", "letmeinletmeinletmein"}
	for _, common := range blocked {
		if strings.EqualFold(password, common) {
			return "", errors.New("choose a less common password")
		}
	}
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	digest, err := derive(password, salt)
	if err != nil {
		return "", err
	}
	return passwordPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(digest), nil
}

func derive(password string, salt []byte) ([]byte, error) {
	select {
	case hashSlots <- struct{}{}:
	default:
		return nil, errors.New("authentication capacity unavailable")
	}
	defer func() { <-hashSlots }()
	return argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32), nil
}

// CheckPassword rejects unrecognized or attacker-controlled hashing parameters.
func CheckPassword(encoded, password string) bool {
	if len(encoded) > 256 || len(password) > 1024 {
		return false
	}
	password = norm.NFC.String(password)
	if len(password) > 1024 || !utf8.ValidString(password) || !strings.HasPrefix(encoded, passwordPrefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, passwordPrefix), "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil || len(salt) != 16 {
		return false
	}
	wanted, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(wanted) != 32 {
		return false
	}
	actual, err := derive(password, salt)
	return err == nil && subtle.ConstantTimeCompare(wanted, actual) == 1
}

// NewTotp delegates secret generation and provisioning format to the OTP library.
func NewTotp(username string) (secret, provisioning string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "QoAuth", AccountName: username, SecretSize: 20})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// TotpStep returns the matched counter; storage must consume it atomically.
// Descending order handles a rare collision conservatively, including future skew.
func TotpStep(secret, code string, now time.Time) (int64, error) {
	if len(code) != 6 {
		return 0, ErrCredential
	}
	for _, value := range code {
		if value < '0' || value > '9' {
			return 0, ErrCredential
		}
	}
	current := now.Unix() / 30
	for _, step := range []int64{current + 1, current, current - 1} {
		if step < 0 {
			continue
		}
		valid, err := hotp.ValidateCustom(code, uint64(step), secret, hotp.ValidateOpts{Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err == nil && valid {
			return step, nil
		}
	}
	return 0, ErrCredential
}
