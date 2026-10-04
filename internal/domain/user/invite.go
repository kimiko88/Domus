// Package user provides domain entities and validation for users, roles, sessions, and family invites.
package user

import (
	"errors"
	"time"
)

var (
	ErrInviteNotFound = errors.New("user: invite not found")
	ErrInviteExpired  = errors.New("user: invite has expired")
	ErrInviteUsed     = errors.New("user: invite has already been used")
	ErrInvalidInvite  = errors.New("user: invalid invite details")
)

// FamilyInvite represents an invitation link allowing a new member to join the family.
type FamilyInvite struct {
	ID              string     `json:"id"`
	Token           string     `json:"token"`
	Role            Role       `json:"role"`
	CreatedByUserID string     `json:"created_by_user_id"`
	ExpiresAt       time.Time  `json:"expires_at"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	UsedByUserID    *string    `json:"used_by_user_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// IsExpired returns true if the invitation validity period has elapsed.
func (inv *FamilyInvite) IsExpired() bool {
	return time.Now().After(inv.ExpiresAt)
}

// IsUsed returns true if the invitation has already been accepted.
func (inv *FamilyInvite) IsUsed() bool {
	return inv.UsedAt != nil
}

// Validate checks the invite entity properties.
func (inv *FamilyInvite) Validate() error {
	if inv.ID == "" || inv.Token == "" {
		return ErrInvalidInvite
	}
	if !inv.Role.Valid() {
		return ErrInvalidRole
	}
	return nil
}
