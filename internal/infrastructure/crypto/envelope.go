// Package crypto provides envelope encryption helpers for securing sensitive database fields at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EncryptedPrefix identifies fields that have been encrypted at rest.
	EncryptedPrefix = "ENC:v1:"
	// KeySize is the required byte length for AES-256 keys (32 bytes = 256 bits).
	KeySize = 32
	// GCMNonceSize is the standard 12-byte nonce length for AES-GCM.
	GCMNonceSize = 12
)

var (
	ErrEmptyMasterKey     = errors.New("envelope: master key cannot be empty")
	ErrCiphertextTooShort = errors.New("envelope: ciphertext payload is too short")
	ErrDecryptionFailed   = errors.New("envelope: authentication failed, ciphertext may be corrupt or key mismatch")
)

// EnvelopeCipher handles transparent encryption and decryption of sensitive strings using AES-256-GCM.
type EnvelopeCipher struct {
	aead cipher.AEAD
}

// NewEnvelopeCipher constructs an EnvelopeCipher from a 32-byte master key.
// If the key length is not 32 bytes, a SHA-256 digest of the key is derived to ensure 256-bit security.
func NewEnvelopeCipher(masterKey []byte) (*EnvelopeCipher, error) {
	if len(masterKey) == 0 {
		return nil, ErrEmptyMasterKey
	}

	key := masterKey
	if len(key) != KeySize {
		digest := sha256.Sum256(masterKey)
		key = digest[:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to initialize aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to initialize gcm: %w", err)
	}

	return &EnvelopeCipher{aead: aead}, nil
}

// Encrypt encrypts a plaintext string and returns a prefixed, URL-safe Base64 encoded ciphertext.
// Format: "ENC:v1:<base64(nonce_12b + ciphertext + auth_tag_16b)>"
func (c *EnvelopeCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// If already encrypted, return as is
	if c.IsEncrypted(plaintext) {
		return plaintext, nil
	}

	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("envelope: failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext and 16-byte authentication tag to nonce
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.RawURLEncoding.EncodeToString(sealed)

	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts a prefixed ciphertext string back to its original plaintext.
// If the string does not have the "ENC:v1:" prefix, it is treated as unencrypted legacy plaintext
// and returned as-is (graceful backward compatibility).
func (c *EnvelopeCipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if !c.IsEncrypted(ciphertext) {
		// Unencrypted legacy plaintext, return verbatim
		return ciphertext, nil
	}

	rawB64 := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	payload, err := base64.RawURLEncoding.DecodeString(rawB64)
	if err != nil {
		return "", fmt.Errorf("envelope: invalid base64 encoding: %w", err)
	}

	if len(payload) < GCMNonceSize+c.aead.Overhead() {
		return "", ErrCiphertextTooShort
	}

	nonce := payload[:GCMNonceSize]
	encryptedData := payload[GCMNonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted returns true if the string begins with the envelope encryption prefix.
func (c *EnvelopeCipher) IsEncrypted(val string) bool {
	return strings.HasPrefix(val, EncryptedPrefix)
}

// GenerateMasterKey generates a fresh, cryptographically secure 32-byte key.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("envelope: failed to generate random key: %w", err)
	}
	return key, nil
}

// LoadOrCreateMasterKey loads a 32-byte master key from the specified filesystem path,
// or generates and persists a new key with restricted permissions (0600) if the file does not exist.
func LoadOrCreateMasterKey(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("envelope: key path cannot be empty")
	}

	load := func() ([]byte, error) {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("envelope: master key path %s must be a regular file, not a symlink or special file", path)
		}
		if info.Mode().Perm() != 0o600 {
			if err := os.Chmod(path, 0o600); err != nil {
				return nil, fmt.Errorf("envelope: failed to secure master key permissions: %w", err)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		// Preserve compatibility with existing raw, hex, and Base64 keys.
		if len(data) == KeySize {
			return data, nil
		}
		trimmed := strings.TrimSpace(string(data))
		if len(trimmed) == KeySize*2 {
			decoded, err := hex.DecodeString(trimmed)
			if err == nil && len(decoded) == KeySize {
				return decoded, nil
			}
		}
		if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == KeySize {
			return decoded, nil
		}
		return nil, fmt.Errorf("envelope: master key at %s is malformed (expected %d raw bytes, %d hex chars, or Base64 for %d bytes)", path, KeySize, KeySize*2, KeySize)
	}

	key, err := load()
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("envelope: failed to read existing master key %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("envelope: failed to create key directory: %w", err)
		}
	}

	key, err = GenerateMasterKey()
	if err != nil {
		return nil, err
	}
	encoded := []byte(base64.StdEncoding.EncodeToString(key) + "\n")

	// O_EXCL prevents concurrent startups from replacing a key another process
	// has just created. If another process wins, load and use its key.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return load()
	}
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to persist master key to %s: %w", path, err)
	}
	writeErr := func() error {
		if _, err := file.Write(encoded); err != nil {
			return err
		}
		if err := file.Sync(); err != nil {
			return err
		}
		return file.Close()
	}()
	if writeErr != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("envelope: failed to persist master key to %s: %w", path, writeErr)
	}

	return key, nil
}
