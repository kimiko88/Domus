// Package auth defines models and interfaces for user authentication and security.
package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTwoFactorNotFound = errors.New("two_factor: not configured")
	ErrInvalidTOTPCode   = errors.New("two_factor: invalid totp code")
	ErrTwoFactorRequired = errors.New("two_factor: 2fa verification required")
	ErrIPAddressBanned   = errors.New("security: ip address temporarily banned")
)

// TwoFactorConfig stores TOTP secret, activation state, and backup recovery codes.
type TwoFactorConfig struct {
	UserID      string    `json:"user_id"`
	Secret      string    `json:"secret"` // Base32 encoded secret
	IsEnabled   bool      `json:"is_enabled"`
	BackupCodes []string  `json:"backup_codes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BannedIP tracks failed authentication attempts and temporary IP locks.
type BannedIP struct {
	IPAddress      string     `json:"ip_address"`
	FailedAttempts int        `json:"failed_attempts"`
	BannedUntil    *time.Time `json:"banned_until,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// IsBanned checks whether the ban expiration timestamp is in the future.
func (b *BannedIP) IsBanned() bool {
	if b.BannedUntil == nil {
		return false
	}
	return time.Now().UTC().Before(*b.BannedUntil)
}

// TwoFactorRepository persists 2FA user configurations.
type TwoFactorRepository interface {
	Get(ctx context.Context, userID string) (*TwoFactorConfig, error)
	Save(ctx context.Context, cfg *TwoFactorConfig) error
	// ConsumeBackupCode atomically removes a matching recovery code. It returns
	// false when the code is absent or has already been consumed.
	ConsumeBackupCode(ctx context.Context, userID, code string) (bool, error)
	Delete(ctx context.Context, userID string) error
}

// BannedIPRepository persists failed attempts and handles brute force rate limiting.
type BannedIPRepository interface {
	Get(ctx context.Context, ip string) (*BannedIP, error)
	RecordFailedAttempt(ctx context.Context, ip string, maxAttempts int, banDuration time.Duration) (bool, error)
	ResetAttempts(ctx context.Context, ip string) error
	ListBanned(ctx context.Context) ([]*BannedIP, error)
	Unban(ctx context.Context, ip string) error
}
