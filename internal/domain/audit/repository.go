// Package audit defines the audit repository interface (port).
package audit

import (
	"context"
	"time"
)

// Repository is the domain port for recording and querying audit logs.
type Repository interface {
	Record(ctx context.Context, entry *Entry) error
	List(ctx context.Context, limit int) ([]*Entry, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
