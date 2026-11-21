package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/xiaomi_home"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const xiaomiCloudIntegrationID = "integration_xiaomi_home_cloud"

type XiaomiCloudHandler struct {
	integrations integration.Repository
	devices device.Repository
	entities entity.Repository
	cipher *crypto.EnvelopeCipher
	bus events.Bus
	mu sync.Mutex
	seenEvents map[string]time.Time
}

type xiaomiCloudConfig struct {
	ClientID string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Region string `json:"region"`
	PublicBaseURL string `json:"public_base_url"`
	RedirectURL string `json:"redirect_url"`
	CallbackSecret string `json:"callback_secret"`
	EventSecret string `json:"event_secret"`
	OAuthState string `json:"oauth_state,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	MacKey string `json:"mac_key,omitempty"`
	MacAlgorithm string `json:"mac_algorithm,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	UserID string `json:"user_id,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}

type xiaomiCloudConfigRequest struct {
	ClientID string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Region string `json:"region"`
	PublicBaseURL string `json:"public_base_url"`
}

type xiaomiPropertyWrite struct {
	ServiceID int `json:"siid"`
	PropertyID int `json:"piid"`
	Value any `json:"value"`
}

func NewXiaomiCloudHandler(integrations integration.Repository, devices device.Repository, entities entity.Repository, cipher *crypto.EnvelopeCipher, bus events.Bus) (*XiaomiCloudHandler, error) {
	if integrations == nil || devices == nil || entities == nil || cipher == nil {
		return nil, errors.New("xiaomi: repositories and encryption are required")
	}
	return &XiaomiCloudHandler{integrations: integrations, devices: devices, entities: entities, cipher: cipher, bus: bus, seenEvents: make(map[string]time.Time)}, nil
}

func (h *XiaomiCloudHandler) RegisterCallback(r chi.Router) {
	r.Get("/integrations/xiaomi-home/cloud/callback/{secret}", h.Callback)
	r.Post("/integrations/xiaomi-home/cloud/events/{secret}", h.EventCallback)
}

func (h *XiaomiCloudHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Put("/config", h.Configure)
	r.Get("/authorize", h.Authorize)
	r.Get("/devices", h.ListDevices)
	r.Get("/homes", h.ListHomes)
	r.Post("/sync", h.SyncDevices)
	r.Get("/devices/{did}/spec", h.GetDeviceSpec)
	r.Get("/devices/{did}/properties", h.GetProperties)
	r.Put("/devices/{did}/properties", h.SetProperties)
	r.Put("/devices/{did}/actions", h.InvokeAction)
	r.Post("/events/subscribe", h.Subscribe)
	r.Post("/disconnect", h.Disconnect)
}

func (h *XiaomiCloudHandler) Configure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	var req xiaomiCloudConfigRequest
	if !decodeLimitedJSON(w, r, &req, 16*1024) { return }
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	req.Region = strings.ToLower(strings.TrimSpace(req.Region))
	req.PublicBaseURL = strings.TrimRight(strings.TrimSpace(req.PublicBaseURL), "/")
	if err := validHTTPSBaseURL(req.PublicBaseURL); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_URL", "Public callback base URL must be HTTPS and cannot contain credentials, query or fragment")
		return
	}
	var callbackSecret, eventSecret [32]byte
	if _, err := rand.Read(callbackSecret[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Could not create Xiaomi callback secret")
		return
	}
	if _, err := rand.Read(eventSecret[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Could not create Xiaomi event secret")
		return
	}
	config := xiaomiCloudConfig{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret, Region: req.Region,
		PublicBaseURL: req.PublicBaseURL,
		CallbackSecret: hex.EncodeToString(callbackSecret[:]),
		EventSecret: hex.EncodeToString(eventSecret[:]),
	}
	config.RedirectURL = config.PublicBaseURL + "/api/v1/integrations/xiaomi-home/cloud/callback/" + config.CallbackSecret
	if _, err := h.newClient(&config); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONFIG", "Invalid Xiaomi Open Platform app ID, secret or region")
		return
	}
	if err := h.saveConfig(r.Context(), config, "disconnected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not save encrypted Xiaomi configuration")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"callback_url": config.RedirectURL,
		"events_callback_url": config.PublicBaseURL + "/api/v1/integrations/xiaomi-home/cloud/events/" + config.EventSecret,
		"region": config.Region,
	})
}

