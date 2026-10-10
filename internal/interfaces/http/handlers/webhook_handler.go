// Package handlers provides HTTP controllers for the Domus REST API.
package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/domus-hub/domus/internal/application/webhook"
	domainWebhook "github.com/domus-hub/domus/internal/domain/webhook"
	"github.com/go-chi/chi/v5"
)

type WebhookHandler struct {
	svc *webhook.Service
}

func NewWebhookHandler(svc *webhook.Service) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) RegisterRoutes(r chi.Router) {
	// Inbound public receiver
	r.HandleFunc("/webhooks/{webhook_id}", h.HandleInbound)

	// Admin management endpoints
	r.Route("/webhook-configs", func(r chi.Router) {
		r.Get("/", h.ListWebhooks)
		r.Post("/", h.CreateWebhook)
		r.Delete("/{id}", h.DeleteWebhook)
	})
}

func (h *WebhookHandler) HandleInbound(w http.ResponseWriter, r *http.Request) {
	webhookID := chi.URLParam(r, "webhook_id")
	if webhookID == "" {
		http.Error(w, "missing webhook_id", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Collect query params
	queryParams := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	wh, err := h.svc.ExecuteWebhook(r.Context(), webhookID, r.Method, r.Header, queryParams, body)
	if err != nil {
		if err == domainWebhook.ErrWebhookNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err == domainWebhook.ErrWebhookDisabled {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if err == domainWebhook.ErrMethodNotAllowed {
			http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			return
		}
		if err == domainWebhook.ErrInvalidSignature {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "ok",
		"webhook_id":    wh.WebhookID,
		"trigger_count": wh.TriggerCount,
	})
}

func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListWebhooks(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

type createWebhookRequest struct {
	Name           string `json:"name"`
	SecretToken    string `json:"secret_token"`
	AllowedMethods string `json:"allowed_methods"`
}

func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	wh, err := h.svc.CreateWebhook(r.Context(), req.Name, req.SecretToken, req.AllowedMethods)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(wh)
}

func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if err := h.svc.DeleteWebhook(r.Context(), id); err != nil {
		if err == domainWebhook.ErrWebhookNotFound {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
