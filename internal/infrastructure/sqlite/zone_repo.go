// Package sqlite provides SQLite implementations of spatial zone and person repositories.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domzone "github.com/domus-hub/domus/internal/domain/zone"
	"github.com/google/uuid"
)

var _ domzone.Repository = (*ZoneRepository)(nil)

// ZoneRepository manages persistence for spatial zones and tracked persons in SQLite.
type ZoneRepository struct {
	db *sql.DB
}

// NewZoneRepository constructs a ZoneRepository.
func NewZoneRepository(db *sql.DB) *ZoneRepository {
	return &ZoneRepository{db: db}
}

// CreateZone inserts a new geographical zone.
func (r *ZoneRepository) CreateZone(ctx context.Context, z *domzone.Zone) error {
	if z.ID == "" {
		z.ID = uuid.NewString()
	}
	if z.CreatedAt.IsZero() {
		z.CreatedAt = time.Now().UTC()
	}

	passiveInt := 0
	if z.Passive {
		passiveInt = 1
	}

	query := `
		INSERT INTO spatial_zones (id, name, latitude, longitude, radius, icon, passive, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		z.ID, z.Name, z.Latitude, z.Longitude, z.Radius, z.Icon, passiveInt,
		z.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert zone: %w", err)
	}
	return nil
}

// GetZone retrieves a zone by ID.
func (r *ZoneRepository) GetZone(ctx context.Context, id string) (*domzone.Zone, error) {
	query := `
		SELECT id, name, latitude, longitude, radius, icon, passive, created_at
		FROM spatial_zones
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		z          domzone.Zone
		passiveInt int
		createdStr string
	)
	err := row.Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Radius, &z.Icon, &passiveInt, &createdStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domzone.ErrZoneNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan zone: %w", err)
	}
	z.Passive = passiveInt == 1
	z.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &z, nil
}

// ListZones returns all geographical zones.
func (r *ZoneRepository) ListZones(ctx context.Context) ([]*domzone.Zone, error) {
	query := `
		SELECT id, name, latitude, longitude, radius, icon, passive, created_at
		FROM spatial_zones
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query zones: %w", err)
	}
	defer rows.Close()

	var zones []*domzone.Zone
	for rows.Next() {
		var (
			z          domzone.Zone
			passiveInt int
			createdStr string
		)
		if err := rows.Scan(&z.ID, &z.Name, &z.Latitude, &z.Longitude, &z.Radius, &z.Icon, &passiveInt, &createdStr); err != nil {
			return nil, err
		}
		z.Passive = passiveInt == 1
		z.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		zones = append(zones, &z)
	}
	return zones, rows.Err()
}

// UpdateZone updates an existing zone.
func (r *ZoneRepository) UpdateZone(ctx context.Context, z *domzone.Zone) error {
	passiveInt := 0
	if z.Passive {
		passiveInt = 1
	}

	query := `
		UPDATE spatial_zones
		SET name = ?, latitude = ?, longitude = ?, radius = ?, icon = ?, passive = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		z.Name, z.Latitude, z.Longitude, z.Radius, z.Icon, passiveInt, z.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update zone: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domzone.ErrZoneNotFound
	}
	return nil
}

// DeleteZone deletes a zone.
func (r *ZoneRepository) DeleteZone(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM spatial_zones WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete zone: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domzone.ErrZoneNotFound
	}
	return nil
}

// CreatePerson inserts a new tracked person profile.
func (r *ZoneRepository) CreatePerson(ctx context.Context, p *domzone.Person) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	p.UpdatedAt = now

	trackersJSON, err := json.Marshal(p.DeviceTrackers)
	if err != nil {
		trackersJSON = []byte("[]")
	}

	query := `
		INSERT INTO tracked_persons (id, name, user_id, state, latitude, longitude, device_trackers_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		p.ID, p.Name, p.UserID, p.State, p.Latitude, p.Longitude, string(trackersJSON),
		now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert person: %w", err)
	}
	return nil
}

// GetPerson retrieves a person by ID.
func (r *ZoneRepository) GetPerson(ctx context.Context, id string) (*domzone.Person, error) {
	query := `
		SELECT id, name, user_id, state, latitude, longitude, device_trackers_json, updated_at
		FROM tracked_persons
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		p            domzone.Person
		userNull     sql.NullString
		latNull      sql.NullFloat64
		lonNull      sql.NullFloat64
		trackersJSON string
		updatedStr   string
	)
	err := row.Scan(&p.ID, &p.Name, &userNull, &p.State, &latNull, &lonNull, &trackersJSON, &updatedStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domzone.ErrPersonNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan person: %w", err)
	}

	p.UserID = userNull.String
	if latNull.Valid {
		v := latNull.Float64
		p.Latitude = &v
	}
	if lonNull.Valid {
		v := lonNull.Float64
		p.Longitude = &v
	}
	_ = json.Unmarshal([]byte(trackersJSON), &p.DeviceTrackers)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &p, nil
}

// ListPersons returns all tracked person profiles.
func (r *ZoneRepository) ListPersons(ctx context.Context) ([]*domzone.Person, error) {
	query := `
		SELECT id, name, user_id, state, latitude, longitude, device_trackers_json, updated_at
		FROM tracked_persons
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query persons: %w", err)
	}
	defer rows.Close()

	var persons []*domzone.Person
	for rows.Next() {
		var (
			p            domzone.Person
			userNull     sql.NullString
			latNull      sql.NullFloat64
			lonNull      sql.NullFloat64
			trackersJSON string
			updatedStr   string
		)
		if err := rows.Scan(&p.ID, &p.Name, &userNull, &p.State, &latNull, &lonNull, &trackersJSON, &updatedStr); err != nil {
			return nil, err
		}
		p.UserID = userNull.String
		if latNull.Valid {
			v := latNull.Float64
			p.Latitude = &v
		}
		if lonNull.Valid {
			v := lonNull.Float64
			p.Longitude = &v
		}
		_ = json.Unmarshal([]byte(trackersJSON), &p.DeviceTrackers)
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		persons = append(persons, &p)
	}
	return persons, rows.Err()
}

// UpdatePerson updates a person's profile and location.
func (r *ZoneRepository) UpdatePerson(ctx context.Context, p *domzone.Person) error {
	p.UpdatedAt = time.Now().UTC()
	trackersJSON, err := json.Marshal(p.DeviceTrackers)
	if err != nil {
		trackersJSON = []byte("[]")
	}

	query := `
		UPDATE tracked_persons
		SET name = ?, user_id = ?, state = ?, latitude = ?, longitude = ?,
		    device_trackers_json = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		p.Name, p.UserID, p.State, p.Latitude, p.Longitude,
		string(trackersJSON), p.UpdatedAt.Format(time.RFC3339),
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update person: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domzone.ErrPersonNotFound
	}
	return nil
}

// DeletePerson deletes a person profile.
func (r *ZoneRepository) DeletePerson(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM tracked_persons WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete person: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domzone.ErrPersonNotFound
	}
	return nil
}