func (h *XiaomiCloudHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, err := h.loadConfig(r.Context())
	if errors.Is(err, integration.ErrIntegrationNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "connected": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not load Xiaomi integration status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"connected": config.AccessToken != "" && config.RefreshToken != "" && config.UserID != "",
		"nickname": config.Nickname,
		"user_id_masked": maskIdentifier(config.UserID),
		"region": config.Region,
		"expires_at": config.ExpiresAt,
		"callback_url": config.RedirectURL,
		"events_callback_url": config.PublicBaseURL + "/api/v1/integrations/xiaomi-home/cloud/events/" + config.EventSecret,
	})
}

func (h *XiaomiCloudHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	config, client, err := h.loadClient(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "NOT_CONFIGURED", "Configure Xiaomi Open Platform credentials first")
		return
	}
	state, err := xiaomi_home.GenerateCloudState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Could not create OAuth state")
		return
	}
	authorizeURL, err := client.AuthorizationURL(state, []string{"1", "6000"})
	if err != nil {
		writeError(w, http.StatusBadRequest, "OAUTH_CONFIG_ERROR", "Could not create Xiaomi authorization URL")
		return
	}
	config.OAuthState = state
	if err := h.saveConfig(r.Context(), *config, "authorizing"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not persist Xiaomi OAuth state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorization_url": authorizeURL})
}

func (h *XiaomiCloudHandler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	secret := chi.URLParam(r, "secret")
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if len(secret) != 64 || len(state) < 32 || len(code) == 0 || len(code) > 2048 {
		http.Error(w, "Invalid Xiaomi OAuth callback", http.StatusBadRequest)
		return
	}
	config, err := h.loadConfig(r.Context())
	if err != nil || subtle.ConstantTimeCompare([]byte(secret), []byte(config.CallbackSecret)) != 1 ||
		!xiaomi_home.CompareCloudState(config.OAuthState, state) {
		http.Error(w, "Invalid Xiaomi OAuth callback", http.StatusNotFound)
		return
	}
	client, err := h.newClient(config)
	if err != nil {
		http.Error(w, "Xiaomi Open Platform is not configured correctly", http.StatusServiceUnavailable)
		return
	}
	token, err := client.ExchangeCode(r.Context(), code)
	if err != nil {
		_ = h.integrations.SetStatus(r.Context(), xiaomiCloudIntegrationID, "error", "Xiaomi OAuth code exchange failed")
		http.Error(w, "Xiaomi authorization failed; return to Domus and retry", http.StatusBadGateway)
		return
	}
	profile, err := client.GetUserProfile(r.Context(), token.AccessToken)
	if err != nil {
		http.Error(w, "Xiaomi authorized the account but its profile could not be read", http.StatusBadGateway)
		return
	}
	config.AccessToken, config.RefreshToken = token.AccessToken, token.RefreshToken
	config.MacKey, config.MacAlgorithm, config.ExpiresAt = token.MacKey, token.MacAlgorithm, token.ExpiresAt
	config.UserID, config.Nickname, config.OAuthState = profile.UserID, profile.Nickname, ""
	if err := h.saveConfig(r.Context(), *config, "connected"); err != nil {
		http.Error(w, "Xiaomi authorization succeeded but encrypted token storage failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected": true, "nickname": profile.Nickname, "message": "Xiaomi Home is connected to Domus. You can close this page."})
}

func (h *XiaomiCloudHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	_, client, err := h.ensureFreshToken(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "XIAOMI_NOT_CONNECTED", "Authorize Xiaomi Home before listing devices")
		return
	}
	devices, err := client.ListDevices(r.Context(), config.AccessToken, config.UserID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_API_ERROR", "Could not load Xiaomi Home devices")
		return
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "total": len(devices), "region": config.Region})
}

func (h *XiaomiCloudHandler) ListHomes(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.ensureFreshToken(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "XIAOMI_NOT_CONNECTED", "Authorize Xiaomi Home before listing homes")
		return
	}
	homes, err := client.ListHomes(r.Context(), config.AccessToken, config.UserID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_API_ERROR", "Could not load Xiaomi homes and rooms")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"homes": homes})
}

