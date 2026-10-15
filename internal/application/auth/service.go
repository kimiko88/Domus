// Package auth provides authentication application logic.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/audit"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/google/uuid"
)

var (
	ErrSetupAlreadyCompleted = errors.New("auth: initial setup has already been completed")
	ErrInvalidCredentials    = errors.New("auth: invalid username or password")
)

// Service coordinates authentication, session lifecycle, and audit logs.
type Service struct {
	userRepo    user.UserRepository
	sessionRepo user.SessionRepository
	inviteRepo  user.InviteRepository
	auditRepo   audit.Repository
	hasher      *crypto.Argon2Hasher
	sessionTTL  time.Duration
}

// NewService constructs an AuthService.
func NewService(
	userRepo user.UserRepository,
	sessionRepo user.SessionRepository,
	auditRepo audit.Repository,
	hasher *crypto.Argon2Hasher,
	sessionTTL time.Duration,
) *Service {
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour
	}
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		hasher:      hasher,
		sessionTTL:  sessionTTL,
	}
}

// WithInvites configures the invite repository on the service.
func (s *Service) WithInvites(inviteRepo user.InviteRepository) *Service {
	s.inviteRepo = inviteRepo
	return s
}

// IsSetupRequired returns true if no users exist in the system yet.
func (s *Service) IsSetupRequired(ctx context.Context) (bool, error) {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("auth service: check user count: %w", err)
	}
	return count == 0, nil
}

// SetupInitialAdmin provisions the first administrator user during onboarding.
func (s *Service) SetupInitialAdmin(ctx context.Context, username, password string) (*user.User, error) {
	required, err := s.IsSetupRequired(ctx)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrSetupAlreadyCompleted
	}

	trimmedUser := strings.TrimSpace(username)
	if len(trimmedUser) < 3 || len(trimmedUser) > 64 {
		return nil, user.ErrInvalidUsername
	}
	if len(password) < 8 {
		return nil, user.ErrInvalidPassword
	}

	hash, err := s.hasher.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("auth service: failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	admin := &user.User{
		ID:           uuid.NewString(),
		Username:     trimmedUser,
		PasswordHash: hash,
		Role:         user.RoleAdmin,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		return nil, fmt.Errorf("auth service: failed to create admin: %w", err)
	}

	// Record audit event
	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorSystem,
		Action:     "auth.setup.admin_created",
		TargetType: "user",
		TargetID:   admin.ID,
		Payload:    fmt.Sprintf(`{"username":%q}`, admin.Username),
		CreatedAt:  now,
	})

	return admin, nil
}

// Login authenticates credentials, establishes a session, and writes audit records.
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (*user.Session, *user.User, error) {
	u, err := s.userRepo.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		// Dummy verify to mitigate timing attacks against non-existent users
		_, _ = s.hasher.VerifyPassword("dummy_password", "$argon2id$v=19$m=32768,t=1,p=1$c2FsdHNhbHRzYWx0$aGFzaGhhc2hoYXNo")
		_ = s.recordLoginAudit(ctx, "auth.login.failed", "", username, ip, userAgent, "user not found")
		return nil, nil, ErrInvalidCredentials
	}

	if !u.Enabled {
		_ = s.recordLoginAudit(ctx, "auth.login.failed", u.ID, username, ip, userAgent, "user disabled")
		return nil, nil, user.ErrUserDisabled
	}

	match, err := s.hasher.VerifyPassword(password, u.PasswordHash)
	if err != nil || !match {
		_ = s.recordLoginAudit(ctx, "auth.login.failed", u.ID, username, ip, userAgent, "password mismatch")
		return nil, nil, ErrInvalidCredentials
	}

	// Create random session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, nil, fmt.Errorf("auth service: failed to generate session token: %w", err)
	}
	sessionID := hex.EncodeToString(tokenBytes)

	now := time.Now().UTC()
	sess := &user.Session{
		ID:        sessionID,
		UserID:    u.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
		IPAddress: ip,
		UserAgent: userAgent,
	}

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, nil, fmt.Errorf("auth service: failed to persist session: %w", err)
	}

	_ = s.recordLoginAudit(ctx, "auth.login.success", u.ID, username, ip, userAgent, "")
	return sess, u, nil
}

