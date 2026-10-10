package crypto_test

import (
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
)

func BenchmarkArgon2Hasher_HashPassword(b *testing.B) {
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	password := "SecurePassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.HashPassword(password)
	}
}

func BenchmarkArgon2Hasher_VerifyPassword(b *testing.B) {
	hasher := crypto.NewArgon2Hasher(crypto.Argon2Params{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	password := "SecurePassword123!"
	hash, _ := hasher.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.VerifyPassword(password, hash)
	}
}
