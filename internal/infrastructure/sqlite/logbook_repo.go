// Package sqlite provides SQLite persistence for logbook entries.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/logbook"
	"github.com/google/uuid"
)

// LogbookRepository implements logbook.Repository using SQLite.
type LogbookRepository struct {
	db *sql.DB
}

// NewLogbookRepository constructs a new LogbookRepository.
func NewLogbookRepository(db *sql.DB) *LogbookRepository {
	return &LogbookRepository{db: db}
}

// Save inserts a new entry into logbook_entries.
func (r *LogbookRepository) Save(ctx context.Context, entry *logbook.Entry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	query := `
		INSERT INTO logbook_entries (
			id, entity_id, entity_name, domain, state, message, context_type, context_id, context_name, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		entry.ID,
		entry.EntityID,
		entry.EntityName,
		entry.Domain,
		entry.State,
		entry.Message,
		entry.ContextType,
		entry.ContextID,
		entry.ContextName,
		entry.Timestamp.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to insert logbook entry: %w", err)
	}
	return nil
}

// List returns filtered logbook entries in reverse chronological order.
func (r *LogbookRepository) List(ctx context.Context, filter logbook.Filter) ([]*logbook.Entry, error) {
	query := `
		SELECT id, entity_id, entity_name, domain, state, message, context_type, context_id, context_name, timestamp
		FROM logbook_entries
		WHERE 1=1
	`
	var args []any

	if filter.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, filter.EntityID)
	}
	if filter.Domain != "" {
		query += " AND domain = ?"
		args = append(args, filter.Domain)
	}
	if !filter.From.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.From.Format(time.RFC3339))
	}
	if !filter.To.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.To.Format(time.RFC3339))
	}

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logbook entries: %w", err)
	}
	defer rows.Close()

	var entries []*logbook.Entry
	for rows.Next() {
		var (
			e            logbook.Entry
			timestampStr string
		)
		err := rows.Scan(
			&e.ID,
			&e.EntityID,
			&e.EntityName,
			&e.Domain,
			&e.State,
			&e.Message,
			&e.ContextType,
			&e.ContextID,
			&e.ContextName,
			&timestampStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan logbook entry: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			e.Timestamp = t
		}
		entries = append(entries, &e)
	}

	return entries, nil
}

// PurgeOlderThan removes logbook entries older than the cutoff timestamp.
func (r *LogbookRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM logbook_entries WHERE timestamp < ?", cutoff.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to purge old logbook entries: %w", err)
	}
	return res.RowsAffected()
}
