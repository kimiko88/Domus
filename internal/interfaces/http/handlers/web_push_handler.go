// Package handlers exposes authenticated browser Web Push endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	appnotification "github.com/domus-hub/domus/internal/application/notification"
	domainnotification "github.com/domus-hub/domus/internal/domain/notification"
	"github.com/domus-hub/domus/internal/interfaces/http/middleware"
)

type WebPushHandler struct {
	service *appnotification.WebPushService
}

func NewWebPushHandler(service *appnotification.WebPushService) *WebPushHandler {
	return &WebPushHandler{service: service}
}

func (h *WebPushHandler) PublicKey(w http.ResponseWriter, r *http.Request) {
	key, ok := h.service.PublicKey()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "WEB_PUSH_NOT_CONFIGURED", "Web Push VAPID keys are not configured on this server")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"public_key": key})
}

func (h *WebPushHandler) RegisterSubscription(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	var req struct {
		Endpoint       string `json:"endpoint"`
		Keys           struct {
			P256DH string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
		ExpirationTime *int64 `json:"expirationTime"`
		Locale         string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed subscription body")
		return
	}
	locale := strings.TrimSpace(req.Locale)
	if locale == "" {
		locale = strings.Split(r.Header.Get("Accept-Language"), ",")[0]
	}
	if locale == "" {
		locale = "it"
	}
	subscription := &domainnotification.WebPushSubscription{
		UserID: currentUser.ID,
		Endpoint: req.Endpoint,
		P256DH: req.Keys.P256DH,
		Auth: req.Keys.Auth,
		Locale: locale,
		ExpirationTime: req.ExpirationTime,
	}
	if err := h.service.RegisterSubscription(r.Context(), subscription); err != nil {
		writeError(w, http.StatusBadRequest, "SUBSCRIPTION_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status":"subscribed"})
}

func (h *WebPushHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	endpoint := r.URL.Query().Get("endpoint")
	if endpoint == "" {
		var req struct { Endpoint string `json:"endpoint"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request body")
			return
		}
		endpoint = req.Endpoint
	}
	if err := h.service.Unsubscribe(r.Context(), currentUser.ID, endpoint); err != nil {
		writeError(w, http.StatusBadRequest, "UNSUBSCRIBE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status":"unsubscribed"})
}

func (h *WebPushHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	preferences, err := h.service.Preferences(r.Context(), currentUser.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PREFERENCES_FAILED", "could not load Web Push preferences")
		return
	}
	writeJSON(w, http.StatusOK, preferences)
}

func (h *WebPushHandler) SavePreferences(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	var preferences domainnotification.WebPushPreferences
	if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "malformed preferences body")
		return
	}
	if err := h.service.SavePreferences(r.Context(), currentUser.ID, preferences); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PREFERENCES", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preferences)
}

func (h *WebPushHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	sent, err := h.service.SendUser(r.Context(), currentUser.ID, "Domus", "Le notifiche push sono configurate correttamente.", "test", map[string]any{"kind":"test","url":"/settings"})
	if err != nil {
		writeError(w, http.StatusBadGateway, "TEST_PUSH_FAILED", err.Error())
		return
	}
	if sent == 0 {
		writeError(w, http.StatusConflict, "NO_SUBSCRIPTION", "no active browser push subscription is registered")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"sent":sent})
}
