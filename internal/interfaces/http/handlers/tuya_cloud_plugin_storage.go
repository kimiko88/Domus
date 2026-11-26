package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
)

var errTuyaCloudPluginInstanceMissing = errors.New("tuya cloud: Plugin Manager instance not found")

func (h *TuyaCloudHandler) managedTuyaInstance(ctx context.Context) (apppluginruntime.Instance, map[string]json.RawMessage, error) {
	if h.pluginManager == nil {
		return apppluginruntime.Instance{}, nil, errors.New("tuya cloud: Plugin Manager is unavailable")
	}
	instances, err := h.pluginManager.ListInstances(ctx)
	if err != nil {
		return apppluginruntime.Instance{}, nil, fmt.Errorf("tuya cloud: list plugin instances: %w", err)
	}
	for _, instance := range instances {
		if instance.Type != tuya_cloud.PluginType {
			continue
		}
		raw, err := h.pluginManager.GetConfig(ctx, instance.ID)
		if err != nil {
			return apppluginruntime.Instance{}, nil, fmt.Errorf("tuya cloud: read encrypted instance configuration: %w", err)
		}
		var config map[string]json.RawMessage
		if err := json.Unmarshal(raw, &config); err != nil {
			return apppluginruntime.Instance{}, nil, fmt.Errorf("tuya cloud: decode instance configuration: %w", err)
		}
		if config == nil {
			config = make(map[string]json.RawMessage)
		}
		return instance, config, nil
	}
	return apppluginruntime.Instance{}, nil, errTuyaCloudPluginInstanceMissing
}

func (h *TuyaCloudHandler) refreshManagedProvider(ctx context.Context, instance apppluginruntime.Instance) error {
	if h.pluginManager == nil {
		return errors.New("tuya cloud: Plugin Manager is unavailable")
	}
	if !instance.Enabled {
		h.mu.Lock()
		h.client = nil
		h.mu.Unlock()
		return nil
	}
	adapter, ok := h.pluginManager.GetAdapter(instance.ID)
	if !ok {
		h.mu.Lock()
		h.client = nil
		h.mu.Unlock()
		return fmt.Errorf("tuya cloud: enabled plugin instance %q has no prepared adapter", instance.ID)
	}
	provider, ok := adapter.(tuya_cloud.Provider)
	if !ok {
		return fmt.Errorf("tuya cloud: plugin instance %q does not implement the provider contract", instance.ID)
	}
	h.mu.Lock()
	h.client = provider
	h.mu.Unlock()
	return nil
}

