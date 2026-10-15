// Package sqlite provides SQLite implementations of domain repositories.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/template_entity"
	"github.com/google/uuid"
)

var _ template_entity.Repository = (*TemplateEntityRepository)(nil)

// TemplateEntityRepository handles SQLite persistence for template entities.
type TemplateEntityRepository struct {
	db *sql.DB
}

// NewTemplateEntityRepository creates a new TemplateEntityRepository.
func NewTemplateEntityRepository(db *sql.DB) *TemplateEntityRepository {
	return &TemplateEntityRepository{db: db}
}

// Create inserts a new template entity.
func (r *TemplateEntityRepository) Create(ctx context.Context, e *template_entity.TemplateEntity) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	sourcesJSON, err := json.Marshal(e.SourceEntityIDs)
	if err != nil {
		sourcesJSON = []byte("[]")
	}

	query := `
		INSERT INTO template_entities (
			id, entity_id, domain, name, state_template, attributes_template,
			unit_of_measurement, device_class, icon, source_entity_ids, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		e.ID, e.EntityID, e.Domain, e.Name, e.StateTemplate, e.AttributesTemplate,
		e.UnitOfMeasurement, e.DeviceClass, e.Icon, string(sourcesJSON),
		e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert template entity: %w", err)
	}
	return nil
}

// GetByID retrieves a template entity by its primary ID.
func (r *TemplateEntityRepository) GetByID(ctx context.Context, id string) (*template_entity.TemplateEntity, error) {
	query := `
		SELECT id, entity_id, domain, name, state_template, attributes_template,
		       unit_of_measurement, device_class, icon, source_entity_ids, created_at, updated_at
		FROM template_entities
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanTemplateEntity(row)
}

// GetByEntityID retrieves a template entity by its unique entity_id.
func (r *TemplateEntityRepository) GetByEntityID(ctx context.Context, entityID string) (*template_entity.TemplateEntity, error) {
	query := `
		SELECT id, entity_id, domain, name, state_template, attributes_template,
		       unit_of_measurement, device_class, icon, source_entity_ids, created_at, updated_at
		FROM template_entities
		WHERE entity_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, entityID)
	return r.scanTemplateEntity(row)
}

// List returns all registered template entities.
func (r *TemplateEntityRepository) List(ctx context.Context) ([]*template_entity.TemplateEntity, error) {
	query := `
		SELECT id, entity_id, domain, name, state_template, attributes_template,
		       unit_of_measurement, device_class, icon, source_entity_ids, created_at, updated_at
		FROM template_entities
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list template entities: %w", err)
	}
	defer rows.Close()

	var list []*template_entity.TemplateEntity
	for rows.Next() {
		e, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// Update updates an existing template entity.
func (r *TemplateEntityRepository) Update(ctx context.Context, e *template_entity.TemplateEntity) error {
	e.UpdatedAt = time.Now().UTC()
	sourcesJSON, err := json.Marshal(e.SourceEntityIDs)
	if err != nil {
		sourcesJSON = []byte("[]")
	}

	query := `
		UPDATE template_entities
		SET name = ?, state_template = ?, attributes_template = ?,
		    unit_of_measurement = ?, device_class = ?, icon = ?,
		    source_entity_ids = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		e.Name, e.StateTemplate, e.AttributesTemplate,
		e.UnitOfMeasurement, e.DeviceClass, e.Icon,
		string(sourcesJSON), e.UpdatedAt.Format(time.RFC3339),
		e.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update template entity: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return template_entity.ErrNotFound
	}
	return nil
}

// Delete removes a template entity by ID.
func (r *TemplateEntityRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM template_entities WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete template entity: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return template_entity.ErrNotFound
	}
	return nil
}

func (r *TemplateEntityRepository) scanTemplateEntity(row *sql.Row) (*template_entity.TemplateEntity, error) {
	var (
		e           template_entity.TemplateEntity
		sourcesStr  string
		createdAtStr string
		updatedAtStr string
		unitNull    sql.NullString
		classNull   sql.NullString
		iconNull    sql.NullString
	)

	err := row.Scan(
		&e.ID, &e.EntityID, &e.Domain, &e.Name, &e.StateTemplate, &e.AttributesTemplate,
		&unitNull, &classNull, &iconNull, &sourcesStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, template_entity.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan template entity: %w", err)
	}

	e.UnitOfMeasurement = unitNull.String
	e.DeviceClass = classNull.String
	e.Icon = iconNull.String
	_ = json.Unmarshal([]byte(sourcesStr), &e.SourceEntityIDs)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &e, nil
}

func (r *TemplateEntityRepository) scanRow(rows *sql.Rows) (*template_entity.TemplateEntity, error) {
	var (
		e           template_entity.TemplateEntity
		sourcesStr  string
		createdAtStr string
		updatedAtStr string
		unitNull    sql.NullString
		classNull   sql.NullString
		iconNull    sql.NullString
	)

	err := rows.Scan(
		&e.ID, &e.EntityID, &e.Domain, &e.Name, &e.StateTemplate, &e.AttributesTemplate,
		&unitNull, &classNull, &iconNull, &sourcesStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to scan template entity row: %w", err)
	}

	e.UnitOfMeasurement = unitNull.String
	e.DeviceClass = classNull.String
	e.Icon = iconNull.String
	_ = json.Unmarshal([]byte(sourcesStr), &e.SourceEntityIDs)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &e, nil
}
