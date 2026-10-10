package mesh_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/domus-hub/domus/internal/application/mesh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupBindingTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE zigbee_bindings (
			id TEXT PRIMARY KEY,
			source_device_id TEXT NOT NULL,
			source_device_name TEXT NOT NULL,
			target_device_id TEXT NOT NULL,
			target_device_name TEXT NOT NULL,
			clusters_json TEXT NOT NULL DEFAULT '["genOnOff"]',
			bound_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)
	return db
}

func TestBindingService_Lifecycle(t *testing.T) {
	db := setupBindingTestDB(t)
	defer db.Close()

	svc := mesh.NewBindingService(db)
	ctx := context.Background()

	// 1. Create binding
	b, err := svc.CreateBinding(ctx, "0x00158d0001", "Telecomando Letto", "0x00158d0002", "Lampada Comodino", []string{"genOnOff", "genLevelCtrl"})
	require.NoError(t, err)
	assert.NotEmpty(t, b.ID)
	assert.Equal(t, "Telecomando Letto", b.SourceDeviceName)
	assert.Equal(t, "Lampada Comodino", b.TargetDeviceName)
	assert.Len(t, b.Clusters, 2)

	// 2. List bindings
	list, err := svc.ListBindings(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// 3. Delete binding
	err = svc.RemoveBinding(ctx, b.ID)
	require.NoError(t, err)

	list, err = svc.ListBindings(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}
