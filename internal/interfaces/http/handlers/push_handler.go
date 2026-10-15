// Package handlers provides HTTP route handlers for companion push notifications.
package handlers

import (
	"encoding/json"
	"net/http"

	appnotification "github.com/domus-hub/domus/internal/application/notification"
	domainnotification "github.com/domus-hub/domus/internal/domain/notification"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
)

// PushHandler handles companion device push registrations and push dispatches.
type PushHandler struct {
	service *appnotification.PushService
}

// NewPushHandler creates a new PushHandler.
func NewPushHandler(service *appnotification.PushService) *PushHandler {
	return &PushHandler{service: service}
}

// RegisterToken handles POST /api/v1/notifications/push/tokens.
func (h *PushHandler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	var req domainnotification.DevicePushToken
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if req.UserID == "" {
		if u := middleware.GetUser(r.Context()); u != nil {
			req.UserID = u.ID
		}
	}

	if err := h.service.RegisterToken(r.Context(), &req); err != nil {
		writeError(w, http.StatusBadRequest, "REGISTRATION_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":       "registered",
		"device_token": req.DeviceToken,
		"platform":     req.Platform,
	})
}

// RevokeToken handles DELETE /api/v1/notifications/push/tokens.
func (h *PushHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			token = body.Token
		}
	}

	if token == "" {
		writeError(w, http.StatusBadRequest, "MISSING_TOKEN", "token parameter is required")
		return
	}

	if err := h.service.RevokeToken(r.Context(), token); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "revoked",
		"token":  token,
	})
}

// ListTokens handles GET /api/v1/notifications/push/tokens.
func (h *PushHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		if u := middleware.GetUser(r.Context()); u != nil {
			userID = u.ID
		}
	}

	all := r.URL.Query().Get("all") == "true"
	var (
		tokens []domainnotification.DevicePushToken
		err    error
	)

	if all || userID == "" {
		tokens, err = h.service.GetAllTokens(r.Context())
	} else {
		tokens, err = h.service.GetTokensByUser(r.Context(), userID)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if tokens == nil {
		tokens = []domainnotification.DevicePushToken{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"total":  len(tokens),
	})
}

// Send handles POST /api/v1/notifications/push/send.
func (h *PushHandler) Send(w http.ResponseWriter, r *http.Request) {
	var msg domainnotification.PushMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if msg.Title == "" && msg.Body == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "title or body must be provided")
		return
	}

	report, err := h.service.Send(r.Context(), msg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DISPATCH_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}
