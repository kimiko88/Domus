// Package config — TDD RED: secrets encryption tests.
package config_test

import (
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveKey_DeterministicFromPassphrase(t *testing.T) {
	k1, err := config.DeriveKey("passphrase", []byte("saltsaltsaltsalt"))
	require.NoError(t, err)

	k2, err := config.DeriveKey("passphrase", []byte("saltsaltsaltsalt"))
	require.NoError(t, err)

	assert.Equal(t, k1, k2, "same passphrase+salt must produce same key")
}

func TestDeriveKey_DifferentSaltsProduceDifferentKeys(t *testing.T) {
	k1, err := config.DeriveKey("passphrase", []byte("saltsaltsaltsalt"))
	require.NoError(t, err)

	k2, err := config.DeriveKey("passphrase", []byte("othersaltothers!"))
	require.NoError(t, err)

	assert.NotEqual(t, k1, k2)
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := []byte("super secret integration config")

	ciphertext, err := config.Encrypt(plaintext, "my-passphrase")
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, string(plaintext), string(ciphertext))

	recovered, err := config.Decrypt(ciphertext, "my-passphrase")
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}

func TestEncryptDecrypt_WrongPassphrase(t *testing.T) {
	ciphertext, err := config.Encrypt([]byte("secret"), "correct-pass")
	require.NoError(t, err)

	_, err = config.Decrypt(ciphertext, "wrong-pass")
	assert.Error(t, err, "decryption with wrong passphrase must fail")
}

func TestEncryptDecrypt_TamperedCiphertext(t *testing.T) {
	ciphertext, err := config.Encrypt([]byte("secret"), "passphrase")
	require.NoError(t, err)

	// Flip a byte in the ciphertext portion (after the header)
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = config.Decrypt(tampered, "passphrase")
	assert.Error(t, err, "decryption of tampered data must fail")
}

func TestEncrypt_IsDifferentEachCall(t *testing.T) {
	// Each encrypt call uses a fresh random nonce → different ciphertext
	c1, err := config.Encrypt([]byte("hello"), "pass")
	require.NoError(t, err)

	c2, err := config.Encrypt([]byte("hello"), "pass")
	require.NoError(t, err)

	assert.NotEqual(t, c1, c2, "two encryptions of same plaintext should differ")
}

func TestDecrypt_TooShortInput(t *testing.T) {
	_, err := config.Decrypt([]byte("tooshort"), "pass")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "too short") || err != nil)
}
