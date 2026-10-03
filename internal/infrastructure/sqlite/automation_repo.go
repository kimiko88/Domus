// Package sqlite provides SQLite implementations of domain repository ports.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/google/uuid"
)

// Ensure AutomationRepository implements automation.Repository.
var _ automation.Repository = (*AutomationRepository)(nil)

// AutomationRepository handles persistence for automations and automation runs.
type AutomationRepository struct {
	db *sql.DB
}

// NewAutomationRepository constructs an AutomationRepository.
func NewAutomationRepository(db *sql.DB) *AutomationRepository {
	return &AutomationRepository{db: db}
}

// Create inserts a new automation record.
func (r *AutomationRepository) Create(ctx context.Context, a *automation.Automation) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}

	triggersJSON, err := json.Marshal(a.Triggers)
	if err != nil {
		return fmt.Errorf("sqlite: failed to marshal triggers: %w", err)
	}
	conditionsJSON, err := json.Marshal(a.Conditions)
	if err != nil {
		return fmt.Errorf("sqlite: failed to marshal conditions: %w", err)
	}
	actionsJSON, err := json.Marshal(a.Actions)
	if err != nil {
		return fmt.Errorf("sqlite: failed to marshal actions: %w", err)
	}

	query := `
		INSERT INTO automations (
			id, name, description, enabled, trigger, conditions, actions,
			execution_mode, cooldown_seconds, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	enabledInt := 0
	if a.Enabled {
		enabledInt = 1
	}

	_, err = r.db.ExecContext(
		ctx, query,
		a.ID, a.Name, a.Description, enabledInt,
		string(triggersJSON), string(conditionsJSON), string(actionsJSON),
		a.ExecutionMode, a.CooldownSeconds,
		a.CreatedAt.Format(time.RFC3339),
		a.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert automation: %w", err)
	}
	return nil
}

// GetByID fetches an automation by ID.
func (r *AutomationRepository) GetByID(ctx context.Context, id string) (*automation.Automation, error) {
	query := `
		SELECT id, name, description, enabled, trigger, conditions, actions,
		       execution_mode, cooldown_seconds, last_run_at, last_status, created_at, updated_at
		FROM automations
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		a                                         automation.Automation
		desc, lastRunAtStr, lastStatus            sql.NullString
		enabledInt                                int
		triggersJSON, conditionsJSON, actionsJSON string
		createdAtStr, updatedAtStr                string
	)

	err := row.Scan(
		&a.ID, &a.Name, &desc, &enabledInt,
		&triggersJSON, &conditionsJSON, &actionsJSON,
		&a.ExecutionMode, &a.CooldownSeconds,
		&lastRunAtStr, &lastStatus,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, automation.ErrAutomationNotFound
		}
		return nil, fmt.Errorf("sqlite: failed to scan automation: %w", err)
	}

	a.Description = desc.String
	a.Enabled = enabledInt == 1
	a.LastStatus = lastStatus.String

	if lastRunAtStr.Valid && lastRunAtStr.String != "" {
		if t, err := time.Parse(time.RFC3339, lastRunAtStr.String); err == nil {
			a.LastRunAt = &t
		}
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	// Unmarshal JSON DSL
	_ = json.Unmarshal([]byte(triggersJSON), &a.Triggers)
	_ = json.Unmarshal([]byte(conditionsJSON), &a.Conditions)
	_ = json.Unmarshal([]byte(actionsJSON), &a.Actions)

	return &a, nil
}

