// Package crypto_test tests Argon2id hashing and password verification (TDD red-to-green phase).
package crypto_test

import (
	"strings"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgon2_HashAndVerify(t *testing.T) {
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      64 * 1024,
		Iterations:  1,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	})

	password := "SecretP@ssw0rd!123"

	// 1. Generate hash
	encodedHash, err := hasher.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, encodedHash)

	// Format must be standard: $argon2id$v=19$m=65536,t=1,p=2$<salt>$<hash>
	assert.True(t, strings.HasPrefix(encodedHash, "$argon2id$v=19$m="))

	// 2. Correct password verifies successfully
	match, err := hasher.VerifyPassword(password, encodedHash)
	require.NoError(t, err)
	assert.True(t, match, "Correct password must match the generated hash")

	// 3. Incorrect password fails verification
	matchWrong, err := hasher.VerifyPassword("WrongPassword!", encodedHash)
	require.NoError(t, err)
	assert.False(t, matchWrong, "Wrong password must not match")

	// 4. Malformed hash string returns error or false
	matchInvalid, err := hasher.VerifyPassword(password, "invalid$hash$format")
	assert.Error(t, err)
	assert.False(t, matchInvalid)
}

func TestArgon2_UniqueSalts(t *testing.T) {
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      32 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	password := "SamePassword123"

	hash1, err := hasher.HashPassword(password)
	require.NoError(t, err)

	hash2, err := hasher.HashPassword(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "Hashing the same password twice must produce different hashes due to unique salts")
}
