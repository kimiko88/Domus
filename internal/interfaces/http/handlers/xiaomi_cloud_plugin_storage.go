package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/plugins/xiaomi_home"
)

var xiaomiManagedConfigFields = []string{
	"client_id",
	"client_secret",
	"region",
	"public_base_url",
	"redirect_url",
	"callback_secret",
	"event_secret",
	"oauth_state",
	"access_token",
	"refresh_token",
	"mac_key",
	"mac_algorithm",
	"expires_at",
	"user_id",
	"nickname",
}

func (h *XiaomiCloudHandler) loadManagedConfig(ctx context.Context) (*xiaomiCloudConfig, error) {
	if h.pluginManager == nil {
		return nil, errors.New("xiaomi: Plugin Manager is unavailable")
	}
	raw, err := h.pluginManager.GetConfig(ctx, xiaomiCloudIntegrationID)
	if err != nil {
		return nil, err
	}
	var config xiaomiCloudConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("xiaomi: decode encrypted plugin configuration: %w", err)
	}
	return &config, nil
}

// saveManagedConfig writes the full account state through the Manager's atomic
// encrypted update path. The stable instance ID is also the historical device
// integration ID, so existing devices and events keep their references.
func (h *XiaomiCloudHandler) saveManagedConfig(ctx context.Context, config xiaomiCloudConfig, status string) error {
	if h.pluginManager == nil {
		return errors.New("xiaomi: Plugin Manager is unavailable")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("xiaomi: encode plugin configuration: %w", err)
	}
	var submitted map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &submitted); err != nil {
		return fmt.Errorf("xiaomi: decode plugin configuration: %w", err)
	}

	_, lookupErr := h.pluginManager.GetConfig(ctx, xiaomiCloudIntegrationID)
	switch {
	case errors.Is(lookupErr, integration.ErrIntegrationNotFound):
		if _, err := h.pluginManager.Create(
			ctx,
			xiaomiCloudIntegrationID,
			xiaomi_home.CloudPluginType,
			"Xiaomi Home Cloud",
			encoded,
			true,
		); err != nil {
			return fmt.Errorf("xiaomi: create encrypted Plugin Manager instance: %w", err)
		}
	case lookupErr != nil:
		return fmt.Errorf("xiaomi: read existing Plugin Manager configuration: %w", lookupErr)
	default:
		if _, err := h.pluginManager.UpdateConfigMutate(ctx, xiaomiCloudIntegrationID, func(current map[string]json.RawMessage) error {
			for _, key := range xiaomiManagedConfigFields {
				if value, ok := submitted[key]; ok {
					current[key] = append(json.RawMessage(nil), value...)
				} else {
					delete(current, key)
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("xiaomi: update encrypted Plugin Manager configuration: %w", err)
		}
	}

	if err := h.integrations.SetStatus(ctx, xiaomiCloudIntegrationID, status, ""); err != nil {
		return fmt.Errorf("xiaomi: update integration status: %w", err)
	}
	return nil
}

var _ = (*apppluginruntime.Manager)(nil)
