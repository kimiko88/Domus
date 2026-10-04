// Package sqlite provides history repository implementations for SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/history"
)

// HistoryRepository implements history.Repository for SQLite.
type HistoryRepository struct {
	db *sql.DB
}

// NewHistoryRepository constructs a HistoryRepository.
func NewHistoryRepository(db *sql.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// SaveBatch inserts a slice of StateHistory entries in a single transaction.
func (r *HistoryRepository) SaveBatch(ctx context.Context, entries []*history.StateHistory) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: failed to begin history batch tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO state_history (entity_id, state, recorded_at)
		VALUES (?, ?, ?);
	`)
	if err != nil {
		return fmt.Errorf("sqlite: failed to prepare history insert: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		recordedAt := entry.RecordedAt
		if recordedAt.IsZero() {
			recordedAt = time.Now().UTC()
		}
		stateStr := entry.State
		if stateStr == "" {
			stateStr = "null"
		}

		res, err := stmt.ExecContext(ctx, entry.EntityID, stateStr, recordedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("sqlite: failed to insert history row for %s: %w", entry.EntityID, err)
		}
		if id, err := res.LastInsertId(); err == nil {
			entry.ID = id
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: failed to commit history batch: %w", err)
	}

	return nil
}

// ListByEntity retrieves historical state snapshots matching the provided filter.
func (r *HistoryRepository) ListByEntity(ctx context.Context, filter history.HistoryFilter) ([]*history.StateHistory, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, entity_id, state, recorded_at
		FROM state_history
		WHERE entity_id = ?
	`
	args := []any{filter.EntityID}

	if !filter.From.IsZero() {
		query += " AND recorded_at >= ?"
		args = append(args, filter.From.UTC().Format(time.RFC3339))
	}
	if !filter.To.IsZero() {
		query += " AND recorded_at <= ?"
		args = append(args, filter.To.UTC().Format(time.RFC3339))
	}

	query += " ORDER BY recorded_at ASC, id ASC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query state history: %w", err)
	}
	defer rows.Close()

	var results []*history.StateHistory
	for rows.Next() {
		var (
			h          history.StateHistory
			recordedAt string
		)
		if err := rows.Scan(&h.ID, &h.EntityID, &h.State, &recordedAt); err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan state history row: %w", err)
		}
		if parsed, err := time.Parse(time.RFC3339, recordedAt); err == nil {
			h.RecordedAt = parsed
		}
		results = append(results, &h)
	}

	return results, rows.Err()
}

// PurgeOlderThan deletes all state history records prior to the cutoff timestamp.
func (r *HistoryRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	query := `DELETE FROM state_history WHERE recorded_at < ?;`
	res, err := r.db.ExecContext(ctx, query, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("sqlite: failed to purge state history: %w", err)
	}
	return res.RowsAffected()
}
