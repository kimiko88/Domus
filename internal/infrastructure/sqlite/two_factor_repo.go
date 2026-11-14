// Package sqlite provides SQLite persistence for two-factor authentication and IP brute force protection.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domauth "github.com/domus-hub/domus/internal/domain/auth"
)

var (
	_ domauth.TwoFactorRepository = (*TwoFactorRepository)(nil)
	_ domauth.BannedIPRepository  = (*BannedIPRepository)(nil)
)

// TwoFactorRepository persists TOTP user configurations in SQLite.
type TwoFactorRepository struct {
	db *sql.DB
}

// NewTwoFactorRepository constructs a TwoFactorRepository.
func NewTwoFactorRepository(db *sql.DB) *TwoFactorRepository {
	return &TwoFactorRepository{db: db}
}

// Get retrieves the 2FA configuration for a user.
func (r *TwoFactorRepository) Get(ctx context.Context, userID string) (*domauth.TwoFactorConfig, error) {
	query := `
		SELECT user_id, secret, is_enabled, backup_codes, created_at, updated_at
		FROM user_two_factor
		WHERE user_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, userID)

	var (
		cfg        domauth.TwoFactorConfig
		isEnabled  int
		codesJSON  string
		createdStr string
		updatedStr string
	)
	err := row.Scan(&cfg.UserID, &cfg.Secret, &isEnabled, &codesJSON, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domauth.ErrTwoFactorNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to query 2fa: %w", err)
	}

	cfg.IsEnabled = isEnabled == 1
	if err := json.Unmarshal([]byte(codesJSON), &cfg.BackupCodes); err != nil {
		return nil, fmt.Errorf("sqlite: failed to decode 2fa backup codes: %w", err)
	}
	cfg.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	cfg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &cfg, nil
}

// Save inserts or updates a user's 2FA configuration.
func (r *TwoFactorRepository) Save(ctx context.Context, cfg *domauth.TwoFactorConfig) error {
	codesJSON, err := json.Marshal(cfg.BackupCodes)
	if err != nil {
		return fmt.Errorf("sqlite: failed to encode 2fa backup codes: %w", err)
	}

	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now

	enabledInt := 0
	if cfg.IsEnabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO user_two_factor (user_id, secret, is_enabled, backup_codes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			secret = excluded.secret,
			is_enabled = excluded.is_enabled,
			backup_codes = excluded.backup_codes,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(ctx, query,
		cfg.UserID, cfg.Secret, enabledInt, string(codesJSON),
		cfg.CreatedAt.Format(time.RFC3339), cfg.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save 2fa config: %w", err)
	}
	return nil
}

// ConsumeBackupCode atomically removes one recovery code. The conditional
// UPDATE prevents two concurrent logins from consuming the same code.
func (r *TwoFactorRepository) ConsumeBackupCode(ctx context.Context, userID, code string) (bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" { return false, nil }
	for attempt := 0; attempt < 8; attempt++ {
		var raw string
		if err := r.db.QueryRowContext(ctx, "SELECT backup_codes FROM user_two_factor WHERE user_id = ?", userID).Scan(&raw); err != nil {
			if errors.Is(err, sql.ErrNoRows) { return false, nil }
			return false, fmt.Errorf("sqlite: failed to load backup codes: %w", err)
		}
		var codes []string
		if err := json.Unmarshal([]byte(raw), &codes); err != nil {
			return false, fmt.Errorf("sqlite: failed to decode backup codes: %w", err)
		}
		found := -1
		for i, candidate := range codes { if strings.ToUpper(strings.TrimSpace(candidate)) == code { found = i; break } }
		if found < 0 { return false, nil }
		codes = append(codes[:found], codes[found+1:]...)
		updated, err := json.Marshal(codes)
		if err != nil { return false, fmt.Errorf("sqlite: failed to encode backup codes: %w", err) }
		res, err := r.db.ExecContext(ctx,
			"UPDATE user_two_factor SET backup_codes = ?, updated_at = ? WHERE user_id = ? AND backup_codes = ?",
			string(updated), time.Now().UTC().Format(time.RFC3339), userID, raw)
		if err != nil { return false, fmt.Errorf("sqlite: failed to consume backup code: %w", err) }
		n, err := res.RowsAffected()
		if err != nil { return false, fmt.Errorf("sqlite: failed to verify backup-code consumption: %w", err) }
		if n == 1 { return true, nil }
	}
	return false, errors.New("sqlite: backup codes changed too frequently; retry authentication")
}

// Delete removes 2FA configuration for a user.
func (r *TwoFactorRepository) Delete(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM user_two_factor WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete 2fa config: %w", err)
	}
	return nil
}

// BannedIPRepository handles failed login counts and IP blocking in SQLite.
type BannedIPRepository struct {
	db *sql.DB
}

// NewBannedIPRepository constructs a BannedIPRepository.
func NewBannedIPRepository(db *sql.DB) *BannedIPRepository {
	return &BannedIPRepository{db: db}
}

// Get queries IP ban record.
func (r *BannedIPRepository) Get(ctx context.Context, ip string) (*domauth.BannedIP, error) {
	query := `
		SELECT ip_address, failed_attempts, banned_until, reason, created_at, updated_at
		FROM banned_ips
		WHERE ip_address = ?
	`
	row := r.db.QueryRowContext(ctx, query, ip)

	var (
		b          domauth.BannedIP
		untilNull  sql.NullString
		reasonNull sql.NullString
		createdStr string
		updatedStr string
	)
	err := row.Scan(&b.IPAddress, &b.FailedAttempts, &untilNull, &reasonNull, &createdStr, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not banned, clean
		}
		return nil, fmt.Errorf("sqlite: failed to query banned ip: %w", err)
	}

	if untilNull.Valid && untilNull.String != "" {
		t, err := time.Parse(time.RFC3339, untilNull.String)
		if err != nil {
			return nil, fmt.Errorf("sqlite: invalid banned-until timestamp for %s: %w", ip, err)
		}
		b.BannedUntil = &t
	}
	b.Reason = reasonNull.String
	b.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &b, nil
}

// RecordFailedAttempt increments attempt count and applies temporary ban if threshold reached.
func (r *BannedIPRepository) RecordFailedAttempt(ctx context.Context, ip string, maxAttempts int, banDuration time.Duration) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if banDuration <= 0 {
		banDuration = 15 * time.Minute
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	banUntilStr := sql.NullString{
		String: now.Add(banDuration).Format(time.RFC3339),
		Valid:  true,
	}
	initialBanUntil := sql.NullString{}
	if maxAttempts <= 1 {
		initialBanUntil = banUntilStr
	}

	// Keep the increment and threshold decision in one SQLite UPSERT. The
	// previous read/modify/write sequence could lose concurrent failures and
	// let parallel password guesses evade the IP ban threshold.
	query := `
		INSERT INTO banned_ips (ip_address, failed_attempts, banned_until, reason, created_at, updated_at)
		VALUES (?, 1, ?, ?, ?, ?)
		ON CONFLICT(ip_address) DO UPDATE SET
			failed_attempts = banned_ips.failed_attempts + 1,
			banned_until = CASE
				WHEN banned_ips.failed_attempts + 1 >= ? THEN ?
				ELSE banned_ips.banned_until
			END,
			reason = CASE
				WHEN banned_ips.failed_attempts + 1 >= ? THEN excluded.reason
				ELSE banned_ips.reason
			END,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		ip, initialBanUntil, "Brute-force protection trigger", nowStr, nowStr,
		maxAttempts, banUntilStr, maxAttempts,
	)
	if err != nil {
		return false, fmt.Errorf("sqlite: failed to record failed attempt: %w", err)
	}

	current, err := r.Get(ctx, ip)
	if err != nil {
		return false, err
	}
	return current != nil && current.IsBanned(), nil
}