func (h *XiaomiCloudHandler) SyncDevices(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.ensureFreshToken(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "XIAOMI_NOT_CONNECTED", "Authorize Xiaomi Home before synchronizing devices")
		return
	}
	cloudDevices, err := client.ListDevices(r.Context(), config.AccessToken, config.UserID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_API_ERROR", "Could not list Xiaomi Home devices")
		return
	}
	synced, skipped, failed := 0, 0, 0
	for _, item := range cloudDevices {
		if item.DID == "" || !xiaomi_home.ValidMIoTDeviceType(item.Type) {
			skipped++
			continue
		}
		record, lookupErr := h.devices.GetByExternalID(r.Context(), xiaomiCloudIntegrationID, item.DID)
		if errors.Is(lookupErr, device.ErrDeviceNotFound) {
			metadata, _ := json.Marshal(map[string]any{
				"source": "xiaomi_open_platform", "miot_type": item.Type,
				"category": item.Category, "region": config.Region, "room_id": item.RoomID,
				"last_update_timestamp": item.LastUpdate,
			})
			record = &device.Device{
				ID: uuid.NewString(), IntegrationID: xiaomiCloudIntegrationID,
				ExternalID: item.DID, Name: item.Name,
				Manufacturer: "Xiaomi", Model: item.Type, Online: item.Online,
				Metadata: string(metadata), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			}
			if err := h.devices.Create(r.Context(), record); err != nil {
				failed++
				continue
			}
		} else if lookupErr != nil {
			failed++
			continue
		} else {
			record.Name, record.Model = item.Name, item.Type
			if err := h.devices.Update(r.Context(), record); err != nil {
				failed++
				continue
			}
			_ = h.devices.SetOnline(r.Context(), record.ID, item.Online)
		}
		if err := h.syncDeviceSpec(r.Context(), client, config, item, record); err != nil {
			failed++
			continue
		}
		synced++
	}
	writeJSON(w, http.StatusOK, map[string]int{"synced": synced, "skipped": skipped, "failed": failed, "total": len(cloudDevices)})
}

func (h *XiaomiCloudHandler) GetDeviceSpec(w http.ResponseWriter, r *http.Request) {
	_, client, err := h.ensureFreshToken(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "XIAOMI_NOT_CONNECTED", "Authorize Xiaomi Home before loading a device specification")
		return
	}
	record, err := h.devices.GetByExternalID(r.Context(), xiaomiCloudIntegrationID, chi.URLParam(r, "did"))
	if err != nil {
		writeError(w, http.StatusNotFound, "DEVICE_NOT_FOUND", "Synchronize this Xiaomi device before loading its specification")
		return
	}
	var metadata struct { MIoTType string `json:"miot_type"` }
	if err := json.Unmarshal([]byte(record.Metadata), &metadata); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "MIOT_SPEC_UNAVAILABLE", "The device has no MIoT specification")
		return
	}
	spec, err := client.FetchDeviceSpec(r.Context(), metadata.MIoTType)
	if err != nil {
		writeError(w, http.StatusBadGateway, "MIOT_SPEC_ERROR", "Could not load MIoT-Spec-V2")
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (h *XiaomiCloudHandler) GetProperties(w http.ResponseWriter, r *http.Request) {
	config, client, record, spec, err := h.getDeviceContext(r)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_DEVICE_ERROR", "Could not load Xiaomi device or MIoT specification")
		return
	}
	ids := make([]string, 0, 64)
	for _, ref := range spec.Properties() {
		if ref.Property.CanRead() {
			ids = append(ids, fmt.Sprintf("%s.%d.%d", record.ExternalID, ref.ServiceID, ref.Property.IID))
		}
		if len(ids) == 64 { break }
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"properties": []any{}})
		return
	}
	values, err := client.GetProperties(r.Context(), config.AccessToken, config.UserID, ids)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_PROPERTY_ERROR", "Could not read Xiaomi device properties")
		return
	}
	for _, value := range values {
		h.updateEntityState(r.Context(), record, spec, value.PID, value.Value, value.Status)
	}
	w.Header().Set("Cache-Control", "no-store, private")
	writeJSON(w, http.StatusOK, map[string]any{"properties": values})
}

