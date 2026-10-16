package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	appCloud "github.com/domus-hub/domus/internal/application/cloud_assistant"
	domainCloud "github.com/domus-hub/domus/internal/domain/cloud_assistant"
	"github.com/go-chi/chi/v5"
)

// CloudAssistantHandler handles cloud voice assistant fulfillment endpoints for Alexa and Google Home.
type CloudAssistantHandler struct {
	svc *appCloud.Service
}

// NewCloudAssistantHandler constructs a new CloudAssistantHandler.
func NewCloudAssistantHandler(svc *appCloud.Service) *CloudAssistantHandler {
	return &CloudAssistantHandler{svc: svc}
}

// RegisterRoutes mounts Alexa and Google fulfillment routes.
func (h *CloudAssistantHandler) RegisterRoutes(r chi.Router) {
	r.Route("/cloud", func(c chi.Router) {
		c.Post("/alexa", h.Alexa)
		c.Post("/google", h.Google)
	})
}

// Alexa handles POST /api/v1/cloud/alexa.
func (h *CloudAssistantHandler) Alexa(w http.ResponseWriter, r *http.Request) {
	if !h.svc.ValidateToken(r.Header.Get("Authorization")) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing bearer token")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "cannot read request body")
		return
	}

	resp, err := h.svc.HandleAlexa(r.Context(), body)
	if err != nil {
		if err == appCloud.ErrCloudDisabled {
			writeError(w, http.StatusForbidden, "DISABLED", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Google handles POST /api/v1/cloud/google.
func (h *CloudAssistantHandler) Google(w http.ResponseWriter, r *http.Request) {
	if !h.svc.ValidateToken(r.Header.Get("Authorization")) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or missing bearer token")
		return
	}

	var req domainCloud.GoogleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed google request")
		return
	}

	resp, err := h.svc.HandleGoogle(r.Context(), &req)
	if err != nil {
		if err == appCloud.ErrCloudDisabled {
			writeError(w, http.StatusForbidden, "DISABLED", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
