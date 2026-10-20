// Package sqlite provides SQLite repository implementations for entities.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/entity"
)

// EntityRepository implements entity.Repository for SQLite.
type EntityRepository struct {
	db     *sql.DB
	readDB *sql.DB
}

// NewEntityRepository constructs an EntityRepository with optional separate read pool.
func NewEntityRepository(db *sql.DB, readDB ...*sql.DB) *EntityRepository {
	r := &EntityRepository{db: db, readDB: db}
	if len(readDB) > 0 && readDB[0] != nil {
		r.readDB = readDB[0]
	}
	return r
}

// Create inserts a new entity record.
func (r *EntityRepository) Create(ctx context.Context, e *entity.Entity) error {
	query := `
		INSERT INTO entities (
			id, device_id, entity_key, domain, name, icon, unit_of_measurement,
			writable, state, attributes, available, area_id, last_changed_at, last_updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	var deviceID, icon, uom, areaID *string
	if e.DeviceID != "" {
		deviceID = &e.DeviceID
	}
	if e.Icon != "" {
		icon = &e.Icon
	}
	if e.UnitOfMeasurement != "" {
		uom = &e.UnitOfMeasurement
	}
	if e.AreaID != "" {
		areaID = &e.AreaID
	}

	writable := 0
	if e.Writable {
		writable = 1
	}
	available := 0
	if e.Available {
		available = 1
	}

	state := e.State
	if state == "" {
		state = "null"
	}
	attrs := e.Attributes
	if attrs == "" {
		attrs = "{}"
	}

	lastChangedAt := e.LastChangedAt.Format(time.RFC3339)
	lastUpdatedAt := e.LastUpdatedAt.Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		e.ID,
		deviceID,
		e.EntityKey,
		e.Domain,
		e.Name,
		icon,
		uom,
		writable,
		state,
		attrs,
		available,
		areaID,
		lastChangedAt,
		lastUpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert entity: %w", err)
	}
	return nil
}

// GetByID retrieves an entity by its UUID.
func (r *EntityRepository) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	query := `
		SELECT id, device_id, entity_key, domain, name, icon, unit_of_measurement,
		       writable, state, attributes, available, area_id, last_changed_at, last_updated_at
		FROM entities
		WHERE id = ?;
	`
	row := r.readDB.QueryRowContext(ctx, query, id)
	return r.scanEntity(row)
}

// GetByKey retrieves an entity by its entity key (e.g. "light.living_room").
func (r *EntityRepository) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	query := `
		SELECT id, device_id, entity_key, domain, name, icon, unit_of_measurement,
		       writable, state, attributes, available, area_id, last_changed_at, last_updated_at
		FROM entities
		WHERE entity_key = ?;
	`
	row := r.readDB.QueryRowContext(ctx, query, key)
	return r.scanEntity(row)
}

// List retrieves entities based on optional filter parameters.
func (r *EntityRepository) List(ctx context.Context, filter entity.ListFilter) ([]*entity.Entity, error) {
	query := `
		SELECT id, device_id, entity_key, domain, name, icon, unit_of_measurement,
		       writable, state, attributes, available, area_id, last_changed_at, last_updated_at
		FROM entities
		WHERE 1=1
	`
	var args []any

	if filter.Domain != "" {
		query += " AND domain = ?"
		args = append(args, filter.Domain)
	}
	if filter.AreaID != "" {
		query += " AND area_id = ?"
		args = append(args, filter.AreaID)
	}
	if filter.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, filter.DeviceID)
	}
	if filter.Writable != nil {
		val := 0
		if *filter.Writable {
			val = 1
		}
		query += " AND writable = ?"
		args = append(args, val)
	}

	query += " ORDER BY domain ASC, name ASC;"

	rows, err := r.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list entities: %w", err)
	}
	defer rows.Close()

	var entities []*entity.Entity
	for rows.Next() {
		var (
			e           entity.Entity
			deviceID    sql.NullString
			icon        sql.NullString
			uom         sql.NullString
			areaID      sql.NullString
			writable    int
			available   int
			lastChanged string
			lastUpdated string
		)
		err := rows.Scan(
			&e.ID,
			&deviceID,
			&e.EntityKey,
			&e.Domain,
			&e.Name,
			&icon,
			&uom,
			&writable,
			&e.State,
			&e.Attributes,
			&available,
			&areaID,
			&lastChanged,
			&lastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("sqlite: scan entity: %w", err)
		}
		if deviceID.Valid {
			e.DeviceID = deviceID.String
		}
		if icon.Valid {
			e.Icon = icon.String
		}
		if uom.Valid {
			e.UnitOfMeasurement = uom.String
		}
		if areaID.Valid {
			e.AreaID = areaID.String
		}
		e.Writable = writable == 1
		e.Available = available == 1
		if parsed, err := time.Parse(time.RFC3339, lastChanged); err == nil {
			e.LastChangedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			e.LastUpdatedAt = parsed
		}
		entities = append(entities, &e)
	}
	return entities, rows.Err()
}

// Update modifies general metadata of an entity.
func (r *EntityRepository) Update(ctx context.Context, e *entity.Entity) error {
	query := `
		UPDATE entities
		SET name = ?, icon = ?, area_id = ?, last_updated_at = ?
		WHERE id = ?;
	`
	var icon, areaID *string
	if e.Icon != "" {
		icon = &e.Icon
	}
	if e.AreaID != "" {
		areaID = &e.AreaID
	}
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query, e.Name, icon, areaID, now, e.ID)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update entity: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

// UpdateState updates state, attributes, availability, and change timestamp for an entity.
func (r *EntityRepository) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	query := `
		UPDATE entities
		SET state = ?, attributes = ?, available = ?, last_changed_at = ?, last_updated_at = ?
		WHERE id = ?;
	`
	availVal := 0
	if available {
		availVal = 1
	}
	ts := at.Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query, stateJSON, attributesJSON, availVal, ts, ts, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update entity state: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

// Delete removes an entity by ID.
func (r *EntityRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM entities WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete entity: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return entity.ErrEntityNotFound
	}
	return nil
}

func (r *EntityRepository) scanEntity(row *sql.Row) (*entity.Entity, error) {
	var (
		e           entity.Entity
		deviceID    sql.NullString
		icon        sql.NullString
		uom         sql.NullString
		areaID      sql.NullString
		writable    int
		available   int
		lastChanged string
		lastUpdated string
	)
	err := row.Scan(
		&e.ID,
		&deviceID,
		&e.EntityKey,
		&e.Domain,
		&e.Name,
		&icon,
		&uom,
		&writable,
		&e.State,
		&e.Attributes,
		&available,
		&areaID,
		&lastChanged,
		&lastUpdated,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, entity.ErrEntityNotFound
		}
		return nil, fmt.Errorf("sqlite: scan entity: %w", err)
	}
	if deviceID.Valid {
		e.DeviceID = deviceID.String
	}
	if icon.Valid {
		e.Icon = icon.String
	}
	if uom.Valid {
		e.UnitOfMeasurement = uom.String
	}
	if areaID.Valid {
		e.AreaID = areaID.String
	}
	e.Writable = writable == 1
	e.Available = available == 1
	if parsed, err := time.Parse(time.RFC3339, lastChanged); err == nil {
		e.LastChangedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
		e.LastUpdatedAt = parsed
	}
	return &e, nil
}