func (h *XiaomiCloudHandler) SetProperties(w http.ResponseWriter, r *http.Request) {
	var req struct { Properties []xiaomiPropertyWrite `json:"properties"` }
	if !decodeLimitedJSON(w, r, &req, 32*1024) { return }
	if len(req.Properties) == 0 || len(req.Properties) > 32 {
		writeError(w, http.StatusBadRequest, "INVALID_PROPERTIES", "Provide between 1 and 32 MIoT properties")
		return
	}
	config, client, record, spec, err := h.getDeviceContext(r)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_DEVICE_ERROR", "Could not load Xiaomi device or MIoT specification")
		return
	}
	updates := make([]xiaomi_home.PropertyUpdate, 0, len(req.Properties))
	seen := make(map[string]bool)
	for _, item := range req.Properties {
		ref, ok := spec.FindProperty(item.ServiceID, item.PropertyID)
		if !ok || !ref.Property.CanWrite() {
			writeError(w, http.StatusForbidden, "PROPERTY_NOT_WRITABLE", "The requested property is not declared writable by this device")
			return
		}
		if err := ref.Property.ValidateValue(item.Value); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_PROPERTY_VALUE", err.Error())
			return
		}
		pid := fmt.Sprintf("%s.%d.%d", record.ExternalID, item.ServiceID, item.PropertyID)
		if seen[pid] {
			writeError(w, http.StatusBadRequest, "DUPLICATE_PROPERTY", "A property can be written only once per request")
			return
		}
		seen[pid] = true
		updates = append(updates, xiaomi_home.PropertyUpdate{PID: pid, Value: item.Value})
	}
	results, err := client.SetProperties(r.Context(), config.AccessToken, config.UserID, updates)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_PROPERTY_ERROR", "Could not write Xiaomi device properties")
		return
	}
	for _, result := range results {
		h.updateEntityState(r.Context(), record, spec, result.PID, result.Value, result.Status)
	}
	writeJSON(w, http.StatusOK, map[string]any{"properties": results})
}

