package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Keys holds the installation signing and data-encryption keys without log methods.
type Keys struct {
	Signing    *rsa.PrivateKey
	Encryption [32]byte
	ID         string
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
}

// CreateKeys exclusively creates a private key file during explicit initialization.
// Serving never calls it or silently regenerates missing keys.
func CreateKeys(directory string) (*Keys, error) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	content, err := json.Marshal(diskKeys{Version: 1, Signing: encoded, Encryption: secret})
	if err != nil {
		return nil, err
	}
	path := filepath.Join(directory, "keys.json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, errors.New("key file already exists or cannot be created")
	}
	succeeded := false
	defer func() {
		file.Close()
		if !succeeded {
			_ = os.Remove(path)
		}
	}()
	if err = restrictFile(path); err != nil {
		return nil, err
	}
	if _, err = file.Write(content); err != nil {
		return nil, errors.New("cannot persist keys")
	}
	if err = file.Sync(); err != nil {
		return nil, errors.New("cannot sync keys")
	}
	succeeded = true
	return LoadKeys(directory)
}

// LoadKeys fails closed on missing, malformed, unsafe or changed key files.
func LoadKeys(directory string) (*Keys, error) {
	path := filepath.Join(directory, "keys.json")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return nil, errors.New("installation keys unavailable")
	}
	if err = checkPrivateFile(path, info); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("installation keys unreadable")
	}
	var stored diskKeys
	if json.Unmarshal(content, &stored) != nil || stored.Version != 1 || len(stored.Encryption) != 32 {
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
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(public)
	keys := &Keys{Signing: private, ID: hex.EncodeToString(digest[:12])}
	copy(keys.Encryption[:], stored.Encryption)
	return keys, nil
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
