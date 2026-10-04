// Package sqlite provides audit log repository implementations for SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/audit"
)

// AuditRepository implements audit.Repository for SQLite.
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository constructs an AuditRepository.
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Record inserts a new audit log entry.
func (r *AuditRepository) Record(ctx context.Context, entry *audit.Entry) error {
	query := `
		INSERT INTO audit_log (actor_type, actor_id, action, target_type, target_id, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	var actorID, targetType, targetID *string
	if entry.ActorID != "" {
		actorID = &entry.ActorID
	}
	if entry.TargetType != "" {
		targetType = &entry.TargetType
	}
	if entry.TargetID != "" {
		targetID = &entry.TargetID
	}
	payload := entry.Payload
	if payload == "" {
		payload = "{}"
	}

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	res, err := r.db.ExecContext(ctx, query,
		string(entry.ActorType),
		actorID,
		entry.Action,
		targetType,
		targetID,
		payload,
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to record audit entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		entry.ID = id
	}

	return nil
}

// List returns the latest audit entries up to the specified limit.
func (r *AuditRepository) List(ctx context.Context, limit int) ([]*audit.Entry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, actor_type, actor_id, action, target_type, target_id, payload, created_at
		FROM audit_log
		ORDER BY id DESC
		LIMIT ?;
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*audit.Entry
	for rows.Next() {
		var (
			e          audit.Entry
			actorType  string
			actorID    sql.NullString
			targetType sql.NullString
			targetID   sql.NullString
			createdAt  string
		)

		err := rows.Scan(
			&e.ID,
			&actorType,
			&actorID,
			&e.Action,
			&targetType,
			&targetID,
			&e.Payload,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("sqlite: scan audit entry: %w", err)
		}

		e.ActorType = audit.ActorType(actorType)
		if actorID.Valid {
			e.ActorID = actorID.String
		}
		if targetType.Valid {
			e.TargetType = targetType.String
		}
		if targetID.Valid {
			e.TargetID = targetID.String
		}
		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			e.CreatedAt = parsed
		}

		entries = append(entries, &e)
	}

	return entries, rows.Err()
}

// PurgeOlderThan deletes all audit log entries prior to the cutoff timestamp.
func (r *AuditRepository) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	query := `DELETE FROM audit_log WHERE created_at < ?;`
	res, err := r.db.ExecContext(ctx, query, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("sqlite: failed to purge audit log: %w", err)
	}
	return res.RowsAffected()
}
