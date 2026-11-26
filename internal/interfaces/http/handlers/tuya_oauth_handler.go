package handlers

import (
	"context"
	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TuyaOAuthHandler struct {
	db           *sql.DB
	pluginManager *apppluginruntime.Manager
	cipher       *crypto.EnvelopeCipher
	client       *tuya_cloud.OAuthClient
	tuyaLocal    *tuya_local.Adapter
	clientID     string
	authorizeURL string
	callbackURL  string
	refreshMu    sync.Mutex
	configMu     sync.Mutex
}

func NewTuyaOAuthHandler(db *sql.DB, cipher *crypto.EnvelopeCipher, client *tuya_cloud.OAuthClient, clientID, authorizeURL, callbackURL string, managers ...*apppluginruntime.Manager) (*TuyaOAuthHandler, error) {
	if db == nil || cipher == nil || client == nil || strings.TrimSpace(clientID) == "" {
		return nil, errors.New("tuya oauth: database, encryption, client and client ID are required")
	}
	authURL, err := url.Parse(authorizeURL)
	if err != nil || authURL.Scheme != "https" || authURL.Host == "" || authURL.User != nil || authURL.Fragment != "" {
		return nil, errors.New("tuya oauth: authorization H5 URL must be absolute HTTPS")
	}
	cb, err := url.Parse(callbackURL)
	if err != nil || cb.Scheme != "https" || cb.Host == "" || cb.User != nil || cb.RawQuery != "" || cb.Fragment != "" {
		return nil, errors.New("tuya oauth: callback URL must be absolute HTTPS without query or fragment")
	}
	h := &TuyaOAuthHandler{db: db, cipher: cipher, client: client, clientID: clientID, authorizeURL: authorizeURL, callbackURL: callbackURL}
	if len(managers) > 0 {
		h.pluginManager = managers[0]
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tuya_user_oauth_states (state TEXT PRIMARY KEY, user_id TEXT NOT NULL, expires_at INTEGER NOT NULL)`)
	if err != nil {
		return nil, fmt.Errorf("tuya oauth: initialize state storage: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tuya_user_oauth_connections (user_id TEXT PRIMARY KEY, tuya_uid TEXT NOT NULL, access_token_enc TEXT NOT NULL, refresh_token_enc TEXT NOT NULL, expires_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`)
	if err != nil {
		return nil, fmt.Errorf("tuya oauth: initialize token storage: %w", err)
	}
	return h, nil
}

// WithLocalAdapter attaches the local LAN adapter for local key handover.
func (h *TuyaOAuthHandler) WithLocalAdapter(adapter *tuya_local.Adapter) *TuyaOAuthHandler {
	h.tuyaLocal = adapter
	return h
}

func (h *TuyaOAuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/authorize", h.Start)
	r.Get("/status", h.Status)
	r.Get("/devices", h.Devices)
	r.Post("/disconnect", h.Disconnect)
	r.Post("/sync-local", h.SyncLocal)
}
func (h *TuyaOAuthHandler) RegisterCallback(r chi.Router) {
	r.Get("/integrations/tuya-oauth/callback", h.Callback)
}

func (h *TuyaOAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	if h.pluginManager != nil {
		h.configMu.Lock()
		_, prepareErr := h.ensureManagedPlugin(r.Context())
		h.configMu.Unlock()
		if prepareErr != nil {
			writeError(w, http.StatusServiceUnavailable, "PLUGIN_UNAVAILABLE", "Tuya Cloud plugin instance could not be prepared")
			return
		}
	}
	state := uuid.NewString() + uuid.NewString()
	_, err := h.db.ExecContext(r.Context(), `DELETE FROM tuya_user_oauth_states WHERE expires_at < ?`, time.Now().Unix())
	if err == nil {
		_, err = h.db.ExecContext(r.Context(), `INSERT INTO tuya_user_oauth_states(state,user_id,expires_at) VALUES(?,?,?)`, state, u.ID, time.Now().Add(10*time.Minute).Unix())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "could not create authorization session")
		return
	}
	authURL, err := url.Parse(h.authorizeURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CONFIG_ERROR", "invalid authorization URL")
		return
	}
	q := authURL.Query()
	q.Set("client_id", h.clientID)
	q.Set("callback_url", h.callbackURL)
	q.Set("state", state)
	authURL.RawQuery = q.Encode()
	writeJSON(w, http.StatusOK, map[string]string{"authorization_url": authURL.String()})
}

