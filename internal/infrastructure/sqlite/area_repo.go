// Package sqlite provides SQLite repository implementations for areas.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/area"
)

// AreaRepository implements area.Repository for SQLite.
type AreaRepository struct {
	db *sql.DB
}

// NewAreaRepository constructs an AreaRepository.
func NewAreaRepository(db *sql.DB) *AreaRepository {
	return &AreaRepository{db: db}
}

// Create inserts a new area record.
func (r *AreaRepository) Create(ctx context.Context, a *area.Area) error {
	query := `
		INSERT INTO areas (id, name, icon, parent_area_id, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	var icon, parent *string
	if a.Icon != "" {
		icon = &a.Icon
	}
	if a.ParentAreaID != "" {
		parent = &a.ParentAreaID
	}

	createdAt := a.CreatedAt.Format(time.RFC3339)
	updatedAt := a.UpdatedAt.Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		a.ID,
		a.Name,
		icon,
		parent,
		a.SortOrder,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert area: %w", err)
	}
	return nil
}

// GetByID retrieves an area by its ID.
func (r *AreaRepository) GetByID(ctx context.Context, id string) (*area.Area, error) {
	query := `
		SELECT id, name, icon, parent_area_id, sort_order, created_at, updated_at
		FROM areas
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		a         area.Area
		icon      sql.NullString
		parent    sql.NullString
		createdAt string
		updatedAt string
	)

	err := row.Scan(&a.ID, &a.Name, &icon, &parent, &a.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, area.ErrAreaNotFound
		}
		return nil, fmt.Errorf("sqlite: scan area: %w", err)
	}

	if icon.Valid {
		a.Icon = icon.String
	}
	if parent.Valid {
		a.ParentAreaID = parent.String
	}
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		a.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		a.UpdatedAt = parsed
	}

	return &a, nil
}

// List retrieves all areas ordered by sort_order and name.
func (r *AreaRepository) List(ctx context.Context) ([]*area.Area, error) {
	query := `
		SELECT id, name, icon, parent_area_id, sort_order, created_at, updated_at
		FROM areas
		ORDER BY sort_order ASC, name ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list areas: %w", err)
	}
	defer rows.Close()

	var areas []*area.Area
	for rows.Next() {
		var (
			a         area.Area
			icon      sql.NullString
			parent    sql.NullString
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&a.ID, &a.Name, &icon, &parent, &a.SortOrder, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("sqlite: scan area: %w", err)
		}
		if icon.Valid {
			a.Icon = icon.String
		}
		if parent.Valid {
			a.ParentAreaID = parent.String
		}
		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			a.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			a.UpdatedAt = parsed
		}
		areas = append(areas, &a)
	}
	return areas, rows.Err()
}

// Update modifies an existing area record.
func (r *AreaRepository) Update(ctx context.Context, a *area.Area) error {
	query := `
		UPDATE areas
		SET name = ?, icon = ?, parent_area_id = ?, sort_order = ?, updated_at = ?
		WHERE id = ?;
	`
	var icon, parent *string
	if a.Icon != "" {
		icon = &a.Icon
	}
	if a.ParentAreaID != "" {
		parent = &a.ParentAreaID
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query,
		a.Name,
		icon,
		parent,
		a.SortOrder,
		updatedAt,
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update area: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return area.ErrAreaNotFound
	}
	return nil
}

// Delete removes an area by its ID.
func (r *AreaRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM areas WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete area: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return area.ErrAreaNotFound
	}
	return nil
}
