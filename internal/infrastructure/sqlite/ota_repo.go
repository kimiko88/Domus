package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/ota"
	"github.com/google/uuid"
)

// OTARepository implements ota.Repository for SQLite.
type OTARepository struct {
	db *sql.DB
}

// NewOTARepository creates a new SQLite-backed OTARepository.
func NewOTARepository(db *sql.DB) *OTARepository {
	return &OTARepository{db: db}
}

// SaveDevice inserts or updates an OTA device record.
func (r *OTARepository) SaveDevice(ctx context.Context, dev *ota.DeviceOTA) error {
	if dev.ID == "" {
		dev.ID = "ota_" + uuid.NewString()[:8]
	}
	if dev.CheckedAt.IsZero() {
		dev.CheckedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO ota_devices (id, name, device_key, platform, current_version, latest_version, upgrade_state, release_notes, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_key) DO UPDATE SET
			name = excluded.name,
			platform = excluded.platform,
			current_version = excluded.current_version,
			latest_version = excluded.latest_version,
			upgrade_state = excluded.upgrade_state,
			release_notes = excluded.release_notes,
			checked_at = excluded.checked_at
	`
	_, err := r.db.ExecContext(
		ctx, query,
		dev.ID,
		dev.Name,
		dev.DeviceKey,
		dev.Platform,
		dev.CurrentVersion,
		dev.LatestVersion,
		string(dev.UpgradeState),
		dev.ReleaseNotes,
		dev.CheckedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save ota device: %w", err)
	}
	return nil
}

// GetDevice retrieves an OTA device by ID.
func (r *OTARepository) GetDevice(ctx context.Context, id string) (*ota.DeviceOTA, error) {
	query := `
		SELECT id, name, device_key, platform, current_version, latest_version, upgrade_state, release_notes, checked_at
		FROM ota_devices
		WHERE id = ?
	`
	var (
		dev          ota.DeviceOTA
		upgradeState string
		releaseNotes sql.NullString
		checkedStr   string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dev.ID,
		&dev.Name,
		&dev.DeviceKey,
		&dev.Platform,
		&dev.CurrentVersion,
		&dev.LatestVersion,
		&upgradeState,
		&releaseNotes,
		&checkedStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ota device not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get ota device: %w", err)
	}

	dev.UpgradeState = ota.UpgradeState(upgradeState)
	dev.ReleaseNotes = releaseNotes.String
	dev.HasUpdate = dev.LatestVersion != "" && dev.LatestVersion != dev.CurrentVersion
	if t, err := time.Parse(time.RFC3339, checkedStr); err == nil {
		dev.CheckedAt = t
	}

	return &dev, nil
}

// ListDevices returns all devices registered for OTA.
func (r *OTARepository) ListDevices(ctx context.Context) ([]*ota.DeviceOTA, error) {
	query := `
		SELECT id, name, device_key, platform, current_version, latest_version, upgrade_state, release_notes, checked_at
		FROM ota_devices
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list ota devices: %w", err)
	}
	defer rows.Close()

	var devices []*ota.DeviceOTA
	for rows.Next() {
		var (
			dev          ota.DeviceOTA
			upgradeState string
			releaseNotes sql.NullString
			checkedStr   string
		)
		if err := rows.Scan(
			&dev.ID,
			&dev.Name,
			&dev.DeviceKey,
			&dev.Platform,
			&dev.CurrentVersion,
			&dev.LatestVersion,
			&upgradeState,
			&releaseNotes,
			&checkedStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan ota device error: %w", err)
		}
		dev.UpgradeState = ota.UpgradeState(upgradeState)
		dev.ReleaseNotes = releaseNotes.String
		dev.HasUpdate = dev.LatestVersion != "" && dev.LatestVersion != dev.CurrentVersion
		if t, err := time.Parse(time.RFC3339, checkedStr); err == nil {
			dev.CheckedAt = t
		}
		devices = append(devices, &dev)
	}

	if devices == nil {
		devices = []*ota.DeviceOTA{}
	}
	return devices, nil
}

// DeleteDevice removes an OTA device record by ID.
func (r *OTARepository) DeleteDevice(ctx context.Context, id string) error {
	query := `DELETE FROM ota_devices WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete ota device: %w", err)
	}
	return nil
}
