package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
)

const smartLifeOAuthConfigKey = "smart_life_oauth"

var (
	errTuyaOAuthNotConnected = errors.New("tuya oauth: Smart Life account is not connected")
	errTuyaPluginInstanceMissing = errors.New("tuya oauth: Tuya Cloud plugin instance not found")
)

type storedSmartLifeOAuth struct {
	UID          string `json:"uid"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// ensureManagedPlugin creates the singleton Tuya Cloud plugin instance on first
// OAuth use. Credentials are persisted through Plugin Manager encryption.
func (h *TuyaOAuthHandler) ensureManagedPlugin(ctx context.Context) (string, error) {
	if h.pluginManager == nil {
		return "", errors.New("tuya oauth: plugin manager is unavailable")
	}
	instances, err := h.pluginManager.ListInstances(ctx)
	if err != nil {
		return "", fmt.Errorf("tuya oauth: list plugin instances: %w", err)
	}
	for _, instance := range instances {
		if instance.Type != tuya_cloud.PluginType {
			continue
		}
		if !instance.Enabled {
			return "", errors.New("tuya oauth: Tuya Cloud plugin instance is disabled")
		}
		return instance.ID, nil
	}

	cfg := h.client.Config()
	initial := map[string]any{
		"client_id": cfg.ClientID,
		"secret": cfg.Secret,
		"base_url": cfg.BaseURL,
		"uid": "",
		"schema": "",
		"oauth_identifier": cfg.Identifier,
	}
	raw, err := json.Marshal(initial)
	if err != nil {
		return "", fmt.Errorf("tuya oauth: encode initial plugin configuration: %w", err)
	}
	created, err := h.pluginManager.Create(ctx, tuya_cloud.PluginType, tuya_cloud.PluginType, "Tuya Cloud / Smart Life", raw, true)
	if err != nil {
		return "", fmt.Errorf("tuya oauth: create encrypted Tuya Cloud instance: %w", err)
	}
	return created.ID, nil
}

func (h *TuyaOAuthHandler) managedPluginConfig(ctx context.Context, create bool) (string, map[string]json.RawMessage, error) {
	if h.pluginManager == nil {
		return "", nil, errors.New("tuya oauth: plugin manager is unavailable")
	}
	instances, err := h.pluginManager.ListInstances(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("tuya oauth: list plugin instances: %w", err)
	}
	instanceID := ""
	for _, instance := range instances {
		if instance.Type == tuya_cloud.PluginType {
			instanceID = instance.ID
			break
		}
	}
	if instanceID == "" {
		if !create {
			return "", nil, errTuyaPluginInstanceMissing
		}
		instanceID, err = h.ensureManagedPlugin(ctx)
		if err != nil {
			return "", nil, err
		}
	}
	raw, err := h.pluginManager.GetConfig(ctx, instanceID)
	if err != nil {
		return "", nil, fmt.Errorf("tuya oauth: read encrypted plugin configuration: %w", err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", nil, fmt.Errorf("tuya oauth: decode plugin configuration: %w", err)
	}
	if config == nil {
		config = make(map[string]json.RawMessage)
	}
	return instanceID, config, nil
}

func (h *TuyaOAuthHandler) storeManagedToken(ctx context.Context, _ string, token tuya_cloud.OAuthToken) error {
	if token.UID == "" || token.AccessToken == "" || token.RefreshToken == "" {
		return errors.New("tuya oauth: refusing to persist an incomplete Smart Life token")
	}
	instanceID, config, err := h.managedPluginConfig(ctx, true)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(time.Duration(token.ExpireTime) * time.Second).Unix()
	if token.ExpireTime < 0 {
		expiresAt = time.Now().Add(-time.Second).Unix()
	}
	encoded, err := json.Marshal(storedSmartLifeOAuth{
		UID: token.UID, AccessToken: token.AccessToken,
		RefreshToken: token.RefreshToken, ExpiresAt: expiresAt,
	})
	if err != nil {
		return fmt.Errorf("tuya oauth: encode Smart Life token: %w", err)
	}
	config[smartLifeOAuthConfigKey] = encoded
	updated, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("tuya oauth: encode Tuya plugin configuration: %w", err)
	}
	if _, err := h.pluginManager.UpdateConfig(ctx, instanceID, updated); err != nil {
		return fmt.Errorf("tuya oauth: persist Smart Life token in encrypted plugin config: %w", err)
	}
	return nil
}

func (h *TuyaOAuthHandler) loadManagedToken(ctx context.Context, userID string) (tuya_cloud.OAuthToken, int64, error) {
	_, config, err := h.managedPluginConfig(ctx, false)
	if err != nil && !errors.Is(err, errTuyaPluginInstanceMissing) {
		return tuya_cloud.OAuthToken{}, 0, err
	}
	if err == nil {
		if raw := config[smartLifeOAuthConfigKey]; len(raw) > 0 {
			var stored storedSmartLifeOAuth
			if err := json.Unmarshal(raw, &stored); err != nil {
				return tuya_cloud.OAuthToken{}, 0, fmt.Errorf("tuya oauth: decode stored Smart Life token: %w", err)
			}
			if stored.UID != "" && stored.AccessToken != "" && stored.RefreshToken != "" && stored.ExpiresAt > 0 {
				token := tuya_cloud.OAuthToken{
					UID: stored.UID, AccessToken: stored.AccessToken,
					RefreshToken: stored.RefreshToken,
					ExpireTime: int64(time.Until(time.Unix(stored.ExpiresAt, 0)).Seconds()),
				}
				return token, stored.ExpiresAt, nil
			}
		}
	}

	// One-time compatibility migration: old per-user tokens are decrypted,
	// written through Plugin Manager (which re-encrypts the full config), and
	// deleted from the legacy table only after the new write succeeds.
	legacy, expiresAt, err := h.loadLegacyToken(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tuya_cloud.OAuthToken{}, 0, errTuyaOAuthNotConnected
		}
		return tuya_cloud.OAuthToken{}, 0, err
	}
	if err := h.storeManagedToken(ctx, userID, legacy); err != nil {
		return tuya_cloud.OAuthToken{}, 0, err
	}
	if _, err := h.db.ExecContext(ctx, "DELETE FROM tuya_user_oauth_connections WHERE user_id=?", userID); err != nil {
		return tuya_cloud.OAuthToken{}, 0, fmt.Errorf("tuya oauth: remove migrated legacy token: %w", err)
	}
	return legacy, expiresAt, nil
}

func (h *TuyaOAuthHandler) loadLegacyToken(ctx context.Context, userID string) (tuya_cloud.OAuthToken, int64, error) {
	var uid, accessEnc, refreshEnc string
	var expires int64
	err := h.db.QueryRowContext(ctx, "SELECT tuya_uid,access_token_enc,refresh_token_enc,expires_at FROM tuya_user_oauth_connections WHERE user_id=?", userID).Scan(&uid, &accessEnc, &refreshEnc, &expires)
	if err != nil {
		return tuya_cloud.OAuthToken{}, 0, err
	}
	access, err := h.cipher.Decrypt(accessEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, 0, err
	}
	refresh, err := h.cipher.Decrypt(refreshEnc)
	if err != nil {
		return tuya_cloud.OAuthToken{}, 0, err
	}
	return tuya_cloud.OAuthToken{
		UID: uid, AccessToken: access, RefreshToken: refresh,
		ExpireTime: int64(time.Until(time.Unix(expires, 0)).Seconds()),
	}, expires, nil
}

func (h *TuyaOAuthHandler) clearManagedToken(ctx context.Context, userID string) error {
	instanceID, config, err := h.managedPluginConfig(ctx, false)
	if err == nil {
		delete(config, smartLifeOAuthConfigKey)
		updated, marshalErr := json.Marshal(config)
		if marshalErr != nil {
			return marshalErr
		}
		if _, updateErr := h.pluginManager.UpdateConfig(ctx, instanceID, updated); updateErr != nil {
			return updateErr
		}
	} else if !errors.Is(err, errTuyaPluginInstanceMissing) {
		return err
	}
	_, err = h.db.ExecContext(ctx, "DELETE FROM tuya_user_oauth_connections WHERE user_id=?", userID)
	return err
}
