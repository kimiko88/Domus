package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/tag"
	"github.com/google/uuid"
)

// TagRepository implements tag.Repository for SQLite.
type TagRepository struct {
	db *sql.DB
}

// NewTagRepository creates a new SQLite-backed TagRepository.
func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

// SaveTag inserts or updates an NFC/QR tag.
func (r *TagRepository) SaveTag(ctx context.Context, t *tag.Tag) error {
	if t.ID == "" {
		t.ID = "tag_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	var lastScannedStr *string
	if t.LastScannedAt != nil {
		s := t.LastScannedAt.Format(time.RFC3339)
		lastScannedStr = &s
	}

	query := `
		INSERT INTO nfc_tags (id, name, tag_id, action_type, action_target_id, last_scanned_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tag_id) DO UPDATE SET
			name = excluded.name,
			action_type = excluded.action_type,
			action_target_id = excluded.action_target_id,
			last_scanned_at = excluded.last_scanned_at
	`
	_, err := r.db.ExecContext(
		ctx, query,
		t.ID,
		t.Name,
		t.TagID,
		string(t.ActionType),
		t.ActionTargetID,
		lastScannedStr,
		t.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save tag: %w", err)
	}
	return nil
}

// GetTagByTagID retrieves a tag by its NFC serial / UID / QR string.
func (r *TagRepository) GetTagByTagID(ctx context.Context, tagID string) (*tag.Tag, error) {
	query := `
		SELECT id, name, tag_id, action_type, action_target_id, last_scanned_at, created_at
		FROM nfc_tags
		WHERE tag_id = ?
	`
	var (
		t              tag.Tag
		actionType     string
		lastScannedStr sql.NullString
		createdStr     string
	)
	err := r.db.QueryRowContext(ctx, query, tagID).Scan(
		&t.ID,
		&t.Name,
		&t.TagID,
		&actionType,
		&t.ActionTargetID,
		&lastScannedStr,
		&createdStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tag not found: %s", tagID)
		}
		return nil, fmt.Errorf("sqlite: failed to get tag: %w", err)
	}

	t.ActionType = tag.ActionType(actionType)
	if t.CreatedAt, err = time.Parse(time.RFC3339, createdStr); err != nil {
		t.CreatedAt = time.Now().UTC()
	}
	if lastScannedStr.Valid {
		if parsed, err := time.Parse(time.RFC3339, lastScannedStr.String); err == nil {
			t.LastScannedAt = &parsed
		}
	}
	return &t, nil
}

// ListTags returns all saved NFC/QR tags.
func (r *TagRepository) ListTags(ctx context.Context) ([]*tag.Tag, error) {
	query := `
		SELECT id, name, tag_id, action_type, action_target_id, last_scanned_at, created_at
		FROM nfc_tags
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []*tag.Tag
	for rows.Next() {
		var (
			t              tag.Tag
			actionType     string
			lastScannedStr sql.NullString
			createdStr     string
		)
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.TagID,
			&actionType,
			&t.ActionTargetID,
			&lastScannedStr,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan tag error: %w", err)
		}
		t.ActionType = tag.ActionType(actionType)
		if parsed, err := time.Parse(time.RFC3339, createdStr); err == nil {
			t.CreatedAt = parsed
		}
		if lastScannedStr.Valid {
			if parsed, err := time.Parse(time.RFC3339, lastScannedStr.String); err == nil {
				t.LastScannedAt = &parsed
			}
		}
		tags = append(tags, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if tags == nil {
		tags = []*tag.Tag{}
	}
	return tags, nil
}

// DeleteTag removes a tag by ID.
func (r *TagRepository) DeleteTag(ctx context.Context, id string) error {
	query := `DELETE FROM nfc_tags WHERE id = ? OR tag_id = ?`
	_, err := r.db.ExecContext(ctx, query, id, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete tag: %w", err)
	}
	return nil
}
