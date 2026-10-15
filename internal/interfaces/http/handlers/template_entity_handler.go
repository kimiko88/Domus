// Package handlers provides HTTP request handlers for template entities.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appte "github.com/domus-hub/domus/internal/application/template_entity"
	domte "github.com/domus-hub/domus/internal/domain/template_entity"
	"github.com/go-chi/chi/v5"
)

// TemplateEntityHandler handles HTTP endpoints for template entities.
type TemplateEntityHandler struct {
	service *appte.Service
	repo    domte.Repository
}

// NewTemplateEntityHandler constructs a TemplateEntityHandler.
func NewTemplateEntityHandler(service *appte.Service, repo domte.Repository) *TemplateEntityHandler {
	return &TemplateEntityHandler{
		service: service,
		repo:    repo,
	}
}

// List handles GET /api/v1/template-entities.
func (h *TemplateEntityHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"template_entities": templates,
		"total":             len(templates),
	})
}

// Get handles GET /api/v1/template-entities/{id}.
func (h *TemplateEntityHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	te, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domte.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "template entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, te)
}

// Create handles POST /api/v1/template-entities.
func (h *TemplateEntityHandler) Create(w http.ResponseWriter, r *http.Request) {
	var te domte.TemplateEntity
	if err := json.NewDecoder(r.Body).Decode(&te); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	if err := h.service.CreateTemplateEntity(r.Context(), &te); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, te)
}

// Update handles PUT /api/v1/template-entities/{id}.
func (h *TemplateEntityHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domte.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "template entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var req domte.TemplateEntity
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	existing.Name = req.Name
	existing.StateTemplate = req.StateTemplate
	existing.AttributesTemplate = req.AttributesTemplate
	existing.UnitOfMeasurement = req.UnitOfMeasurement
	existing.DeviceClass = req.DeviceClass
	existing.Icon = req.Icon
	if len(req.SourceEntityIDs) > 0 {
		existing.SourceEntityIDs = req.SourceEntityIDs
	}

	if err := h.service.UpdateTemplateEntity(r.Context(), existing); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/template-entities/{id}.
func (h *TemplateEntityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteTemplateEntity(r.Context(), id); err != nil {
		if errors.Is(err, domte.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "template entity not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PreviewRequest body for expression simulation.
type PreviewRequest struct {
	Template   string            `json:"template"`
	MockStates map[string]string `json:"mock_states,omitempty"`
}

// Preview handles POST /api/v1/template-entities/preview.
func (h *TemplateEntityHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
		return
	}

	result, err := h.service.Preview(r.Context(), req.Template, req.MockStates)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PREVIEW_ERROR", err.Error())
		return
	}

	extractedSources := appte.ExtractSourceEntityIDs(req.Template)
	writeJSON(w, http.StatusOK, map[string]any{
		"evaluated_state": result,
		"source_entities": extractedSources,
	})
}
