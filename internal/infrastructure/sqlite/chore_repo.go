package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/chore"
)

type choreRepo struct {
	db *sql.DB
}

// NewChoreRepository creates a new SQLite-backed chore repository.
func NewChoreRepository(db *sql.DB) chore.Repository {
	return &choreRepo{db: db}
}

func (r *choreRepo) Create(ctx context.Context, c *chore.Chore) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	var dueStr *string
	if c.DueDate != nil {
		s := c.DueDate.UTC().Format(time.RFC3339)
		dueStr = &s
	}

	var areaID *string
	if c.AreaID != "" {
		areaID = &c.AreaID
	}

	var userID *string
	if c.AssignedUserID != "" {
		userID = &c.AssignedUserID
	}

	query := `
		INSERT INTO chores (
			id, title, description, area_id, assigned_user_id, priority, due_date,
			recurrence_type, recurrence_rule, status, last_completed_at, last_completed_by,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Title, c.Description, areaID, userID,
		string(c.Priority), dueStr, string(c.RecurrenceType), c.RecurrenceRule,
		string(c.Status), nil, nil,
		c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *choreRepo) GetByID(ctx context.Context, id string) (*chore.Chore, error) {
	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(area_id, ''), COALESCE(assigned_user_id, ''),
		       priority, due_date, recurrence_type, COALESCE(recurrence_rule, ''), status,
		       last_completed_at, COALESCE(last_completed_by, ''), created_at, updated_at
		FROM chores WHERE id = ?;
	`
	return r.scanChore(r.db.QueryRowContext(ctx, query, id))
}

func (r *choreRepo) List(ctx context.Context, f chore.Filter) ([]*chore.Chore, error) {
	var conditions []string
	var args []any

	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(f.Status))
	}
	if f.AssignedUserID != "" {
		conditions = append(conditions, "assigned_user_id = ?")
		args = append(args, f.AssignedUserID)
	}
	if f.AreaID != "" {
		conditions = append(conditions, "area_id = ?")
		args = append(args, f.AreaID)
	}

	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(area_id, ''), COALESCE(assigned_user_id, ''),
		       priority, due_date, recurrence_type, COALESCE(recurrence_rule, ''), status,
		       last_completed_at, COALESCE(last_completed_by, ''), created_at, updated_at
		FROM chores
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY CASE WHEN due_date IS NULL THEN 1 ELSE 0 END, due_date ASC, created_at DESC;"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chores []*chore.Chore
	for rows.Next() {
		c, err := r.scanChore(rows)
		if err != nil {
			return nil, err
		}
		chores = append(chores, c)
	}
	return chores, rows.Err()
}

func (r *choreRepo) Update(ctx context.Context, c *chore.Chore) error {
	c.UpdatedAt = time.Now().UTC()

	var dueStr *string
	if c.DueDate != nil {
		s := c.DueDate.UTC().Format(time.RFC3339)
		dueStr = &s
	}

	var completedStr *string
	if c.LastCompletedAt != nil {
		s := c.LastCompletedAt.UTC().Format(time.RFC3339)
		completedStr = &s
	}

	var areaID *string
	if c.AreaID != "" {
		areaID = &c.AreaID
	}

	var userID *string
	if c.AssignedUserID != "" {
		userID = &c.AssignedUserID
	}

	var completedBy *string
	if c.LastCompletedBy != "" {
		completedBy = &c.LastCompletedBy
	}

	query := `
		UPDATE chores
		SET title = ?, description = ?, area_id = ?, assigned_user_id = ?,
		    priority = ?, due_date = ?, recurrence_type = ?, recurrence_rule = ?,
		    status = ?, last_completed_at = ?, last_completed_by = ?, updated_at = ?
		WHERE id = ?;
	`
	res, err := r.db.ExecContext(ctx, query,
		c.Title, c.Description, areaID, userID,
		string(c.Priority), dueStr, string(c.RecurrenceType), c.RecurrenceRule,
		string(c.Status), completedStr, completedBy,
		c.UpdatedAt.Format(time.RFC3339), c.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return chore.ErrChoreNotFound
	}
	return nil
}

func (r *choreRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM chores WHERE id = ?;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return chore.ErrChoreNotFound
	}
	return nil
}

func (r *choreRepo) GetPendingDueBefore(ctx context.Context, cutoff time.Time) ([]*chore.Chore, error) {
	query := `
		SELECT id, title, COALESCE(description, ''), COALESCE(area_id, ''), COALESCE(assigned_user_id, ''),
		       priority, due_date, recurrence_type, COALESCE(recurrence_rule, ''), status,
		       last_completed_at, COALESCE(last_completed_by, ''), created_at, updated_at
		FROM chores
		WHERE status = 'pending' AND due_date IS NOT NULL AND due_date <= ?
		ORDER BY due_date ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chores []*chore.Chore
	for rows.Next() {
		c, err := r.scanChore(rows)
		if err != nil {
			return nil, err
		}
		chores = append(chores, c)
	}
	return chores, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *choreRepo) scanChore(s scanner) (*chore.Chore, error) {
	var c chore.Chore
	var priorityStr, recurrenceStr, statusStr string
	var dueStr, completedStr *string
	var createdStr, updatedStr string

	err := s.Scan(
		&c.ID, &c.Title, &c.Description, &c.AreaID, &c.AssignedUserID,
		&priorityStr, &dueStr, &recurrenceStr, &c.RecurrenceRule, &statusStr,
		&completedStr, &c.LastCompletedBy, &createdStr, &updatedStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, chore.ErrChoreNotFound
		}
		return nil, err
	}

	c.Priority = chore.Priority(priorityStr)
	c.RecurrenceType = chore.RecurrenceType(recurrenceStr)
	c.Status = chore.Status(statusStr)

	if dueStr != nil && *dueStr != "" {
		if t, err := time.Parse(time.RFC3339, *dueStr); err == nil {
			c.DueDate = &t
		}
	}
	if completedStr != nil && *completedStr != "" {
		if t, err := time.Parse(time.RFC3339, *completedStr); err == nil {
			c.LastCompletedAt = &t
		}
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

	return &c, nil
}
