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

// InviteRepository implements user.InviteRepository for SQLite.
type InviteRepository struct {
	db *sql.DB
}

// NewInviteRepository constructs an InviteRepository.
func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{db: db}
}

// Create inserts a new family invitation record.
func (r *InviteRepository) Create(ctx context.Context, inv *user.FamilyInvite) error {
	query := `
		INSERT INTO family_invites (
			id, token, role, created_by_user_id, expires_at, used_at, used_by_user_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`
	var usedAtStr *string
	if inv.UsedAt != nil {
		s := inv.UsedAt.UTC().Format(time.RFC3339)
		usedAtStr = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		inv.ID,
		inv.Token,
		string(inv.Role),
		inv.CreatedByUserID,
		inv.ExpiresAt.UTC().Format(time.RFC3339),
		usedAtStr,
		inv.UsedByUserID,
		inv.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert family invite: %w", err)
	}
	return nil
}

// GetByToken retrieves an invitation by its unique secret token string.
func (r *InviteRepository) GetByToken(ctx context.Context, token string) (*user.FamilyInvite, error) {
	query := `
		SELECT id, token, role, created_by_user_id, expires_at, used_at, used_by_user_id, created_at
		FROM family_invites
		WHERE token = ?;
	`
	row := r.db.QueryRowContext(ctx, query, token)
	return r.scanInvite(row)
}

// GetByID retrieves an invitation by its UUID.
func (r *InviteRepository) GetByID(ctx context.Context, id string) (*user.FamilyInvite, error) {
	query := `
		SELECT id, token, role, created_by_user_id, expires_at, used_at, used_by_user_id, created_at
		FROM family_invites
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanInvite(row)
}

// ListActive returns all unconsumed, non-expired family invitations.
func (r *InviteRepository) ListActive(ctx context.Context) ([]*user.FamilyInvite, error) {
	query := `
		SELECT id, token, role, created_by_user_id, expires_at, used_at, used_by_user_id, created_at
		FROM family_invites
		WHERE used_at IS NULL AND expires_at > (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		ORDER BY created_at DESC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list active invites: %w", err)
	}
	defer rows.Close()

	var invites []*user.FamilyInvite
	for rows.Next() {
		inv, err := r.scanInviteRow(rows)
		if err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, nil
}

// MarkUsed flags an invitation as consumed by a specific newly registered user.
func (r *InviteRepository) MarkUsed(ctx context.Context, id string, usedByUserID string, usedAt time.Time) error {
	query := `
		UPDATE family_invites
		SET used_at = ?, used_by_user_id = ?
		WHERE id = ? AND used_at IS NULL;
	`
	res, err := r.db.ExecContext(ctx, query, usedAt.UTC().Format(time.RFC3339), usedByUserID, id)
	if err != nil {
		return fmt.Errorf("sqlite: mark invite used: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return user.ErrInviteUsed
	}
	return nil
}

// Delete permanently removes an invitation record.
func (r *InviteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM family_invites WHERE id = ?;`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete invite: %w", err)
	}
	return nil
}

func (r *InviteRepository) scanInvite(row *sql.Row) (*user.FamilyInvite, error) {
	var (
		inv          user.FamilyInvite
		roleStr      string
		createdByID  sql.NullString
		expiresAt    string
		usedAt       sql.NullString
		usedByUserID sql.NullString
		createdAt    string
	)

	err := row.Scan(
		&inv.ID,
		&inv.Token,
		&roleStr,
		&createdByID,
		&expiresAt,
		&usedAt,
		&usedByUserID,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrInviteNotFound
		}
		return nil, fmt.Errorf("sqlite: scan invite: %w", err)
	}

	inv.Role = user.Role(roleStr)
	if createdByID.Valid {
		inv.CreatedByUserID = createdByID.String
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		inv.ExpiresAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		inv.CreatedAt = parsed
	}
	if usedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, usedAt.String); err == nil {
			inv.UsedAt = &parsed
		}
	}
	if usedByUserID.Valid {
		inv.UsedByUserID = &usedByUserID.String
	}

	return &inv, nil
}

func (r *InviteRepository) scanInviteRow(rows *sql.Rows) (*user.FamilyInvite, error) {
	var (
		inv          user.FamilyInvite
		roleStr      string
		createdByID  sql.NullString
		expiresAt    string
		usedAt       sql.NullString
		usedByUserID sql.NullString
		createdAt    string
	)

	err := rows.Scan(
		&inv.ID,
		&inv.Token,
		&roleStr,
		&createdByID,
		&expiresAt,
		&usedAt,
		&usedByUserID,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: scan invite row: %w", err)
	}

	inv.Role = user.Role(roleStr)
	if createdByID.Valid {
		inv.CreatedByUserID = createdByID.String
	}
	if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		inv.ExpiresAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		inv.CreatedAt = parsed
	}
	if usedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339, usedAt.String); err == nil {
			inv.UsedAt = &parsed
		}
	}
	if usedByUserID.Valid {
		inv.UsedByUserID = &usedByUserID.String
	}

	return &inv, nil
}
