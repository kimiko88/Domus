// Package handlers provides HTTP request handlers for spatial zones and person tracking.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appzone "github.com/domus-hub/domus/internal/application/zone"
	domzone "github.com/domus-hub/domus/internal/domain/zone"
	"github.com/go-chi/chi/v5"
)

// ZoneHandler handles HTTP endpoints for zones and tracked persons.
type ZoneHandler struct {
	service *appzone.Service
	repo    domzone.Repository
}

// NewZoneHandler constructs a ZoneHandler.
func NewZoneHandler(service *appzone.Service, repo domzone.Repository) *ZoneHandler {
	return &ZoneHandler{
		service: service,
		repo:    repo,
	}
}

// ListZones handles GET /api/v1/zones.
func (h *ZoneHandler) ListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.repo.ListZones(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"zones": zones,
		"total": len(zones),
	})
}

// GetZone handles GET /api/v1/zones/{id}.
func (h *ZoneHandler) GetZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	z, err := h.repo.GetZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, domzone.ErrZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "zone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, z)
}

// CreateZone handles POST /api/v1/zones.
func (h *ZoneHandler) CreateZone(w http.ResponseWriter, r *http.Request) {
	var z domzone.Zone
	if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	if err := z.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	if err := h.repo.CreateZone(r.Context(), &z); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, z)
}

// UpdateZone handles PUT /api/v1/zones/{id}.
func (h *ZoneHandler) UpdateZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	z, err := h.repo.GetZone(r.Context(), id)
	if err != nil {
		if errors.Is(err, domzone.ErrZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "zone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req domzone.Zone
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	z.Name = req.Name
	z.Latitude = req.Latitude
	z.Longitude = req.Longitude
	z.Radius = req.Radius
	z.Icon = req.Icon
	z.Passive = req.Passive

	if err := h.repo.UpdateZone(r.Context(), z); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, z)
}

// DeleteZone handles DELETE /api/v1/zones/{id}.
func (h *ZoneHandler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteZone(r.Context(), id); err != nil {
		if errors.Is(err, domzone.ErrZoneNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "zone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListPersons handles GET /api/v1/persons.
func (h *ZoneHandler) ListPersons(w http.ResponseWriter, r *http.Request) {
	persons, err := h.repo.ListPersons(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"persons": persons,
		"total":   len(persons),
	})
}

// GetPerson handles GET /api/v1/persons/{id}.
func (h *ZoneHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.repo.GetPerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, domzone.ErrPersonNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "person not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// CreatePerson handles POST /api/v1/persons.
func (h *ZoneHandler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	var p domzone.Person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}
	if p.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name is required")
		return
	}
	if err := h.repo.CreatePerson(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// UpdatePerson handles PUT /api/v1/persons/{id}.
func (h *ZoneHandler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.repo.GetPerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, domzone.ErrPersonNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "person not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req domzone.Person
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	p.Name = req.Name
	p.UserID = req.UserID
	p.DeviceTrackers = req.DeviceTrackers

	if err := h.repo.UpdatePerson(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// DeletePerson handles DELETE /api/v1/persons/{id}.
func (h *ZoneHandler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.DeletePerson(r.Context(), id); err != nil {
		if errors.Is(err, domzone.ErrPersonNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "person not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// LocationUpdateRequest payload.
type LocationUpdateRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// UpdatePersonLocation handles POST /api/v1/persons/{id}/location.
func (h *ZoneHandler) UpdatePersonLocation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	updated, err := h.service.UpdatePersonLocation(r.Context(), id, req.Latitude, req.Longitude)
	if err != nil {
		if errors.Is(err, domzone.ErrPersonNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "person not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
