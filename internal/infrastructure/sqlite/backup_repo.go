package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/backup"
)

// RemoteBackupConfigModel represents database storage for offsite backup target configuration.
type RemoteBackupConfigModel struct {
	ID                string                    `json:"id"`
	Enabled           bool                      `json:"enabled"`
	Destination       backup.OffsiteDestination `json:"destination"`
	RetentionDays     int                       `json:"retention_days"`
	EncryptionEnabled bool                      `json:"encryption_enabled"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

// BackupRecordModel represents a single completed or failed backup archive record.
type BackupRecordModel struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	SizeBytes    int64     `json:"size_bytes"`
	Checksum     string    `json:"checksum"`
	Destination  string    `json:"destination"`
	Status       string    `json:"status"` // "completed", "failed"
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// BackupRepository handles SQLite persistence for offsite backup settings and history records.
type BackupRepository struct {
	db *sql.DB
}

// NewBackupRepository creates a new SQLite BackupRepository.
func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// GetConfig loads the offsite backup configuration from SQLite.
func (r *BackupRepository) GetConfig(ctx context.Context) (*RemoteBackupConfigModel, error) {
	query := `
		SELECT id, enabled, destination_type, endpoint, path, bucket, retention_days, encryption_enabled, updated_at
		FROM remote_backup_configs
		WHERE id = 'default';
	`
	row := r.db.QueryRowContext(ctx, query)

	var (
		cfg        RemoteBackupConfigModel
		enabledInt int
		encInt     int
		destType   string
		updatedStr string
	)
	if err := row.Scan(
		&cfg.ID, &enabledInt, &destType, &cfg.Destination.Endpoint,
		&cfg.Destination.Path, &cfg.Destination.Bucket, &cfg.RetentionDays,
		&encInt, &updatedStr,
	); err != nil {
		if err == sql.ErrNoRows {
			// Return default configuration
			return &RemoteBackupConfigModel{
				ID:      "default",
				Enabled: false,
				Destination: backup.OffsiteDestination{
					Type:     backup.DestWebDAV,
					Endpoint: "",
					Path:     "/DomusBackups",
				},
				RetentionDays:     7,
				EncryptionEnabled: true,
				UpdatedAt:         time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("sqlite: failed to query backup config: %w", err)
	}

	cfg.Enabled = enabledInt == 1
	cfg.EncryptionEnabled = encInt == 1
	cfg.Destination.Type = backup.DestinationType(destType)
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		cfg.UpdatedAt = t
	}
	return &cfg, nil
}

// SaveConfig persists the offsite backup settings into SQLite.
func (r *BackupRepository) SaveConfig(ctx context.Context, cfg *RemoteBackupConfigModel) error {
	if cfg.ID == "" {
		cfg.ID = "default"
	}
	enabledInt := 0
	if cfg.Enabled {
		enabledInt = 1
	}
	encInt := 0
	if cfg.EncryptionEnabled {
		encInt = 1
	}
	now := time.Now().UTC()
	cfg.UpdatedAt = now

	query := `
		INSERT INTO remote_backup_configs (id, enabled, destination_type, endpoint, path, bucket, retention_days, encryption_enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			destination_type = excluded.destination_type,
			endpoint = excluded.endpoint,
			path = excluded.path,
			bucket = excluded.bucket,
			retention_days = excluded.retention_days,
			encryption_enabled = excluded.encryption_enabled,
			updated_at = excluded.updated_at;
	`
	_, err := r.db.ExecContext(ctx, query,
		cfg.ID, enabledInt, string(cfg.Destination.Type), cfg.Destination.Endpoint,
		cfg.Destination.Path, cfg.Destination.Bucket, cfg.RetentionDays, encInt,
		now.Format(time.RFC3339),
	)
	return err
}

// SaveRecord logs a backup archive execution into SQLite.
func (r *BackupRepository) SaveRecord(ctx context.Context, rec *BackupRecordModel) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO backup_records (id, filename, size_bytes, checksum, destination, status, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			error_message = excluded.error_message;
	`
	_, err := r.db.ExecContext(ctx, query,
		rec.ID, rec.Filename, rec.SizeBytes, rec.Checksum, rec.Destination,
		rec.Status, rec.ErrorMessage, rec.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// ListRecords returns recent backup execution logs from SQLite.
func (r *BackupRepository) ListRecords(ctx context.Context, limit int) ([]*BackupRecordModel, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, filename, size_bytes, checksum, destination, status, error_message, created_at
		FROM backup_records
		ORDER BY created_at DESC
		LIMIT ?;
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*BackupRecordModel
	for rows.Next() {
		var (
			rec        BackupRecordModel
			createdStr string
		)
		if err := rows.Scan(
			&rec.ID, &rec.Filename, &rec.SizeBytes, &rec.Checksum,
			&rec.Destination, &rec.Status, &rec.ErrorMessage, &createdStr,
		); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			rec.CreatedAt = t
		}
		records = append(records, &rec)
	}
	if records == nil {
		records = []*BackupRecordModel{}
	}
	return records, nil
}
