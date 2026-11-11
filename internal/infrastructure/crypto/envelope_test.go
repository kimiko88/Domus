package crypto_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeCipher_EncryptAndDecrypt(t *testing.T) {
	key, err := crypto.GenerateMasterKey()
	require.NoError(t, err)
	require.Len(t, key, 32)

	cipher, err := crypto.NewEnvelopeCipher(key)
	require.NoError(t, err)

	original := "sk_live_super_secret_webhook_token_98765"
	encrypted, err := cipher.Encrypt(original)
	require.NoError(t, err)
	assert.True(t, cipher.IsEncrypted(encrypted))
	assert.NotEqual(t, original, encrypted)

	// Decrypt
	decrypted, err := cipher.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)

	// Re-encrypting already encrypted string should be idempotent
	reEncrypted, err := cipher.Encrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, encrypted, reEncrypted)
}

func TestEnvelopeCipher_LegacyPlaintextFallback(t *testing.T) {
	key, err := crypto.GenerateMasterKey()
	require.NoError(t, err)

	cipher, err := crypto.NewEnvelopeCipher(key)
	require.NoError(t, err)

	legacyPlaintext := "unencrypted_legacy_api_key_12345"
	assert.False(t, cipher.IsEncrypted(legacyPlaintext))

	// Decrypting legacy plaintext should gracefully return it as-is
	res, err := cipher.Decrypt(legacyPlaintext)
	require.NoError(t, err)
	assert.Equal(t, legacyPlaintext, res)
}

func TestEnvelopeCipher_TamperedCiphertextFails(t *testing.T) {
	key1, _ := crypto.GenerateMasterKey()
	key2, _ := crypto.GenerateMasterKey()

	cipher1, err := crypto.NewEnvelopeCipher(key1)
	require.NoError(t, err)
	cipher2, err := crypto.NewEnvelopeCipher(key2)
	require.NoError(t, err)

	encrypted, err := cipher1.Encrypt("confidential_data")
	require.NoError(t, err)

	// Decrypting with different key must fail
	_, err = cipher2.Decrypt(encrypted)
	assert.ErrorIs(t, err, crypto.ErrDecryptionFailed)

	// Tampered ciphertext must fail
	tampered := encrypted[:len(encrypted)-3] + "xyz"
	_, err = cipher1.Decrypt(tampered)
	assert.Error(t, err)
}

func TestEnvelopeCipher_LoadOrCreateMasterKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "master.key")

	// 1. Create fresh key
	key1, err := crypto.LoadOrCreateMasterKey(keyPath)
	require.NoError(t, err)
	require.Len(t, key1, 32)

	// 2. Load existing key from same file
	key2, err := crypto.LoadOrCreateMasterKey(keyPath)
	require.NoError(t, err)
	assert.Equal(t, key1, key2, "Loading from existing path must yield identical key")

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Existing keys with broad permissions are tightened without changing bytes.
	require.NoError(t, os.Chmod(keyPath, 0644))
	key3, err := crypto.LoadOrCreateMasterKey(keyPath)
	require.NoError(t, err)
	assert.Equal(t, key1, key3)
	info, err = os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}


func TestLoadOrCreateMasterKey_RejectsMalformedExistingKey(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "master.key")
	require.NoError(t, os.WriteFile(keyPath, []byte("truncated-key"), 0600))

	_, err := crypto.LoadOrCreateMasterKey(keyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")
	data, readErr := os.ReadFile(keyPath)
	require.NoError(t, readErr)
	assert.Equal(t, "truncated-key", string(data), "a malformed key must never be silently replaced")
}
