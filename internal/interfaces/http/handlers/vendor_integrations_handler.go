package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/imou_life"
	"github.com/domus-hub/domus/internal/plugins/xiaomi_home"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	imouIntegrationID = "integration_imou_life"
	xiaomiIntegrationID = "integration_xiaomi_home_local"
)

type ImouLifeHandler struct {
	repo integration.Repository
	cipher *crypto.EnvelopeCipher
	bus events.Bus
	mu sync.Mutex
	seenEvents map[string]time.Time
}

type imouStoredConfig struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	DataCenter string `json:"data_center"`
	AuthorizationURL string `json:"authorization_url"`
	PublicBaseURL string `json:"public_base_url"`
	CallbackSecret string `json:"callback_secret"`
	EventSecret string `json:"event_secret,omitempty"`
	PushEnabled bool `json:"push_enabled,omitempty"`
	PushFlags []string `json:"push_flags,omitempty"`
	AdminAccessToken string `json:"admin_access_token,omitempty"`
	AdminExpiresAt time.Time `json:"admin_expires_at,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type imouConfigRequest struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	DataCenter string `json:"data_center"`
	AuthorizationURL string `json:"authorization_url"`
	PublicBaseURL string `json:"public_base_url"`
}

type imouDeviceListResponse struct {
	Devices []imou_life.Device `json:"devices"`
	Count int `json:"count"`
	Page int `json:"page"`
	PageSize int `json:"page_size"`
}

type XiaomiHomeHandler struct {
	repo integration.Repository
	cipher *crypto.EnvelopeCipher
	client *xiaomi_home.Client
	mu sync.Mutex
}

type xiaomiStoredConfig struct {
	Devices []xiaomiStoredDevice `json:"devices"`
}

type xiaomiStoredDevice struct {
	ID string `json:"id"`
	Name string `json:"name"`
	IP string `json:"ip"`
	Token string `json:"token"`
	Model string `json:"model,omitempty"`
	Firmware string `json:"firmware,omitempty"`
	MAC string `json:"mac,omitempty"`
	Hardware string `json:"hardware,omitempty"`
	DeviceID uint32 `json:"device_id,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