// List returns automations matching the given filter.
func (r *AutomationRepository) List(ctx context.Context, filter automation.ListFilter) ([]*automation.Automation, error) {
	query := `
		SELECT id, name, description, enabled, trigger, conditions, actions,
		       execution_mode, cooldown_seconds, last_run_at, last_status, created_at, updated_at
		FROM automations
		WHERE 1=1
	`
	var args []any
	if filter.Enabled != nil {
		enabledInt := 0
		if *filter.Enabled {
			enabledInt = 1
		}
		query += " AND enabled = ?"
		args = append(args, enabledInt)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query automations: %w", err)
	}
	defer rows.Close()

	var result []*automation.Automation
	for rows.Next() {
		var (
			a                                         automation.Automation
			desc, lastRunAtStr, lastStatus            sql.NullString
			enabledInt                                int
			triggersJSON, conditionsJSON, actionsJSON string
			createdAtStr, updatedAtStr                string
		)
		if err := rows.Scan(
			&a.ID, &a.Name, &desc, &enabledInt,
			&triggersJSON, &conditionsJSON, &actionsJSON,
			&a.ExecutionMode, &a.CooldownSeconds,
			&lastRunAtStr, &lastStatus,
			&createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan automation row: %w", err)
		}

		a.Description = desc.String
		a.Enabled = enabledInt == 1
		a.LastStatus = lastStatus.String

		if lastRunAtStr.Valid && lastRunAtStr.String != "" {
			if t, err := time.Parse(time.RFC3339, lastRunAtStr.String); err == nil {
				a.LastRunAt = &t
			}
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		_ = json.Unmarshal([]byte(triggersJSON), &a.Triggers)
		_ = json.Unmarshal([]byte(conditionsJSON), &a.Conditions)
		_ = json.Unmarshal([]byte(actionsJSON), &a.Actions)

		result = append(result, &a)
	}
	return result, nil
}

// Update saves modifications to an automation.
func (r *AutomationRepository) Update(ctx context.Context, a *automation.Automation) error {
	now := time.Now().UTC()
	a.UpdatedAt = now

	triggersJSON, _ := json.Marshal(a.Triggers)
	conditionsJSON, _ := json.Marshal(a.Conditions)
	actionsJSON, _ := json.Marshal(a.Actions)

	enabledInt := 0
	if a.Enabled {
		enabledInt = 1
	}

	query := `
		UPDATE automations
		SET name = ?, description = ?, enabled = ?, trigger = ?, conditions = ?, actions = ?,
		    execution_mode = ?, cooldown_seconds = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(
		ctx, query,
		a.Name, a.Description, enabledInt,
		string(triggersJSON), string(conditionsJSON), string(actionsJSON),
		a.ExecutionMode, a.CooldownSeconds,
		a.UpdatedAt.Format(time.RFC3339),
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update automation: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return automation.ErrAutomationNotFound
	}
	return nil
}

// UpdateLastRun records the timestamp and status of the latest automation run.
func (r *AutomationRepository) UpdateLastRun(ctx context.Context, id string, status string) error {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	query := `
		UPDATE automations
		SET last_run_at = ?, last_status = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, nowStr, status, id)
	return err
}

// Delete removes an automation record.
func (r *AutomationRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM automations WHERE id = ?"
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete automation: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return automation.ErrAutomationNotFound
	}
	return nil
}

// CreateRun inserts an automation run execution record.
func (r *AutomationRepository) CreateRun(ctx context.Context, run *automation.AutomationRun) error {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	query := `
		INSERT INTO automation_runs (
			id, automation_id, started_at, status, trigger_context, log
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx, query,
		run.ID, run.AutomationID,
		run.StartedAt.Format(time.RFC3339),
		run.Status, run.TriggerContext, run.Log,
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to insert automation run: %w", err)
	}
	return nil
}

// UpdateRun updates status, finish time, and log for an automation run.
func (r *AutomationRepository) UpdateRun(ctx context.Context, run *automation.AutomationRun) error {
	var finishedAtStr sql.NullString
	if run.FinishedAt != nil {
		finishedAtStr = sql.NullString{String: run.FinishedAt.Format(time.RFC3339), Valid: true}
	}

	query := `
		UPDATE automation_runs
		SET finished_at = ?, status = ?, log = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, finishedAtStr, run.Status, run.Log, run.ID)
	if err != nil {
		return fmt.Errorf("sqlite: failed to update automation run: %w", err)
	}
	return nil
}

// ListRuns returns run records for a specific automation, newest first.
func (r *AutomationRepository) ListRuns(ctx context.Context, automationID string, limit int) ([]*automation.AutomationRun, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, automation_id, started_at, finished_at, status, trigger_context, log
		FROM automation_runs
		WHERE automation_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, automationID, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query automation runs: %w", err)
	}
	defer rows.Close()

	var runs []*automation.AutomationRun
	for rows.Next() {
		var (
			run           automation.AutomationRun
			startedAtStr  string
			finishedAtStr sql.NullString
		)
		if err := rows.Scan(
			&run.ID, &run.AutomationID, &startedAtStr, &finishedAtStr,
			&run.Status, &run.TriggerContext, &run.Log,
		); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan automation run: %w", err)
		}
		run.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		if finishedAtStr.Valid && finishedAtStr.String != "" {
			if t, err := time.Parse(time.RFC3339, finishedAtStr.String); err == nil {
				run.FinishedAt = &t
				run.DurationMs = t.Sub(run.StartedAt).Milliseconds()
			}
		}
		runs = append(runs, &run)
	}
	return runs, nil
}
