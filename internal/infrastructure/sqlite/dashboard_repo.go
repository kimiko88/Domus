package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/google/uuid"
)

var _ dashboard.Repository = (*DashboardRepository)(nil)

// DashboardRepository implements dashboard.Repository for SQLite.
type DashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository constructs a new DashboardRepository.
func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// Create inserts a new dashboard.
func (r *DashboardRepository) Create(ctx context.Context, d *dashboard.Dashboard) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}
	if d.LayoutJSON == "" {
		d.LayoutJSON = "[]"
	}
	if d.KioskConfigJSON == "" {
		d.KioskConfigJSON = `{"enabled": false, "screensaver_timeout_sec": 120, "wake_on_presence": true}`
	}

	query := `
		INSERT INTO custom_dashboards (id, title, icon, is_default, sort_order, layout_json, kiosk_config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx, query,
		d.ID, d.Title, d.Icon, d.IsDefault, d.SortOrder, d.LayoutJSON, d.KioskConfigJSON,
		d.CreatedAt.Format(time.RFC3339),
		d.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert dashboard: %w", err)
	}
	return nil
}

// GetByID finds a dashboard by ID.
func (r *DashboardRepository) GetByID(ctx context.Context, id string) (*dashboard.Dashboard, error) {
	query := `
		SELECT id, title, icon, is_default, sort_order, layout_json, kiosk_config_json, created_at, updated_at
		FROM custom_dashboards
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var d dashboard.Dashboard
	var createdAtStr, updatedAtStr string
	err := row.Scan(&d.ID, &d.Title, &d.Icon, &d.IsDefault, &d.SortOrder, &d.LayoutJSON, &d.KioskConfigJSON, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dashboard.ErrDashboardNotFound
		}
		return nil, fmt.Errorf("sqlite: query dashboard failed: %w", err)
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &d, nil
}

// List returns all dashboards ordered by sort_order.
func (r *DashboardRepository) List(ctx context.Context) ([]*dashboard.Dashboard, error) {
	query := `
		SELECT id, title, icon, is_default, sort_order, layout_json, kiosk_config_json, created_at, updated_at
		FROM custom_dashboards
		ORDER BY sort_order ASC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list dashboards failed: %w", err)
	}
	defer rows.Close()

	var list []*dashboard.Dashboard
	for rows.Next() {
		var d dashboard.Dashboard
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&d.ID, &d.Title, &d.Icon, &d.IsDefault, &d.SortOrder, &d.LayoutJSON, &d.KioskConfigJSON, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("sqlite: scan dashboard failed: %w", err)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &d)
	}
	return list, rows.Err()
}

// Update updates an existing dashboard.
func (r *DashboardRepository) Update(ctx context.Context, d *dashboard.Dashboard) error {
	d.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE custom_dashboards
		SET title = ?, icon = ?, is_default = ?, sort_order = ?, layout_json = ?, kiosk_config_json = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(
		ctx, query,
		d.Title, d.Icon, d.IsDefault, d.SortOrder, d.LayoutJSON, d.KioskConfigJSON,
		d.UpdatedAt.Format(time.RFC3339),
		d.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: update dashboard failed: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return dashboard.ErrDashboardNotFound
	}
	return nil
}

// Delete removes a dashboard by ID.
func (r *DashboardRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM custom_dashboards WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete dashboard failed: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return dashboard.ErrDashboardNotFound
	}
	return nil
}

// SetDefault clears default on other dashboards and marks the specified one as default.
func (r *DashboardRepository) SetDefault(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE custom_dashboards SET is_default = FALSE`); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `UPDATE custom_dashboards SET is_default = TRUE WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return dashboard.ErrDashboardNotFound
	}

	return tx.Commit()
}
