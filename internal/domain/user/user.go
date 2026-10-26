// Package user defines the user, role, session domain entities and validation rules.
package user

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Role defines the authorization level of a user.
type Role string

const (
	// RoleAdmin has full permissions including system configuration and user management.
	RoleAdmin Role = "admin"
	// RoleOperator can control devices, execute actions, and activate scenes.
	RoleOperator Role = "operator"
	// RoleViewer can only inspect entities and read telemetry.
	RoleViewer Role = "viewer"
	// RoleGuest has restricted area access and temporary validity.
	RoleGuest Role = "guest"
	// RoleKid has child-safe permissions (only own room entities, family chores, media).
	RoleKid Role = "kid"
)

// Valid returns true if the role is one of the recognized role types.
func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer, RoleGuest, RoleKid:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidUsername = errors.New("user: username must be between 3 and 64 alphanumeric characters")
	ErrInvalidPassword = errors.New("user: password must be at least 8 characters long")
	ErrInvalidRole     = errors.New("user: invalid role specified")
	ErrUserDisabled    = errors.New("user: user account is disabled")
	ErrUserNotFound    = errors.New("user: user not found")
	ErrSessionExpired  = errors.New("user: session has expired")
	ErrSessionNotFound = errors.New("user: session not found")
	ErrGuestExpired    = errors.New("user: guest account has expired")
)

// User represents an authenticated identity within Domus.
type User struct {
	ID               string     `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email,omitempty"`
	PasswordHash     string     `json:"-"`
	Role             Role       `json:"role"`
	Enabled          bool       `json:"enabled"`
	IsGuest          bool       `json:"is_guest"`
	GuestExpiresAt   *time.Time `json:"guest_expires_at,omitempty"`
	PinCode          string     `json:"pin_code,omitempty"`
	KioskMode        bool       `json:"kiosk_mode"`
	AllowedViews     []string   `json:"allowed_views,omitempty"`
	AllowedAreaIDs   []string   `json:"allowed_area_ids,omitempty"`
	AllowedEntityIDs []string   `json:"allowed_entity_ids,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// IsGuestExpired returns true if the user is a guest whose expiration timestamp has passed.
func (u *User) IsGuestExpired() bool {
	if !u.IsGuest || u.GuestExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.GuestExpiresAt)
}

// Validate checks the user's properties for domain compliance.
func (u *User) Validate() error {
	trimmedUser := strings.TrimSpace(u.Username)
	if len(trimmedUser) < 3 || len(trimmedUser) > 64 {
		return ErrInvalidUsername
	}
	if !u.Role.Valid() {
		return ErrInvalidRole
	}
	if u.PasswordHash == "" {
		return fmt.Errorf("user: password hash cannot be empty")
	}
	return nil
}

// Session represents an active authenticated session.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

// IsExpired returns true if the session lifetime has elapsed.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
