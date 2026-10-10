// Package handlers provides HTTP controllers for the Domus REST API.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/application/assistant"
	"github.com/go-chi/chi/v5"
)

type AssistantHandler struct {
	svc *assistant.Service
}

func NewAssistantHandler(svc *assistant.Service) *AssistantHandler {
	return &AssistantHandler{svc: svc}
}

func (h *AssistantHandler) RegisterRoutes(r chi.Router) {
	r.Route("/assistant", func(r chi.Router) {
		r.Post("/chat", h.Chat)
		r.Get("/config", h.GetConfig)
		r.Put("/config", h.UpdateConfig)
		r.Get("/satellites", h.ListSatellites)
		r.Post("/satellites", h.RegisterSatellite)
		r.Post("/satellites/{id}/stream", h.StreamVoice)
	})
}

type chatRequest struct {
	Prompt string `json:"prompt"`
}

func (h *AssistantHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.ProcessPrompt(r.Context(), req.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AssistantHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.svc.GetConfig()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

func (h *AssistantHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var cfg assistant.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.svc.UpdateConfig(cfg)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.svc.GetConfig())
}

func (h *AssistantHandler) ListSatellites(w http.ResponseWriter, r *http.Request) {
	list := h.svc.ListSatellites()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *AssistantHandler) RegisterSatellite(w http.ResponseWriter, r *http.Request) {
	var sat assistant.VoiceSatellite
	if err := json.NewDecoder(r.Body).Decode(&sat); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	registered := h.svc.RegisterSatellite(sat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(registered)
}

type satelliteStreamRequest struct {
	Prompt string `json:"prompt"`
}

func (h *AssistantHandler) StreamVoice(w http.ResponseWriter, r *http.Request) {
	satID := chi.URLParam(r, "id")
	var req satelliteStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.ProcessSatelliteVoice(r.Context(), satID, req.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