type xiaomiDeviceView struct {
	ID string `json:"id"`
	Name string `json:"name"`
	IP string `json:"ip"`
	Model string `json:"model,omitempty"`
	Firmware string `json:"firmware,omitempty"`
	MAC string `json:"mac,omitempty"`
	Hardware string `json:"hardware,omitempty"`
	DeviceID uint32 `json:"device_id,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

func NewImouLifeHandler(repo integration.Repository, cipher *crypto.EnvelopeCipher) (*ImouLifeHandler, error) {
	if repo == nil || cipher == nil {
		return nil, errors.New("imou: integration repository and encryption are required")
	}
	return &ImouLifeHandler{repo: repo, cipher: cipher, seenEvents: make(map[string]time.Time)}, nil
}

// WithEventBus connects provider notifications to Domus automations and notifications.
func (h *ImouLifeHandler) WithEventBus(bus events.Bus) *ImouLifeHandler {
	h.bus = bus
	return h
}

func NewXiaomiHomeHandler(repo integration.Repository, cipher *crypto.EnvelopeCipher) (*XiaomiHomeHandler, error) {
	if repo == nil || cipher == nil {
		return nil, errors.New("xiaomi: integration repository and encryption are required")
	}
	return &XiaomiHomeHandler{repo: repo, cipher: cipher, client: xiaomi_home.NewClient(2*time.Second)}, nil
}

func (h *ImouLifeHandler) RegisterCallback(r chi.Router) {
	r.Get("/integrations/imou-life/callback/{secret}", h.Callback)
	r.Post("/integrations/imou-life/callback/{secret}", h.Callback)
	r.Post("/integrations/imou-life/events/{secret}", h.EventCallback)
}

func (h *ImouLifeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Put("/config", h.Configure)
	r.Get("/authorize", h.Authorization)
	r.Get("/devices", h.Devices)
	r.Get("/devices/{deviceID}/stream", h.StreamURL)
	r.Get("/devices/{deviceID}/capabilities", h.Capabilities)
	r.Get("/devices/{deviceID}/properties", h.ReadProperties)
	r.Put("/devices/{deviceID}/properties", h.WriteProperties)
	r.Post("/devices/{deviceID}/services/{serviceID}", h.InvokeService)
	r.Get("/devices/{deviceID}/messages", h.MessageHistory)
	r.Get("/devices/{deviceID}/recordings", h.Recordings)
	r.Post("/events/config", h.ConfigureEvents)
	r.Get("/events/status", h.EventStatus)
	r.Post("/disconnect", h.Disconnect)
}

func (h *XiaomiHomeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Get("/devices", h.ListDevices)
	r.Post("/devices", h.AddDevice)
	r.Get("/devices/{id}", h.GetDevice)
	r.Delete("/devices/{id}", h.DeleteDevice)
	r.Post("/devices/{id}/probe", h.ProbeDevice)
}

func (h *ImouLifeHandler) Configure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	h.mu.Lock()
	defer h.mu.Unlock()
	var req imouConfigRequest
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	req.AccessKey = strings.TrimSpace(req.AccessKey)
	req.SecretKey = strings.TrimSpace(req.SecretKey)
	req.DataCenter = strings.ToLower(strings.TrimSpace(req.DataCenter))
	if req.AccessKey == "" || req.SecretKey == "" {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", "Imou project access key and secret are required")
		return
	}
	if _, err := imou_life.NewClient(imou_life.Config{AccessKey:req.AccessKey, SecretKey:req.SecretKey, DataCenter:req.DataCenter}, nil); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", "Invalid Imou project credentials or data centre")
		return
	}
	if !validHTTPSURL(req.AuthorizationURL, true) || !validHTTPSURL(req.PublicBaseURL, false) {
		writeError(w, http.StatusBadRequest, "INVALID_URL", "Authorization URL and public callback base URL must be absolute HTTPS URLs")
		return
	}
	base, _ := url.Parse(strings.TrimRight(req.PublicBaseURL, "/"))
	if base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		writeError(w, http.StatusBadRequest, "INVALID_URL", "Public callback base URL cannot contain credentials, query or fragment")
		return
	}
	var secretBytes [32]byte
	if _, err := rand.Read(secretBytes[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Could not create callback secret")
		return
	}
	config := imouStoredConfig{
		AccessKey:req.AccessKey,
		SecretKey:req.SecretKey,
		DataCenter:req.DataCenter,
		AuthorizationURL:strings.TrimSpace(req.AuthorizationURL),
		PublicBaseURL:strings.TrimRight(req.PublicBaseURL, "/"),
		CallbackSecret:hex.EncodeToString(secretBytes[:]),
		EventSecret:hex.EncodeToString(secretBytes[:]),
	}
	if _, err := imou_life.NewClient(imou_life.Config{AccessKey:config.AccessKey, SecretKey:config.SecretKey, DataCenter:config.DataCenter}, nil); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", "Invalid Imou API configuration")
		return
	}
	if err := h.saveConfig(r.Context(), config, "disconnected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not save encrypted Imou configuration")
		return
	}
	callbackURL := config.PublicBaseURL + "/api/v1/integrations/imou-life/callback/" + config.CallbackSecret
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":true,
		"callback_url":callbackURL,
		"events_callback_url":config.PublicBaseURL + "/api/v1/integrations/imou-life/events/" + config.EventSecret,
		"authorization_url":config.AuthorizationURL,
		"message":"Register this exact HTTPS callback URL in the Imou Open Platform project, then start user authorization.",
	})
}

func (h *ImouLifeHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if err != nil {
		if errors.Is(err, integration.ErrIntegrationNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"configured":false,"connected":false})
			return
		}
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not load Imou integration status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":true,
		"connected":config.AccessToken != "" && config.RefreshToken != "",
		"access_token_expires_at":config.ExpiresAt,
		"data_center":config.DataCenter,
		"expires_at":config.ExpiresAt,
		"authorization_url":config.AuthorizationURL,
		"callback_url":config.PublicBaseURL + "/api/v1/integrations/imou-life/callback/" + config.CallbackSecret,
		"events_callback_url":config.PublicBaseURL + "/api/v1/integrations/imou-life/events/" + config.EventSecret,
		"push_enabled":config.PushEnabled,
		"push_flags":config.PushFlags,
		"admin_token_expires_at":config.AdminExpiresAt,
	})
}

func (h *ImouLifeHandler) Authorization(w http.ResponseWriter, r *http.Request) {
	config, err := h.loadConfig(r)
	if err != nil {
		writeError(w, http.StatusConflict, "NOT_CONFIGURED", "Configure Imou project credentials before authorization")
		return
	}
	// Return the user-consent H5 URL configured in the Imou developer console.
	// The provider binds the callback URL to the project; Domus never collects an Imou password.
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"authorization_url":config.AuthorizationURL})
}

func (h *ImouLifeHandler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	secret := chi.URLParam(r, "secret")
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if r.Method == http.MethodPost {
		contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
		if contentType != "application/x-www-form-urlencoded" {
			http.Error(w, "Invalid Imou authorization callback", http.StatusBadRequest)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid Imou authorization callback", http.StatusBadRequest)
			return
		}
		if code == "" {
			code = strings.TrimSpace(r.PostForm.Get("code"))
		}
	}
	if len(secret) != 64 || code == "" || len(code) > 2048 {
		http.Error(w, "Invalid Imou authorization callback", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.loadConfig(r)
	if err != nil || subtle.ConstantTimeCompare([]byte(secret), []byte(config.CallbackSecret)) != 1 {
		http.Error(w, "Invalid Imou authorization callback", http.StatusNotFound)
		return
	}
	client, err := imou_life.NewClient(imou_life.Config{AccessKey:config.AccessKey, SecretKey:config.SecretKey, DataCenter:config.DataCenter}, nil)
	if err != nil {
		http.Error(w, "Imou integration is not configured correctly", http.StatusServiceUnavailable)
		return
	}
	callbackURL := config.PublicBaseURL + "/api/v1/integrations/imou-life/callback/" + config.CallbackSecret
	token, err := client.ExchangeCode(r.Context(), code, callbackURL)
	if err != nil {
		_ = h.repo.SetStatus(r.Context(), imouIntegrationID, "error", "OAuth code exchange failed")
		http.Error(w, "Imou authorization failed; return to Domus and retry", http.StatusBadGateway)
		return
	}
	config.AccessToken = token.AccessToken
	config.RefreshToken = token.RefreshToken
	config.ExpiresAt = token.ExpiresAt
	if err := h.saveConfig(r.Context(), config, "connected"); err != nil {
		http.Error(w, "Imou authorization succeeded but token storage failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"connected":true,"message":"Imou Life connected to Domus. You can close this page."})
}


func (h *ImouLifeHandler) ConfigureEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req struct {
		Enabled bool     `json:"enabled"`
		Flags   []string `json:"flags"`
	}
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Configure and authorize Imou Life before enabling events")
		return
	}
	callbackURL := config.PublicBaseURL + "/api/v1/integrations/imou-life/events/" + config.EventSecret
	adminToken, err := h.ensureAdminToken(r, config, client)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_ADMIN_TOKEN_ERROR", "Could not obtain an Imou project token for event registration")
		return
	}
	if err := client.ConfigureMessageCallback(r.Context(), adminToken, callbackURL, req.Enabled, req.Flags); err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CALLBACK_CONFIG_ERROR", "Imou rejected the event callback configuration")
		return
	}
	config.PushEnabled = req.Enabled
	config.PushFlags = append([]string(nil), req.Flags...)
	if err := h.saveConfig(r.Context(), *config, "connected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not persist Imou event configuration")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled":config.PushEnabled, "flags":config.PushFlags, "callback_url":callbackURL})
}

func (h *ImouLifeHandler) EventStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "Imou Life is not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": config.PushEnabled,
		"flags": config.PushFlags,
		"callback_url": config.PublicBaseURL + "/api/v1/integrations/imou-life/events/" + config.EventSecret,
		"admin_token_expires_at": config.AdminExpiresAt,
	})
}

func (h *ImouLifeHandler) EventCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Referrer-Policy", "no-referrer")
	secret := chi.URLParam(r, "secret")
	if len(secret) != 64 || r.Method != http.MethodPost {
		http.Error(w, "Invalid Imou event callback", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var message imou_life.DeviceMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		http.Error(w, "Invalid Imou event payload", http.StatusBadRequest)
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid Imou event payload", http.StatusBadRequest)
		return
	}
	config, err := h.loadConfig(r)
	if err != nil || subtle.ConstantTimeCompare([]byte(secret), []byte(config.EventSecret)) != 1 {
		http.Error(w, "Invalid Imou event callback", http.StatusNotFound)
		return
	}
	if message.AppID != "" && subtle.ConstantTimeCompare([]byte(message.AppID), []byte(config.AccessKey)) != 1 {
		http.Error(w, "Invalid Imou event application", http.StatusForbidden)
		return
	}
	deviceID := message.DeviceID
	if deviceID == "" {
		deviceID = message.LegacyDeviceID
	}
	channelID := message.ChannelID
	if channelID == "" {
		channelID = message.LegacyChannelID
	}
	if deviceID == "" || len(deviceID) > 128 {
		http.Error(w, "Invalid Imou event device", http.StatusBadRequest)
		return
	}
	eventKey := message.ID
	if eventKey == "" {
		eventKey = strings.Join([]string{deviceID, channelID, message.MessageType, strconv.FormatInt(message.OccurredUnix, 10), message.AlarmID}, ":")
	}
	if !h.acceptEvent(eventKey) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if h.bus == nil {
		http.Error(w, "Domus event bus is unavailable", http.StatusServiceUnavailable)
		return
	}
	occurredAt := time.Now().UTC()
	if message.OccurredUnix > 0 {
		occurredAt = time.Unix(message.OccurredUnix, 0).UTC()
	}
	payload := map[string]any{
		"source": "imou_life",
		"device_id": deviceID,
		"channel_id": channelID,
		"message_type": message.MessageType,
		"alarm_id": message.AlarmID,
		"label_type": message.LabelType,
		"device_status": message.DeviceStatus,
		"local_time": message.LocalTime,
	}
	eventType := events.WebhookReceived
	messageType := strings.ToLower(strings.TrimSpace(message.MessageType))
	if messageType == "devicestatus" {
		status := strings.ToLower(strings.TrimSpace(message.DeviceStatus))
		switch status {
		case "online", "1", "on":
			eventType = events.DeviceOnline
		case "offline", "0", "off":
			eventType = events.DeviceOffline
		}
	} else if messageType == "alarm" || messageType == "capabilityevent" ||
		strings.Contains(messageType, "motion") || strings.Contains(messageType, "person") ||
		strings.Contains(messageType, "human") || strings.Contains(messageType, "intrusion") {
		eventType = events.CameraDetected
		label := strings.ToLower(strings.TrimSpace(message.LabelType))
		if label == "" {
			label = messageType
		}
		payload["label"] = label
		payload["confidence"] = 1.0
		payload["camera_id"] = deviceID
	}
	if err := h.bus.Publish(r.Context(), events.Event{
		ID: uuid.NewString(), Type: eventType, OccurredAt: occurredAt,
		Source: "imou_life", Payload: payload,
	}); err != nil {
		http.Error(w, "Domus could not queue Imou event", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ImouLifeHandler) acceptEvent(key string) bool {
	now := time.Now().UTC()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seenEvents == nil {
		h.seenEvents = make(map[string]time.Time)
	}
	for id, seenAt := range h.seenEvents {
		if now.Sub(seenAt) > 24*time.Hour {
			delete(h.seenEvents, id)
		}
	}
	if _, exists := h.seenEvents[key]; exists {
		return false
	}
	if len(h.seenEvents) >= 10000 {
		// Evict the oldest key to keep callback memory bounded.
		var oldestKey string
		var oldest time.Time
		for id, seenAt := range h.seenEvents {
			if oldest.IsZero() || seenAt.Before(oldest) {
				oldestKey, oldest = id, seenAt
			}
		}
		delete(h.seenEvents, oldestKey)
	}
	h.seenEvents[key] = now
	return true
}

func (h *ImouLifeHandler) loadClient(r *http.Request) (*imouStoredConfig, *imou_life.Client, error) {
	config, err := h.loadConfig(r)
	if err != nil {
		return nil, nil, err
	}
	if config.AccessToken == "" || config.RefreshToken == "" {
		return nil, nil, errors.New("imou: OAuth user token is not configured")
	}
	client, err := imou_life.NewClient(imou_life.Config{
		AccessKey: config.AccessKey, SecretKey: config.SecretKey, DataCenter: config.DataCenter,
	}, nil)
	if err != nil {
		return nil, nil, err
	}
	return config, client, nil
}

func (h *ImouLifeHandler) ensureAdminToken(r *http.Request, config *imouStoredConfig, client *imou_life.Client) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if config.AdminAccessToken != "" && time.Until(config.AdminExpiresAt) > 5*time.Minute {
		return config.AdminAccessToken, nil
	}
	token, err := client.GetAdminToken(r.Context())
	if err != nil {
		return "", err
	}
	config.AdminAccessToken = token.AccessToken
	config.AdminExpiresAt = token.ExpiresAt
	if err := h.saveConfig(r.Context(), *config, "connected"); err != nil {
		return "", err
	}
	return config.AdminAccessToken, nil
}

func (h *ImouLifeHandler) Capabilities(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before loading capabilities")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	channelID := r.URL.Query().Get("channel")
	capabilities, err := client.ListCapabilities(r.Context(), config.AccessToken, deviceID, channelID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CAPABILITIES_ERROR", "Could not discover Imou device capabilities")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, map[string]any{"device_id":deviceID,"channel_id":channelID,"capabilities":capabilities})
}

func (h *ImouLifeHandler) ReadProperties(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before reading properties")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	channelID := r.URL.Query().Get("channel")
	capabilities, err := client.ListCapabilities(r.Context(), config.AccessToken, deviceID, channelID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CAPABILITIES_ERROR", "Could not verify property permissions")
		return
	}
	allowed := make(map[string]bool)
	for _, capability := range capabilities {
		for _, property := range capability.Properties {
			if strings.Contains(strings.ToLower(property.AccessMode), "r") {
				allowed[property.PropertyID] = true
			}
		}
	}
	for _, id := range ids {
		if !allowed[strings.TrimSpace(id)] {
			writeError(w, http.StatusForbidden, "PROPERTY_NOT_READABLE", "Requested property is not advertised as readable by this device")
			return
		}
	}
	values, err := client.ReadProperties(r.Context(), config.AccessToken, deviceID, channelID, ids)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_PROPERTY_READ_ERROR", "Could not read device properties")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, values)
}

func (h *ImouLifeHandler) WriteProperties(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChannelID string         `json:"channel_id"`
		Values    map[string]any `json:"values"`
	}
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before writing properties")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	capabilities, err := client.ListCapabilities(r.Context(), config.AccessToken, deviceID, req.ChannelID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CAPABILITIES_ERROR", "Could not verify property permissions")
		return
	}
	allowed := make(map[string]imou_life.CapabilityProperty)
	for _, capability := range capabilities {
		for _, property := range capability.Properties {
			if strings.Contains(strings.ToLower(property.AccessMode), "w") {
				allowed[property.PropertyID] = property
			}
		}
	}
	for id := range req.Values {
		if _, ok := allowed[id]; !ok {
			writeError(w, http.StatusForbidden, "PROPERTY_NOT_WRITABLE", "Requested property is not advertised as writable by this device")
			return
		}
	}
	result, err := client.WriteProperties(r.Context(), config.AccessToken, deviceID, req.ChannelID, req.Values)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_PROPERTY_WRITE_ERROR", "Could not write device properties")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, result)
}

func (h *ImouLifeHandler) InvokeService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChannelID string         `json:"channel_id"`
		Input     map[string]any `json:"input"`
	}
	if !decodeLimitedJSON(w, r, &req, 16*1024) {
		return
	}
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before controlling the device")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	serviceID := chi.URLParam(r, "serviceID")
	capabilities, err := client.ListCapabilities(r.Context(), config.AccessToken, deviceID, req.ChannelID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CAPABILITIES_ERROR", "Could not verify service availability")
		return
	}
	allowed := false
	for _, capability := range capabilities {
		for _, service := range capability.Services {
			if service.ServiceID == serviceID {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "SERVICE_NOT_SUPPORTED", "Service is not advertised by this device")
		return
	}
	result, err := client.InvokeService(r.Context(), config.AccessToken, deviceID, req.ChannelID, serviceID, req.Input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_SERVICE_ERROR", "Imou device rejected the requested service")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, result)
}

func (h *ImouLifeHandler) MessageHistory(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before reading messages")
		return
	}
	begin, errBegin := time.Parse(time.RFC3339, r.URL.Query().Get("begin"))
	end, errEnd := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
	if errBegin != nil || errEnd != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TIME_RANGE", "begin and end must be RFC3339 timestamps")
		return
	}
	page, err := client.ListMessageHistory(r.Context(), config.AccessToken, chi.URLParam(r, "deviceID"), r.URL.Query().Get("channel"), begin, end, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_MESSAGE_HISTORY_ERROR", "Could not load Imou event history")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, page)
}

func (h *ImouLifeHandler) Recordings(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.loadClient(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before searching recordings")
		return
	}
	begin, errBegin := time.Parse(time.RFC3339, r.URL.Query().Get("begin"))
	end, errEnd := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
	if errBegin != nil || errEnd != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TIME_RANGE", "begin and end must be RFC3339 timestamps")
		return
	}
	recordType := 0
	limit := 50
	if value := r.URL.Query().Get("type"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { recordType = parsed } else { recordType = -1 }
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { limit = parsed } else { limit = -1 }
	}
	deviceID := chi.URLParam(r, "deviceID")
	channelID := r.URL.Query().Get("channel")
	capabilities, err := client.ListCapabilities(r.Context(), config.AccessToken, deviceID, channelID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_CAPABILITIES_ERROR", "Could not discover recording capabilities")
		return
	}
	supported := false
	for _, capability := range capabilities {
		for _, service := range capability.Services {
			if service.ServiceID == "record.file.list" { supported = true }
		}
	}
	if !supported {
		writeError(w, http.StatusNotImplemented, "RECORDINGS_NOT_SUPPORTED", "This camera does not advertise local recording search")
		return
	}
	records, sessionID, err := client.ListLocalRecordings(r.Context(), config.AccessToken, deviceID, channelID, begin, end, recordType, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_RECORDINGS_ERROR", "Could not search Imou recordings")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, map[string]any{"records":records,"session_id":sessionID})
}

func (h *ImouLifeHandler) Devices(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.ensureFreshToken(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before listing devices")
		return
	}
	page := 1
	pageSize := 50
	if value := r.URL.Query().Get("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { page = parsed } else { page = 0 }
	}
	if value := r.URL.Query().Get("page_size"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { pageSize = parsed } else { pageSize = 0 }
	}
	devices, count, err := client.ListDevices(r.Context(), config.AccessToken, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_API_ERROR", "Imou device list request failed")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, imouDeviceListResponse{Devices:devices,Count:count,Page:page,PageSize:pageSize})
}

func (h *ImouLifeHandler) StreamURL(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.ensureFreshToken(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before requesting a stream")
		return
	}
	deviceID := chi.URLParam(r, "deviceID")
	channel := 0
	stream := 0
	if value := r.URL.Query().Get("channel"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { channel = parsed } else { channel = -1 }
	}
	if value := r.URL.Query().Get("stream"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil { stream = parsed } else { stream = -1 }
	}
	streamURL, err := client.GetStreamURL(r.Context(), config.AccessToken, deviceID, channel, stream)
	if err != nil {
		writeError(w, http.StatusBadGateway, "IMOU_STREAM_ERROR", "Could not obtain Imou stream URL")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]string{"url":streamURL,"warning":"This RTSP URL is a secret. Do not share it or store it in browser logs."})
}

func (h *ImouLifeHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.loadConfig(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "Imou integration is not configured")
		return
	}
	config.AccessToken = ""
	config.RefreshToken = ""
	config.ExpiresAt = time.Time{}
	if err := h.saveConfig(r.Context(), config, "disconnected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not disconnect Imou account")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"connected":false})
}

func (h *ImouLifeHandler) ensureFreshToken(r *http.Request) (*imouStoredConfig, *imou_life.Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.loadConfig(r)
	if err != nil { return nil, nil, err }
	client, err := imou_life.NewClient(imou_life.Config{AccessKey:config.AccessKey,SecretKey:config.SecretKey,DataCenter:config.DataCenter}, nil)
	if err != nil { return nil, nil, err }
	if config.AccessToken == "" || config.RefreshToken == "" { return nil,nil,errors.New("imou: no delegated account token") }
	if time.Until(config.ExpiresAt) < 2*time.Minute {
		token, refreshErr := client.RefreshToken(r.Context(), config.RefreshToken)
		if refreshErr != nil {
			_ = h.repo.SetStatus(r.Context(), imouIntegrationID, "error", "OAuth token refresh failed")
			return nil,nil,refreshErr
		}
		config.AccessToken = token.AccessToken
		config.RefreshToken = token.RefreshToken
		config.ExpiresAt = token.ExpiresAt
		if err := h.saveConfig(r.Context(), *config, "connected"); err != nil { return nil,nil,err }
	}
	return config, client, nil
}

func (h *ImouLifeHandler) loadConfig(r *http.Request) (*imouStoredConfig, error) {
	record, err := h.repo.GetByID(r.Context(), imouIntegrationID)
	if err != nil { return nil, err }
	plain, err := h.cipher.Decrypt(string(record.ConfigEncrypted))
	if err != nil { return nil, fmt.Errorf("imou: decrypt stored configuration: %w", err) }
	var config imouStoredConfig
	if err := json.Unmarshal([]byte(plain), &config); err != nil { return nil, errors.New("imou: invalid stored configuration") }
	return &config, nil
}

func (h *ImouLifeHandler) saveConfig(ctx context.Context, config imouStoredConfig, status string) error {
	encoded, err := json.Marshal(config)
	if err != nil { return err }
	encrypted, err := h.cipher.Encrypt(string(encoded))
	if err != nil { return err }
	record, getErr := h.repo.GetByID(ctx, imouIntegrationID)
	now := time.Now().UTC()
	if errors.Is(getErr, integration.ErrIntegrationNotFound) {
		record = &integration.Integration{ID:imouIntegrationID,Type:"imou_life",Name:"Imou Life",Enabled:true,Status:status,CreatedAt:now,UpdatedAt:now}
	} else if getErr != nil { return getErr } else {
		record.ConfigEncrypted = []byte(encrypted)
		record.Name = "Imou Life"
		record.Enabled = true
		record.UpdatedAt = now
		if err := h.repo.Update(ctx, record); err != nil { return err }
		return h.repo.SetStatus(ctx, imouIntegrationID, status, "")
	}
	record.ConfigEncrypted = []byte(encrypted)
	if err := h.repo.Create(ctx, record); err != nil { return err }
	return h.repo.SetStatus(ctx, imouIntegrationID, status, "")
}

func (h *XiaomiHomeHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if err != nil {
		if errors.Is(err, integration.ErrIntegrationNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{"configured":false,"devices":[]any{}})
			return
		}
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not load Xiaomi Home configuration")
		return
	}
	views := make([]xiaomiDeviceView, 0, len(config.Devices))
	for _, d := range config.Devices { views = append(views, viewXiaomiDevice(d)) }
	writeJSON(w, http.StatusOK, map[string]any{"configured":true,"mode":"local_miio","devices":views})
}

func (h *XiaomiHomeHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r)
	if errors.Is(err, integration.ErrIntegrationNotFound) {
		writeJSON(w, http.StatusOK, []xiaomiDeviceView{})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not load Xiaomi Home devices")
		return
	}
	views := make([]xiaomiDeviceView, 0, len(config.Devices))
	for _, d := range config.Devices { views = append(views, viewXiaomiDevice(d)) }
	writeJSON(w, http.StatusOK, views)
}

func (h *XiaomiHomeHandler) AddDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req struct {
		IP string `json:"ip"`
		Name string `json:"name"`
		Token string `json:"token"`
	}
	if !decodeLimitedJSON(w,r,&req,16*1024) { return }
	req.IP = strings.TrimSpace(req.IP)
	req.Name = strings.TrimSpace(req.Name)
	req.Token = strings.TrimSpace(req.Token)
	if req.Name == "" || len(req.Name) > 80 {
		writeError(w,http.StatusBadRequest,"INVALID_DEVICE","A device name of at most 80 characters is required")
		return
	}
	info, err := h.client.Info(r.Context(), req.IP, req.Token)
	if err != nil {
		writeError(w,http.StatusBadRequest,"PAIRING_FAILED","Could not authenticate to the local Xiaomi device; check its LAN address and 32-character miIO token")
		return
	}
	handshake, err := h.client.Handshake(r.Context(), req.IP)
	if err != nil {
		writeError(w,http.StatusBadRequest,"PAIRING_FAILED","Xiaomi local handshake failed")
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.loadConfig(r)
	if errors.Is(err, integration.ErrIntegrationNotFound) { config = &xiaomiStoredConfig{} } else if err != nil {
		writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","Could not load Xiaomi Home configuration")
		return
	}
	for _, existing := range config.Devices {
		if existing.IP == req.IP {
			writeError(w,http.StatusConflict,"ALREADY_PAIRED","This Xiaomi device is already paired")
			return
		}
	}
	config.Devices = append(config.Devices, xiaomiStoredDevice{
		ID:uuid.NewString(),Name:req.Name,IP:info.IP,Token:req.Token,Model:info.Model,Firmware:info.Firmware,
		MAC:info.MAC,Hardware:info.Hardware,DeviceID:handshake.DeviceID,AddedAt:time.Now().UTC(),
	})
	if err := h.saveConfig(r.Context(), *config); err != nil {
		writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","Could not store encrypted Xiaomi device credentials")
		return
	}
	writeJSON(w,http.StatusCreated,viewXiaomiDevice(config.Devices[len(config.Devices)-1]))
}

func (h *XiaomiHomeHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	config, err := h.loadConfig(r)
	if err != nil {
		writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found")
		return
	}
	for _, d := range config.Devices {
		if d.ID == chi.URLParam(r,"id") { writeJSON(w,http.StatusOK,viewXiaomiDevice(d)); return }
	}
	writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found")
}

func (h *XiaomiHomeHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.loadConfig(r)
	if err != nil { writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found"); return }
	id := chi.URLParam(r,"id")
	kept := make([]xiaomiStoredDevice,0,len(config.Devices))
	found := false
	for _, d := range config.Devices {
		if d.ID == id { found = true; continue }
		kept = append(kept,d)
	}
	if !found { writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found"); return }
	config.Devices = kept
	if err := h.saveConfig(r.Context(),*config); err != nil { writeError(w,http.StatusInternalServerError,"STORAGE_ERROR","Could not update Xiaomi Home configuration"); return }
	writeJSON(w,http.StatusOK,map[string]string{"status":"deleted","id":id})
}

func (h *XiaomiHomeHandler) ProbeDevice(w http.ResponseWriter, r *http.Request) {
	config, err := h.loadConfig(r)
	if err != nil { writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found"); return }
	id := chi.URLParam(r,"id")
	for _, d := range config.Devices {
		if d.ID != id { continue }
		info, err := h.client.Info(r.Context(),d.IP,d.Token)
		if err != nil { writeError(w,http.StatusBadGateway,"DEVICE_UNAVAILABLE","Xiaomi device did not respond to authenticated miIO info"); return }
		writeJSON(w,http.StatusOK,info)
		return
	}
	writeError(w,http.StatusNotFound,"NOT_FOUND","Xiaomi device not found")
}

func (h *XiaomiHomeHandler) loadConfig(r *http.Request) (*xiaomiStoredConfig,error) {
	record,err := h.repo.GetByID(r.Context(),xiaomiIntegrationID)
	if err != nil { return nil,err }
	plain,err := h.cipher.Decrypt(string(record.ConfigEncrypted))
	if err != nil { return nil,fmt.Errorf("xiaomi: decrypt stored configuration: %w",err) }
	var config xiaomiStoredConfig
	if err := json.Unmarshal([]byte(plain),&config); err != nil { return nil,errors.New("xiaomi: invalid stored configuration") }
	if config.Devices == nil { config.Devices = []xiaomiStoredDevice{} }
	return &config,nil
}

func (h *XiaomiHomeHandler) saveConfig(ctx context.Context,config xiaomiStoredConfig) error {
	encoded,err := json.Marshal(config)
	if err != nil { return err }
	encrypted,err := h.cipher.Encrypt(string(encoded))
	if err != nil { return err }
	record,getErr := h.repo.GetByID(ctx,xiaomiIntegrationID)
	now:=time.Now().UTC()
	if errors.Is(getErr,integration.ErrIntegrationNotFound) {
		record=&integration.Integration{ID:xiaomiIntegrationID,Type:"xiaomi_home_local",Name:"Xiaomi Home (LAN)",Enabled:true,Status:"connected",CreatedAt:now,UpdatedAt:now}
	} else if getErr!=nil { return getErr } else {
		record.ConfigEncrypted=[]byte(encrypted); record.Name="Xiaomi Home (LAN)"; record.Enabled=true; record.UpdatedAt=now
		if err:=h.repo.Update(ctx,record); err!=nil{return err}
		return h.repo.SetStatus(ctx,xiaomiIntegrationID,"connected","")
	}
	record.ConfigEncrypted=[]byte(encrypted)
	if err:=h.repo.Create(ctx,record); err!=nil{return err}
	return h.repo.SetStatus(ctx,xiaomiIntegrationID,"connected","")
}

func viewXiaomiDevice(d xiaomiStoredDevice) xiaomiDeviceView {
	return xiaomiDeviceView{ID:d.ID,Name:d.Name,IP:d.IP,Model:d.Model,Firmware:d.Firmware,MAC:d.MAC,Hardware:d.Hardware,DeviceID:d.DeviceID,AddedAt:d.AddedAt}
}

func decodeLimitedJSON(w http.ResponseWriter, r *http.Request, dst any, max int64) bool {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is required")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, max)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid or oversized JSON request")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Only one JSON value is allowed")
		return false
	}
	return true
}

func validHTTPSURL(raw string, allowQuery bool) bool {
	u,err:=url.Parse(strings.TrimSpace(raw))
	if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.Fragment!="" { return false }
	if !allowQuery && u.RawQuery!="" { return false }
	return true
}
