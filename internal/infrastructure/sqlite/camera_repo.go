package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/camera"
	"github.com/google/uuid"
)

// CameraRepository implements camera.Repository for SQLite.
type CameraRepository struct {
	db *sql.DB
}

// NewCameraRepository creates a new SQLite-backed CameraRepository.
func NewCameraRepository(db *sql.DB) *CameraRepository {
	return &CameraRepository{db: db}
}

// SaveCamera inserts or updates a camera configuration.
func (r *CameraRepository) SaveCamera(ctx context.Context, cam *camera.Camera) error {
	if cam.ID == "" {
		cam.ID = "cam_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if cam.CreatedAt.IsZero() {
		cam.CreatedAt = now
	}
	cam.UpdatedAt = now

	detEnabledInt := 0
	if cam.DetectionEnabled {
		detEnabledInt = 1
	}

	query := `
		INSERT INTO cameras (id, name, area_id, stream_url, snapshot_url, stream_type, detection_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			area_id = excluded.area_id,
			stream_url = excluded.stream_url,
			snapshot_url = excluded.snapshot_url,
			stream_type = excluded.stream_type,
			detection_enabled = excluded.detection_enabled,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(
		ctx, query,
		cam.ID,
		cam.Name,
		cam.AreaID,
		cam.StreamURL,
		cam.SnapshotURL,
		string(cam.StreamType),
		detEnabledInt,
		cam.CreatedAt.Format(time.RFC3339),
		cam.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save camera: %w", err)
	}
	return nil
}

// GetCamera retrieves a camera by ID.
func (r *CameraRepository) GetCamera(ctx context.Context, id string) (*camera.Camera, error) {
	query := `
		SELECT id, name, area_id, stream_url, snapshot_url, stream_type, detection_enabled, created_at, updated_at
		FROM cameras
		WHERE id = ?
	`
	var (
		cam           camera.Camera
		areaID        sql.NullString
		snapshotURL   sql.NullString
		streamType    string
		detEnabledInt int
		createdStr    string
		updatedStr    string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cam.ID,
		&cam.Name,
		&areaID,
		&cam.StreamURL,
		&snapshotURL,
		&streamType,
		&detEnabledInt,
		&createdStr,
		&updatedStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("camera not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get camera: %w", err)
	}

	cam.StreamType = camera.StreamType(streamType)
	cam.DetectionEnabled = detEnabledInt == 1
	cam.SnapshotURL = snapshotURL.String
	if areaID.Valid {
		cam.AreaID = &areaID.String
	}
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		cam.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		cam.UpdatedAt = t
	}

	return &cam, nil
}

// ListCameras returns all registered cameras.
func (r *CameraRepository) ListCameras(ctx context.Context) ([]*camera.Camera, error) {
	query := `
		SELECT id, name, area_id, stream_url, snapshot_url, stream_type, detection_enabled, created_at, updated_at
		FROM cameras
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list cameras: %w", err)
	}
	defer rows.Close()

	var cameras []*camera.Camera
	for rows.Next() {
		var (
			cam           camera.Camera
			areaID        sql.NullString
			snapshotURL   sql.NullString
			streamType    string
			detEnabledInt int
			createdStr    string
			updatedStr    string
		)
		if err := rows.Scan(
			&cam.ID,
			&cam.Name,
			&areaID,
			&cam.StreamURL,
			&snapshotURL,
			&streamType,
			&detEnabledInt,
			&createdStr,
			&updatedStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan camera error: %w", err)
		}
		cam.StreamType = camera.StreamType(streamType)
		cam.DetectionEnabled = detEnabledInt == 1
		cam.SnapshotURL = snapshotURL.String
		if areaID.Valid {
			cam.AreaID = &areaID.String
		}
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			cam.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			cam.UpdatedAt = t
		}
		cameras = append(cameras, &cam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if cameras == nil {
		cameras = []*camera.Camera{}
	}
	return cameras, nil
}

// DeleteCamera removes a camera by ID.
func (r *CameraRepository) DeleteCamera(ctx context.Context, id string) error {
	query := `DELETE FROM cameras WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete camera: %w", err)
	}
	return nil
}

// SaveEvent inserts a computer vision detection event.
func (r *CameraRepository) SaveEvent(ctx context.Context, evt *camera.DetectionEvent) error {
	if evt.ID == "" {
		evt.ID = "cev_" + uuid.NewString()[:8]
	}
	if evt.DetectedAt.IsZero() {
		evt.DetectedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO camera_events (id, camera_id, label, score, thumbnail_url, detected_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx, query,
		evt.ID,
		evt.CameraID,
		evt.Label,
		evt.Score,
		evt.ThumbnailURL,
		evt.DetectedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save camera event: %w", err)
	}
	return nil
}

// ListEvents returns detection events for a camera, limited to limit.
func (r *CameraRepository) ListEvents(ctx context.Context, cameraID string, limit int) ([]*camera.DetectionEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	var query string
	var args []any
	if cameraID != "" {
		query = `
			SELECT id, camera_id, label, score, thumbnail_url, detected_at
			FROM camera_events
			WHERE camera_id = ?
			ORDER BY detected_at DESC
			LIMIT ?
		`
		args = []any{cameraID, limit}
	} else {
		query = `
			SELECT id, camera_id, label, score, thumbnail_url, detected_at
			FROM camera_events
			ORDER BY detected_at DESC
			LIMIT ?
		`
		args = []any{limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list camera events: %w", err)
	}
	defer rows.Close()

	var events []*camera.DetectionEvent
	for rows.Next() {
		var (
			evt          camera.DetectionEvent
			thumbnailURL sql.NullString
			detectedStr  string
		)
		if err := rows.Scan(
			&evt.ID,
			&evt.CameraID,
			&evt.Label,
			&evt.Score,
			&thumbnailURL,
			&detectedStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan camera event error: %w", err)
		}
		evt.ThumbnailURL = thumbnailURL.String
		if t, err := time.Parse(time.RFC3339, detectedStr); err == nil {
			evt.DetectedAt = t
		}
		events = append(events, &evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows error: %w", err)
	}

	if events == nil {
		events = []*camera.DetectionEvent{}
	}
	return events, nil
}
