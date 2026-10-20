// Package sqlite provides SQLite implementations of domain repository ports.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/history"
	"github.com/google/uuid"
)

var _ history.StatisticsRepository = (*StatisticsRepository)(nil)

// StatisticsRepository handles persistence for long-term pre-aggregated statistics.
type StatisticsRepository struct {
	db     *sql.DB
	readDB *sql.DB
}

// NewStatisticsRepository creates a new SQLite StatisticsRepository with optional separate read pool.
func NewStatisticsRepository(db *sql.DB, readDB ...*sql.DB) *StatisticsRepository {
	r := &StatisticsRepository{db: db, readDB: db}
	if len(readDB) > 0 && readDB[0] != nil {
		r.readDB = readDB[0]
	}
	return r
}

// InsertOrUpdateBatch writes a batch of aggregated statistics records using upsert logic.
func (r *StatisticsRepository) InsertOrUpdateBatch(ctx context.Context, stats []*history.StatisticsRecord) error {
	if len(stats) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: failed to begin stats tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO long_term_statistics (
			id, entity_id, period, period_start, period_end,
			mean, min, max, sum, count, unit, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_id, period, period_start) DO UPDATE SET
			period_end = excluded.period_end,
			mean = excluded.mean,
			min = excluded.min,
			max = excluded.max,
			sum = excluded.sum,
			count = excluded.count,
			unit = excluded.unit
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("sqlite: failed to prepare stats upsert: %w", err)
	}
	defer stmt.Close()

	for _, s := range stats {
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now().UTC()
		}

		_, err := stmt.ExecContext(ctx,
			s.ID, s.EntityID, s.Period,
			s.PeriodStart.Format(time.RFC3339), s.PeriodEnd.Format(time.RFC3339),
			s.Mean, s.Min, s.Max, s.Sum, s.Count, s.Unit,
			s.CreatedAt.Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("sqlite: failed to exec stats upsert: %w", err)
		}
	}

	return tx.Commit()
}

// GetStatistics queries long-term aggregated metrics matching the filter criteria.
func (r *StatisticsRepository) GetStatistics(ctx context.Context, filter history.StatisticsFilter) ([]*history.StatisticsRecord, error) {
	var (
		where []string
		args  []any
	)

	if len(filter.EntityIDs) > 0 {
		placeholders := make([]string, len(filter.EntityIDs))
		for i, id := range filter.EntityIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		where = append(where, fmt.Sprintf("entity_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.Period != "" {
		where = append(where, "period = ?")
		args = append(args, filter.Period)
	}

	if !filter.From.IsZero() {
		where = append(where, "period_start >= ?")
		args = append(args, filter.From.UTC().Format(time.RFC3339))
	}

	if !filter.To.IsZero() {
		where = append(where, "period_end <= ?")
		args = append(args, filter.To.UTC().Format(time.RFC3339))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, entity_id, period, period_start, period_end,
		       mean, min, max, sum, count, unit, created_at
		FROM long_term_statistics
		%s
		ORDER BY period_start ASC
	`, whereClause)

	rows, err := r.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query statistics: %w", err)
	}
	defer rows.Close()

	var results []*history.StatisticsRecord
	for rows.Next() {
		var (
			s          history.StatisticsRecord
			startStr   string
			endStr     string
			createdStr string
			meanNull   sql.NullFloat64
			minNull    sql.NullFloat64
			maxNull    sql.NullFloat64
			sumNull    sql.NullFloat64
			unitNull   sql.NullString
		)

		err := rows.Scan(
			&s.ID, &s.EntityID, &s.Period, &startStr, &endStr,
			&meanNull, &minNull, &maxNull, &sumNull, &s.Count, &unitNull, &createdStr,
		)
		if err != nil {
			return nil, fmt.Errorf("sqlite: failed to scan statistics record: %w", err)
		}

		s.PeriodStart, _ = time.Parse(time.RFC3339, startStr)
		s.PeriodEnd, _ = time.Parse(time.RFC3339, endStr)
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		if meanNull.Valid {
			val := meanNull.Float64
			s.Mean = &val
		}
		if minNull.Valid {
			val := minNull.Float64
			s.Min = &val
		}
		if maxNull.Valid {
			val := maxNull.Float64
			s.Max = &val
		}
		if sumNull.Valid {
			val := sumNull.Float64
			s.Sum = &val
		}
		s.Unit = unitNull.String

		results = append(results, &s)
	}
	return results, rows.Err()
}
