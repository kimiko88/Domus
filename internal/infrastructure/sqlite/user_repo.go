// Package sqlite provides SQLite repository implementations for Domus domain entities.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
)

// UserRepository implements user.UserRepository for SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, role, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`
	var email *string
	if u.Email != "" {
		email = &u.Email
	}

	enabled := 0
	if u.Enabled {
		enabled = 1
	}

	createdAt := u.CreatedAt.Format(time.RFC3339)
	updatedAt := u.UpdatedAt.Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		u.ID,
		u.Username,
		email,
		u.PasswordHash,
		string(u.Role),
		enabled,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by their unique UUID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		FROM users
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanUser(row)
}

// GetByUsername retrieves a user by their unique username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, created_at, updated_at
		FROM users
		WHERE username = ?;
	`
	row := r.db.QueryRowContext(ctx, query, username)
	return r.scanUser(row)
}

// Count returns the total number of registered users.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users;`
	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("sqlite: failed to count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) scanUser(row *sql.Row) (*user.User, error) {
	var (
		u         user.User
		email     sql.NullString
		roleStr   string
		enabled   int
		createdAt string
		updatedAt string
	)

	err := row.Scan(
		&u.ID,
		&u.Username,
		&email,
		&u.PasswordHash,
		&roleStr,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}
		return nil, fmt.Errorf("sqlite: scan user: %w", err)
	}

	if email.Valid {
		u.Email = email.String
	}
	u.Role = user.Role(roleStr)
	u.Enabled = enabled == 1

	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		u.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		u.UpdatedAt = parsed
	}

	return &u, nil
}