func (h *TuyaOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || len(state) > 128 || code == "" || len(code) > 2048 {
		http.Error(w, "Autorizzazione Tuya non valida. Torna a Domus e riprova.", http.StatusBadRequest)
		return
	}
	var userID string
	var expires int64
	err := h.db.QueryRowContext(r.Context(), `SELECT user_id,expires_at FROM tuya_user_oauth_states WHERE state=?`, state).Scan(&userID, &expires)
	if err != nil || expires < time.Now().Unix() {
		_, _ = h.db.ExecContext(r.Context(), `DELETE FROM tuya_user_oauth_states WHERE state=?`, state)
		http.Error(w, "Sessione OAuth scaduta o non valida. Torna a Domus e riprova.", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM tuya_user_oauth_states WHERE state=?`, state)
	if err != nil {
		http.Error(w, "Impossibile completare la sessione OAuth.", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		http.Error(w, "Sessione OAuth già utilizzata.", http.StatusBadRequest)
		return
	}
	h.refreshMu.Lock()
	token, err := h.client.ExchangeCode(r.Context(), code)
	if err != nil {
		h.refreshMu.Unlock()
		logTuyaOAuthFailure("exchange_code", err)
		http.Error(w, "Tuya non ha completato l'autorizzazione. Verifica le impostazioni OAuth e riprova.", http.StatusBadGateway)
		return
	}
	err = h.storeToken(r.Context(), userID, token)
	h.refreshMu.Unlock()
	if err != nil {
		logTuyaOAuthFailure("store_token", err)
		http.Error(w, "Token ricevuto ma non è stato possibile salvarlo in modo sicuro.", http.StatusInternalServerError)
		return
	}
	writeOAuthSuccessPage(w)
}

func (h *TuyaOAuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	if h.pluginManager != nil {
		token, expires, err := h.loadManagedToken(r.Context(), u.ID)
		if errors.Is(err, errTuyaOAuthNotConnected) || errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusOK, map[string]any{"connected": false})
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "could not read connection status")
			return
		}
		_ = token
		writeJSON(w, http.StatusOK, map[string]any{"connected": true, "expires_at": time.Unix(expires, 0).UTC()})
		return
	}
	var expires int64
	err := h.db.QueryRowContext(r.Context(), `SELECT expires_at FROM tuya_user_oauth_connections WHERE user_id=?`, u.ID).Scan(&expires)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]any{"connected": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "could not read connection status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected": true, "expires_at": time.Unix(expires, 0).UTC()})
}

func (h *TuyaOAuthHandler) Devices(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	token, err := h.validToken(r.Context(), u.ID)
	if err != nil {
		logTuyaOAuthFailure("prepare_device_request", err)
		writeError(w, http.StatusBadGateway, "TUYA_OAUTH_ERROR", "Smart Life account is not connected or token renewal failed")
		return
	}
	devices, err := h.client.Devices(r.Context(), token)
	if err != nil && isTuyaTokenRejection(err) {
		if refreshed, refreshErr := h.refreshRejectedToken(r.Context(), u.ID, token); refreshErr == nil {
			token = refreshed
			devices, err = h.client.Devices(r.Context(), token)
		} else {
			logTuyaOAuthFailure("refresh_rejected_device_token", refreshErr)
			err = refreshErr
		}
	}
	if err != nil {
		logTuyaOAuthFailure("list_user_devices", err)
		writeError(w, http.StatusBadGateway, "TUYA_OAUTH_ERROR", tuyaOAuthDeviceErrorMessage(err))
		return
	}

	type enrichedDevice struct {
		tuya_cloud.Device
		InferredCategory string `json:"inferred_category"`
		EntityKey        string `json:"entity_key"`
		ClimateSource    bool   `json:"climate_source"`
		LocalConfigured  bool   `json:"local_configured"`
		LocalActive      bool   `json:"local_active"`
		LocalIP          string `json:"local_ip,omitempty"`
	}
	enriched := make([]enrichedDevice, 0, len(devices))
	for _, dev := range devices {
		dev = fetchMissingTuyaOAuthDeviceDetails(r.Context(), h.client, token, dev, false)
		category, entityKey, _, _ := inferTuyaDeviceCategoryAndTelemetry(dev)
		item := enrichedDevice{
			Device: dev,
			InferredCategory: string(category),
			EntityKey: entityKey,
			ClimateSource: isTuyaClimateSource(dev, category),
		}
		if h.tuyaLocal != nil {
			item.LocalConfigured = h.tuyaLocal.IsDeviceConfigured(dev.ID)
			item.LocalActive = h.tuyaLocal.IsDeviceConnected(dev.ID)
			item.LocalIP = h.tuyaLocal.GetDeviceIP(dev.ID)
		}
		enriched = append(enriched, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"devices": enriched})
}

// fetchMissingTuyaOAuthDeviceDetails hydrates Smart Life device summaries that
// omit category, product details, telemetry or a Local Key.
func fetchMissingTuyaOAuthDeviceDetails(ctx context.Context, client *tuya_cloud.OAuthClient, token tuya_cloud.OAuthToken, dev tuya_cloud.Device, needLocalKey bool) tuya_cloud.Device {
	if client == nil || (dev.Category != "" && len(dev.Status) > 0 && dev.ProductName != "" &&
		(!needLocalKey || dev.LocalKey != "")) {
		return dev
	}
	detailed, err := client.GetDevice(ctx, token, dev.ID)
	if err != nil || detailed == nil {
		return dev
	}
	if dev.Name == "" { dev.Name = detailed.Name }
	if dev.LocalKey == "" { dev.LocalKey = detailed.LocalKey }
	if dev.IP == "" { dev.IP = detailed.IP }
	if dev.ProductID == "" { dev.ProductID = detailed.ProductID }
	if dev.ProductName == "" { dev.ProductName = detailed.ProductName }
	if dev.Category == "" { dev.Category = detailed.Category }
	if len(dev.Status) == 0 { dev.Status = detailed.Status }
	if !dev.Online && detailed.Online { dev.Online = true }
	return dev
}

// SyncLocal fetches all Smart Life devices authorized by the user, retrieves their local_key,
// and provisions them in the Tuya Local adapter for direct, offline-first LAN communication.
func (h *TuyaOAuthHandler) SyncLocal(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	if h.tuyaLocal == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "tuya local adapter not initialized")
		return
	}

	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}
	hasSelection := false
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid Smart Life device selection payload")
			return
		}
		hasSelection = req.DeviceIDs != nil
	}
	selected := make(map[string]struct{}, len(req.DeviceIDs))
	for _, id := range req.DeviceIDs {
		id = strings.TrimSpace(id)
		if id == "" || len(id) > 128 || strings.ContainsAny(id, "/?#") {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid device ID in Smart Life selection")
			return
		}
		selected[id] = struct{}{}
	}

	token, err := h.validToken(r.Context(), u.ID)
	if err != nil {
		logTuyaOAuthFailure("sync_local_prepare_token", err)
		writeError(w, http.StatusBadGateway, "TUYA_OAUTH_ERROR", "Smart Life account is not connected")
		return
	}

	devices, err := h.client.Devices(r.Context(), token)
	if err != nil && isTuyaTokenRejection(err) {
		if refreshed, refreshErr := h.refreshRejectedToken(r.Context(), u.ID, token); refreshErr == nil {
			token = refreshed
			devices, err = h.client.Devices(r.Context(), token)
		}
	}
	if err != nil {
		logTuyaOAuthFailure("sync_local_list_devices", err)
		writeError(w, http.StatusBadGateway, "TUYA_OAUTH_ERROR", tuyaOAuthDeviceErrorMessage(err))
		return
	}

	if hasSelection {
		known := make(map[string]struct{}, len(devices))
		for _, dev := range devices { known[dev.ID] = struct{}{} }
		for id := range selected {
			if _, ok := known[id]; !ok {
				writeError(w, http.StatusBadRequest, "UNKNOWN_DEVICE", fmt.Sprintf("device %q does not belong to the connected Smart Life account", id))
				return
			}
		}
	}

	synced := make([]tuya_local.DeviceConfig, 0, len(devices))
	localControlCount := 0
	climateSourceCount := 0
	for _, dev := range devices {
		if hasSelection {
			if _, keep := selected[dev.ID]; !keep { continue }
		}

		dev = fetchMissingTuyaOAuthDeviceDetails(r.Context(), h.client, token, dev, true)
		localKey := strings.TrimSpace(dev.LocalKey)
		if localKey == "" {
			// The list/detail API may no longer return a Local Key even though the
			// adapter has a previously paired and encrypted copy.
			for _, existing := range h.tuyaLocal.GetConfiguredDevices() {
				if existing.DeviceID == dev.ID {
					localKey = existing.LocalKey
					break
				}
			}
		}

		category, entityKey, initialState, initialAttributes := inferTuyaDeviceCategoryAndTelemetry(dev)
		climateSource := isTuyaClimateSource(dev, category)
		if localKey == "" {
			if !climateSource { continue }
			cfg := tuya_local.DeviceConfig{
				DeviceID: dev.ID, Name: dev.Name, Version: "3.3",
				Category: category, EntityKey: entityKey,
				IntegrationID: "tuya_oauth", Online: dev.Online,
				InitialState: initialState, InitialAttributes: initialAttributes,
			}
			if err := h.tuyaLocal.ProvisionCloudDevice(r.Context(), cfg, dev.Name); err != nil {
				continue
			}
			synced = append(synced, cfg)
			climateSourceCount++
			continue
		}

		cfg := tuya_local.DeviceConfig{
			DeviceID: dev.ID, Name: dev.Name, IP: dev.IP, LocalKey: localKey, Version: "3.3",
			Category: category, EntityKey: entityKey, Online: dev.Online,
			InitialState: initialState, InitialAttributes: initialAttributes,
		}
		if err := h.tuyaLocal.RegisterAndPersistDevice(r.Context(), cfg, dev.Name); err != nil {
			continue
		}
		cfg.IP = h.tuyaLocal.GetDeviceIP(dev.ID)
		synced = append(synced, cfg)
		localControlCount++
		if climateSource { climateSourceCount++ }
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"synced_count": len(synced),
		"local_control_count": localControlCount,
		"climate_sources_count": climateSourceCount,
		"message": fmt.Sprintf("%d dispositivi Smart Life sincronizzati: %d con controllo LAN, %d sorgenti climatiche", len(synced), localControlCount, climateSourceCount),
		"devices": synced,
	})
}

func tuyaOAuthDeviceErrorMessage(err error) string {
	var providerErr *tuya_cloud.OAuthAPIError
	if errors.As(err, &providerErr) {
		message := strings.ToLower(strings.TrimSpace(providerErr.Message))
		if providerErr.Code == "1010" && (strings.Contains(message, "scope") || strings.Contains(message, "not allow")) {
			return "Tuya ha completato il collegamento OAuth, ma il progetto non è autorizzato a leggere i dispositivi. Nel progetto Tuya verifica il pacchetto User Permission Package e l'abilitazione dell'API Get User's Device List per l'app autorizzata; poi scollega e ricollega Smart Life."
		}
		if strings.Contains(message, "permission") || strings.Contains(message, "not authorized") {
			return "Tuya ha negato l'accesso ai dispositivi. Verifica le autorizzazioni del progetto e dell'account Smart Life nella console Tuya, poi ripeti l'autorizzazione."
		}
	}
	return "Impossibile recuperare i dispositivi Smart Life. Verifica l'autorizzazione Tuya; se il token è stato revocato, scollega e ricollega l'account."
}

func isTuyaTokenRejection(err error) bool {
	var providerErr *tuya_cloud.OAuthAPIError
	if !errors.As(err, &providerErr) {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(providerErr.Message))
	// Tuya's numeric code 1010 is overloaded/inconsistent across API
	// responses: "token scope not allow" is an authorization failure, not
	// an expired token. The provider message must take precedence.
	if strings.Contains(message, "scope") || strings.Contains(message, "permission") ||
		strings.Contains(message, "not allow") || strings.Contains(message, "not authorized") {
		return false
	}
	if strings.Contains(message, "token invalid") ||
		strings.Contains(message, "invalid token") ||
		strings.Contains(message, "token expired") ||
		strings.Contains(message, "token has expired") {
		return true
	}
	return providerErr.Code == "1011" || providerErr.Code == "1012"
}

// refreshRejectedToken serializes refresh-token rotation. If another request has
// already refreshed the rejected access token, it reuses the newly stored token
// instead of consuming the one-time refresh token a second time.
func (h *TuyaOAuthHandler) refreshRejectedToken(ctx context.Context, userID string, rejected tuya_cloud.OAuthToken) (tuya_cloud.OAuthToken, error) {
	if h.pluginManager != nil {
		h.refreshMu.Lock()
		defer h.refreshMu.Unlock()
		current, _, err := h.loadManagedToken(ctx, userID)
		if err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		if current.AccessToken != rejected.AccessToken {
			return current, nil
		}
		next, err := h.client.Refresh(ctx, current.RefreshToken)
		if err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		if next.UID != current.UID {
			return tuya_cloud.OAuthToken{}, errors.New("tuya oauth: refreshed token UID mismatch")
		}
		if err = h.storeToken(ctx, userID, next); err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		return next, nil
	}
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()

	var uid, accessEnc, refreshEnc string
	var expires int64
	if err := h.db.QueryRowContext(ctx, `SELECT tuya_uid,access_token_enc,refresh_token_enc,expires_at FROM tuya_user_oauth_connections WHERE user_id=?`, userID).Scan(&uid, &accessEnc, &refreshEnc, &expires); err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	access, err := h.cipher.Decrypt(accessEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	refresh, err := h.cipher.Decrypt(refreshEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}

	if access != rejected.AccessToken {
		return tuya_cloud.OAuthToken{UID: uid, AccessToken: access, RefreshToken: refresh, ExpireTime: int64(time.Until(time.Unix(expires, 0)).Seconds())}, nil
	}
	next, err := h.client.Refresh(ctx, refresh)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	if next.UID != uid {
		return tuya_cloud.OAuthToken{}, errors.New("tuya oauth: refreshed token UID mismatch")
	}
	if err = h.storeToken(ctx, userID, next); err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	return next, nil
}

func logTuyaOAuthFailure(operation string, err error) {
	var providerErr *tuya_cloud.OAuthAPIError
	if errors.As(err, &providerErr) {
		slog.Error("Tuya Smart Life OAuth provider request failed",
			"operation", operation,
			"upstream_status", providerErr.HTTPStatus,
			"tuya_code", providerErr.Code,
			"tuya_message", providerErr.Message)
		return
	}
	// Do not log arbitrary transport errors: their URLs may contain refresh tokens.
	slog.Error("Tuya Smart Life OAuth operation failed",
		"operation", operation,
		"error_type", fmt.Sprintf("%T", err))
}

func (h *TuyaOAuthHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	if h.pluginManager != nil {
		if err := h.clearManagedToken(r.Context(), u.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "could not remove connection")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM tuya_user_oauth_connections WHERE user_id=?`, u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "could not remove connection")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *TuyaOAuthHandler) validToken(ctx context.Context, userID string) (tuya_cloud.OAuthToken, error) {
	if h.pluginManager != nil {
		h.refreshMu.Lock()
		defer h.refreshMu.Unlock()
		token, expires, err := h.loadManagedToken(ctx, userID)
		if err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		if time.Until(time.Unix(expires, 0)) > 5*time.Minute {
			token.ExpireTime = int64(time.Until(time.Unix(expires, 0)).Seconds())
			return token, nil
		}
		next, err := h.client.Refresh(ctx, token.RefreshToken)
		if err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		if next.UID != token.UID {
			return tuya_cloud.OAuthToken{}, errors.New("tuya oauth: refreshed token UID mismatch")
		}
		if err = h.storeToken(ctx, userID, next); err != nil {
			return tuya_cloud.OAuthToken{}, err
		}
		return next, nil
	}
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	var uid, accessEnc, refreshEnc string
	var expires int64
	err := h.db.QueryRowContext(ctx, `SELECT tuya_uid,access_token_enc,refresh_token_enc,expires_at FROM tuya_user_oauth_connections WHERE user_id=?`, userID).Scan(&uid, &accessEnc, &refreshEnc, &expires)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	access, err := h.cipher.Decrypt(accessEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	refresh, err := h.cipher.Decrypt(refreshEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	if time.Until(time.Unix(expires, 0)) > 5*time.Minute {
		return tuya_cloud.OAuthToken{UID: uid, AccessToken: access, RefreshToken: refresh, ExpireTime: int64(time.Until(time.Unix(expires, 0)).Seconds())}, nil
	}
	next, err := h.client.Refresh(ctx, refresh)
	if err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	if next.UID != uid {
		return tuya_cloud.OAuthToken{}, errors.New("tuya oauth: refreshed token UID mismatch")
	}
	if err = h.storeToken(ctx, userID, next); err != nil {
		return tuya_cloud.OAuthToken{}, err
	}
	return next, nil
}

func (h *TuyaOAuthHandler) storeToken(ctx context.Context, userID string, token tuya_cloud.OAuthToken) error {
	if h.pluginManager != nil {
		h.configMu.Lock()
		defer h.configMu.Unlock()
		return h.storeManagedToken(ctx, userID, token)
	}
	access, err := h.cipher.Encrypt(token.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := h.cipher.Encrypt(token.RefreshToken)
	if err != nil {
		return err
	}
	_, err = h.db.ExecContext(ctx, `INSERT INTO tuya_user_oauth_connections(user_id,tuya_uid,access_token_enc,refresh_token_enc,expires_at,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET tuya_uid=excluded.tuya_uid,access_token_enc=excluded.access_token_enc,refresh_token_enc=excluded.refresh_token_enc,expires_at=excluded.expires_at,updated_at=excluded.updated_at`, userID, token.UID, access, refresh, time.Now().Add(time.Duration(token.ExpireTime)*time.Second).Unix(), time.Now().Unix())
	return err
}

// writeOAuthSuccessPage confirms completion without redirecting the browser to the
// public callback hostname. Domus itself remains reachable only over the user's
// private network (for example, Tailscale).
func writeOAuthSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="it">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Smart Life collegato — Domus</title>
</head>
<body>
<main>
<h1>Autorizzazione Smart Life completata</h1>
<p>L'account Smart Life è stato collegato a Domus e i token sono stati salvati in modo sicuro.</p>
<p>Puoi chiudere questa scheda e tornare all'interfaccia Domus tramite la tua connessione Tailscale.</p>
</main>
</body>
</html>`))
}
