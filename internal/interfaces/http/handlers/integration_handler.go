// Package handlers provides HTTP route handlers for Integrations.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/domus-hub/domus/internal/application/brand"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/go-chi/chi/v5"
)

// IntegrationHandler handles HTTP operations for Integration entities and Brand device auto-discovery.
type IntegrationHandler struct {
	repo         integration.Repository
	bus          events.Bus
	brandService *brand.Service
}

// NewIntegrationHandler constructs an IntegrationHandler.
func NewIntegrationHandler(repo integration.Repository, bus events.Bus) *IntegrationHandler {
	return &IntegrationHandler{
		repo: repo,
		bus:  bus,
	}
}

// WithBrandService attaches the smart home brand auto-discovery service.
func (h *IntegrationHandler) WithBrandService(svc *brand.Service) *IntegrationHandler {
	h.brandService = svc
	return h
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

// GetCatalog returns the full smart home brand integration catalog.
func (h *IntegrationHandler) GetCatalog(w http.ResponseWriter, r *http.Request) {
	if h.brandService != nil {
		writeJSON(w, http.StatusOK, h.brandService.GetCatalog())
		return
	}
	writeJSON(w, http.StatusOK, brand.DefaultCatalog())
}

// ListDiscovered returns devices discovered on the LAN ready for 1-click pairing.
func (h *IntegrationHandler) ListDiscovered(w http.ResponseWriter, r *http.Request) {
	if h.brandService == nil {
		writeJSON(w, http.StatusOK, []*brand.DiscoveredDevice{})
		return
	}
	devices := h.brandService.ListDiscovered()
	if devices == nil {
		devices = []*brand.DiscoveredDevice{}
	}
	writeJSON(w, http.StatusOK, devices)
}

type scanRequest struct {
	Target string `json:"target"`
}

// Scan triggers a network scan or specific IP probe for smart home devices.
func (h *IntegrationHandler) Scan(w http.ResponseWriter, r *http.Request) {
	if h.brandService == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "brand service not configured")
		return
	}

	var req scanRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	devices, err := h.brandService.Scan(r.Context(), req.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, "scan_failed", err.Error())
		return
	}

	if devices == nil {
		devices = []*brand.DiscoveredDevice{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"devices": devices,
	})
}

// Pair executes 1-click provisioning of a brand device into Domus.
func (h *IntegrationHandler) Pair(w http.ResponseWriter, r *http.Request) {
	if h.brandService == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "brand service not configured")
		return
	}

	var req brand.PairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid pair request payload")
		return
	}

	dev, entities, err := h.brandService.PairDevice(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pairing_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":   "paired",
		"device":   dev,
		"entities": entities,
	})
}
