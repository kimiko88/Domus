// Package audit defines the audit repository interface (port).
package audit

import "context"

// Repository is the domain port for recording and querying audit logs.
type Repository interface {
	Record(ctx context.Context, entry *Entry) error
	List(ctx context.Context, limit int) ([]*Entry, error)
}
