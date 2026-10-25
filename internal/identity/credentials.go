// Package identity implements bounded credential handling using established primitives.
package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
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
var ErrPasswordPolicy = errors.New("password policy")
var hashSlots = make(chan struct{}, 4)

const passwordPrefix = "$argon2id$v=19$m=19456,t=2,p=1$"
const MinPasswordCharacters = 15
const MaxPasswordBytes = 1024

// PasswordPolicy describes public creation rules without changing login verification.
type PasswordPolicy struct {
	MinCharacters int `json:"min_characters"`
	MaxBytes      int `json:"max_bytes"`
}

// CurrentPasswordPolicy is shared by every password-creation entry point and the UI.
func CurrentPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinCharacters: MinPasswordCharacters, MaxBytes: MaxPasswordBytes}
}

// PasswordHash accepts long passwords and uses a fixed bounded Argon2id profile.
func PasswordHash(password string) (string, error) {
	if err := ValidateNewPassword(password); err != nil {
		return "", err
	}
	password = norm.NFC.String(password)
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	digest, err := derive(password, salt)
	if err != nil {
		return "", err
	}
	return passwordPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(digest), nil
}

// ValidateNewPassword counts normalized Unicode code points and bounds raw/normalized bytes.
func ValidateNewPassword(password string) error {
	if len(password) > MaxPasswordBytes || !utf8.ValidString(password) {
		return fmt.Errorf("%w: use valid UTF-8 within %d bytes", ErrPasswordPolicy, MaxPasswordBytes)
	}
	password = norm.NFC.String(password)
	if utf8.RuneCountInString(password) < MinPasswordCharacters || len(password) > MaxPasswordBytes {
		return fmt.Errorf("%w: at least %d characters after Unicode normalization and at most %d UTF-8 bytes", ErrPasswordPolicy, MinPasswordCharacters, MaxPasswordBytes)
	}
	blocked := []string{"passwordpassword", "123456789012345", "qwertyuiopasdfgh", "letmeinletmeinletmein"}
	for _, common := range blocked {
		if strings.EqualFold(password, common) {
			return fmt.Errorf("%w: choose a less common password", ErrPasswordPolicy)
		}
	}
	return nil
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
	if len(encoded) > 256 || len(password) > MaxPasswordBytes {
		return false
	}
	password = norm.NFC.String(password)
	if len(password) > MaxPasswordBytes || !utf8.ValidString(password) || !strings.HasPrefix(encoded, passwordPrefix) {
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
