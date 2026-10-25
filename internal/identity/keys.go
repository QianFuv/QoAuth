package identity

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Keys holds the installation signing and data-encryption keys without log methods.
type Keys struct {
	Signing    *rsa.PrivateKey
	Encryption [32]byte
	ID         string
	Previous   *VerificationKey
	Replaces   string
}

// IDTokenLifetime is the fixed maximum signed-token lifetime of this provider.
const IDTokenLifetime = 5 * time.Minute

// VerificationWindow includes the maximum lifetime and five minutes of RP clock skew.
const VerificationWindow = IDTokenLifetime + 5*time.Minute

// VerificationKey retains public material only during a bounded rotation overlap.
type VerificationKey struct {
	Public *rsa.PublicKey
	ID     string
	Until  int64
}

// Binding detects replacement of either operational key when opening a database.
func (keys *Keys) Binding() string {
	public, _ := x509.MarshalPKIXPublicKey(&keys.Signing.PublicKey)
	digest := sha256.Sum256(append(public, keys.Encryption[:]...))
	return hex.EncodeToString(digest[:])
}

type diskKeys struct {
	Version    int
	Signing    []byte
	Encryption []byte
	Previous   []byte `json:",omitempty"`
	Until      int64  `json:",omitempty"`
	Replaces   string `json:",omitempty"`
}

// CreateKeys exclusively creates a private key file during explicit initialization.
// Serving never calls it or silently regenerates missing keys.
func CreateKeys(directory string) (*Keys, error) {
	keys, err := freshSigningKey()
	if err != nil {
		return nil, err
	}
	_, _ = rand.Read(keys.Encryption[:])
	if err = WriteKeys(directory, keys); err != nil {
		return nil, err
	}
	return LoadKeys(directory)
}

func freshSigningKey() (*Keys, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &Keys{Signing: private, ID: publicID(&private.PublicKey)}, nil
}

func publicID(public *rsa.PublicKey) string {
	encoded, _ := x509.MarshalPKIXPublicKey(public)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:12])
}

// WriteKeys exclusively creates the key file in a new private installation.
func WriteKeys(directory string, keys *Keys) error {
	return writeKeys(filepath.Join(directory, "keys.json"), keys)
}

func writeKeys(path string, keys *Keys) error {
	content, err := encodeKeys(keys)
	if err != nil {
		return err
	}
	return WritePrivate(path, bytes.NewReader(content))
}

func encodeKeys(keys *Keys) ([]byte, error) {
	encoded, err := x509.MarshalPKCS8PrivateKey(keys.Signing)
	if err != nil {
		return nil, err
	}
	stored := diskKeys{Version: 2, Signing: encoded, Encryption: keys.Encryption[:], Replaces: keys.Replaces}
	if keys.Previous != nil {
		stored.Previous, err = x509.MarshalPKIXPublicKey(keys.Previous.Public)
		if err != nil {
			return nil, err
		}
		stored.Until = keys.Previous.Until
	}
	return json.Marshal(stored)
}

func writeCandidate(directory string, input io.Reader) error {
	temporary := filepath.Join(directory, "keys.staging.json")
	if _, err := os.Lstat(temporary); !os.IsNotExist(err) {
		file, err := OpenPrivate(temporary)
		if err != nil {
			return errors.New("unsafe interrupted key staging file; preserve it for inspection")
		}
		file.Close()
		if err = RemoveMarker(temporary); err != nil {
			return err
		}
	}
	if err := WritePrivate(temporary, input); err != nil {
		return err
	}
	return replaceFile(temporary, filepath.Join(directory, "keys.next.json"))
}

// LoadKeys fails closed on missing, malformed, unsafe or changed key files.
func LoadKeys(directory string) (*Keys, error) {
	return loadKeys(filepath.Join(directory, "keys.json"))
}

