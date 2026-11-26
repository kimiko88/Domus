package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/user"
	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	domusmw "github.com/domus-hub/domus/internal/interfaces/http/middleware"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
)

// TuyaCloudHandler exposes server-side Tuya OpenAPI integration, device discovery,
// and automatic provisioning into the local-first LAN adapter.
type TuyaCloudHandler struct {
	client    tuya_cloud.Provider
	pluginManager *apppluginruntime.Manager
	tuyaLocal *tuya_local.Adapter
	db        *sql.DB
	cipher    *crypto.EnvelopeCipher
	mu        sync.RWMutex
}

// NewTuyaCloudHandler creates a new TuyaCloudHandler instance.
func NewTuyaCloudHandler(client tuya_cloud.Provider, managers ...*apppluginruntime.Manager) *TuyaCloudHandler {
	h := &TuyaCloudHandler{client: client}
	if len(managers) > 0 { h.pluginManager = managers[0] }
	return h
}

// WithLocalAdapter attaches the local LAN adapter for zero-cloud device handover.
func (h *TuyaCloudHandler) WithLocalAdapter(adapter *tuya_local.Adapter) *TuyaCloudHandler {
	h.tuyaLocal = adapter
	return h
}

// WithStorage attaches database and envelope cipher for persisting Tuya credentials.
func (h *TuyaCloudHandler) WithStorage(db *sql.DB, cipher *crypto.EnvelopeCipher) *TuyaCloudHandler {
	h.db = db
	h.cipher = cipher
	if db == nil {
		return h
	}
	if h.pluginManager != nil {
		if err := h.migrateLegacyConfig(context.Background()); err != nil {
			// The legacy row is retained on every migration failure.
			slog.Error("tuya cloud: could not migrate the legacy encrypted configuration", "error", err)
		}
		return h
	}

	// Compatibility path for installations without the Plugin Manager.
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS tuya_cloud_config (id TEXT PRIMARY KEY, client_id TEXT NOT NULL, secret_enc TEXT NOT NULL, uid TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at INTEGER NOT NULL)")
	if err != nil {
		slog.Error("tuya cloud: could not initialize the legacy configuration table", "error", err)
		return h
	}
	if err := h.loadStoredConfig(context.Background()); err != nil {
		slog.Error("tuya cloud: could not restore the encrypted legacy configuration", "error", err)
	}
	return h
}
func (h *TuyaCloudHandler) loadStoredConfig(ctx context.Context) error {
	if h.db == nil || h.cipher == nil {
		return nil
	}
	var clientID, secretEnc, uid, endpoint string
	var updatedAt int64
	err := h.db.QueryRowContext(ctx, `SELECT client_id, secret_enc, uid, endpoint, updated_at FROM tuya_cloud_config WHERE id = 'default'`).Scan(&clientID, &secretEnc, &uid, &endpoint, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	secret, err := h.cipher.Decrypt(secretEnc)
	if err != nil {
		return fmt.Errorf("decrypt secret: %w", err)
	}

	cfg := tuya_cloud.Config{ClientID: clientID, Secret: secret, BaseURL: endpoint, UID: uid}
	h.mu.RLock()
	current := h.client
	h.mu.RUnlock()
	if current != nil {
		current.SetCredentials(clientID, secret, endpoint, uid)
		return nil
	}
	provider, err := h.providerForConfig(ctx, cfg)
	if err != nil { return err }
	h.mu.Lock()
	if h.client == nil { h.client = provider } else { h.client.SetCredentials(clientID, secret, endpoint, uid) }
	h.mu.Unlock()
	return nil
}


func (h *TuyaCloudHandler) providerForConfig(ctx context.Context, cfg tuya_cloud.Config) (tuya_cloud.Provider, error) {
	if h.pluginManager == nil {
		return tuya_cloud.New(cfg)
	}

	// The legacy Tuya settings page predates persisted plugin instance IDs.
	// Prefer any already-prepared singleton provider, including one created
	// with a custom instance ID from the Plugin Manager UI.
	useAdapter := func(provider tuya_cloud.Provider) (tuya_cloud.Provider, error) {
		if provider == nil {
			return nil, errors.New("tuya cloud: prepared plugin returned a nil provider")
		}
		provider.SetCredentials(cfg.ClientID, cfg.Secret, cfg.BaseURL, cfg.UID)
		return provider, nil
	}

	fallback := func(cause error) (tuya_cloud.Provider, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, err := tuya_cloud.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("tuya cloud: supervised provider failed (%v) and direct client initialization failed: %w", cause, err)
		}
		slog.Warn("tuya cloud: supervised plugin unavailable; using the validated direct OpenAPI client", "error", cause)
		return client, nil
	}

	if adapter, found := h.pluginManager.GetAdapter(tuya_cloud.PluginType); found {
		provider, ok := adapter.(tuya_cloud.Provider)
		if !ok {
			return fallback(errors.New("default plugin instance does not implement the Tuya provider contract"))
		}
		return useAdapter(provider)
	}
	if adapter, found := h.pluginManager.GetAdapterByType(tuya_cloud.PluginType); found {
		provider, ok := adapter.(tuya_cloud.Provider)
		if !ok {
			return fallback(errors.New("prepared plugin instance does not implement the Tuya provider contract"))
		}
		return useAdapter(provider)
	}

	instances, err := h.pluginManager.ListInstances(ctx)
	if err != nil {
		return fallback(fmt.Errorf("inspect saved plugin instances: %w", err))
	}

	// A saved singleton may have a custom ID. Re-prepare that exact ID if its
	// startup restore failed, rather than trying to register a second instance.
	for _, instance := range instances {
		if instance.Type != tuya_cloud.PluginType {
			continue
		}
		if !instance.Enabled {
			return nil, errors.New("tuya cloud: plugin instance is disabled; enable it in the Plugin Manager")
		}
		if adapter, found := h.pluginManager.GetAdapter(instance.ID); found {
			provider, ok := adapter.(tuya_cloud.Provider)
			if !ok {
				return fallback(errors.New("saved plugin instance does not implement the Tuya provider contract"))
			}
			return useAdapter(provider)
		}

		raw, marshalErr := json.Marshal(map[string]any{
			"client_id": cfg.ClientID, "secret": cfg.Secret, "base_url": cfg.BaseURL,
			"uid": cfg.UID, "schema": cfg.Schema,
		})
		if marshalErr != nil {
			return fallback(fmt.Errorf("encode Tuya plugin configuration: %w", marshalErr))
		}
		adapter, prepareErr := h.pluginManager.Prepare(ctx, tuya_cloud.PluginType, instance.ID, raw)
		if prepareErr != nil {
			return fallback(fmt.Errorf("prepare saved instance %q: %w", instance.ID, prepareErr))
		}
		provider, ok := adapter.(tuya_cloud.Provider)
		if !ok {
			return fallback(errors.New("prepared saved instance does not implement the Tuya provider contract"))
		}
		return provider, nil
	}

	// Do not create an unpersisted singleton adapter here. The compatibility
	// settings flow can use a direct client until the Plugin Manager instance is
	// created through its durable encrypted configuration path.
	return fallback(errors.New("no persisted Tuya Cloud plugin instance found"))
}