// Logout terminates an active session and records the action.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	sess, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil // idempotent logout
	}

	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("auth service: failed to delete session: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    sess.UserID,
		Action:     "auth.logout",
		TargetType: "session",
		TargetID:   sessionID,
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

// ValidateSession verifies if a session exists, is not expired, and user is active.
func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*user.User, *user.Session, error) {
	if sessionID == "" {
		return nil, nil, user.ErrSessionNotFound
	}

	sess, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	if sess.IsExpired() {
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, nil, user.ErrSessionExpired
	}

	u, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}

	if !u.Enabled {
		return nil, nil, user.ErrUserDisabled
	}

	if u.IsGuestExpired() {
		_ = s.sessionRepo.Delete(ctx, sessionID)
		return nil, nil, user.ErrGuestExpired
	}

	return u, sess, nil
}

// LoginWithPin authenticates a guest or user quickly with a numeric PIN for wall-mounted tablet kiosks.
func (s *Service) LoginWithPin(ctx context.Context, pin, ip, userAgent string) (*user.Session, *user.User, error) {
	cleanPin := strings.TrimSpace(pin)
	if cleanPin == "" {
		return nil, nil, ErrInvalidCredentials
	}

	u, err := s.userRepo.GetByPinCode(ctx, cleanPin)
	if err != nil {
		_ = s.recordLoginAudit(ctx, "auth.kiosk_login.failed", "", "kiosk_pin", ip, userAgent, "invalid PIN")
		return nil, nil, ErrInvalidCredentials
	}

	if !u.Enabled {
		_ = s.recordLoginAudit(ctx, "auth.kiosk_login.failed", u.ID, u.Username, ip, userAgent, "user disabled")
		return nil, nil, user.ErrUserDisabled
	}

	if u.IsGuestExpired() {
		_ = s.recordLoginAudit(ctx, "auth.kiosk_login.failed", u.ID, u.Username, ip, userAgent, "guest expired")
		return nil, nil, user.ErrGuestExpired
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, nil, fmt.Errorf("auth service: failed to generate session token: %w", err)
	}
	sessionID := hex.EncodeToString(tokenBytes)

	now := time.Now().UTC()
	sess := &user.Session{
		ID:        sessionID,
		UserID:    u.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
		IPAddress: ip,
		UserAgent: userAgent,
	}

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, nil, fmt.Errorf("auth service: failed to persist session: %w", err)
	}

	_ = s.recordLoginAudit(ctx, "auth.kiosk_login.success", u.ID, u.Username, ip, userAgent, "PIN unlocked")
	return sess, u, nil
}

// KioskLogin is an alias for LoginWithPin to support wall tablets and kiosk PIN authentication.
func (s *Service) KioskLogin(ctx context.Context, pin, ip, userAgent string) (*user.Session, *user.User, error) {
	return s.LoginWithPin(ctx, pin, ip, userAgent)
}

// CreateGuest creates a guest user with PIN, optional expiration, and kiosk mode.
func (s *Service) CreateGuest(ctx context.Context, username, pin string, validHours int, kioskMode bool) (*user.User, error) {
	trimmedUser := strings.TrimSpace(username)
	if trimmedUser == "" {
		return nil, fmt.Errorf("auth: guest username cannot be empty")
	}

	now := time.Now().UTC()
	var expiresAt *time.Time
	if validHours > 0 {
		exp := now.Add(time.Duration(validHours) * time.Hour)
		expiresAt = &exp
	}

	dummyHash, _ := s.hasher.HashPassword("guest_" + uuid.NewString())

	u := &user.User{
		ID:             uuid.NewString(),
		Username:       trimmedUser,
		PasswordHash:   dummyHash,
		Role:           user.RoleGuest,
		Enabled:        true,
		IsGuest:        true,
		GuestExpiresAt: expiresAt,
		PinCode:        strings.TrimSpace(pin),
		KioskMode:      kioskMode,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) recordLoginAudit(ctx context.Context, action, userID, username, ip, ua, reason string) error {
	payload := fmt.Sprintf(`{"username":%q,"ip":%q,"user_agent":%q`, username, ip, ua)
	if reason != "" {
		payload += fmt.Sprintf(`,"reason":%q}`, reason)
	} else {
		payload += "}"
	}

	actorType := audit.ActorLocalUser
	if userID == "" {
		actorType = audit.ActorSystem
	}

	return s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  actorType,
		ActorID:    userID,
		Action:     action,
		TargetType: "auth",
		Payload:    payload,
		CreatedAt:  time.Now().UTC(),
	})
}
