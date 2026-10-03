// Package sqlite provides SQLite repository implementations for devices.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
)

// DeviceRepository implements device.Repository for SQLite.
type DeviceRepository struct {
	db *sql.DB
}

// NewDeviceRepository constructs a DeviceRepository.
func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create inserts a new device record.
func (r *DeviceRepository) Create(ctx context.Context, d *device.Device) error {
	query := `
		INSERT INTO devices (id, integration_id, external_id, name, manufacturer, model, area_id, online, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	var manufacturer, model, areaID *string
	if d.Manufacturer != "" {
		manufacturer = &d.Manufacturer
	}
	if d.Model != "" {
		model = &d.Model
	}
	if d.AreaID != "" {
		areaID = &d.AreaID
	}
	online := 0
	if d.Online {
		online = 1
	}
	metadata := d.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	createdAt := d.CreatedAt.Format(time.RFC3339)
	updatedAt := d.UpdatedAt.Format(time.RFC3339)

	_, err := r.db.ExecContext(ctx, query,
		d.ID,
		d.IntegrationID,
		d.ExternalID,
		d.Name,
		manufacturer,
		model,
		areaID,
		online,
		metadata,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert device: %w", err)
	}
	return nil
}

// GetByID retrieves a device by its UUID.
func (r *DeviceRepository) GetByID(ctx context.Context, id string) (*device.Device, error) {
	query := `
		SELECT id, integration_id, external_id, name, manufacturer, model, area_id, online, metadata, created_at, updated_at
		FROM devices
		WHERE id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanDevice(row)
}

// GetByExternalID retrieves a device by integration ID and external ID.
func (r *DeviceRepository) GetByExternalID(ctx context.Context, integrationID, externalID string) (*device.Device, error) {
	query := `
		SELECT id, integration_id, external_id, name, manufacturer, model, area_id, online, metadata, created_at, updated_at
		FROM devices
		WHERE integration_id = ? AND external_id = ?;
	`
	row := r.db.QueryRowContext(ctx, query, integrationID, externalID)
	return r.scanDevice(row)
}

// List retrieves devices matching the optional filter criteria.
func (r *DeviceRepository) List(ctx context.Context, filter device.ListFilter) ([]*device.Device, error) {
	query := `
		SELECT id, integration_id, external_id, name, manufacturer, model, area_id, online, metadata, created_at, updated_at
		FROM devices
		WHERE 1=1
	`
	var args []any

	if filter.IntegrationID != "" {
		query += " AND integration_id = ?"
		args = append(args, filter.IntegrationID)
	}
	if filter.AreaID != "" {
		query += " AND area_id = ?"
		args = append(args, filter.AreaID)
	}
	if filter.Online != nil {
		onlineVal := 0
		if *filter.Online {
			onlineVal = 1
		}
		query += " AND online = ?"
		args = append(args, onlineVal)
	}

	query += " ORDER BY name ASC;"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list devices: %w", err)
	}
	defer rows.Close()

	var devices []*device.Device
	for rows.Next() {
		var (
			d            device.Device
			manufacturer sql.NullString
			model        sql.NullString
			areaID       sql.NullString
			online       int
			createdAt    string
			updatedAt    string
		)
		err := rows.Scan(
			&d.ID,
			&d.IntegrationID,
			&d.ExternalID,
			&d.Name,
			&manufacturer,
			&model,
			&areaID,
			&online,
			&d.Metadata,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("sqlite: scan device: %w", err)
		}
		if manufacturer.Valid {
			d.Manufacturer = manufacturer.String
		}
		if model.Valid {
			d.Model = model.String
		}
		if areaID.Valid {
			d.AreaID = areaID.String
		}
		d.Online = online == 1
		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			d.CreatedAt = parsed
		}
		if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			d.UpdatedAt = parsed
		}
		devices = append(devices, &d)
	}
	return devices, rows.Err()
}

// Update modifies an existing device.
func (r *DeviceRepository) Update(ctx context.Context, d *device.Device) error {
	query := `
		UPDATE devices
		SET name = ?, manufacturer = ?, model = ?, area_id = ?, metadata = ?, updated_at = ?
		WHERE id = ?;
	`
	var manufacturer, model, areaID *string
	if d.Manufacturer != "" {
		manufacturer = &d.Manufacturer
	}
	if d.Model != "" {
		model = &d.Model
	}
	if d.AreaID != "" {
		areaID = &d.AreaID
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query,
		d.Name,
		manufacturer,
		model,
		areaID,
		d.Metadata,
		updatedAt,
		d.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update device: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

// SetOnline updates the online status of a device.
func (r *DeviceRepository) SetOnline(ctx context.Context, id string, online bool) error {
	query := `
		UPDATE devices
		SET online = ?, updated_at = ?
		WHERE id = ?;
	`
	onlineVal := 0
	if online {
		onlineVal = 1
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, query, onlineVal, updatedAt, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to set device online status: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

// Delete removes a device and associated cascades.
func (r *DeviceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete device: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return device.ErrDeviceNotFound
	}
	return nil
}

func (r *DeviceRepository) scanDevice(row *sql.Row) (*device.Device, error) {
	var (
		d            device.Device
		manufacturer sql.NullString
		model        sql.NullString
		areaID       sql.NullString
		online       int
		createdAt    string
		updatedAt    string
	)
	err := row.Scan(
		&d.ID,
		&d.IntegrationID,
		&d.ExternalID,
		&d.Name,
		&manufacturer,
		&model,
		&areaID,
		&online,
		&d.Metadata,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, device.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("sqlite: scan device: %w", err)
	}
	if manufacturer.Valid {
		d.Manufacturer = manufacturer.String
	}
	if model.Valid {
		d.Model = model.String
	}
	if areaID.Valid {
		d.AreaID = areaID.String
	}
	d.Online = online == 1
	if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
		d.CreatedAt = parsed
	}
	if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		d.UpdatedAt = parsed
	}
	return &d, nil
}
