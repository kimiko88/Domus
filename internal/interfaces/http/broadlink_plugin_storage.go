package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/plugins/broadlink"
)

// restoreLegacyBroadlinkInstance adopts pre-Plugin-Manager installations only
// when they contain paired devices. Device keys and learned IR/RF commands
// remain in their existing encrypted tables; the manager persists lifecycle
// and the stable integration identity.
func restoreLegacyBroadlinkInstance(ctx context.Context, manager *apppluginruntime.Manager, db *sql.DB) error {
	if manager == nil || db == nil {
		return nil
	}

	instances, err := manager.ListInstances(ctx)
	if err != nil {
		return fmt.Errorf("list plugin instances before BroadLink adoption: %w", err)
	}
	for _, instance := range instances {
		if instance.Type == broadlink.PluginType {
			// This includes intentionally disabled instances; never silently
			// re-enable an instance the administrator disabled.
			return nil
		}
		if instance.ID == broadlink.DefaultIntegrationID {
			return fmt.Errorf("BroadLink stable instance ID %q is already used by plugin type %q", instance.ID, instance.Type)
		}
	}

	var tableCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='broadlink_devices'`).Scan(&tableCount); err != nil {
		return fmt.Errorf("inspect legacy BroadLink device storage: %w", err)
	}
	if tableCount == 0 {
		return nil
	}

	var deviceCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM broadlink_devices`).Scan(&deviceCount); err != nil {
		return fmt.Errorf("count legacy BroadLink devices: %w", err)
	}
	if deviceCount == 0 {
		return nil
	}

	_, err = manager.Create(
		ctx,
		broadlink.DefaultIntegrationID,
		broadlink.PluginType,
		"BroadLink Local",
		json.RawMessage(`{}`),
		true,
	)
	if err != nil {
		return fmt.Errorf("persist legacy BroadLink plugin instance: %w", err)
	}
	return nil
}
