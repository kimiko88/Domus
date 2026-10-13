// Package handlers contains HTTP route handlers for Domus.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
)

// AuthHandler handles authentication and onboarding HTTP requests.
type AuthHandler struct {
	authService  *auth.Service
	cookieSecure bool
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authService *auth.Service, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieSecure: cookieSecure,
	}
}

type setupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Setup handles initial administrator account creation on first boot.
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	admin, err := h.authService.SetupInitialAdmin(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrSetupAlreadyCompleted) {
			h.writeError(w, http.StatusConflict, "conflict", "setup has already been completed")
			return
		}
		if errors.Is(err, user.ErrInvalidUsername) || errors.Is(err, user.ErrInvalidPassword) {
			h.writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to complete setup")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":         admin.ID,
		"username":   admin.Username,
		"role":       admin.Role,
		"created_at": admin.CreatedAt,
	})
}

// SetupStatus returns whether initial onboarding setup is still required.
func (h *AuthHandler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	required, err := h.authService.IsSetupRequired(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_error", "failed to check setup status")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"setup_required": required,
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login authenticates a user and attaches an HttpOnly session cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	ip := extractClientIP(r)
	ua := r.UserAgent()

	sess, u, err := h.authService.Login(r.Context(), req.Username, req.Password, ip, ua)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
			return
		}
		if errors.Is(err, user.ErrUserDisabled) {
			h.writeError(w, http.StatusForbidden, "user_disabled", "account is disabled")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	// Set session cookie
	maxAge := int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})

	// Set CSRF token cookie readable by client JavaScript
	csrfToken := generateRandomToken()
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})

	h.writeJSON(w, http.StatusOK, map[string]any{
		"token": sess.ID,
		"user": map[string]any{
			"id":       u.ID,
			"username": u.Username,
			"role":     u.Role,
			"email":    u.Email,
		},
		"csrf_token": csrfToken,
	})
}

type kioskLoginRequest struct {
	Pin string `json:"pin"`
}

// KioskLogin allows quick authentication with a numeric PIN for wall-mounted tablet kiosk dashboards.
func (h *AuthHandler) KioskLogin(w http.ResponseWriter, r *http.Request) {
	var req kioskLoginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	ip := extractClientIP(r)
	ua := r.UserAgent()

	sess, u, err := h.authService.LoginWithPin(r.Context(), req.Pin, ip, ua)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.writeError(w, http.StatusUnauthorized, "invalid_credentials", "PIN non valido")
			return
		}
		if errors.Is(err, user.ErrGuestExpired) {
			h.writeError(w, http.StatusForbidden, "guest_expired", "account ospite scaduto")
			return
		}
		if errors.Is(err, user.ErrUserDisabled) {
			h.writeError(w, http.StatusForbidden, "user_disabled", "account disabilitato")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error", "login fallito")
		return
	}

	maxAge := int(sess.ExpiresAt.Sub(sess.CreatedAt).Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})

	csrfToken := generateRandomToken()
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})

	h.writeJSON(w, http.StatusOK, map[string]any{
		"token": sess.ID,
		"user": map[string]any{
			"id":         u.ID,
			"username":   u.Username,
			"role":       u.Role,
			"is_guest":   u.IsGuest,
			"kiosk_mode": u.KioskMode,
		},
		"csrf_token": csrfToken,
	})
}

// Logout terminates the current session and clears the cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if sess := domusmw.GetSession(r.Context()); sess != nil {
		_ = h.authService.Logout(r.Context(), sess.ID)
	} else if cookie, err := r.Cookie(domusmw.SessionCookieName); err == nil && cookie.Value != "" {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	// Clear CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.CSRFCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Me returns the identity of the currently authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"role":       u.Role,
		"email":      u.Email,
		"created_at": u.CreatedAt,
	})
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, data)
}

func (h *AuthHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	writeError(w, status, code, message)
}

// extractClientIP is a package-level helper that extracts client IP from the request.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		candidate := strings.TrimSpace(parts[0])
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func generateRandomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
