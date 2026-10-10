// Package handlers provides HTTP controllers for the Domus REST API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/application/mesh"
	"github.com/go-chi/chi/v5"
)

type BindingHandler struct {
	svc *mesh.BindingService
}

func NewBindingHandler(svc *mesh.BindingService) *BindingHandler {
	return &BindingHandler{svc: svc}
}

func (h *BindingHandler) RegisterRoutes(r chi.Router) {
	r.Route("/devices/bindings", func(r chi.Router) {
		r.Get("/", h.ListBindings)
		r.Post("/", h.CreateBinding)
		r.Delete("/{id}", h.RemoveBinding)
	})
}

func (h *BindingHandler) ListBindings(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListBindings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

type createBindingRequest struct {
	SourceDeviceID   string   `json:"source_device_id"`
	SourceDeviceName string   `json:"source_device_name"`
	TargetDeviceID   string   `json:"target_device_id"`
	TargetDeviceName string   `json:"target_device_name"`
	Clusters         []string `json:"clusters"`
}

func (h *BindingHandler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	var req createBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	b, err := h.svc.CreateBinding(r.Context(), req.SourceDeviceID, req.SourceDeviceName, req.TargetDeviceID, req.TargetDeviceName, req.Clusters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(b)
}

func (h *BindingHandler) RemoveBinding(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if err := h.svc.RemoveBinding(r.Context(), id); err != nil {
		if err == mesh.ErrBindingNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
