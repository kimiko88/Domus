// Package sqlite provides SQLite implementation of the scene repository port.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/scene"
	"github.com/google/uuid"
)

// Ensure SceneRepository implements scene.Repository.
var _ scene.Repository = (*SceneRepository)(nil)

// SceneRepository handles persistence for scene configurations.
type SceneRepository struct {
	db *sql.DB
}

// NewSceneRepository constructs a SceneRepository.
func NewSceneRepository(db *sql.DB) *SceneRepository {
	return &SceneRepository{db: db}
}

// Create inserts a new scene.
func (r *SceneRepository) Create(ctx context.Context, s *scene.Scene) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}

	actionsJSON, err := json.Marshal(s.Actions)
	if err != nil {
		return fmt.Errorf("sqlite: failed to marshal scene actions: %w", err)
	}

	query := `
		INSERT INTO scenes (id, name, icon, actions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(
		ctx, query,
		s.ID, s.Name, s.Icon, string(actionsJSON),
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert scene: %w", err)
	}
	return nil
}

// GetByID fetches a scene by ID.
func (r *SceneRepository) GetByID(ctx context.Context, id string) (*scene.Scene, error) {
	query := `
		SELECT id, name, icon, actions, created_at, updated_at
		FROM scenes
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		s                          scene.Scene
		icon                       sql.NullString
		actionsJSON                string
		createdAtStr, updatedAtStr string
	)

	err := row.Scan(&s.ID, &s.Name, &icon, &actionsJSON, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, scene.ErrSceneNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan scene: %w", err)
	}

	s.Icon = icon.String
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	_ = json.Unmarshal([]byte(actionsJSON), &s.Actions)

	return &s, nil
}

// List returns all scenes ordered by name.
func (r *SceneRepository) List(ctx context.Context) ([]*scene.Scene, error) {
	query := `
		SELECT id, name, icon, actions, created_at, updated_at
		FROM scenes
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query scenes: %w", err)
	}
	defer rows.Close()

	var result []*scene.Scene
	for rows.Next() {
		var (
			s                          scene.Scene
			icon                       sql.NullString
			actionsJSON                string
			createdAtStr, updatedAtStr string
		)
		if err := rows.Scan(&s.ID, &s.Name, &icon, &actionsJSON, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan scene row: %w", err)
		}
		s.Icon = icon.String
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		_ = json.Unmarshal([]byte(actionsJSON), &s.Actions)

		result = append(result, &s)
	}
	return result, nil
}

// Update updates an existing scene.
func (r *SceneRepository) Update(ctx context.Context, s *scene.Scene) error {
	now := time.Now().UTC()
	s.UpdatedAt = now

	actionsJSON, _ := json.Marshal(s.Actions)
	query := `
		UPDATE scenes
		SET name = ?, icon = ?, actions = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, s.Name, s.Icon, string(actionsJSON), s.UpdatedAt.Format(time.RFC3339), s.ID)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update scene: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return scene.ErrSceneNotFound
	}
	return nil
}

// Delete removes a scene.
func (r *SceneRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM scenes WHERE id = ?"
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete scene: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return scene.ErrSceneNotFound
	}
	return nil
}
