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
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	domauth "github.com/domus-hub/domus/internal/domain/auth"
	"github.com/domus-hub/domus/internal/domain/user"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/go-chi/chi/v5"
)

// AuthHandler handles authentication, 2FA, brute-force defense, and onboarding HTTP requests.
type AuthHandler struct {
	authService   *auth.Service
	cookieSecure  bool
	twoFactorRepo domauth.TwoFactorRepository
	bannedRepo    domauth.BannedIPRepository
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authService *auth.Service, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieSecure: cookieSecure,
	}
}

// WithSecurity attaches 2FA and brute force IP banning repositories.
func (h *AuthHandler) WithSecurity(twoFactorRepo domauth.TwoFactorRepository, bannedRepo domauth.BannedIPRepository) *AuthHandler {
	h.twoFactorRepo = twoFactorRepo
	h.bannedRepo = bannedRepo
	return h
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
	Username      string `json:"username"`
	Password      string `json:"password"`
	TwoFactorCode string `json:"two_factor_code,omitempty"`
}

// Login authenticates a user, verifies optional 2FA TOTP, and attaches an HttpOnly session cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := extractClientIP(r)

	// 1. Check IP Ban
	if h.bannedRepo != nil {
		banned, banErr := h.bannedRepo.Get(r.Context(), ip)
		if banErr != nil {
			h.writeError(w, http.StatusServiceUnavailable, "BRUTE_FORCE_PROTECTION_UNAVAILABLE", "impossibile verificare lo stato di protezione accessi")
			return
		}
		if banned != nil && banned.IsBanned() {
			h.writeError(w, http.StatusForbidden, "IP_BANNED", "Indirizzo IP bloccato per troppi tentativi falliti")
			return
		}
	}

	var req loginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	ua := r.UserAgent()

	sess, u, err := h.authService.Login(r.Context(), req.Username, req.Password, ip, ua)
	if err != nil {
		if h.bannedRepo != nil {
			if _, recordErr := h.bannedRepo.RecordFailedAttempt(r.Context(), ip, 5, 15*time.Minute); recordErr != nil {
				h.writeError(w, http.StatusServiceUnavailable, "BRUTE_FORCE_PROTECTION_UNAVAILABLE", "impossibile registrare il tentativo di accesso")
				return
			}
		}
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

	// 2. Check Two-Factor Authentication. Repository failures must never
	// silently downgrade an account to password-only authentication.
	if h.twoFactorRepo != nil {
		twoFactorCfg, cfgErr := h.twoFactorRepo.Get(r.Context(), u.ID)
		if cfgErr != nil && !errors.Is(cfgErr, domauth.ErrTwoFactorNotFound) {
			_ = h.authService.Logout(r.Context(), sess.ID)
			h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare la configurazione 2FA")
			return
		}
		if cfgErr == nil && twoFactorCfg != nil && twoFactorCfg.IsEnabled {
			if req.TwoFactorCode == "" {
				_ = h.authService.Logout(r.Context(), sess.ID)
				h.writeJSON(w, http.StatusAccepted, map[string]any{"requires_2fa": true, "message": "Codice di verifica 2FA richiesto"})
				return
			}
			validSecondFactor := auth.VerifyTOTP(twoFactorCfg.Secret, req.TwoFactorCode, time.Now().UTC())
			if !validSecondFactor {
				consumed, consumeErr := h.twoFactorRepo.ConsumeBackupCode(r.Context(), u.ID, req.TwoFactorCode)
				if consumeErr != nil {
					_ = h.authService.Logout(r.Context(), sess.ID)
					h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare il codice di recupero")
					return
				}
				validSecondFactor = consumed
			}
			if !validSecondFactor {
				_ = h.authService.Logout(r.Context(), sess.ID)
				if h.bannedRepo != nil {
					if _, recordErr := h.bannedRepo.RecordFailedAttempt(r.Context(), ip, 5, 15*time.Minute); recordErr != nil {
						h.writeError(w, http.StatusServiceUnavailable, "BRUTE_FORCE_PROTECTION_UNAVAILABLE", "impossibile registrare il tentativo di accesso")
						return
					}
				}
				h.writeError(w, http.StatusUnauthorized, "INVALID_2FA_CODE", "Codice di verifica 2FA non valido")
				return
			}
		}
	}

	// Reset failed attempts before issuing the session cookie. If persistence
	// fails, revoke the just-created session rather than returning a login that
	// the global IP-ban middleware may immediately reject.
	if h.bannedRepo != nil {
		if resetErr := h.bannedRepo.ResetAttempts(r.Context(), ip); resetErr != nil {
			_ = h.authService.Logout(r.Context(), sess.ID)
			h.writeError(w, http.StatusServiceUnavailable, "BRUTE_FORCE_PROTECTION_UNAVAILABLE", "impossibile azzerare i tentativi di accesso")
			return
		}
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
	})
}