func (h *XiaomiCloudHandler) InvokeAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID int `json:"siid"`
		ActionID int `json:"aiid"`
		Input []any `json:"input"`
	}
	if !decodeLimitedJSON(w, r, &req, 16*1024) { return }
	config, client, record, spec, err := h.getDeviceContext(r)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_DEVICE_ERROR", "Could not load Xiaomi device or MIoT specification")
		return
	}
	action, ok := spec.FindAction(req.ServiceID, req.ActionID)
	if !ok || len(req.Input) != len(action.In) {
		writeError(w, http.StatusForbidden, "ACTION_NOT_SUPPORTED", "The action or its input list is not declared by the device specification")
		return
	}
	for index, inputPropertyID := range action.In {
		ref, exists := spec.FindProperty(req.ServiceID, inputPropertyID)
		if !exists {
			writeError(w, http.StatusForbidden, "ACTION_INPUT_UNAVAILABLE", "An action input is not declared in the MIoT service")
			return
		}
		if err := ref.Property.ValidateValue(req.Input[index]); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_ACTION_INPUT", err.Error())
			return
		}
	}
	result, err := client.InvokeAction(r.Context(), config.AccessToken, config.UserID,
		fmt.Sprintf("%s.%d.%d", record.ExternalID, req.ServiceID, req.ActionID), req.Input)
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_ACTION_ERROR", "The Xiaomi device did not accept the action")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *XiaomiCloudHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Topic string `json:"topic"`
		IDs []string `json:"ids"`
	}
	if !decodeLimitedJSON(w, r, &req, 16*1024) { return }
	config, client, err := h.ensureFreshToken(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, "XIAOMI_NOT_CONNECTED", "Authorize Xiaomi Home before subscribing to events")
		return
	}
	callbackURL := config.PublicBaseURL + "/api/v1/integrations/xiaomi-home/cloud/events/" + config.EventSecret
	result, err := client.Subscribe(r.Context(), config.AccessToken, config.UserID, callbackURL, req.Topic, req.IDs, map[string]any{"secret": config.EventSecret})
	if err != nil {
		writeError(w, http.StatusBadGateway, "XIAOMI_SUBSCRIPTION_ERROR", "Could not subscribe to Xiaomi MIoT events")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *XiaomiCloudHandler) EventCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")
	secret := chi.URLParam(r, "secret")
	if len(secret) != 64 {
		http.Error(w, "Invalid Xiaomi event callback", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	var callback struct {
		Topic string `json:"topic"`
		OID string `json:"oid"`
		CustomData map[string]any `json:"custom-data"`
		Properties []struct {
			PID string `json:"pid"`
			Value any `json:"value"`
			Status int `json:"status"`
		} `json:"properties"`
		Events []struct {
			EID string `json:"eid"`
			Arguments []any `json:"arguments"`
		} `json:"events"`
		Devices []struct {
			DID string `json:"did"`
			Online bool `json:"online"`
		} `json:"devices"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&callback); err != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid Xiaomi event payload", http.StatusBadRequest)
		return
	}
	config, err := h.loadConfig(r.Context())
	if err != nil || subtle.ConstantTimeCompare([]byte(secret), []byte(config.EventSecret)) != 1 {
		http.Error(w, "Invalid Xiaomi event callback", http.StatusNotFound)
		return
	}
	provided, _ := callback.CustomData["secret"].(string)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(config.EventSecret)) != 1 {
		http.Error(w, "Invalid Xiaomi subscription", http.StatusForbidden)
		return
	}
	if h.bus == nil {
		http.Error(w, "Domus event bus unavailable", http.StatusServiceUnavailable)
		return
	}
	eventKey := callback.OID
	if eventKey == "" {
		encoded, _ := json.Marshal(callback)
		digest := sha256.Sum256(encoded)
		eventKey = hex.EncodeToString(digest[:])
	}
	if !h.acceptEvent(eventKey) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	publish := func(eventType events.EventType, payload map[string]any) error {
		return h.bus.Publish(r.Context(), events.Event{ID: uuid.NewString(), Type: eventType, OccurredAt: time.Now().UTC(), Source: "xiaomi_home", Payload: payload})
	}
	publishErr := error(nil)
	publishOne := func(eventType events.EventType, payload map[string]any) {
		if err := publish(eventType, payload); err != nil && publishErr == nil { publishErr = err }
	}
	switch callback.Topic {
	case "properties-changed":
		_, client, tokenErr := h.ensureFreshToken(r.Context())
		if tokenErr != nil {
			h.forgetEvent(eventKey)
			http.Error(w, "Xiaomi cloud token is unavailable", http.StatusServiceUnavailable)
			return
		}
		for _, item := range callback.Properties {
			parts := strings.Split(item.PID, ".")
			if len(parts) != 3 { continue }
			record, err := h.devices.GetByExternalID(r.Context(), xiaomiCloudIntegrationID, parts[0])
			if err != nil { continue }
			var metadata struct { MIoTType string `json:"miot_type"` }
			if json.Unmarshal([]byte(record.Metadata), &metadata) != nil { continue }
			spec, err := client.FetchDeviceSpec(r.Context(), metadata.MIoTType)
			if err != nil { continue }
			h.updateEntityState(r.Context(), record, spec, item.PID, item.Value, item.Status)
			publishOne(events.XiaomiPropertyChanged, map[string]any{"did": parts[0], "pid": item.PID, "value": item.Value, "status": item.Status})
		}
	case "events-occured":
		for _, item := range callback.Events {
			publishOne(events.XiaomiMIoTEvent, map[string]any{"eid": item.EID, "arguments": item.Arguments})
		}
	case "devices-status-changed":
		for _, item := range callback.Devices {
			record, err := h.devices.GetByExternalID(r.Context(), xiaomiCloudIntegrationID, item.DID)
			if err != nil { continue }
			_ = h.devices.SetOnline(r.Context(), record.ID, item.Online)
			eventType := events.DeviceOffline
			if item.Online { eventType = events.DeviceOnline }
			publishOne(eventType, map[string]any{"device_id": record.ID, "did": item.DID, "online": item.Online})
			publishOne(events.XiaomiDeviceStatusChanged, map[string]any{"device_id": record.ID, "did": item.DID, "online": item.Online})
		}
	default:
		publishOne(events.WebhookReceived, map[string]any{"source": "xiaomi_home", "topic": callback.Topic, "oid": callback.OID})
	}
	if publishErr != nil {
		h.forgetEvent(eventKey)
		http.Error(w, "Domus could not queue Xiaomi event", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ControlEntity lets MIoT-backed entities use the standard Domus command path.
func (h *XiaomiCloudHandler) ControlEntity(ctx context.Context, entityKey, service string, params map[string]any) error {
	ent, err := h.entities.GetByKey(ctx, entityKey)
	if err != nil { return err }
	var attrs map[string]any
	if err := json.Unmarshal([]byte(ent.Attributes), &attrs); err != nil || attrs["source"] != "xiaomi_home" {
		return errors.New("xiaomi: entity is not a cloud MIoT entity")
	}
	siid, ok1 := numericInt(attrs["siid"])
	piid, ok2 := numericInt(attrs["piid"])
	if !ok1 || !ok2 { return errors.New("xiaomi: entity has invalid MIoT identifiers") }
	config, client, err := h.ensureFreshToken(ctx)
	if err != nil { return err }
	record, err := h.devices.GetByID(ctx, ent.DeviceID)
	if err != nil { return err }
	var metadata map[string]any
	if err := json.Unmarshal([]byte(record.Metadata), &metadata); err != nil { return err }
	miotType, _ := metadata["miot_type"].(string)
	spec, err := client.FetchDeviceSpec(ctx, miotType)
	if err != nil { return err }

	if service == "miot_action" {
		actionService, ok1 := numericInt(params["siid"])
		actionID, ok2 := numericInt(params["aiid"])
		input, ok3 := params["input"].([]any)
		if !ok1 || !ok2 || !ok3 { return errors.New("xiaomi: miot_action requires siid, aiid and input") }
		action, ok := spec.FindAction(actionService, actionID)
		if !ok || len(action.In) != len(input) { return errors.New("xiaomi: action is unsupported or its input count is invalid") }
		for i, propertyID := range action.In {
			ref, exists := spec.FindProperty(actionService, propertyID)
			if !exists { return errors.New("xiaomi: action input is absent from the MIoT specification") }
			if err := ref.Property.ValidateValue(input[i]); err != nil { return err }
		}
		_, err := client.InvokeAction(ctx, config.AccessToken, config.UserID, fmt.Sprintf("%s.%d.%d", record.ExternalID, actionService, actionID), input)
		return err
	}

	ref, ok := spec.FindProperty(siid, piid)
	if !ok || !ref.Property.CanWrite() { return errors.New("xiaomi: property is not writable") }
	var value any
	switch service {
	case "turn_on":
		value = true
	case "turn_off":
		value = false
	case "toggle":
		var current any
		if err := json.Unmarshal([]byte(ent.State), &current); err != nil { return errors.New("xiaomi: current entity state is invalid") }
		flag, ok := current.(bool)
		if !ok { return errors.New("xiaomi: toggle is only available for boolean MIoT properties") }
		value = !flag
	case "set_value", "set_temperature", "set_percentage":
		value = params["value"]
		if value == nil { value = params["temperature"] }
	case "select_option":
		value = params["option"]
	default:
		return fmt.Errorf("xiaomi: unsupported entity command %q", service)
	}
	if value == nil { return errors.New("xiaomi: command value is required") }
	if err := ref.Property.ValidateValue(value); err != nil { return err }
	pid := fmt.Sprintf("%s.%d.%d", record.ExternalID, siid, piid)
	results, err := client.SetProperties(ctx, config.AccessToken, config.UserID, []xiaomi_home.PropertyUpdate{{PID: pid, Value: value}})
	if err != nil { return err }
	if len(results) == 0 || results[0].Status != 0 {
		return errors.New("xiaomi: device rejected the property update")
	}
	h.updateEntityState(ctx, record, spec, pid, value, 0)
	return nil
}

func numericInt(value any) (int, bool) {
	switch n := value.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		if n == float64(int(n)) { return int(n), true }
	case json.Number:
		parsed, err := n.Int64()
		return int(parsed), err == nil
	}
	return 0, false
}

func (h *XiaomiCloudHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	config, err := h.loadConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_CONFIGURED", "Xiaomi Open Platform is not configured")
		return
	}
	config.AccessToken, config.RefreshToken, config.MacKey, config.MacAlgorithm = "", "", "", ""
	config.UserID, config.Nickname, config.OAuthState = "", "", ""
	config.ExpiresAt = time.Time{}
	if err := h.saveConfig(r.Context(), *config, "disconnected"); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Could not disconnect Xiaomi account")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"connected": false})
}

func (h *XiaomiCloudHandler) getDeviceContext(r *http.Request) (*xiaomiCloudConfig, *xiaomi_home.CloudClient, *device.Device, *xiaomi_home.MIoTDeviceSpec, error) {
	config, client, err := h.ensureFreshToken(r.Context())
	if err != nil { return nil, nil, nil, nil, err }
	record, err := h.devices.GetByExternalID(r.Context(), xiaomiCloudIntegrationID, chi.URLParam(r, "did"))
	if err != nil { return nil, nil, nil, nil, err }
	var metadata struct { MIoTType string `json:"miot_type"` }
	if err := json.Unmarshal([]byte(record.Metadata), &metadata); err != nil { return nil, nil, nil, nil, err }
	spec, err := client.FetchDeviceSpec(r.Context(), metadata.MIoTType)
	if err != nil { return nil, nil, nil, nil, err }
	return config, client, record, spec, nil
}

func (h *XiaomiCloudHandler) syncDeviceSpec(ctx context.Context, client *xiaomi_home.CloudClient, config *xiaomiCloudConfig, cloudDevice xiaomi_home.CloudDevice, record *device.Device) error {
	spec, err := client.FetchDeviceSpec(ctx, cloudDevice.Type)
	if err != nil { return err }
	pids := make([]string, 0, 64)
	for _, ref := range spec.Properties() {
		if !ref.Property.CanRead() && !ref.Property.CanWrite() { continue }
		domain, writable := xiaomi_home.MIoTEntityDomain(ref.Property)
		key := xiaomiEntityKey(cloudDevice.DID, ref.ServiceID, ref.Property.IID, domain)
		attributes, _ := json.Marshal(map[string]any{
			"source": "xiaomi_home", "miot_did": cloudDevice.DID,
			"miot_type": cloudDevice.Type, "siid": ref.ServiceID,
			"piid": ref.Property.IID, "format": ref.Property.Format,
			"access": ref.Property.Access, "value_range": ref.Property.ValueRange,
			"value_list": ref.Property.ValueList,
		})
		if _, lookupErr := h.entities.GetByKey(ctx, key); errors.Is(lookupErr, entity.ErrEntityNotFound) {
			item := &entity.Entity{
				ID: uuid.NewString(), DeviceID: record.ID, EntityKey: key,
				Domain: domain, Name: ref.Property.Description,
				UnitOfMeasurement: xiaomi_home.MIoTUnitName(ref.Property.Unit),
				Writable: writable, State: "null", Attributes: string(attributes),
				Available: cloudDevice.Online,
				LastChangedAt: time.Now().UTC(), LastUpdatedAt: time.Now().UTC(),
			}
			if createErr := h.entities.Create(ctx, item); createErr != nil { return createErr }
		} else if lookupErr != nil { return lookupErr }
		if ref.Property.CanRead() && len(pids) < 64 {
			pids = append(pids, fmt.Sprintf("%s.%d.%d", cloudDevice.DID, ref.ServiceID, ref.Property.IID))
		}
	}
	if len(pids) > 0 {
		values, err := client.GetProperties(ctx, config.AccessToken, config.UserID, pids)
		if err == nil {
			for _, item := range values { h.updateEntityState(ctx, record, spec, item.PID, item.Value, item.Status) }
		}
	}
	return nil
}

func (h *XiaomiCloudHandler) updateEntityState(ctx context.Context, record *device.Device, spec *xiaomi_home.MIoTDeviceSpec, pid string, value any, status int) {
	parts := strings.Split(pid, ".")
	if len(parts) != 3 || parts[0] != record.ExternalID { return }
	siid, e1 := strconv.Atoi(parts[1])
	piid, e2 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil { return }
	ref, ok := spec.FindProperty(siid, piid)
	if !ok { return }
	domain, _ := xiaomi_home.MIoTEntityDomain(ref.Property)
	key := xiaomiEntityKey(record.ExternalID, siid, piid, domain)
	current, err := h.entities.GetByKey(ctx, key)
	if err != nil { return }
	encoded, err := json.Marshal(value)
	if err != nil { return }
	attrs := map[string]any{}
	_ = json.Unmarshal([]byte(current.Attributes), &attrs)
	attrs["miot_status"] = status
	attrsJSON, _ := json.Marshal(attrs)
	if status == 0 {
		_ = h.entities.UpdateState(ctx, current.ID, string(encoded), string(attrsJSON), true, time.Now().UTC())
		if h.bus != nil {
			_ = h.bus.Publish(ctx, events.Event{
				ID: uuid.NewString(), Type: events.EntityStateChanged, OccurredAt: time.Now().UTC(),
				Source: "xiaomi_home", Payload: map[string]any{"entity_key": key, "state": value, "attributes": attrs},
			})
		}
	} else {
		_ = h.entities.UpdateState(ctx, current.ID, current.State, string(attrsJSON), false, time.Now().UTC())
	}
}

func (h *XiaomiCloudHandler) ensureFreshToken(ctx context.Context) (*xiaomiCloudConfig, *xiaomi_home.CloudClient, error) {
	config, err := h.loadConfig(ctx)
	if err != nil { return nil, nil, err }
	if config.AccessToken == "" || config.RefreshToken == "" || config.UserID == "" {
		return nil, nil, errors.New("xiaomi: account is not connected")
	}
	client, err := h.newClient(config)
	if err != nil { return nil, nil, err }
	if !config.ExpiresAt.IsZero() && time.Until(config.ExpiresAt) < 2*time.Minute {
		token, err := client.RefreshToken(ctx, config.RefreshToken)
		if err != nil {
			_ = h.integrations.SetStatus(ctx, xiaomiCloudIntegrationID, "error", "Xiaomi OAuth token refresh failed")
			return nil, nil, err
		}
		config.AccessToken, config.RefreshToken = token.AccessToken, token.RefreshToken
		config.MacKey, config.MacAlgorithm, config.ExpiresAt = token.MacKey, token.MacAlgorithm, token.ExpiresAt
		if err := h.saveConfig(ctx, *config, "connected"); err != nil { return nil, nil, err }
	}
	return config, client, nil
}

func (h *XiaomiCloudHandler) loadClient(ctx context.Context) (*xiaomiCloudConfig, *xiaomi_home.CloudClient, error) {
	config, err := h.loadConfig(ctx)
	if err != nil { return nil, nil, err }
	client, err := h.newClient(config)
	return config, client, err
}

func (h *XiaomiCloudHandler) newClient(config *xiaomiCloudConfig) (*xiaomi_home.CloudClient, error) {
	return xiaomi_home.NewCloudClient(xiaomi_home.CloudConfig{
		ClientID: config.ClientID, ClientSecret: config.ClientSecret,
		RedirectURL: config.RedirectURL, Region: config.Region,
	}, nil)
}

func (h *XiaomiCloudHandler) loadConfig(ctx context.Context) (*xiaomiCloudConfig, error) {
	record, err := h.integrations.GetByID(ctx, xiaomiCloudIntegrationID)
	if err != nil { return nil, err }
	plain, err := h.cipher.Decrypt(string(record.ConfigEncrypted))
	if err != nil { return nil, fmt.Errorf("xiaomi: decrypt stored cloud configuration: %w", err) }
	var config xiaomiCloudConfig
	if err := json.Unmarshal([]byte(plain), &config); err != nil { return nil, errors.New("xiaomi: invalid stored cloud configuration") }
	return &config, nil
}

func (h *XiaomiCloudHandler) saveConfig(ctx context.Context, config xiaomiCloudConfig, status string) error {
	encoded, err := json.Marshal(config)
	if err != nil { return err }
	encrypted, err := h.cipher.Encrypt(string(encoded))
	if err != nil { return err }
	record, getErr := h.integrations.GetByID(ctx, xiaomiCloudIntegrationID)
	now := time.Now().UTC()
	if errors.Is(getErr, integration.ErrIntegrationNotFound) {
		record = &integration.Integration{
			ID: xiaomiCloudIntegrationID, Type: "xiaomi_home_cloud", Name: "Xiaomi Home Cloud",
			Enabled: true, Status: status, CreatedAt: now, UpdatedAt: now,
		}
	} else if getErr != nil { return getErr } else {
		record.ConfigEncrypted, record.Name, record.Enabled, record.UpdatedAt = []byte(encrypted), "Xiaomi Home Cloud", true, now
		if err := h.integrations.Update(ctx, record); err != nil { return err }
		return h.integrations.SetStatus(ctx, xiaomiCloudIntegrationID, status, "")
	}
	record.ConfigEncrypted = []byte(encrypted)
	if err := h.integrations.Create(ctx, record); err != nil { return err }
	return h.integrations.SetStatus(ctx, xiaomiCloudIntegrationID, status, "")
}

func (h *XiaomiCloudHandler) forgetEvent(key string) {
	h.mu.Lock()
	delete(h.seenEvents, key)
	h.mu.Unlock()
}

func (h *XiaomiCloudHandler) acceptEvent(key string) bool {
	now := time.Now().UTC()
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, at := range h.seenEvents { if now.Sub(at) > 24*time.Hour { delete(h.seenEvents, id) } }
	if _, exists := h.seenEvents[key]; exists { return false }
	if len(h.seenEvents) >= 10000 {
		var oldestID string
		var oldest time.Time
		for id, at := range h.seenEvents { if oldest.IsZero() || at.Before(oldest) { oldestID, oldest = id, at } }
		delete(h.seenEvents, oldestID)
	}
	h.seenEvents[key] = now
	return true
}

func xiaomiEntityKey(did string, siid, piid int, domain string) string {
	var slug strings.Builder
	for _, ch := range strings.ToLower(did) {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' { slug.WriteRune(ch) } else if slug.Len() > 0 { slug.WriteByte('_') }
	}
	name := strings.Trim(slug.String(), "_")
	if name == "" { name = "device" }
	return fmt.Sprintf("%s.xiaomi_%s_s%d_p%d", domain, name, siid, piid)
}

func validHTTPSBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("not a valid HTTPS base URL")
	}
	return nil
}

func maskIdentifier(value string) string {
	if len(value) <= 4 { return "****" }
	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}
