package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/application/camera"
)

// NVRRepository handles SQLite persistence for camera AI vision, family faces,
// authorized license plates, motion zones, and event clips.
type NVRRepository struct {
	db *sql.DB
}

// NewNVRRepository creates a new SQLite NVR repository.
func NewNVRRepository(db *sql.DB) *NVRRepository {
	return &NVRRepository{db: db}
}

// SaveFamilyFace persists an enrolled family member's face embedding vector.
func (r *NVRRepository) SaveFamilyFace(ctx context.Context, f camera.FamilyFace) error {
	embJSON, err := json.Marshal(f.Embedding)
	if err != nil {
		embJSON = []byte("[]")
	}

	query := `
		INSERT INTO camera_family_faces (id, name, role, photo_url, embedding, enrolled_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			role = excluded.role,
			photo_url = excluded.photo_url,
			embedding = excluded.embedding,
			enrolled_at = excluded.enrolled_at;
	`
	_, err = r.db.ExecContext(ctx, query, f.ID, f.Name, f.Role, f.PhotoURL, string(embJSON), f.EnrolledAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("sqlite: failed to save family face: %w", err)
	}
	return nil
}

// ListFamilyFaces returns all enrolled biometric face profiles from SQLite.
func (r *NVRRepository) ListFamilyFaces(ctx context.Context) ([]camera.FamilyFace, error) {
	query := `SELECT id, name, role, photo_url, embedding, enrolled_at FROM camera_family_faces ORDER BY enrolled_at DESC;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list family faces: %w", err)
	}
	defer rows.Close()

	var faces []camera.FamilyFace
	for rows.Next() {
		var (
			f           camera.FamilyFace
			embStr      string
			enrolledStr string
		)
		if err := rows.Scan(&f.ID, &f.Name, &f.Role, &f.PhotoURL, &embStr, &enrolledStr); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan family face: %w", err)
		}
		_ = json.Unmarshal([]byte(embStr), &f.Embedding)
		if t, err := time.Parse(time.RFC3339, enrolledStr); err == nil {
			f.EnrolledAt = t
		}
		faces = append(faces, f)
	}
	if faces == nil {
		faces = []camera.FamilyFace{}
	}
	return faces, nil
}

// DeleteFamilyFace deletes an enrolled face template from SQLite.
func (r *NVRRepository) DeleteFamilyFace(ctx context.Context, id string) error {
	query := `DELETE FROM camera_family_faces WHERE id = ?;`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// SaveAuthorizedPlate records an authorized vehicle license plate in SQLite.
func (r *NVRRepository) SaveAuthorizedPlate(ctx context.Context, v camera.AuthorizedVehicle) error {
	gateInt := 0
	if v.AutoOpenGate {
		gateInt = 1
	}

	query := `
		INSERT INTO camera_authorized_plates (plate_number, owner_name, vehicle_model, auto_open_gate, enrolled_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(plate_number) DO UPDATE SET
			owner_name = excluded.owner_name,
			vehicle_model = excluded.vehicle_model,
			auto_open_gate = excluded.auto_open_gate,
			enrolled_at = excluded.enrolled_at;
	`
	_, err := r.db.ExecContext(ctx, query, v.PlateNumber, v.OwnerName, v.VehicleModel, gateInt, v.EnrolledAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("sqlite: failed to save authorized plate: %w", err)
	}
	return nil
}

// ListAuthorizedPlates returns all enrolled license plates from SQLite.
func (r *NVRRepository) ListAuthorizedPlates(ctx context.Context) ([]camera.AuthorizedVehicle, error) {
	query := `SELECT plate_number, owner_name, vehicle_model, auto_open_gate, enrolled_at FROM camera_authorized_plates ORDER BY enrolled_at DESC;`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list authorized plates: %w", err)
	}
	defer rows.Close()

	var plates []camera.AuthorizedVehicle
	for rows.Next() {
		var (
			v           camera.AuthorizedVehicle
			gateInt     int
			enrolledStr string
		)
		if err := rows.Scan(&v.PlateNumber, &v.OwnerName, &v.VehicleModel, &gateInt, &enrolledStr); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan plate: %w", err)
		}
		v.AutoOpenGate = gateInt == 1
		if t, err := time.Parse(time.RFC3339, enrolledStr); err == nil {
			v.EnrolledAt = t
		}
		plates = append(plates, v)
	}
	if plates == nil {
		plates = []camera.AuthorizedVehicle{}
	}
	return plates, nil
}

// DeleteAuthorizedPlate removes an authorized plate from SQLite.
func (r *NVRRepository) DeleteAuthorizedPlate(ctx context.Context, plate string) error {
	query := `DELETE FROM camera_authorized_plates WHERE plate_number = ?;`
	_, err := r.db.ExecContext(ctx, query, plate)
	return err
}

// SaveZone saves or updates an intelligent motion/AI zone in SQLite.
func (r *NVRRepository) SaveZone(ctx context.Context, z camera.MotionZone) error {
	labelsJSON, _ := json.Marshal(z.Labels)
	coordsJSON, _ := json.Marshal(z.Coordinates)
	enabledInt := 0
	if z.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO camera_motion_zones (id, camera_id, name, sensitivity, labels, coordinates, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			sensitivity = excluded.sensitivity,
			labels = excluded.labels,
			coordinates = excluded.coordinates,
			enabled = excluded.enabled;
	`
	_, err := r.db.ExecContext(ctx, query, z.ID, z.CameraID, z.Name, z.Sensitivity, string(labelsJSON), string(coordsJSON), enabledInt)
	return err
}

// GetZones retrieves motion zones configured for a camera from SQLite.
func (r *NVRRepository) GetZones(ctx context.Context, cameraID string) ([]camera.MotionZone, error) {
	query := `SELECT id, camera_id, name, sensitivity, labels, coordinates, enabled FROM camera_motion_zones WHERE camera_id = ?;`
	rows, err := r.db.QueryContext(ctx, query, cameraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []camera.MotionZone
	for rows.Next() {
		var (
			z          camera.MotionZone
			labelsStr  string
			coordsStr  string
			enabledInt int
		)
		if err := rows.Scan(&z.ID, &z.CameraID, &z.Name, &z.Sensitivity, &labelsStr, &coordsStr, &enabledInt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labelsStr), &z.Labels)
		_ = json.Unmarshal([]byte(coordsStr), &z.Coordinates)
		z.Enabled = enabledInt == 1
		zones = append(zones, z)
	}
	if zones == nil {
		zones = []camera.MotionZone{}
	}
	return zones, nil
}

// SaveEventClip records a generated video clip snippet in SQLite.
func (r *NVRRepository) SaveEventClip(ctx context.Context, c *camera.EventClip) error {
	query := `
		INSERT INTO camera_event_clips (event_id, camera_id, duration_sec, thumbnail_url, clip_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(event_id) DO UPDATE SET
			duration_sec = excluded.duration_sec,
			thumbnail_url = excluded.thumbnail_url,
			clip_url = excluded.clip_url;
	`
	_, err := r.db.ExecContext(ctx, query, c.EventID, c.CameraID, c.DurationSec, c.ThumbnailURL, c.ClipURL, c.CreatedAt.Format(time.RFC3339))
	return err
}

// GetEventClip fetches a video clip record by event ID from SQLite.
func (r *NVRRepository) GetEventClip(ctx context.Context, eventID string) (*camera.EventClip, error) {
	query := `SELECT event_id, camera_id, duration_sec, thumbnail_url, clip_url, created_at FROM camera_event_clips WHERE event_id = ?;`
	row := r.db.QueryRowContext(ctx, query, eventID)

	var (
		c          camera.EventClip
		createdStr string
	)
	if err := row.Scan(&c.EventID, &c.CameraID, &c.DurationSec, &c.ThumbnailURL, &c.ClipURL, &createdStr); err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		c.CreatedAt = t
	}
	return &c, nil
}
