// Package user defines the repository interfaces (ports) for users, sessions, and family invites.
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
	List(ctx context.Context) ([]*User, error)
	UpdateRole(ctx context.Context, id string, role Role) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// SessionRepository is the domain port for session storage operations.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByID(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

// InviteRepository is the domain port for family invitation storage operations.
type InviteRepository interface {
	Create(ctx context.Context, inv *FamilyInvite) error
	GetByToken(ctx context.Context, token string) (*FamilyInvite, error)
	GetByID(ctx context.Context, id string) (*FamilyInvite, error)
	ListActive(ctx context.Context) ([]*FamilyInvite, error)
	MarkUsed(ctx context.Context, id string, usedByUserID string, usedAt time.Time) error
	Delete(ctx context.Context, id string) error
}
