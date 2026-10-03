// Package sqlite provides repository container and database helpers.
package sqlite

import (
	"database/sql"

	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/user"
)

// Repositories aggregates all initialized SQLite repositories.
type Repositories struct {
	Users    user.UserRepository
	Sessions user.SessionRepository
	Audit    audit.Repository
}

// NewRepositories initializes and returns all SQLite repositories.
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Users:    NewUserRepository(db),
		Sessions: NewSessionRepository(db),
		Audit:    NewAuditRepository(db),
	}
}