// KioskLogin handles quick PIN authentication for wall tablets and dashboard displays.
func (h *AuthHandler) KioskLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON payload")
		return
	}

	if req.PIN == "" {
		h.writeError(w, http.StatusBadRequest, "bad_request", "PIN is required")
		return
	}

	ip := extractClientIP(r)
	ua := r.UserAgent()

	sess, u, err := h.authService.KioskLogin(r.Context(), req.PIN, ip, ua)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid_pin", "invalid kiosk PIN")
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
	http.SetCookie(w, &http.Cookie{
		Name: domusmw.CSRFCookieName, Value: generateRandomToken(), Path: "/",
		HttpOnly: false, Secure: h.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge,
	})

	h.writeJSON(w, http.StatusOK, map[string]any{
		"token": sess.ID,
		"user": map[string]any{
			"id":         u.ID,
			"username":   u.Username,
			"role":       u.Role,
			"kiosk_mode": true,
		},
	})
}

// Logout terminates the current user session and clears authentication cookies.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess := domusmw.GetSession(r.Context())
	var logoutErr error
	if sess != nil {
		logoutErr = h.authService.Logout(r.Context(), sess.ID)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     domusmw.CSRFCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	if logoutErr != nil {
		h.writeError(w, http.StatusInternalServerError, "LOGOUT_FAILED", "impossibile revocare la sessione sul server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the identity of the currently authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	twoFactorActive := false
	if h.twoFactorRepo != nil {
		cfg, err := h.twoFactorRepo.Get(r.Context(), u.ID)
		if err != nil && !errors.Is(err, domauth.ErrTwoFactorNotFound) {
			h.writeError(w, http.StatusServiceUnavailable, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare lo stato della 2FA")
			return
		}
		if cfg != nil && cfg.IsEnabled {
			twoFactorActive = true
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"role":       u.Role,
		"email":      u.Email,
		"two_factor": twoFactorActive,
		"created_at": u.CreatedAt,
	})
}

// Setup2FA handles POST /api/v1/auth/2fa/setup.
func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if h.twoFactorRepo == nil {
		h.writeError(w, http.StatusServiceUnavailable, "TWO_FACTOR_UNAVAILABLE", "servizio 2FA non disponibile")
		return
	}
	if existing, err := h.twoFactorRepo.Get(r.Context(), u.ID); err == nil && existing != nil && existing.IsEnabled {
		h.writeError(w, http.StatusConflict, "TWO_FACTOR_ALREADY_ENABLED", "disattivare la 2FA con password e secondo fattore prima di configurarla nuovamente")
		return
	} else if err != nil && !errors.Is(err, domauth.ErrTwoFactorNotFound) {
		h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare la configurazione 2FA")
		return
	}

	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate 2fa secret")
		return
	}

	backupCodes := auth.GenerateBackupCodes(8)
	qrURL := auth.BuildQRCodeURL(u.Username, secret, "Domus")

	// Store pending configuration
	cfg := &domauth.TwoFactorConfig{
		UserID:      u.ID,
		Secret:      secret,
		IsEnabled:   false,
		BackupCodes: backupCodes,
	}
	if err := h.twoFactorRepo.Save(r.Context(), cfg); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save 2fa configuration")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"secret":       secret,
		"qr_url":       qrURL,
		"backup_codes": backupCodes,
	})
}