// ResetAttempts clears failure count after a successful login.
func (r *BannedIPRepository) ResetAttempts(ctx context.Context, ip string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM banned_ips WHERE ip_address = ?", ip)
	return err
}

// ListBanned returns all currently locked IP addresses.
func (r *BannedIPRepository) ListBanned(ctx context.Context) ([]*domauth.BannedIP, error) {
	query := `
		SELECT ip_address, failed_attempts, banned_until, reason, created_at, updated_at
		FROM banned_ips
		WHERE banned_until IS NOT NULL
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query banned ips: %w", err)
	}
	defer rows.Close()

	var list []*domauth.BannedIP
	now := time.Now().UTC()
	for rows.Next() {
		var (
			b          domauth.BannedIP
			untilNull  sql.NullString
			reasonNull sql.NullString
			createdStr string
			updatedStr string
		)
		if err := rows.Scan(&b.IPAddress, &b.FailedAttempts, &untilNull, &reasonNull, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		if untilNull.Valid && untilNull.String != "" {
			if t, err := time.Parse(time.RFC3339, untilNull.String); err == nil {
				b.BannedUntil = &t
			}
		}
		b.Reason = reasonNull.String
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

		// Only include still actively banned
		if b.BannedUntil != nil && now.Before(*b.BannedUntil) {
			list = append(list, &b)
		}
	}
	return list, rows.Err()
}

// Unban manually removes a ban.
func (r *BannedIPRepository) Unban(ctx context.Context, ip string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM banned_ips WHERE ip_address = ?", ip)
	return err
}
