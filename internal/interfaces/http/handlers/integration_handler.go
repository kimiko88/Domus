// Package handlers provides HTTP route handlers for Integrations.
package handlers

import (
	"errors"
	"net/http"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/go-chi/chi/v5"
)

// IntegrationHandler handles HTTP operations for Integration entities.
type IntegrationHandler struct {
	repo integration.Repository
	bus  events.Bus
}

// NewIntegrationHandler constructs an IntegrationHandler.
func NewIntegrationHandler(repo integration.Repository, bus events.Bus) *IntegrationHandler {
	return &IntegrationHandler{
		repo: repo,
		bus:  bus,
	}
}

// List returns all registered integrations.
func (h *IntegrationHandler) List(w http.ResponseWriter, r *http.Request) {
	integrations, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list integrations")
		return
	}

	if integrations == nil {
		integrations = []*integration.Integration{}
	}

	writeJSON(w, http.StatusOK, integrations)
}

// GetByID returns an integration by its ID.
func (h *IntegrationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	in, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, integration.ErrIntegrationNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "integration not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get integration")
		return
	}

	writeJSON(w, http.StatusOK, in)
}

// Reconnect initiates a reconnection attempt for an integration.
func (h *IntegrationHandler) Reconnect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	in, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, integration.ErrIntegrationNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "integration not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get integration")
		return
	}

	if h.bus != nil {
		_ = h.bus.Publish(r.Context(), events.Event{
			Type: events.IntegrationStateChanged,
			Payload: map[string]any{
				"id":     in.ID,
				"action": "reconnect",
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "reconnecting",
		"message": "reconnection initiated",
		"id":      in.ID,
	})
}
