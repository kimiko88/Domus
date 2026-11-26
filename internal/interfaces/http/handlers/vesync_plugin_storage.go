package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/plugins/vesync"
)

func (h *VeSyncHandler) saveManagedConfig(ctx context.Context, config vesyncStoredConfig, status string) error {
	if h.pluginManager == nil {
		return errors.New("vesync: Plugin Manager is unavailable")
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("vesync: encode managed session: %w", err)
	}

	_, lookupErr := h.pluginManager.GetConfig(ctx, vesyncIntegrationID)
	switch {
	case errors.Is(lookupErr, integration.ErrIntegrationNotFound):
		if _, err := h.pluginManager.Create(
			ctx,
			vesyncIntegrationID,
			vesync.PluginType,
			"VeSync Cloud",
			encoded,
			status != "disconnected",
		); err != nil {
			return fmt.Errorf("vesync: create encrypted Plugin Manager instance: %w", err)
		}
	case lookupErr != nil:
		return fmt.Errorf("vesync: read existing Plugin Manager configuration: %w", lookupErr)
	default:
		var submitted map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &submitted); err != nil {
			return fmt.Errorf("vesync: decode managed session: %w", err)
		}
		_, err := h.pluginManager.UpdateConfigMutate(ctx, vesyncIntegrationID, func(current map[string]json.RawMessage) error {
			current["session"] = append(json.RawMessage(nil), submitted["session"]...)
			return nil
		})
		if err != nil {
			return fmt.Errorf("vesync: update encrypted Plugin Manager session: %w", err)
		}
	}

	// Disconnection clears the session before disabling the adapter. A later
	// login explicitly re-enables the same stable instance ID.
	if status == "disconnected" {
		if err := h.pluginManager.SetEnabled(ctx, vesyncIntegrationID, false); err != nil {
			return fmt.Errorf("vesync: disable disconnected plugin instance: %w", err)
		}
	} else {
		if err := h.pluginManager.SetEnabled(ctx, vesyncIntegrationID, true); err != nil {
			return fmt.Errorf("vesync: enable configured plugin instance: %w", err)
		}
	}
	if err := h.repo.SetStatus(ctx, vesyncIntegrationID, status, ""); err != nil {
		return fmt.Errorf("vesync: update integration status: %w", err)
	}
	return nil
}
