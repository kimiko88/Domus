// Package config provides AES-256-GCM encryption/decryption for sensitive secrets.
//
// Wire format (binary):
//
//	[ salt (16 bytes) | nonce (12 bytes) | ciphertext+tag ]
//
// Key derivation: PBKDF2-SHA256 with 100_000 iterations → 32-byte key.
package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltLen  = 16
	nonceLen = 12
	keyLen   = 32 // AES-256
	pbkdf2It = 100_000
)

// DeriveKey derives a 32-byte AES key from passphrase + salt using PBKDF2-SHA256.
// The salt must be exactly saltLen bytes. Use this function when you need to
// verify or derive a key independently (e.g., tests).
func DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) != saltLen {
		return nil, fmt.Errorf("secrets: salt must be %d bytes, got %d", saltLen, len(salt))
	}
	key := pbkdf2.Key([]byte(passphrase), salt, pbkdf2It, keyLen, sha256.New)
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a freshly generated random
// salt and nonce. The passphrase is used to derive the encryption key via PBKDF2.
//
// Returns the binary wire-format blob (salt + nonce + ciphertext+tag).
func Encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("secrets: failed to generate salt: %w", err)
	}

	key, err := DeriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("secrets: failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Assemble: salt | nonce | ciphertext+tag
	result := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypt decrypts a blob produced by Encrypt using the provided passphrase.
// Returns an error if the blob is malformed, the passphrase is wrong, or the
// authentication tag does not match (i.e., data was tampered).
func Decrypt(blob []byte, passphrase string) ([]byte, error) {
	minLen := saltLen + nonceLen + 1 // at least 1 byte of ciphertext
	if len(blob) < minLen {
		return nil, fmt.Errorf("secrets: ciphertext too short (%d bytes)", len(blob))
	}

	salt := blob[:saltLen]
	nonce := blob[saltLen : saltLen+nonceLen]
	ciphertext := blob[saltLen+nonceLen:]

	key, err := DeriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("secrets: decryption failed (wrong passphrase or tampered data): %w", err)
	}

	return plaintext, nil
}
