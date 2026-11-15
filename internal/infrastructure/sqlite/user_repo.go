// Package sqlite provides SQLite repository implementations for Domus domain entities.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
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
		INSERT INTO users (id, username, email, password_hash, role, enabled, is_guest, guest_expires_at, pin_code, kiosk_mode, created_at, updated_at, permissions_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	var email *string
	if u.Email != "" {
		email = &u.Email
	}

	enabled := 0
	if u.Enabled {
		enabled = 1
	}

	isGuest := 0
	if u.IsGuest {
		isGuest = 1
	}

	kioskMode := 0
	if u.KioskMode {
		kioskMode = 1
	}

	var guestExpiresAt *string
	if u.GuestExpiresAt != nil {
		exp := u.GuestExpiresAt.Format(time.RFC3339)
		guestExpiresAt = &exp
	}

	var pinCode *string
	if u.PinCode != "" {
		pinCode = &u.PinCode
	}

	createdAt := u.CreatedAt.Format(time.RFC3339)
	updatedAt := u.UpdatedAt.Format(time.RFC3339)
	permissions := u.Permissions
	if permissions == nil {
		permissions = user.DefaultPermissions(u.ID, u.Role)
	}
	permissions.UserID = u.ID
	permissions.Role = u.Role
	u.Permissions = permissions
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("sqlite: marshal user permissions: %w", err)
	}

	_, err := r.db.ExecContext(ctx, query,
		u.ID,
		u.Username,
		email,
		u.PasswordHash,
		string(u.Role),
		enabled,
		isGuest,
		guestExpiresAt,
		pinCode,
		kioskMode,
		createdAt,
		updatedAt,
		string(permissionsJSON),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by their unique UUID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, is_guest, guest_expires_at, pin_code, kiosk_mode, created_at, updated_at, permissions_json
		FROM users
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanUser(row)
}

// GetByUsername retrieves a user by their unique username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, is_guest, guest_expires_at, pin_code, kiosk_mode, created_at, updated_at, permissions_json
		FROM users
		WHERE username = ?;
	`
	row := r.db.QueryRowContext(ctx, query, username)
	return r.scanUser(row)
}

// GetByPinCode finds an active user or guest by their numeric PIN code.
func (r *UserRepository) GetByPinCode(ctx context.Context, pin string) (*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, is_guest, guest_expires_at, pin_code, kiosk_mode, created_at, updated_at, permissions_json
		FROM users
		WHERE pin_code = ? AND pin_code IS NOT NULL AND pin_code != '';
	`
	row := r.db.QueryRowContext(ctx, query, pin)
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

// List returns all registered users in chronological order.
func (r *UserRepository) List(ctx context.Context) ([]*user.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, enabled, is_guest, guest_expires_at, pin_code, kiosk_mode, created_at, updated_at, permissions_json
		FROM users
		ORDER BY created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list users: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		var (
			u              user.User
			email          sql.NullString
			roleStr        string
			enabled        int
			isGuest        int
			guestExpiresAt sql.NullString
			pinCode        sql.NullString
			kioskMode      int
			createdAt      string
			updatedAt      string
			permissionsJSON string
		)
		if err := rows.Scan(
			&u.ID, &u.Username, &email, &u.PasswordHash, &roleStr, &enabled,
			&isGuest, &guestExpiresAt, &pinCode, &kioskMode,
			&createdAt, &updatedAt, &permissionsJSON,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan user row: %w", err)
		}
		if email.Valid {
			u.Email = email.String
		}
		u.Role = user.Role(roleStr)
		u.Enabled = enabled == 1
		u.IsGuest = isGuest == 1
		if guestExpiresAt.Valid {
			if parsed, err := time.Parse(time.RFC3339, guestExpiresAt.String); err == nil {
				u.GuestExpiresAt = &parsed
			}
		}
		if pinCode.Valid {
			u.PinCode = pinCode.String
		}
		u.KioskMode = kioskMode == 1
		u.Permissions = decodePermissions(permissionsJSON, u.ID, u.Role)

		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			u.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			u.UpdatedAt = parsed
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

// UpdateRole updates the authorization role of a user.
func (r *UserRepository) UpdateRole(ctx context.Context, id string, role user.Role) error {
	permissionsJSON, err := json.Marshal(user.DefaultPermissions(id, role))
	if err != nil {
		return fmt.Errorf("sqlite: marshal default user permissions: %w", err)
	}
	query := `
		UPDATE users
		SET role = ?, permissions_json = ?, updated_at = ?
		WHERE id = ?;
	`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, query, string(role), string(permissionsJSON), now, id)
	if err != nil {
		return fmt.Errorf("sqlite: update user role: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

// UpdatePermissions stores the granular access policy for a user.
func (r *UserRepository) UpdatePermissions(ctx context.Context, id string, permissions *user.UserPermissions) error {
	if permissions == nil {
		return fmt.Errorf("sqlite: user permissions cannot be nil")
	}
	permissions.UserID = id
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("sqlite: marshal user permissions: %w", err)
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET permissions_json = ?, updated_at = ? WHERE id = ?`,
		string(permissionsJSON), time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update user permissions: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

func decodePermissions(raw, userID string, role user.Role) *user.UserPermissions {
	var permissions user.UserPermissions
	if raw == "" || raw == "{}" || json.Unmarshal([]byte(raw), &permissions) != nil || permissions.Role == "" {
		return user.DefaultPermissions(userID, role)
	}
	permissions.UserID = userID
	permissions.Role = role
	return &permissions
}

// Delete permanently removes a user record and cascades sessions.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete user: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) scanUser(row *sql.Row) (*user.User, error) {
	var (
		u              user.User
		email          sql.NullString
		roleStr        string
		enabled        int
		isGuest        int
		guestExpiresAt sql.NullString
		pinCode        sql.NullString
		kioskMode      int
		createdAt      string
		updatedAt      string
		permissionsJSON string
	)

	err := row.Scan(
		&u.ID,
		&u.Username,
		&email,
		&u.PasswordHash,
		&roleStr,
		&enabled,
		&isGuest,
		&guestExpiresAt,
		&pinCode,
		&kioskMode,
		&createdAt,
		&updatedAt,
		&permissionsJSON,
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
	u.IsGuest = isGuest == 1
	if guestExpiresAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, guestExpiresAt.String); err == nil {
			u.GuestExpiresAt = &parsed
		}
	}
	if pinCode.Valid {
		u.PinCode = pinCode.String
	}
	u.KioskMode = kioskMode == 1
	u.Permissions = decodePermissions(permissionsJSON, u.ID, u.Role)

	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		u.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		u.UpdatedAt = parsed
	}

	return &u, nil
}
