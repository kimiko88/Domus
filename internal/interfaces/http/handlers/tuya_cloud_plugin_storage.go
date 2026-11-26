package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
)

const legacyTuyaCloudConfigTable = "tuya_cloud_config"

var errTuyaCloudPluginDisabled = errors.New("tuya cloud: plugin instance is disabled in Plugin Manager")

type tuyaCloudOpenAPIConfig struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
	UID      string `json:"uid"`
	Endpoint string `json:"base_url"`
	Schema   string `json:"schema"`
}

func rawJSONString(value string) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func readRawJSONString(config map[string]json.RawMessage, key string) string {
	raw := config[key]
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func (h *TuyaCloudHandler) managedTuyaInstance(ctx context.Context) (apppluginruntime.Instance, bool, error) {
	if h.pluginManager == nil {
		return apppluginruntime.Instance{}, false, errors.New("tuya cloud: Plugin Manager is unavailable")
	}
	instances, err := h.pluginManager.ListInstances(ctx)
	if err != nil {
		return apppluginruntime.Instance{}, false, fmt.Errorf("tuya cloud: list plugin instances: %w", err)
	}
	var found *apppluginruntime.Instance
	for i := range instances {
		if instances[i].Type != tuya_cloud.PluginType {
			continue
		}
		if found != nil {
			return apppluginruntime.Instance{}, false, errors.New("tuya cloud: multiple provider instances found; expected a singleton")
		}
		copy := instances[i]
		found = &copy
	}
	if found == nil {
		return apppluginruntime.Instance{}, false, nil
	}
	return *found, true, nil
}

func (h *TuyaCloudHandler) managedOpenAPIConfig(ctx context.Context, instanceID string) (map[string]json.RawMessage, error) {
	raw, err := h.pluginManager.GetConfig(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("tuya cloud: read encrypted Plugin Manager config: %w", err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("tuya cloud: decode Plugin Manager config: %w", err)
	}
	if config == nil {
		config = make(map[string]json.RawMessage)
	}
	return config, nil
}

func (h *TuyaCloudHandler) bindManagedProvider(instanceID string) error {
	adapter, ok := h.pluginManager.GetAdapter(instanceID)
	if !ok {
		return errors.New("tuya cloud: persisted instance has no prepared runtime adapter")
	}
	provider, ok := adapter.(tuya_cloud.Provider)
	if !ok {
		return fmt.Errorf("tuya cloud: adapter %q does not implement the provider contract", instanceID)
	}
	h.mu.Lock()
	h.client = provider
	h.mu.Unlock()
	return nil
}

func (h *TuyaCloudHandler) persistManagedOpenAPIConfig(ctx context.Context, cfg tuyaCloudOpenAPIConfig) (string, error) {
	if h.pluginManager == nil {
		return "", errors.New("tuya cloud: Plugin Manager is unavailable")
	}
	if _, err := tuya_cloud.New(tuya_cloud.Config{
		ClientID: cfg.ClientID, Secret: cfg.Secret, BaseURL: cfg.Endpoint,
		UID: cfg.UID, Schema: cfg.Schema,
	}); err != nil {
		return "", fmt.Errorf("tuya cloud: invalid OpenAPI configuration: %w", err)
	}

	instance, exists, err := h.managedTuyaInstance(ctx)
	if err != nil {
		return "", err
	}
	config := map[string]json.RawMessage{
		"client_id": rawJSONString(cfg.ClientID),
		"secret": rawJSONString(cfg.Secret),
		"base_url": rawJSONString(cfg.Endpoint),
	}
	if strings.TrimSpace(cfg.UID) != "" {
		config["uid"] = rawJSONString(cfg.UID)
	}
	if strings.TrimSpace(cfg.Schema) != "" {
		config["schema"] = rawJSONString(cfg.Schema)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	if !exists {
		created, err := h.pluginManager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud OpenAPI", encoded, true)
		if err != nil {
			return "", fmt.Errorf("tuya cloud: create managed provider instance: %w", err)
		}
		if err := h.bindManagedProvider(created.ID); err != nil {
			return "", err
		}
		return created.ID, nil
	}
	if !instance.Enabled {
		return "", errTuyaCloudPluginDisabled
	}
	_, err = h.pluginManager.UpdateConfigMutate(ctx, instance.ID, func(current map[string]json.RawMessage) error {
		for key, value := range config {
			current[key] = append(json.RawMessage(nil), value...)
		}
		if _, ok := config["uid"]; !ok {
			delete(current, "uid")
		}
		if _, ok := config["schema"]; !ok {
			delete(current, "schema")
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("tuya cloud: persist OpenAPI config in Plugin Manager: %w", err)
	}
	if err := h.bindManagedProvider(instance.ID); err != nil {
		return "", err
	}
	return instance.ID, nil
}

func (h *TuyaCloudHandler) removeManagedOpenAPIConfig(ctx context.Context) error {
	if h.pluginManager == nil {
		return nil
	}
	instance, exists, err := h.managedTuyaInstance(ctx)
	if err != nil || !exists {
		return err
	}
	_, err = h.pluginManager.UpdateConfigMutate(ctx, instance.ID, func(config map[string]json.RawMessage) error {
		delete(config, "client_id")
		delete(config, "secret")
		delete(config, "uid")
		delete(config, "schema")
		// Keep base_url and all Internal fields: OAuth needs the provider region,
		// and the OAuth token map must survive removal of OpenAPI credentials.
		return nil
	})
	if err != nil {
		return fmt.Errorf("tuya cloud: remove OpenAPI settings: %w", err)
	}
	adapter, ok := h.pluginManager.GetAdapter(instance.ID)
	if !ok {
		return errors.New("tuya cloud: updated instance has no prepared runtime adapter")
	}
	provider, ok := adapter.(tuya_cloud.Provider)
	if !ok {
		return fmt.Errorf("tuya cloud: adapter %q does not implement the provider contract", instance.ID)
	}
	h.mu.Lock()
	h.client = provider
	h.mu.Unlock()
	return nil
}

func (h *TuyaCloudHandler) readManagedOpenAPIConfig(ctx context.Context) (tuyaCloudOpenAPIConfig, bool, string, error) {
	instance, exists, err := h.managedTuyaInstance(ctx)
	if err != nil || !exists {
		return tuyaCloudOpenAPIConfig{}, exists, "", err
	}
	config, err := h.managedOpenAPIConfig(ctx, instance.ID)
	if err != nil {
		return tuyaCloudOpenAPIConfig{}, true, instance.ID, err
	}
	value := tuyaCloudOpenAPIConfig{
		ClientID: readRawJSONString(config, "client_id"),
		Secret: readRawJSONString(config, "secret"),
		UID: readRawJSONString(config, "uid"),
		Endpoint: readRawJSONString(config, "base_url"),
		Schema: readRawJSONString(config, "schema"),
	}
	if value.Endpoint == "" {
		value.Endpoint = "https://openapi.tuyaeu.com"
	}
	if instance.Enabled {
		if adapter, ok := h.pluginManager.GetAdapter(instance.ID); ok {
			if provider, ok := adapter.(tuya_cloud.Provider); ok {
				h.mu.Lock()
				h.client = provider
				h.mu.Unlock()
			}
		}
	}
	return value, true, instance.ID, nil
}

// migrateLegacyTuyaCloudConfig migrates the single legacy row into the
// encrypted Plugin Manager configuration. The old table is dropped only after
// the new record has been persisted or an already-complete managed record has
// been confirmed.
func (h *TuyaCloudHandler) migrateLegacyTuyaCloudConfig(ctx context.Context) error {
	if h.pluginManager == nil || h.db == nil || h.cipher == nil {
		return nil
	}
	var tableCount int
	if err := h.db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
		legacyTuyaCloudConfigTable,
	).Scan(&tableCount); err != nil {
		return fmt.Errorf("tuya cloud: inspect legacy config table: %w", err)
	}
	if tableCount == 0 {
		return nil
	}

	var clientID, secretEncrypted, uid, endpoint string
	var updatedAt int64
	err := h.db.QueryRowContext(ctx,
		"SELECT client_id, secret_enc, uid, endpoint, updated_at FROM tuya_cloud_config WHERE id='default'",
	).Scan(&clientID, &secretEncrypted, &uid, &endpoint, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		_, dropErr := h.db.ExecContext(ctx, "DROP TABLE tuya_cloud_config")
		return dropErr
	}
	if err != nil {
		return fmt.Errorf("tuya cloud: read legacy config: %w", err)
	}
	secret, err := h.cipher.Decrypt(secretEncrypted)
	if err != nil {
		return fmt.Errorf("tuya cloud: decrypt legacy OpenAPI secret: %w", err)
	}
	legacy := tuyaCloudOpenAPIConfig{
		ClientID: clientID, Secret: secret, UID: uid, Endpoint: endpoint,
	}
	if legacy.Endpoint == "" {
		legacy.Endpoint = "https://openapi.tuyaeu.com"
	}

	instance, exists, err := h.managedTuyaInstance(ctx)
	if err != nil {
		return err
	}
	if exists {
		current, err := h.managedOpenAPIConfig(ctx, instance.ID)
		if err != nil {
			return err
		}
		if readRawJSONString(current, "client_id") == "" || readRawJSONString(current, "secret") == "" {
			if !instance.Enabled {
				return errTuyaCloudPluginDisabled
			}
			if _, err := h.persistManagedOpenAPIConfig(ctx, legacy); err != nil {
				return err
			}
		} else if err := h.bindManagedProvider(instance.ID); err != nil {
			return err
		}
	} else {
		if _, err := h.persistManagedOpenAPIConfig(ctx, legacy); err != nil {
			return err
		}
	}

	// A successful save/confirmation means the Plugin Manager is now the source
	// of truth. Remove the legacy row and table only at this point.
	if _, err := h.db.ExecContext(ctx, "DROP TABLE tuya_cloud_config"); err != nil {
		return fmt.Errorf("tuya cloud: drop migrated legacy config table: %w", err)
	}
	_ = updatedAt
	return nil
}

func (h *TuyaCloudHandler) legacyOpenAPIConfigTableExists(ctx context.Context) (bool, error) {
	if h.db == nil {
		return false, nil
	}
	var count int
	err := h.db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
		legacyTuyaCloudConfigTable,
	).Scan(&count)
	return count > 0, err
}

func (h *TuyaCloudHandler) syncClientFromManagedConfig(ctx context.Context) error {
	cfg, exists, id, err := h.readManagedOpenAPIConfig(ctx)
	if err != nil || !exists {
		return err
	}
	if cfg.ClientID == "" || cfg.Secret == "" {
		h.mu.Lock()
		h.client = nil
		h.mu.Unlock()
		return nil
	}
	return h.bindManagedProvider(id)
}

func (h *TuyaCloudHandler) nowUTC() int64 { return time.Now().UTC().Unix() }
