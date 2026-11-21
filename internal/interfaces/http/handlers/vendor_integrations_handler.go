package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/integration"
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
	mu sync.Mutex
}

type imouStoredConfig struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	DataCenter string `json:"data_center"`
	AuthorizationURL string `json:"authorization_url"`
	PublicBaseURL string `json:"public_base_url"`
	CallbackSecret string `json:"callback_secret"`
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
	return &ImouLifeHandler{repo: repo, cipher: cipher}, nil
}

func NewXiaomiHomeHandler(repo integration.Repository, cipher *crypto.EnvelopeCipher) (*XiaomiHomeHandler, error) {
	if repo == nil || cipher == nil {
		return nil, errors.New("xiaomi: integration repository and encryption are required")
	}
	return &XiaomiHomeHandler{repo: repo, cipher: cipher, client: xiaomi_home.NewClient(2*time.Second)}, nil
}

func (h *ImouLifeHandler) RegisterCallback(r chi.Router) {
	r.Get("/integrations/imou-life/callback/{secret}", h.Callback)
}

func (h *ImouLifeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.Status)
	r.Put("/config", h.Configure)
	r.Get("/authorize", h.Authorization)
	r.Get("/devices", h.Devices)
	r.Get("/devices/{deviceID}/stream", h.StreamURL)
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
		"authorization_url":config.AuthorizationURL,
		"message":"Register this exact HTTPS callback URL in the Imou Open Platform project, then start user authorization.",
	})
}

func (h *ImouLifeHandler) Status(w http.ResponseWriter, r *http.Request) {
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
		"connected":config.AccessToken != "" && time.Now().Before(config.ExpiresAt),
		"data_center":config.DataCenter,
		"expires_at":config.ExpiresAt,
		"authorization_url":config.AuthorizationURL,
		"callback_url":config.PublicBaseURL + "/api/v1/integrations/imou-life/callback/" + config.CallbackSecret,
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
	secret := chi.URLParam(r, "secret")
	code := strings.TrimSpace(r.URL.Query().Get("code"))
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

func (h *ImouLifeHandler) Devices(w http.ResponseWriter, r *http.Request) {
	config, client, err := h.ensureFreshToken(r)
	if err != nil {
		writeError(w, http.StatusConflict, "IMOU_NOT_CONNECTED", "Authorize Imou Life before listing devices")
		return
	}
	page := 1
	pageSize := 50
	if value := r.URL.Query().Get("page"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &page); err != nil { page = 0 }
	}
	if value := r.URL.Query().Get("page_size"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &pageSize); err != nil { pageSize = 0 }
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
		if _, err := fmt.Sscanf(value, "%d", &channel); err != nil { channel = -1 }
	}
	if value := r.URL.Query().Get("stream"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &stream); err != nil { stream = -1 }
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

func (h *ImouLifeHandler) RegisterRoutes(r chi.Router) { // kept as an explicit router capability
	r.Get("/status", h.Status)
	r.Put("/config", h.Configure)
	r.Get("/authorize", h.Authorization)
	r.Get("/devices", h.Devices)
	r.Get("/devices/{deviceID}/stream", h.StreamURL)
	r.Post("/disconnect", h.Disconnect)
}

func (h *XiaomiHomeHandler) Status(w http.ResponseWriter, r *http.Request) {
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
		ID:uuid.NewString(),Name:req.Name,IP:req.IP,Token:req.Token,Model:info.Model,Firmware:info.Firmware,
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

func decodeLimitedJSON(w http.ResponseWriter,r *http.Request,dst any,max int64) bool {
	if r.Body == nil { writeError(w,http.StatusBadRequest,"INVALID_JSON","Request body is required"); return false }
	r.Body=http.MaxBytesReader(w,r.Body,max)
	decoder:=json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(dst); err!=nil { writeError(w,http.StatusBadRequest,"INVALID_JSON","Invalid or oversized JSON request"); return false }
	if decoder.Decode(new(any)) == nil { writeError(w,http.StatusBadRequest,"INVALID_JSON","Only one JSON value is allowed"); return false }
	return true
}

func validHTTPSURL(raw string, allowQuery bool) bool {
	u,err:=url.Parse(strings.TrimSpace(raw))
	if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.Fragment!="" { return false }
	if !allowQuery && u.RawQuery!="" { return false }
	return true
}
