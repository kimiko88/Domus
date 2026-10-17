// Package auth provides authentication, session management, and RFC 6238 TOTP utilities.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	// TOTPPeriod defines the time step window in seconds (standard: 30s).
	TOTPPeriod = 30
	// TOTPDigits is the standard 6-digit passcode length.
	TOTPDigits = 6
)

// GenerateTOTPSecret creates a secure, random Base32-encoded 160-bit key.
func GenerateTOTPSecret() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}

// GenerateBackupCodes creates 8 random 8-character recovery alphanumeric codes.
func GenerateBackupCodes(count int) []string {
	if count <= 0 {
		count = 8
	}
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code := make([]byte, 8)
		randBytes := make([]byte, 8)
		_, _ = rand.Read(randBytes)
		for j := 0; j < 8; j++ {
			code[j] = chars[int(randBytes[j])%len(chars)]
		}
		// Format: XXXX-XXXX
		codes[i] = fmt.Sprintf("%s-%s", string(code[:4]), string(code[4:]))
	}
	return codes
}

// BuildQRCodeURL constructs the standard otpauth URI compatible with Google Authenticator, Authy, etc.
func BuildQRCodeURL(username, secret, issuer string) string {
	if issuer == "" {
		issuer = "Domus"
	}
	label := fmt.Sprintf("%s:%s", issuer, username)
	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		url.PathEscape(label),
		secret,
		url.QueryEscape(issuer),
		TOTPDigits,
		TOTPPeriod,
	)
}

// VerifyTOTP validates a 6-digit passcode against the Base32 secret using RFC 6238 HMAC-SHA1.
// Checks current step, previous step (-30s), and next step (+30s) to handle clock skew.
func VerifyTOTP(secret, passcode string, t time.Time) bool {
	passcode = strings.TrimSpace(passcode)
	if len(passcode) != TOTPDigits {
		return false
	}

	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		// Try with standard padding if needed
		key, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return false
		}
	}

	currentTimeStep := t.Unix() / TOTPPeriod

	// Check window [-1, 0, +1]
	for _, delta := range []int64{0, -1, 1} {
		step := currentTimeStep + delta
		expectedCode := generateHOTP(key, step)
		if expectedCode == passcode {
			return true
		}
	}
	return false
}

func generateHOTP(key []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	digest := mac.Sum(nil)

	// Dynamic truncation
	offset := digest[len(digest)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	otp := binaryCode % uint32(math.Pow10(TOTPDigits))

	return fmt.Sprintf("%0*d", TOTPDigits, otp)
}

// GenerateTOTPPasscode generates the 6-digit TOTP code for the given secret at time t.
func GenerateTOTPPasscode(secret string, t time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return ""
		}
	}
	step := t.Unix() / TOTPPeriod
	return generateHOTP(key, step)
}
