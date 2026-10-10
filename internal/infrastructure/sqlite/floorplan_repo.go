package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/floorplan"
)

type floorplanRepo struct {
	db *sql.DB
}

// NewFloorplanRepository creates a new SQLite-backed floorplan repository.
func NewFloorplanRepository(db *sql.DB) floorplan.Repository {
	return &floorplanRepo{db: db}
}

func (r *floorplanRepo) CreateFloor(ctx context.Context, f *floorplan.Floor) error {
	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now

	query := `
		INSERT INTO floors (id, name, level, plan_svg, plan_image, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := r.db.ExecContext(ctx, query,
		f.ID, f.Name, f.Level, f.PlanSVG, f.PlanImage, f.SortOrder,
		f.CreatedAt.Format(time.RFC3339), f.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *floorplanRepo) GetFloor(ctx context.Context, id string) (*floorplan.Floor, error) {
	query := `
		SELECT id, name, level, COALESCE(plan_svg, ''), COALESCE(plan_image, ''), sort_order, created_at, updated_at
		FROM floors WHERE id = ?;
	`
	var f floorplan.Floor
	var createdStr, updatedStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.Name, &f.Level, &f.PlanSVG, &f.PlanImage, &f.SortOrder, &createdStr, &updatedStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, floorplan.ErrFloorNotFound
		}
		return nil, err
	}
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &f, nil
}

func (r *floorplanRepo) ListFloors(ctx context.Context) ([]*floorplan.Floor, error) {
	query := `
		SELECT id, name, level, COALESCE(plan_svg, ''), COALESCE(plan_image, ''), sort_order, created_at, updated_at
		FROM floors ORDER BY level ASC, sort_order ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var floors []*floorplan.Floor
	for rows.Next() {
		var f floorplan.Floor
		var createdStr, updatedStr string
		if err := rows.Scan(
			&f.ID, &f.Name, &f.Level, &f.PlanSVG, &f.PlanImage, &f.SortOrder, &createdStr, &updatedStr,
		); err != nil {
			return nil, err
		}
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		floors = append(floors, &f)
	}
	return floors, rows.Err()
}

func (r *floorplanRepo) UpdateFloor(ctx context.Context, f *floorplan.Floor) error {
	f.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE floors
		SET name = ?, level = ?, plan_svg = ?, plan_image = ?, sort_order = ?, updated_at = ?
		WHERE id = ?;
	`
	res, err := r.db.ExecContext(ctx, query,
		f.Name, f.Level, f.PlanSVG, f.PlanImage, f.SortOrder, f.UpdatedAt.Format(time.RFC3339), f.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return floorplan.ErrFloorNotFound
	}
	return nil
}

func (r *floorplanRepo) DeleteFloor(ctx context.Context, id string) error {
	query := `DELETE FROM floors WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return floorplan.ErrFloorNotFound
	}
	return nil
}

func (r *floorplanRepo) ListMarkers(ctx context.Context, floorID string) ([]*floorplan.Marker, error) {
	query := `
		SELECT id, floor_id, entity_key, x_percent, y_percent, COALESCE(icon_override, ''), show_badge, created_at, updated_at
		FROM floor_markers WHERE floor_id = ?;
	`
	rows, err := r.db.QueryContext(ctx, query, floorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markers []*floorplan.Marker
	for rows.Next() {
		var m floorplan.Marker
		var showBadgeInt int
		var createdStr, updatedStr string
		if err := rows.Scan(
			&m.ID, &m.FloorID, &m.EntityKey, &m.XPercent, &m.YPercent,
			&m.IconOverride, &showBadgeInt, &createdStr, &updatedStr,
		); err != nil {
			return nil, err
		}
		m.ShowBadge = (showBadgeInt == 1)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		markers = append(markers, &m)
	}
	return markers, rows.Err()
}

func (r *floorplanRepo) SaveMarker(ctx context.Context, m *floorplan.Marker) error {
	now := time.Now().UTC()
	m.UpdatedAt = now
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}

	showBadgeInt := 0
	if m.ShowBadge {
		showBadgeInt = 1
	}

	query := `
		INSERT INTO floor_markers (id, floor_id, entity_key, x_percent, y_percent, icon_override, show_badge, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(floor_id, entity_key) DO UPDATE SET
			x_percent = excluded.x_percent,
			y_percent = excluded.y_percent,
			icon_override = excluded.icon_override,
			show_badge = excluded.show_badge,
			updated_at = excluded.updated_at;
	`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.FloorID, m.EntityKey, m.XPercent, m.YPercent,
		m.IconOverride, showBadgeInt, m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: save marker failed: %w", err)
	}
	return nil
}

func (r *floorplanRepo) DeleteMarker(ctx context.Context, id string) error {
	query := `DELETE FROM floor_markers WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return floorplan.ErrMarkerNotFound
	}
	return nil
}