func loadKeys(path string) (*Keys, error) {
	file, err := OpenPrivate(path)
	if err != nil {
		return nil, errors.New("installation keys unavailable")
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 16385))
	if err != nil || len(content) > 16384 {
		return nil, errors.New("installation keys unreadable")
	}
	var stored diskKeys
	if json.Unmarshal(content, &stored) != nil || (stored.Version != 1 && stored.Version != 2) || len(stored.Encryption) != 32 {
		return nil, errors.New("installation keys invalid")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(stored.Signing)
	if err != nil {
		return nil, errors.New("signing key invalid")
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok || private.N.BitLen() != 2048 || private.Validate() != nil {
		return nil, errors.New("signing key invalid")
	}
	keys := &Keys{Signing: private, ID: publicID(&private.PublicKey), Replaces: stored.Replaces}
	copy(keys.Encryption[:], stored.Encryption)
	if len(stored.Previous) != 0 {
		parsed, err := x509.ParsePKIXPublicKey(stored.Previous)
		if err != nil {
			return nil, errors.New("previous verification key invalid")
		}
		public, ok := parsed.(*rsa.PublicKey)
		if !ok || public.N.BitLen() != 2048 || public.E != 65537 || stored.Until <= 0 || publicID(public) == keys.ID {
			return nil, errors.New("previous verification key invalid")
		}
		keys.Previous = &VerificationKey{Public: public, ID: publicID(public), Until: stored.Until}
	} else if stored.Until != 0 {
		return nil, errors.New("previous verification key missing")
	}
	return keys, nil
}

// HasPendingRotation forces startup to stop after an interrupted key publication.
func HasPendingRotation(directory string) bool {
	_, err := os.Lstat(filepath.Join(directory, "keys.next.json"))
	return !os.IsNotExist(err)
}

// StageRotation publishes only a complete synced candidate under the installation lock.
// Unpublished staging bytes have no database effect and are reclaimed on explicit retry.
func StageRotation(directory string, current *Keys, now time.Time) (*Keys, error) {
	path := filepath.Join(directory, "keys.next.json")
	if HasPendingRotation(directory) {
		next, err := loadKeys(path)
		if err != nil || next.Replaces != current.Binding() || next.Encryption != current.Encryption || next.Previous == nil || next.Previous.ID != current.ID {
			return nil, errors.New("pending signing rotation is invalid; preserve files and restore a verified backup")
		}
		return next, nil
	}
	if current.Previous != nil && current.Previous.Until > now.Unix() {
		return nil, errors.New("previous signing key is still in its verification window")
	}
	next, err := FreshRestoreKeys(current)
	if err != nil {
		return nil, err
	}
	next.Replaces = current.Binding()
	next.Previous = &VerificationKey{Public: &current.Signing.PublicKey, ID: current.ID, Until: now.Add(VerificationWindow).Unix()}
	content, err := encodeKeys(next)
	if err != nil {
		return nil, err
	}
	if err = writeCandidate(directory, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	return next, nil
}

// FreshRestoreKeys preserves AES data custody but discards all old signing trust.
func FreshRestoreKeys(previous *Keys) (*Keys, error) {
	keys, err := freshSigningKey()
	if err != nil {
		return nil, err
	}
	keys.Encryption = previous.Encryption
	return keys, nil
}

// PublishRotation atomically replaces the old file after its DB binding commits.
// A failed publication keeps startup closed; the same command can resume it.
func PublishRotation(directory string) error {
	return replaceFile(filepath.Join(directory, "keys.next.json"), filepath.Join(directory, "keys.json"))
}

func (keys *Keys) aead() cipher.AEAD {
	block, _ := aes.NewCipher(keys.Encryption[:])
	aead, _ := cipher.NewGCM(block)
	return aead
}

// Seal encrypts a factor with authenticated user-specific context.
func (keys *Keys) Seal(user, secret string) []byte {
	aead := keys.aead()
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)
	return aead.Seal(nonce, nonce, []byte(secret), []byte("totp:"+user))
}

// Open rejects ciphertext moved to another account or altered in storage.
func (keys *Keys) Open(user string, sealed []byte) (string, error) {
	aead := keys.aead()
	if len(sealed) < aead.NonceSize()+aead.Overhead() {
		return "", ErrCredential
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], []byte("totp:"+user))
	if err != nil {
		return "", ErrCredential
	}
	return string(plain), nil
}
