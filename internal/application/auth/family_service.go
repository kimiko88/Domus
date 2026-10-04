// Package auth provides family member management and invitation lifecycle logic.
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
	"github.com/google/uuid"
)

var (
	ErrCannotDemoteLastAdmin = errors.New("auth: cannot demote the only remaining administrator")
	ErrCannotRemoveLastAdmin = errors.New("auth: cannot remove the only remaining administrator")
	ErrCannotRemoveSelf      = errors.New("auth: cannot remove own account")
	ErrUsernameTaken         = errors.New("auth: username is already taken")
	ErrInviteRepoNotSet      = errors.New("auth: invite repository not configured")
)

// CreateFamilyInvite generates a new invitation token for a designated role.
func (s *Service) CreateFamilyInvite(ctx context.Context, creatorID string, role user.Role, ttl time.Duration) (*user.FamilyInvite, error) {
	if s.inviteRepo == nil {
		return nil, ErrInviteRepoNotSet
	}
	if !role.Valid() {
		return nil, user.ErrInvalidRole
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour // default 7 days
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("auth: failed to generate invite token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now().UTC()
	invite := &user.FamilyInvite{
		ID:              uuid.NewString(),
		Token:           token,
		Role:            role,
		CreatedByUserID: creatorID,
		ExpiresAt:       now.Add(ttl),
		CreatedAt:       now,
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("auth: failed to store invite: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    creatorID,
		Action:     "family.invite_created",
		TargetType: "family_invite",
		TargetID:   invite.ID,
		Payload:    fmt.Sprintf(`{"role":%q,"expires_at":%q}`, string(role), invite.ExpiresAt.Format(time.RFC3339)),
		CreatedAt:  now,
	})

	return invite, nil
}

// GetInviteByToken retrieves invitation details for verification prior to registration.
func (s *Service) GetInviteByToken(ctx context.Context, token string) (*user.FamilyInvite, error) {
	if s.inviteRepo == nil {
		return nil, ErrInviteRepoNotSet
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return nil, user.ErrInviteNotFound
	}
	return s.inviteRepo.GetByToken(ctx, trimmed)
}

// ListFamilyInvites retrieves all pending active invitations.
func (s *Service) ListFamilyInvites(ctx context.Context) ([]*user.FamilyInvite, error) {
	if s.inviteRepo == nil {
		return nil, ErrInviteRepoNotSet
	}
	return s.inviteRepo.ListActive(ctx)
}

// RevokeFamilyInvite removes an active invitation.
func (s *Service) RevokeFamilyInvite(ctx context.Context, id string) error {
	if s.inviteRepo == nil {
		return ErrInviteRepoNotSet
	}
	if err := s.inviteRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("auth: failed to delete invite: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorSystem,
		Action:     "family.invite_revoked",
		TargetType: "family_invite",
		TargetID:   id,
		Payload:    "{}",
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

// RegisterWithInvite registers a new family member account using a valid invitation token.
func (s *Service) RegisterWithInvite(
	ctx context.Context,
	token, username, password, email, ip, userAgent string,
) (*user.User, *user.Session, error) {
	if s.inviteRepo == nil {
		return nil, nil, ErrInviteRepoNotSet
	}

	invite, err := s.inviteRepo.GetByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, nil, err
	}

	if invite.IsUsed() {
		return nil, nil, user.ErrInviteUsed
	}
	if invite.IsExpired() {
		return nil, nil, user.ErrInviteExpired
	}

	trimmedUser := strings.TrimSpace(username)
	if len(trimmedUser) < 3 || len(trimmedUser) > 64 {
		return nil, nil, user.ErrInvalidUsername
	}
	if len(password) < 8 {
		return nil, nil, user.ErrInvalidPassword
	}

	// Verify username availability
	existing, err := s.userRepo.GetByUsername(ctx, trimmedUser)
	if err == nil && existing != nil {
		return nil, nil, ErrUsernameTaken
	}

	hash, err := s.hasher.HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: hash password: %w", err)
	}

	now := time.Now().UTC()
	newUser := &user.User{
		ID:           uuid.NewString(),
		Username:     trimmedUser,
		Email:        strings.TrimSpace(email),
		PasswordHash: hash,
		Role:         invite.Role, // Enforce role decided by family creator
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, nil, fmt.Errorf("auth: create user: %w", err)
	}

	// Mark invite as used
	if err := s.inviteRepo.MarkUsed(ctx, invite.ID, newUser.ID, now); err != nil {
		return nil, nil, fmt.Errorf("auth: consume invite: %w", err)
	}

	// Create active session immediately
	sessionTokenBytes := make([]byte, 32)
	if _, err := rand.Read(sessionTokenBytes); err != nil {
		return nil, nil, fmt.Errorf("auth: generate session: %w", err)
	}
	sessionID := hex.EncodeToString(sessionTokenBytes)

	session := &user.Session{
		ID:        sessionID,
		UserID:    newUser.ID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
		IPAddress: ip,
		UserAgent: userAgent,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("auth: create session: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    newUser.ID,
		Action:     "family.member_joined",
		TargetType: "user",
		TargetID:   newUser.ID,
		Payload:    fmt.Sprintf(`{"username":%q,"role":%q,"invite_id":%q}`, newUser.Username, string(newUser.Role), invite.ID),
		CreatedAt:  now,
	})

	return newUser, session, nil
}

// ListFamilyMembers returns all registered users belonging to the family.
func (s *Service) ListFamilyMembers(ctx context.Context) ([]*user.User, error) {
	return s.userRepo.List(ctx)
}

// UpdateFamilyMemberRole updates a member's authorization role.
func (s *Service) UpdateFamilyMemberRole(ctx context.Context, adminID, targetUserID string, newRole user.Role) error {
	if !newRole.Valid() {
		return user.ErrInvalidRole
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	// If demoting an admin, ensure at least one other admin remains
	if target.Role == user.RoleAdmin && newRole != user.RoleAdmin {
		allUsers, err := s.userRepo.List(ctx)
		if err != nil {
			return err
		}
		adminCount := 0
		for _, u := range allUsers {
			if u.Role == user.RoleAdmin && u.Enabled {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrCannotDemoteLastAdmin
		}
	}

	if err := s.userRepo.UpdateRole(ctx, targetUserID, newRole); err != nil {
		return fmt.Errorf("auth: update role: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    adminID,
		Action:     "family.role_updated",
		TargetType: "user",
		TargetID:   targetUserID,
		Payload:    fmt.Sprintf(`{"old_role":%q,"new_role":%q}`, string(target.Role), string(newRole)),
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

// RemoveFamilyMember deletes a family member account and revokes active sessions.
func (s *Service) RemoveFamilyMember(ctx context.Context, adminID, targetUserID string) error {
	if adminID == targetUserID {
		return ErrCannotRemoveSelf
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	if target.Role == user.RoleAdmin {
		allUsers, err := s.userRepo.List(ctx)
		if err != nil {
			return err
		}
		adminCount := 0
		for _, u := range allUsers {
			if u.Role == user.RoleAdmin && u.Enabled {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	// Revoke sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, targetUserID)

	// Remove user
	if err := s.userRepo.Delete(ctx, targetUserID); err != nil {
		return fmt.Errorf("auth: delete user: %w", err)
	}

	_ = s.auditRepo.Record(ctx, &audit.Entry{
		ActorType:  audit.ActorLocalUser,
		ActorID:    adminID,
		Action:     "family.member_removed",
		TargetType: "user",
		TargetID:   targetUserID,
		Payload:    fmt.Sprintf(`{"username":%q,"role":%q}`, target.Username, string(target.Role)),
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}