func (h *TuyaCloudHandler) saveManagedOpenAPIConfig(ctx context.Context, cfg tuya_cloud.Config) (bool, error) {
	if h.pluginManager == nil {
		return false, errors.New("tuya cloud: Plugin Manager is unavailable")
	}
	instance, _, err := h.managedTuyaInstance(ctx)
	if errors.Is(err, errTuyaCloudPluginInstanceMissing) {
		values := map[string]any{
			"client_id": cfg.ClientID,
			"secret": cfg.Secret,
			"base_url": cfg.BaseURL,
		}
		if strings.TrimSpace(cfg.UID) != "" {
			values["uid"] = cfg.UID
		}
		if strings.TrimSpace(cfg.Schema) != "" {
			values["schema"] = cfg.Schema
		}
		raw, marshalErr := json.Marshal(values)
		if marshalErr != nil {
			return false, fmt.Errorf("tuya cloud: encode initial configuration: %w", marshalErr)
		}
		created, createErr := h.pluginManager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud OpenAPI", raw, true)
		if createErr != nil {
			return false, fmt.Errorf("tuya cloud: create encrypted plugin instance: %w", createErr)
		}
		if err := h.refreshManagedProvider(ctx, created); err != nil {
			return created.Enabled, err
		}
		return created.Enabled, nil
	}
	if err != nil {
		return false, err
	}

	updated, err := h.pluginManager.UpdateConfigMutate(ctx, instance.ID, func(config map[string]json.RawMessage) error {
		values := map[string]string{
			"client_id": cfg.ClientID,
			"secret": cfg.Secret,
			"base_url": cfg.BaseURL,
		}
		for key, value := range values {
			encoded, err := json.Marshal(value)
			if err != nil {
				return err
			}
			config[key] = encoded
		}
		if strings.TrimSpace(cfg.UID) == "" {
			delete(config, "uid")
		} else {
			encoded, err := json.Marshal(strings.TrimSpace(cfg.UID))
			if err != nil {
				return err
			}
			config["uid"] = encoded
		}
		// "schema" is intentionally retained: the legacy settings form does not
		// expose it, so an unrelated credential update must not erase it.
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("tuya cloud: update encrypted plugin configuration: %w", err)
	}
	if err := h.refreshManagedProvider(ctx, updated); err != nil {
		return updated.Enabled, err
	}
	return updated.Enabled, nil
}

// deleteManagedOpenAPIConfig clears only Tuya OpenAPI credentials. The same
// integration instance may also own Smart Life OAuth state, which must survive.
func (h *TuyaCloudHandler) deleteManagedOpenAPIConfig(ctx context.Context) error {
	instance, _, err := h.managedTuyaInstance(ctx)
	if errors.Is(err, errTuyaCloudPluginInstanceMissing) {
		h.mu.Lock()
		h.client = nil
		h.mu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	updated, err := h.pluginManager.UpdateConfigMutate(ctx, instance.ID, func(config map[string]json.RawMessage) error {
		delete(config, "client_id")
		delete(config, "secret")
		delete(config, "uid")
		delete(config, "schema")
		if len(config["base_url"]) == 0 {
			config["base_url"], _ = json.Marshal("https://openapi.tuyaeu.com")
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("tuya cloud: remove OpenAPI credentials: %w", err)
	}
	return h.refreshManagedProvider(ctx, updated)
}

// migrateLegacyConfig transfers the old single-row OpenAPI settings record into
// the encrypted plugin instance. The legacy row is deleted only after the new
// configuration has been committed successfully.
func (h *TuyaCloudHandler) migrateLegacyConfig(ctx context.Context) error {
	if h.pluginManager == nil || h.db == nil || h.cipher == nil {
		return nil
	}
	var tableExists int
	if err := h.db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='tuya_cloud_config'",
	).Scan(&tableExists); err != nil {
		return fmt.Errorf("tuya cloud: inspect legacy configuration table: %w", err)
	}
	if tableExists == 0 {
		if instance, _, err := h.managedTuyaInstance(ctx); err == nil {
			return h.refreshManagedProvider(ctx, instance)
		} else if errors.Is(err, errTuyaCloudPluginInstanceMissing) {
			return nil
		} else {
			return err
		}
	}

	var clientID, secretEnc, uid, endpoint string
	err := h.db.QueryRowContext(ctx,
		"SELECT client_id, secret_enc, uid, endpoint FROM tuya_cloud_config WHERE id='default'",
	).Scan(&clientID, &secretEnc, &uid, &endpoint)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("tuya cloud: read legacy configuration: %w", err)
	}
	if err == nil {
		secret, decryptErr := h.cipher.Decrypt(secretEnc)
		if decryptErr != nil {
			return fmt.Errorf("tuya cloud: decrypt legacy OpenAPI secret: %w", decryptErr)
		}
		if strings.TrimSpace(endpoint) == "" {
			endpoint = "https://openapi.tuyaeu.com"
		}
		cfg := tuya_cloud.Config{
			ClientID: strings.TrimSpace(clientID),
			Secret: secret,
			UID: strings.TrimSpace(uid),
			BaseURL: strings.TrimSpace(endpoint),
		}
		if _, validationErr := tuya_cloud.New(cfg); validationErr != nil {
			return fmt.Errorf("tuya cloud: legacy OpenAPI configuration is invalid: %w", validationErr)
		}
		if _, saveErr := h.saveManagedOpenAPIConfig(ctx, cfg); saveErr != nil {
			return fmt.Errorf("tuya cloud: migrate legacy configuration: %w", saveErr)
		}
		if _, deleteErr := h.db.ExecContext(ctx, "DELETE FROM tuya_cloud_config WHERE id='default'"); deleteErr != nil {
			return fmt.Errorf("tuya cloud: remove migrated legacy configuration: %w", deleteErr)
		}
	}

	instance, _, err := h.managedTuyaInstance(ctx)
	if err != nil {
		if errors.Is(err, errTuyaCloudPluginInstanceMissing) {
			return nil
		}
		return err
	}
	return h.refreshManagedProvider(ctx, instance)
}

func readTuyaConfigString(config map[string]json.RawMessage, key string) (string, error) {
	raw := config[key]
	if len(raw) == 0 {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("tuya cloud: invalid %s setting: %w", key, err)
	}
	return value, nil
}
