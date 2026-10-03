// Package sqlite provides session repository implementations for SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
)

// SessionRepository implements user.SessionRepository for SQLite.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository constructs a SessionRepository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create stores a new session record.
func (r *SessionRepository) Create(ctx context.Context, s *user.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, expires_at, created_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?);
	`
	expiresAt := s.ExpiresAt.Format(time.RFC3339)
	createdAt := s.CreatedAt.Format(time.RFC3339)

	var ip, ua *string
	if s.IPAddress != "" {
		ip = &s.IPAddress
	}
	if s.UserAgent != "" {
		ua = &s.UserAgent
	}

	_, err := r.db.ExecContext(ctx, query,
		s.ID,
		s.UserID,
		expiresAt,
		createdAt,
		ip,
		ua,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert session: %w", err)
	}
	return nil
}

// GetByID retrieves a session by its token ID.
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*user.Session, error) {
	query := `
		SELECT id, user_id, expires_at, created_at, ip_address, user_agent
		FROM sessions
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		s         user.Session
		expiresAt string
		createdAt string
		ip        sql.NullString
		ua        sql.NullString
	)

	err := row.Scan(
		&s.ID,
		&s.UserID,
		&expiresAt,
		&createdAt,
		&ip,
		&ua,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrSessionNotFound
		}
		return nil, fmt.Errorf("sqlite: scan session: %w", err)
	}

	if ip.Valid {
		s.IPAddress = ip.String
	}
	if ua.Valid {
		s.UserAgent = ua.String
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		s.ExpiresAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		s.CreatedAt = parsed
	}

	return &s, nil
}

// Delete removes an active session by ID.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = ?;`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete session: %w", err)
	}
	return nil
}

// DeleteExpired removes all sessions that expired before the specified timestamp.
func (r *SessionRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	query := `DELETE FROM sessions WHERE expires_at < ?;`
	_, err := r.db.ExecContext(ctx, query, before.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete expired sessions: %w", err)
	}
	return nil
}
