// Package handlers provides HTTP controllers for the Domus REST API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/application/alarm"
	domainAlarm "github.com/domus-hub/domus/internal/domain/alarm"
	"github.com/go-chi/chi/v5"
)

type AlarmHandler struct {
	svc *alarm.Service
}

func NewAlarmHandler(svc *alarm.Service) *AlarmHandler {
	return &AlarmHandler{svc: svc}
}

func (h *AlarmHandler) RegisterRoutes(r chi.Router) {
	r.Route("/alarm", func(r chi.Router) {
		r.Get("/", h.GetPanel)
		r.Post("/arm", h.Arm)
		r.Post("/disarm", h.Disarm)
		r.Post("/trigger", h.Trigger)
		r.Put("/config", h.UpdateConfig)
	})
}

func (h *AlarmHandler) GetPanel(w http.ResponseWriter, r *http.Request) {
	panel, err := h.svc.GetPanel(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(panel)
}

type armRequest struct {
	Mode string `json:"mode"`
	Pin  string `json:"pin"`
}

func (h *AlarmHandler) Arm(w http.ResponseWriter, r *http.Request) {
	var req armRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user := "user"
	panel, err := h.svc.Arm(r.Context(), domainAlarm.State(req.Mode), req.Pin, user)
	if err != nil {
		if err == domainAlarm.ErrInvalidPin {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err == domainAlarm.ErrInvalidMode {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(panel)
}

type disarmRequest struct {
	Pin string `json:"pin"`
}

func (h *AlarmHandler) Disarm(w http.ResponseWriter, r *http.Request) {
	var req disarmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user := "user"
	panel, err := h.svc.Disarm(r.Context(), req.Pin, user)
	if err != nil {
		if err == domainAlarm.ErrInvalidPin {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(panel)
}

type triggerRequest struct {
	Reason string `json:"reason"`
}

func (h *AlarmHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req triggerRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "Pulsante SOS Panico manuale"
	}

	panel, err := h.svc.Trigger(r.Context(), req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(panel)
}

type alarmConfigRequest struct {
	EntryDelaySeconds int      `json:"entry_delay_seconds"`
	ExitDelaySeconds  int      `json:"exit_delay_seconds"`
	PerimeterSensors  []string `json:"perimeter_sensors"`
	InteriorSensors   []string `json:"interior_sensors"`
	SirenEntities     []string `json:"siren_entities"`
	Pin               string   `json:"pin"`
}

func (h *AlarmHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req alarmConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	panel, err := h.svc.UpdateConfig(r.Context(), req.EntryDelaySeconds, req.ExitDelaySeconds, req.PerimeterSensors, req.InteriorSensors, req.SirenEntities, req.Pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(panel)
}