func (h *TuyaCloudHandler) RegisterRoutes(r chi.Router) {
	r.Get("/config", h.GetConfig)
	r.Post("/config", h.SaveConfig)
	r.Delete("/config", h.DeleteConfig)
	r.Get("/devices", h.ListDevices)
	r.Get("/devices/{deviceID}/functions", h.ListFunctions)
	r.Post("/devices/{deviceID}/commands", h.SendCommands)
	r.Post("/sync-local", h.SyncLocal)
}

func (h *TuyaCloudHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	configured := false
	clientID, uid := "", ""
	endpoint := "https://openapi.tuyaeu.com"
	source := "none"

	if h.pluginManager != nil {
		_, config, err := h.managedTuyaInstance(r.Context())
		if err == nil {
			var decodeErr error
			if clientID, decodeErr = readTuyaConfigString(config, "client_id"); decodeErr != nil {
				writeError(w, http.StatusInternalServerError, "CONFIG_READ_FAILED", "could not read Tuya Cloud configuration")
				return
			}
			secret, secretErr := readTuyaConfigString(config, "secret")
			if secretErr != nil {
				writeError(w, http.StatusInternalServerError, "CONFIG_READ_FAILED", "could not read Tuya Cloud configuration")
				return
			}
			if uid, decodeErr = readTuyaConfigString(config, "uid"); decodeErr != nil {
				writeError(w, http.StatusInternalServerError, "CONFIG_READ_FAILED", "could not read Tuya Cloud configuration")
				return
			}
			savedEndpoint, endpointErr := readTuyaConfigString(config, "base_url")
			if endpointErr != nil {
				writeError(w, http.StatusInternalServerError, "CONFIG_READ_FAILED", "could not read Tuya Cloud configuration")
				return
			}
			if savedEndpoint != "" {
				endpoint = savedEndpoint
			}
			configured = clientID != "" && secret != ""
			source = "plugin_manager"
		} else if !errors.Is(err, errTuyaCloudPluginInstanceMissing) {
			writeError(w, http.StatusInternalServerError, "CONFIG_READ_FAILED", "could not read Tuya Cloud configuration")
			return
		}
	} else {
		h.mu.RLock()
		client := h.client
		h.mu.RUnlock()
		if client != nil {
			cfg := client.Config()
			if cfg.ClientID != "" && cfg.Secret != "" {
				configured = true
				clientID, uid = cfg.ClientID, cfg.UID
				if cfg.BaseURL != "" {
					endpoint = cfg.BaseURL
				}
				source = "environment"
			}
		}
	}

	// Read-only compatibility fallback for an unmigrated legacy row.
	if !configured && h.db != nil {
		var dbClientID, dbUID, dbEndpoint string
		err := h.db.QueryRowContext(r.Context(), "SELECT client_id, uid, endpoint FROM tuya_cloud_config WHERE id='default'").Scan(&dbClientID, &dbUID, &dbEndpoint)
		if err == nil && dbClientID != "" {
			configured = true
			clientID, uid = dbClientID, dbUID
			if dbEndpoint != "" {
				endpoint = dbEndpoint
			}
			source = "database"
		}
	}

	maskedClientID := clientID
	if len(clientID) > 8 {
		maskedClientID = clientID[:4] + "••••••••" + clientID[len(clientID)-4:]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": configured,
		"client_id": clientID,
		"client_id_masked": maskedClientID,
		"uid": uid,
		"endpoint": endpoint,
		"source": source,
	})
}
func (h *TuyaCloudHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req struct {
		ClientID string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		UID string `json:"uid"`
		Endpoint string `json:"endpoint"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid configuration payload")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "unexpected trailing JSON content")
		return
	}

	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	req.UID = strings.TrimSpace(req.UID)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.Endpoint == "" {
		req.Endpoint = "https://openapi.tuyaeu.com"
	}
	if req.ClientID == "" || req.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_id and client_secret are required")
		return
	}

	testClient, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID: req.ClientID, Secret: req.ClientSecret,
		BaseURL: req.Endpoint, UID: req.UID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "CONFIG_ERROR", err.Error())
		return
	}

	enabled := true
	if h.pluginManager != nil {
		enabled, err = h.saveManagedOpenAPIConfig(r.Context(), testClient.Config())
		if err != nil {
			slog.Error("tuya cloud: encrypted plugin configuration update failed", "error", err)
			writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "failed to save configuration in Plugin Manager")
			return
		}
	} else {
		// Legacy fallback for installations without the Plugin Manager.
		if h.cipher != nil && h.db != nil {
			secretEnc, encryptErr := h.cipher.Encrypt(req.ClientSecret)
			if encryptErr != nil {
				writeError(w, http.StatusInternalServerError, "CRYPTO_ERROR", "failed to encrypt secret")
				return
			}
			_, err = h.db.ExecContext(r.Context(), "INSERT INTO tuya_cloud_config(id, client_id, secret_enc, uid, endpoint, updated_at) VALUES('default', ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET client_id=excluded.client_id, secret_enc=excluded.secret_enc, uid=excluded.uid, endpoint=excluded.endpoint, updated_at=excluded.updated_at", req.ClientID, secretEnc, req.UID, req.Endpoint, time.Now().Unix())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "STORAGE_ERROR", "failed to save configuration")
				return
			}
		}
		h.mu.Lock()
		if h.client == nil {
			h.client = testClient
		} else {
			h.client.SetCredentials(req.ClientID, req.ClientSecret, req.Endpoint, req.UID)
		}
		h.mu.Unlock()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"enabled": enabled,
		"message": "Configurazione Tuya Smart Life salvata con successo",
	})
}
func (h *TuyaCloudHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	u := domusmw.GetUser(r.Context())
	if u == nil || u.Role != user.RoleAdmin {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "administrator role required")
		return
	}
	if h.db != nil {
		_, _ = h.db.ExecContext(r.Context(), `DELETE FROM tuya_cloud_config WHERE id = 'default'`)
	}
	h.mu.Lock()
	if h.client != nil {
		h.client.SetCredentials("", "", "", "")
	}
	h.client = nil
	h.mu.Unlock()
	if h.pluginManager != nil {
		if instances, err := h.pluginManager.ListInstances(r.Context()); err == nil {
			for _, instance := range instances {
				if instance.Type == tuya_cloud.PluginType {
					_ = h.pluginManager.Delete(r.Context(), instance.ID)
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Configurazione Tuya Smart Life rimossa",
	})
}

func (h *TuyaCloudHandler) getClient() tuya_cloud.Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}

func containsAnyTuyaCode(codes []string, wanted ...string) bool {
	set := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		set[strings.ToLower(strings.TrimSpace(code))] = struct{}{}
	}
	for _, code := range wanted {
		if _, ok := set[strings.ToLower(code)]; ok {
			return true
		}
	}
	return false
}

// isTuyaClimateSource reports whether a device classified as a sensor exposes
// temperature/humidity telemetry or belongs to a known Tuya thermo-hygrometer
// product category. Generic motion/contact/battery sensors are deliberately excluded.
func isTuyaClimateSource(dev tuya_cloud.Device, category tuya_local.DeviceCategory) bool {
	if category == tuya_local.CategoryClimate {
		return true
	}
	if category != tuya_local.CategorySensor {
		return false
	}
	rawCategory := strings.ToLower(strings.TrimSpace(dev.Category))
	if rawCategory == "wsdcg" || rawCategory == "wsdcgq" || rawCategory == "wsdcgq11lm" ||
		rawCategory == "temperature_sensor" || rawCategory == "humidity_sensor" ||
		rawCategory == "thermo_hygrometer" {
		return true
	}
	name := strings.ToLower(dev.Name + " " + dev.ProductName)
	if strings.Contains(name, "temperature") || strings.Contains(name, "temperatura") ||
		strings.Contains(name, "humidity") || strings.Contains(name, "umid") ||
		strings.Contains(name, "thermometer") || strings.Contains(name, "termometro") ||
		strings.Contains(name, "hygrometer") || strings.Contains(name, "termoigrometro") ||
		strings.Contains(name, "termo-igrometro") || strings.Contains(name, "thermo-hygrometer") {
		return true
	}
	for _, status := range dev.Status {
		code := strings.ToLower(strings.TrimSpace(status.Code))
		if containsAnyTuyaCode([]string{code},
			"va_temperature", "current_temperature", "temp_current", "current_temp", "temperature",
			"va_humidity", "current_humidity", "humidity_value", "humidity") {
			return true
		}
	}
	return false
}

func inferTuyaDeviceCategoryAndTelemetry(dev tuya_cloud.Device) (tuya_local.DeviceCategory, string, string, string) {
	statusCodes := make([]string, 0, len(dev.Status))
	for _, st := range dev.Status {
		statusCodes = append(statusCodes, st.Code)
	}
	rawCategory := strings.ToLower(strings.TrimSpace(dev.Category))
	name := strings.ToLower(dev.Name + " " + dev.ProductName)

	// Tuya product categories are not identical to Domus entity domains.
	// Keep the common thermostat and thermo-hygrometer category codes explicit,
	// then use reported datapoints and descriptive names for vendor-specific codes.
	cat := tuya_local.CategorySwitch
	switch rawCategory {
	case "wk", "climate", "thermostat":
		cat = tuya_local.CategoryClimate
	case "sensor", "wsdcg", "wsdcgq", "wsdcgq11lm", "temperature_sensor", "humidity_sensor", "thermo_hygrometer":
		cat = tuya_local.CategorySensor
	default:
		cat = tuya_local.ParseCategory(rawCategory)
	}

	hasClimateSetpoint := containsAnyTuyaCode(statusCodes,
		"temp_set", "target_temp", "target_temperature", "upper_temp", "set_temp")
	hasClimateMode := containsAnyTuyaCode(statusCodes, "hvac_mode", "work_mode")
	hasSensorTelemetry := containsAnyTuyaCode(statusCodes,
		"va_temperature", "current_temperature", "temp_current", "current_temp", "temperature",
		"va_humidity", "current_humidity", "humidity_value", "humidity")
	isThermostatName := strings.Contains(name, "thermostat") || strings.Contains(name, "termostato") ||
		strings.Contains(name, "climate") || strings.Contains(name, "radiator") || strings.Contains(name, "heater")
	isClimateName := isThermostatName
	isSensorName := strings.Contains(name, "temperature sensor") || strings.Contains(name, "humidity sensor") ||
		strings.Contains(name, "sensore temperatura") || strings.Contains(name, "sensore umid") ||
		strings.Contains(name, "termoigrometro") || strings.Contains(name, "termo-igrometro") ||
		strings.Contains(name, "thermo-hygrometer") || strings.Contains(name, "thermometer") ||
		strings.Contains(name, "hygrometer") || strings.Contains(name, "temperature and humidity")

	switch {
	case rawCategory == "wk" || rawCategory == "climate" || rawCategory == "thermostat" ||
		hasClimateSetpoint || hasClimateMode || isClimateName:
		cat = tuya_local.CategoryClimate
	case rawCategory == "sensor" || rawCategory == "wsdcg" || rawCategory == "wsdcgq" ||
		rawCategory == "wsdcgq11lm" || hasSensorTelemetry || isSensorName:
		cat = tuya_local.CategorySensor
	case cat == tuya_local.CategorySwitch && (strings.Contains(name, "light") ||
		strings.Contains(name, "lamp") || strings.Contains(name, "bulb")):
		cat = tuya_local.CategoryLight
	case cat == tuya_local.CategorySwitch && (strings.Contains(name, "plug") || strings.Contains(name, "socket")):
		cat = tuya_local.CategoryPlug
	}

	entityKey := fmt.Sprintf("switch.tuya_%s", dev.ID)
	switch cat {
	case tuya_local.CategoryLight:
		entityKey = fmt.Sprintf("light.tuya_%s", dev.ID)
	case tuya_local.CategoryClimate:
		entityKey = fmt.Sprintf("climate.tuya_%s", dev.ID)
	case tuya_local.CategorySensor:
		entityKey = fmt.Sprintf("sensor.tuya_%s", dev.ID)
	}

	state := "active"
	attrs := make(map[string]any)
	if cat == tuya_local.CategorySensor {
		if value, ok := tuyaStatusNumber(dev.Status,
			"va_temperature", "current_temperature", "temp_current", "current_temp", "temperature"); ok {
			if value > 70 && value <= 1000 {
				value /= 10
			}
			if value >= -50 && value <= 70 {
				attrs["current_temperature"] = value
				attrs["temperature"] = value
			}
		}
		if value, ok := tuyaStatusNumber(dev.Status,
			"va_humidity", "current_humidity", "humidity_value", "humidity"); ok {
			if value > 100 && value <= 1000 {
				value /= 10
			}
			if value >= 0 && value <= 100 {
				attrs["current_humidity"] = value
				attrs["humidity"] = value
			}
		}
		temperatureSensor := containsAnyTuyaCode(statusCodes,
			"va_temperature", "current_temperature", "temp_current", "current_temp", "temperature") ||
			rawCategory == "wsdcg" || rawCategory == "wsdcgq" || rawCategory == "wsdcgq11lm" ||
			rawCategory == "temperature_sensor" || rawCategory == "thermo_hygrometer" ||
			strings.Contains(name, "temperature") || strings.Contains(name, "temperatura") ||
			strings.Contains(name, "termometro") || strings.Contains(name, "thermometer") ||
			strings.Contains(name, "termoigrometro") || strings.Contains(name, "termo-igrometro") ||
			strings.Contains(name, "thermo-hygrometer")
		humiditySensor := containsAnyTuyaCode(statusCodes,
			"va_humidity", "current_humidity", "humidity_value", "humidity") ||
			rawCategory == "wsdcg" || rawCategory == "wsdcgq" || rawCategory == "wsdcgq11lm" ||
			rawCategory == "humidity_sensor" || rawCategory == "thermo_hygrometer" ||
			strings.Contains(name, "humidity") || strings.Contains(name, "umid")
		if temperatureSensor || humiditySensor {
			attrs["tuya_climate_source"] = true
			switch {
			case temperatureSensor:
				attrs["device_class"] = "temperature"
				attrs["unit_of_measurement"] = "°C"
				attrs["tuya_sensor_kind"] = "temperature"
			case humiditySensor:
				attrs["device_class"] = "humidity"
				attrs["unit_of_measurement"] = "%"
				attrs["tuya_sensor_kind"] = "humidity"
			}
			if humiditySensor && temperatureSensor {
				attrs["tuya_sensor_kind"] = "temperature_humidity"
			}
		}
		attrsBytes, _ := json.Marshal(attrs)
		return cat, entityKey, state, string(attrsBytes)
	}

	if cat != tuya_local.CategoryClimate {
		return cat, entityKey, "", ""
	}

	state = "heat"
	attrs["hvac_mode"] = "heat"
	for _, st := range dev.Status {
		code := strings.ToLower(strings.TrimSpace(st.Code))
		switch code {
		case "switch", "power", "switch_1":
			if b, ok := st.Value.(bool); ok && !b {
				state = "off"
				attrs["hvac_mode"] = "off"
			}
		case "mode", "work_mode", "hvac_mode":
			if m, ok := st.Value.(string); ok && m != "" {
				ml := strings.ToLower(m)
				if ml == "auto" || ml == "automatic" || ml == "program" {
					state = "auto"
					attrs["hvac_mode"] = "auto"
				} else if ml == "off" {
					state = "off"
					attrs["hvac_mode"] = "off"
				} else {
					state = "heat"
					attrs["hvac_mode"] = "heat"
				}
			}
		case "temp_set", "target_temp", "target_temperature", "upper_temp", "set_temp":
			if num, ok := tuyaNumericValue(st.Value); ok {
				if num > 50 {
					num = num / 10.0
				}
				attrs["target_temperature"] = num
				attrs["temperature"] = num
			}
		case "temp_current", "current_temp", "current_temperature", "va_temperature", "lower_temp":
			if num, ok := tuyaNumericValue(st.Value); ok {
				if num > 50 {
					num = num / 10.0
				}
				attrs["current_temperature"] = num
			}
		case "va_humidity", "current_humidity", "humidity_value", "humidity":
			if num, ok := tuyaNumericValue(st.Value); ok && num >= 0 && num <= 100 {
				attrs["current_humidity"] = num
			}
		}
	}

	attrsBytes, _ := json.Marshal(attrs)
	return cat, entityKey, state, string(attrsBytes)
}

func tuyaStatusNumber(statuses []tuya_cloud.Status, codes ...string) (float64, bool) {
	wanted := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		wanted[strings.ToLower(strings.TrimSpace(code))] = struct{}{}
	}
	for _, status := range statuses {
		if _, ok := wanted[strings.ToLower(strings.TrimSpace(status.Code))]; !ok {
			continue
		}
		if value, ok := tuyaNumericValue(status.Value); ok {
			return value, true
		}
	}
	return 0, false
}

func (h *TuyaCloudHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	client := h.getClient()
	if client == nil || client.Config().ClientID == "" {
		writeError(w, http.StatusBadRequest, "TUYA_NOT_CONFIGURED", "Tuya Cloud non configurato. Inserisci le credenziali del progetto e l'UID Smart Life.")
		return
	}

	devices, err := client.Devices(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
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
		dev = fetchMissingTuyaDeviceDetails(r.Context(), client, dev, false)
		cat, entityKey, initState, initAttrs := inferTuyaDeviceCategoryAndTelemetry(dev)
		ed := enrichedDevice{
			Device:           dev,
			InferredCategory: string(cat),
			EntityKey:        entityKey,
			ClimateSource:    isTuyaClimateSource(dev, cat),
		}
		if h.tuyaLocal != nil {
			// Refresh records already configured for local LAN control with current
			// cloud telemetry. Listing devices does not silently enable new devices.
			if h.tuyaLocal.IsDeviceConfigured(dev.ID) {
				localKey := dev.LocalKey
				if localKey == "" {
					for _, existing := range h.tuyaLocal.GetConfiguredDevices() {
						if existing.DeviceID == dev.ID {
							localKey = existing.LocalKey
							break
						}
					}
				}
				if localKey != "" {
					_ = h.tuyaLocal.RegisterAndPersistDevice(r.Context(), tuya_local.DeviceConfig{
						DeviceID:          dev.ID,
						Name:              dev.Name,
						IP:                h.tuyaLocal.GetDeviceIP(dev.ID),
						LocalKey:          localKey,
						Version:           "3.3",
						Category:          cat,
						EntityKey:         entityKey,
						Online:            dev.Online,
						InitialState:      initState,
						InitialAttributes: initAttrs,
					}, dev.Name)
				}
			}
			ed.LocalConfigured = h.tuyaLocal.IsDeviceConfigured(dev.ID)
			ed.LocalActive = h.tuyaLocal.IsDeviceConnected(dev.ID)
			ed.LocalIP = h.tuyaLocal.GetDeviceIP(dev.ID)
		}
		enriched = append(enriched, ed)
	}

	writeJSON(w, http.StatusOK, map[string]any{"devices": enriched})
}

// fetchMissingTuyaDeviceDetails hydrates list responses that omit the product
// category or current status datapoints. The cloud list and detail APIs expose
// different subsets of the same device record.
func fetchMissingTuyaDeviceDetails(ctx context.Context, client tuya_cloud.Provider, dev tuya_cloud.Device, needLocalKey bool) tuya_cloud.Device {
	if client == nil || (dev.Category != "" && len(dev.Status) > 0 && dev.ProductName != "" && (!needLocalKey || dev.LocalKey != "")) {
		return dev
	}
	detailed, err := client.GetDevice(ctx, dev.ID)
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

func (h *TuyaCloudHandler) ListFunctions(w http.ResponseWriter, r *http.Request) {
	client := h.getClient()
	if client == nil || client.Config().ClientID == "" {
		writeError(w, http.StatusBadRequest, "TUYA_NOT_CONFIGURED", "Tuya Cloud non configurato")
		return
	}

	functions, err := client.Functions(r.Context(), chi.URLParam(r, "deviceID"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"functions": functions})
}

// SendCommands routes commands locally over LAN whenever the device has a local_key,
// completely avoiding the cloud for day-to-day operation.
func (h *TuyaCloudHandler) SendCommands(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req struct {
		Commands []tuya_cloud.Command `json:"commands"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid command payload")
		return
	}
	if len(req.Commands) == 0 || len(req.Commands) > 32 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "between 1 and 32 commands are required")
		return
	}

	deviceID := chi.URLParam(r, "deviceID")

	client := h.getClient()
	if h.tuyaLocal != nil && h.tuyaLocal.IsDeviceConfigured(deviceID) && client != nil && client.Config().ClientID != "" {
		functions, err := client.Functions(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
			return
		}
		functionByCode := make(map[string]tuya_cloud.Function, len(functions))
		for _, fn := range functions {
			functionByCode[fn.Code] = fn
		}

		localCommands := make([]struct {
			command string
			params map[string]any
		}, 0, len(req.Commands))
		localSupported := true
		for _, cmd := range req.Commands {
			fn, allowed := functionByCode[cmd.Code]
			if !allowed {
				writeError(w, http.StatusBadRequest, "UNSUPPORTED_COMMAND", fmt.Sprintf("Tuya function %q is not declared for this device", cmd.Code))
				return
			}
			code := strings.ToLower(cmd.Code)
			localCmd := ""
			params := make(map[string]any)
			switch {
			case (strings.Contains(code, "switch") || strings.Contains(code, "power")):
				if value, ok := cmd.Value.(bool); ok {
					if value { localCmd = "turn_on" } else { localCmd = "turn_off" }
				}
			case isLocalTuyaSetpointFunction(code):
				if value, ok := tuyaNumericValue(cmd.Value); ok {
					var spec struct { Scale int `json:"scale"` }
					_ = json.Unmarshal([]byte(fn.Values), &spec)
					if spec.Scale < 0 || spec.Scale > 4 {
						writeError(w, http.StatusBadRequest, "INVALID_COMMAND_VALUE", "unsupported temperature scale")
						return
					}
					temperature := value / math.Pow10(spec.Scale)
					localCmd = "set_temperature"
					params["temperature"] = temperature
					params["scale"] = spec.Scale
				}
			case code == "mode" || code == "work_mode" || strings.HasSuffix(code, "_mode"):
				switch value := cmd.Value.(type) {
				case string:
					if strings.EqualFold(value, "auto") || strings.EqualFold(value, "automatic") || strings.EqualFold(value, "manual") {
						localCmd = "set_mode"
						params["mode"] = value
					}
				case float64:
					if value == 0 || value == 1 {
						localCmd = "set_dp"
						params["dp_id"], params["value"] = 4, value
					}
				}
			}
			if localCmd == "" {
				localSupported = false
				break
			}
			localCommands = append(localCommands, struct {
				command string
				params map[string]any
			}{command: localCmd, params: params})
		}

		// Only use the local adapter when every requested function has a safe,
		// known DPS mapping. Unknown device functions remain available through Cloud API.
		if localSupported {
			deviceResponse := make(map[string]any)
			for _, localCmd := range localCommands {
				response, err := h.tuyaLocal.SendCommand(r.Context(), deviceID, localCmd.command, localCmd.params)
				if err != nil {
					if response != nil {
						writeJSON(w, http.StatusBadGateway, map[string]any{
							"success": false, "confirmed": false, "transport": "lan_local",
							"code": "TUYA_LOCAL_ERROR", "message": err.Error(), "device_response": response,
						})
						return
					}
					writeError(w, http.StatusBadGateway, "TUYA_LOCAL_ERROR", fmt.Sprintf("local LAN command was not confirmed: %v", err))
					return
				}
				for dp, value := range response {
					deviceResponse[dp] = value
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "confirmed": true, "transport": "lan_local", "device_response": deviceResponse})
			return
		}
	}

	if client == nil || client.Config().ClientID == "" {
		writeError(w, http.StatusBadRequest, "TUYA_NOT_CONFIGURED", "Tuya Cloud non configurato")
		return
	}

	// Use the Cloud API for declared functions without a safe LocalTuya DPS mapping.
	if err := client.SendCommands(r.Context(), deviceID, req.Commands); err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "transport": "cloud"})
}

