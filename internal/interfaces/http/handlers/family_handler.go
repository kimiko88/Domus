// Package handlers provides HTTP transport handlers for family management and invitations.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
)

// FamilyHandler manages family member permissions and registration invitations.
type FamilyHandler struct {
	authService  *auth.Service
	cookieSecure bool
}

// NewFamilyHandler constructs a new FamilyHandler.
func NewFamilyHandler(authService *auth.Service, cookieSecure bool) *FamilyHandler {
	return &FamilyHandler{
		authService:  authService,
		cookieSecure: cookieSecure,
	}
}

type createInviteRequest struct {
	Role           string `json:"role"`
	ExpiresInHours int    `json:"expires_in_hours"`
}

// CreateInvite handles POST /api/v1/family/invites.
func (h *FamilyHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "failed to parse request body")
		return
	}

	role := user.Role(req.Role)
	if !role.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_role", "invalid family member role")
		return
	}

	ttl := 7 * 24 * time.Hour
	if req.ExpiresInHours > 0 {
		ttl = time.Duration(req.ExpiresInHours) * time.Hour
	}

	invite, err := h.authService.CreateFamilyInvite(r.Context(), u.ID, role, ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         invite.ID,
		"token":      invite.Token,
		"role":       string(invite.Role),
		"expires_at": invite.ExpiresAt.Format(time.RFC3339),
		"invite_url": "/register?token=" + invite.Token,
	})
}

// ListInvites handles GET /api/v1/family/invites.
func (h *FamilyHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	invites, err := h.authService.ListFamilyInvites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	if invites == nil {
		invites = []*user.FamilyInvite{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"invites": invites,
		"total":   len(invites),
	})
}

// RevokeInvite handles DELETE /api/v1/family/invites/{id}.
func (h *FamilyHandler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "invite ID is required")
		return
	}

	if err := h.authService.RevokeFamilyInvite(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ValidateInvite handles GET /api/v1/auth/invite/{token}.
func (h *FamilyHandler) ValidateInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "missing_token", "invite token is required")
		return
	}

	invite, err := h.authService.GetInviteByToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusNotFound, "invite_not_found", "invitation not found or expired")
		return
	}

	if invite.IsUsed() {
		writeError(w, http.StatusBadRequest, "invite_used", "invitation has already been accepted")
		return
	}
	if invite.IsExpired() {
		writeError(w, http.StatusBadRequest, "invite_expired", "invitation validity period has elapsed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":      true,
		"role":       string(invite.Role),
		"expires_at": invite.ExpiresAt.Format(time.RFC3339),
	})
}

type registerWithInviteRequest struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// RegisterWithInvite handles POST /api/v1/auth/register.
func (h *FamilyHandler) RegisterWithInvite(w http.ResponseWriter, r *http.Request) {
	var req registerWithInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "failed to parse request body")
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	newUser, sess, err := h.authService.RegisterWithInvite(
		r.Context(),
		req.Token,
		req.Username,
		req.Password,
		req.Email,
		ip,
		userAgent,
	)
	if err != nil {
		if errors.Is(err, user.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "invite_not_found", "invitation not found")
			return
		}
		if errors.Is(err, user.ErrInviteUsed) {
			writeError(w, http.StatusBadRequest, "invite_used", "invitation has already been used")
			return
		}
		if errors.Is(err, user.ErrInviteExpired) {
			writeError(w, http.StatusBadRequest, "invite_expired", "invitation has expired")
			return
		}
		if errors.Is(err, auth.ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "username_taken", "username is already registered")
			return
		}
		if errors.Is(err, user.ErrInvalidUsername) || errors.Is(err, user.ErrInvalidPassword) {
			writeError(w, http.StatusBadRequest, "invalid_credentials", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Issue the session and matching double-submit CSRF cookies so the new
	// account can immediately use authenticated browser API requests.
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.CSRFCookieName,
		Value:    generateRandomToken(),
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":         newUser.ID,
			"username":   newUser.Username,
			"role":       string(newUser.Role),
			"email":      newUser.Email,
			"created_at": newUser.CreatedAt.Format(time.RFC3339),
		},
	})
}

// ListMembers handles GET /api/v1/family/members.
func (h *FamilyHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	members, err := h.authService.ListFamilyMembers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	result := make([]map[string]any, 0, len(members))
	for _, m := range members {
		result = append(result, map[string]any{
			"id":         m.ID,
			"username":   m.Username,
			"email":      m.Email,
			"role":       string(m.Role),
			"enabled":    m.Enabled,
			"created_at": m.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"members": result,
		"total":   len(result),
	})
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

// UpdateMemberRole handles PUT /api/v1/family/members/{id}/role.
func (h *FamilyHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	admin := domusmw.GetUser(r.Context())
	if admin == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "member ID is required")
		return
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "failed to parse request body")
		return
	}

	newRole := user.Role(req.Role)
	if !newRole.Valid() {
		writeError(w, http.StatusBadRequest, "invalid_role", "invalid role specified")
		return
	}

	if err := h.authService.UpdateFamilyMemberRole(r.Context(), admin.ID, targetID, newRole); err != nil {
		if errors.Is(err, auth.ErrCannotDemoteLastAdmin) {
			writeError(w, http.StatusBadRequest, "cannot_demote_last_admin", err.Error())
			return
		}
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user_not_found", "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// RemoveMember handles DELETE /api/v1/family/members/{id}.
func (h *FamilyHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	admin := domusmw.GetUser(r.Context())
	if admin == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "member ID is required")
		return
	}

	if err := h.authService.RemoveFamilyMember(r.Context(), admin.ID, targetID); err != nil {
		if errors.Is(err, auth.ErrCannotRemoveSelf) {
			writeError(w, http.StatusBadRequest, "cannot_remove_self", err.Error())
			return
		}
		if errors.Is(err, auth.ErrCannotRemoveLastAdmin) {
			writeError(w, http.StatusBadRequest, "cannot_remove_last_admin", err.Error())
			return
		}
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user_not_found", "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type createGuestRequest struct {
	Username   string `json:"username"`
	Pin        string `json:"pin"`
	ValidHours int    `json:"valid_hours"`
	KioskMode  bool   `json:"kiosk_mode"`
}

// CreateGuest handles POST /api/v1/family/guests.
func (h *FamilyHandler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	var req createGuestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "failed to parse request body")
		return
	}

	guest, err := h.authService.CreateGuest(r.Context(), req.Username, req.Pin, req.ValidHours, req.KioskMode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, guest)
}

// ListGuests handles GET /api/v1/family/guests.
func (h *FamilyHandler) ListGuests(w http.ResponseWriter, r *http.Request) {
	members, err := h.authService.ListFamilyMembers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	var guests []*user.User
	for _, m := range members {
		if m.IsGuest || m.Role == user.RoleGuest {
			guests = append(guests, m)
		}
	}
	if guests == nil {
		guests = []*user.User{}
	}

	writeJSON(w, http.StatusOK, guests)
}
