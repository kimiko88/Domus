// Package user defines the repository interfaces (ports) for users and sessions.
package user

import (
	"context"
	"time"
)

// UserRepository is the domain port for user storage operations.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Count(ctx context.Context) (int, error)
}

// SessionRepository is the domain port for session storage operations.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByID(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}