// SyncLocal fetches all devices and their local_key credentials from the cloud API
// and transfers them to the Tuya Local adapter for exclusive offline LAN control.
func (h *TuyaCloudHandler) SyncLocal(w http.ResponseWriter, r *http.Request) {
	if h.tuyaLocal == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "tuya local adapter not initialized")
		return
	}

	// A present device_ids array is an explicit allow-list. An omitted array
	// keeps the legacy behaviour of provisioning every locally controllable
	// device plus climate-capable sensors as read-only cloud entities.
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}
	hasSelection := false
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid Tuya device selection payload")
			return
		}
		hasSelection = req.DeviceIDs != nil
	}
	selected := make(map[string]struct{}, len(req.DeviceIDs))
	for _, id := range req.DeviceIDs {
		id = strings.TrimSpace(id)
		if id == "" || len(id) > 128 || strings.ContainsAny(id, "/?#") {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid device ID in Tuya selection")
			return
		}
		selected[id] = struct{}{}
	}

	client := h.getClient()
	if client == nil || client.Config().ClientID == "" {
		writeError(w, http.StatusBadRequest, "TUYA_NOT_CONFIGURED", "Tuya Cloud non configurato. Inserisci le credenziali del progetto e l'UID Smart Life.")
		return
	}

	devices, err := client.Devices(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "TUYA_CLOUD_ERROR", fmt.Sprintf("failed to list cloud devices: %v", err))
		return
	}

	if hasSelection {
		known := make(map[string]struct{}, len(devices))
		for _, dev := range devices {
			known[dev.ID] = struct{}{}
		}
		for id := range selected {
			if _, ok := known[id]; !ok {
				writeError(w, http.StatusBadRequest, "UNKNOWN_DEVICE", fmt.Sprintf("device %q does not belong to the configured Tuya account", id))
				return
			}
		}
		// Only remove devices from the local-control allow-list. Cloud-only
		// temperature/humidity source entities remain available to Climate & History.
		for _, cfg := range h.tuyaLocal.GetConfiguredDevices() {
			if _, belongsToAccount := known[cfg.DeviceID]; belongsToAccount {
				if _, keep := selected[cfg.DeviceID]; !keep {
					if err := h.tuyaLocal.UnregisterDevice(r.Context(), cfg.DeviceID); err != nil {
						writeError(w, http.StatusInternalServerError, "LOCAL_DEVICE_REMOVE_FAILED", "failed to remove a deselected LocalTuya device")
						return
					}
				}
			}
		}
	}

	synced := make([]tuya_local.DeviceConfig, 0, len(devices))
	localControlCount := 0
	climateSourceCount := 0
	for _, dev := range devices {
		if hasSelection {
			if _, keep := selected[dev.ID]; !keep {
				continue
			}
		}

		dev = fetchMissingTuyaDeviceDetails(r.Context(), client, dev, true)
		cat, entityKey, initState, initAttrs := inferTuyaDeviceCategoryAndTelemetry(dev)
		climateSource := isTuyaClimateSource(dev, cat)
		localKey := strings.TrimSpace(dev.LocalKey)
		if localKey == "" {
			// Reuse a Local Key already stored by this Domus installation when
			// Tuya omits it from a later account/device-list response.
			for _, existing := range h.tuyaLocal.GetConfiguredDevices() {
				if existing.DeviceID == dev.ID {
					localKey = existing.LocalKey
					break
				}
			}
		}

		// Tuya does not guarantee a Local Key for every product or account
		// permission package. Temperature/humidity sensors and thermostats must
		// still become visible and collect Cloud telemetry without LAN control.
		if localKey == "" {
			if !climateSource {
				continue
			}
			cfg := tuya_local.DeviceConfig{
				DeviceID:          dev.ID,
				Name:              dev.Name,
				Version:           "3.3",
				Category:          cat,
				EntityKey:         entityKey,
				IntegrationID:     "tuya_cloud",
				Online:            dev.Online,
				InitialState:      initState,
				InitialAttributes: initAttrs,
			}
			if err := h.tuyaLocal.ProvisionCloudDevice(r.Context(), cfg, dev.Name); err != nil {
				continue
			}
			synced = append(synced, cfg)
			climateSourceCount++
			continue
		}

		cfg := tuya_local.DeviceConfig{
			DeviceID:          dev.ID,
			Name:              dev.Name,
			IP:                "", // only trusted LAN discovery can provide this
			LocalKey:          localKey,
			Version:           "3.3",
			Category:          cat,
			EntityKey:         entityKey,
			Online:            dev.Online,
			InitialState:      initState,
			InitialAttributes: initAttrs,
		}
		if err := h.tuyaLocal.RegisterAndPersistDevice(r.Context(), cfg, dev.Name); err != nil {
			continue
		}
		cfg.IP = h.tuyaLocal.GetDeviceIP(dev.ID)
		synced = append(synced, cfg)
		localControlCount++
		if climateSource {
			climateSourceCount++
		}
	}

	message := fmt.Sprintf("%d dispositivi sincronizzati: %d con controllo LAN, %d sorgenti climatiche", len(synced), localControlCount, climateSourceCount)
	writeJSON(w, http.StatusOK, map[string]any{
		"success":              true,
		"synced_count":         len(synced),
		"local_control_count":  localControlCount,
		"climate_sources_count": climateSourceCount,
		"message":              message,
		"devices":               synced,
	})
}

func isLocalTuyaSetpointFunction(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "temp_set", "target_temp", "target_temperature":
		return true
	default:
		return false
	}
}

func tuyaNumericValue(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