// VerifyAndEnable2FA handles POST /api/v1/auth/2fa/verify-and-enable.
func (h *AuthHandler) VerifyAndEnable2FA(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if h.twoFactorRepo == nil {
		h.writeError(w, http.StatusServiceUnavailable, "TWO_FACTOR_UNAVAILABLE", "servizio 2FA non disponibile")
		return
	}

	var req struct {
		Passcode string `json:"passcode"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	cfg, err := h.twoFactorRepo.Get(r.Context(), u.ID)
	if errors.Is(err, domauth.ErrTwoFactorNotFound) {
		h.writeError(w, http.StatusBadRequest, "2FA_NOT_INITIALIZED", "run 2fa setup first")
		return
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile leggere la configurazione 2FA")
		return
	}

	if !auth.VerifyTOTP(cfg.Secret, req.Passcode, time.Now().UTC()) {
		h.writeError(w, http.StatusBadRequest, "INVALID_PASSCODE", "codice non valido o scaduto")
		return
	}

	cfg.IsEnabled = true
	if err := h.twoFactorRepo.Save(r.Context(), cfg); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enable 2fa")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Autenticazione a due fattori attivata con successo",
	})
}

// Disable2FA handles POST /api/v1/auth/2fa/disable. It requires the
// current password and, when enabled, a fresh TOTP or one-time recovery code.
func (h *AuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if h.twoFactorRepo == nil {
		h.writeError(w, http.StatusServiceUnavailable, "TWO_FACTOR_UNAVAILABLE", "servizio 2FA non disponibile")
		return
	}
	var req struct {
		Password string `json:"password"`
		TwoFactorCode string `json:"two_factor_code"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	validPassword, err := h.authService.VerifyPassword(r.Context(), u.ID, req.Password)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "REAUTHENTICATION_FAILED", "impossibile verificare la password")
		return
	}
	if !validPassword {
		h.writeError(w, http.StatusUnauthorized, "INVALID_PASSWORD", "password attuale non valida")
		return
	}
	cfg, cfgErr := h.twoFactorRepo.Get(r.Context(), u.ID)
	if cfgErr != nil && !errors.Is(cfgErr, domauth.ErrTwoFactorNotFound) {
		h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare la configurazione 2FA")
		return
	}
	if cfgErr == nil && cfg != nil && cfg.IsEnabled {
		validSecondFactor := auth.VerifyTOTP(cfg.Secret, req.TwoFactorCode, time.Now().UTC())
		if !validSecondFactor {
			consumed, consumeErr := h.twoFactorRepo.ConsumeBackupCode(r.Context(), u.ID, req.TwoFactorCode)
			if consumeErr != nil {
				h.writeError(w, http.StatusInternalServerError, "TWO_FACTOR_UNAVAILABLE", "impossibile verificare il codice di recupero")
				return
			}
			validSecondFactor = consumed
		}
		if !validSecondFactor {
			h.writeError(w, http.StatusUnauthorized, "INVALID_2FA_CODE", "codice 2FA attuale non valido")
			return
		}
	}
	if err := h.twoFactorRepo.Delete(r.Context(), u.ID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "impossibile disattivare la 2FA")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status":"success", "message":"Autenticazione a due fattori disattivata"})
}

// ListBannedIPs handles GET /api/v1/security/banned-ips.
func (h *AuthHandler) ListBannedIPs(w http.ResponseWriter, r *http.Request) {
	if h.bannedRepo == nil {
		h.writeJSON(w, http.StatusOK, []any{})
		return
	}
	banned, err := h.bannedRepo.ListBanned(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, banned)
}

// UnbanIP handles DELETE /api/v1/security/banned-ips/{ip}.
func (h *AuthHandler) UnbanIP(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")
	if h.bannedRepo == nil {
		h.writeError(w, http.StatusServiceUnavailable, "BRUTE_FORCE_PROTECTION_UNAVAILABLE", "servizio di protezione accessi non disponibile")
		return
	}
	if err := h.bannedRepo.Unban(r.Context(), ip); err != nil {
		h.writeError(w, http.StatusInternalServerError, "UNBAN_FAILED", "impossibile rimuovere il blocco IP")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, data)
}

func (h *AuthHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	writeError(w, status, code, message)
}

// extractClientIP returns the verified client IP from r.RemoteAddr.
// X-Forwarded-For / X-Real-IP headers are intentionally ignored to prevent
// IP-spoofing (CVE-2026-114437). The chi RealIP middleware (when present)
// is the single authorised place to rewrite RemoteAddr from a trusted proxy.
func extractClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func generateRandomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
