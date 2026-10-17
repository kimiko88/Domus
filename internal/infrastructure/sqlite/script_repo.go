// Package sqlite provides SQLite repository implementation for reusable scripts.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/script"
	"github.com/google/uuid"
)

// ScriptRepository persists scripts in the scripts SQLite table.
type ScriptRepository struct {
	db *sql.DB
}

// NewScriptRepository creates a new ScriptRepository.
func NewScriptRepository(db *sql.DB) *ScriptRepository {
	return &ScriptRepository{db: db}
}

// Create inserts a new script into SQLite.
func (r *ScriptRepository) Create(ctx context.Context, s *script.Script) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now

	fieldsJSON, err := json.Marshal(s.Fields)
	if err != nil {
		fieldsJSON = []byte("{}")
	}

	seqJSON, err := json.Marshal(s.Sequence)
	if err != nil {
		seqJSON = []byte("[]")
	}

	query := `
		INSERT INTO scripts (
			id, name, description, icon, mode, max_runs, fields_json, sequence_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		s.ID,
		s.Name,
		s.Description,
		s.Icon,
		s.Mode,
		s.MaxRuns,
		string(fieldsJSON),
		string(seqJSON),
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert script: %w", err)
	}
	return nil
}

// Get retrieves a script by its ID.
func (r *ScriptRepository) Get(ctx context.Context, id string) (*script.Script, error) {
	query := `
		SELECT id, name, description, icon, mode, max_runs, fields_json, sequence_json, created_at, updated_at
		FROM scripts
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		s            script.Script
		fieldsJSON   string
		seqJSON      string
		createdAtStr string
		updatedAtStr string
	)

	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Description,
		&s.Icon,
		&s.Mode,
		&s.MaxRuns,
		&fieldsJSON,
		&seqJSON,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, script.ErrScriptNotFound
		}
		return nil, fmt.Errorf("failed to scan script: %w", err)
	}

	_ = json.Unmarshal([]byte(fieldsJSON), &s.Fields)
	var seq []automation.Action
	if err := json.Unmarshal([]byte(seqJSON), &seq); err == nil {
		s.Sequence = seq
	}

	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		s.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		s.UpdatedAt = t
	}

	return &s, nil
}

// List returns all persisted scripts.
func (r *ScriptRepository) List(ctx context.Context) ([]*script.Script, error) {
	query := `
		SELECT id, name, description, icon, mode, max_runs, fields_json, sequence_json, created_at, updated_at
		FROM scripts
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scripts: %w", err)
	}
	defer rows.Close()

	var results []*script.Script
	for rows.Next() {
		var (
			s            script.Script
			fieldsJSON   string
			seqJSON      string
			createdAtStr string
			updatedAtStr string
		)
		err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Description,
			&s.Icon,
			&s.Mode,
			&s.MaxRuns,
			&fieldsJSON,
			&seqJSON,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan script row: %w", err)
		}

		_ = json.Unmarshal([]byte(fieldsJSON), &s.Fields)
		var seq []automation.Action
		if err := json.Unmarshal([]byte(seqJSON), &seq); err == nil {
			s.Sequence = seq
		}

		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			s.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			s.UpdatedAt = t
		}

		results = append(results, &s)
	}

	return results, nil
}

// Update modifies an existing script.
func (r *ScriptRepository) Update(ctx context.Context, s *script.Script) error {
	s.UpdatedAt = time.Now().UTC()
	fieldsJSON, _ := json.Marshal(s.Fields)
	seqJSON, _ := json.Marshal(s.Sequence)

	query := `
		UPDATE scripts SET
			name = ?, description = ?, icon = ?, mode = ?, max_runs = ?,
			fields_json = ?, sequence_json = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		s.Name,
		s.Description,
		s.Icon,
		s.Mode,
		s.MaxRuns,
		string(fieldsJSON),
		string(seqJSON),
		s.UpdatedAt.Format(time.RFC3339),
		s.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update script: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return script.ErrScriptNotFound
	}
	return nil
}

// Delete removes a script by ID.
func (r *ScriptRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM scripts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete script: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return script.ErrScriptNotFound
	}
	return nil
}
