package mesh_test

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/mesh"
	"github.com/domus-hub/domus/internal/domain/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDeviceRepo struct {
	devices []*device.Device
}

func (m *mockDeviceRepo) Create(ctx context.Context, d *device.Device) error {
	m.devices = append(m.devices, d)
	return nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, id string) (*device.Device, error) {
	for _, d := range m.devices {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, device.ErrDeviceNotFound
}
func (m *mockDeviceRepo) GetByExternalID(ctx context.Context, integrationID, externalID string) (*device.Device, error) {
	return nil, device.ErrDeviceNotFound
}
func (m *mockDeviceRepo) List(ctx context.Context, filter device.ListFilter) ([]*device.Device, error) {
	return m.devices, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, d *device.Device) error { return nil }
func (m *mockDeviceRepo) SetOnline(ctx context.Context, id string, online bool) error {
	return nil
}
func (m *mockDeviceRepo) Delete(ctx context.Context, id string) error { return nil }

func TestMeshService_GetTopology(t *testing.T) {
	ctx := context.Background()

	// 1. Empty devices
	emptySvc := mesh.NewService()
	emptyTopo, err := emptySvc.GetTopology(ctx, "zigbee")
	require.NoError(t, err)
	assert.Equal(t, "zigbee", emptyTopo.Protocol)
	assert.Empty(t, emptyTopo.Nodes)
	assert.Empty(t, emptyTopo.Links)

	// 2. With real devices in repo
	mockRepo := &mockDeviceRepo{
		devices: []*device.Device{
			{
				ID:            "dev_zigbee_1",
				Name:          "Presa Smart Salotto",
				IntegrationID: "zigbee2mqtt",
				Model:         "TS011F Router Plug",
				Online:        true,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}
	svc := mesh.NewService(mockRepo)

	topo, err := svc.GetTopology(ctx, "zigbee")
	require.NoError(t, err)
	assert.Equal(t, "zigbee", topo.Protocol)
	assert.Len(t, topo.Nodes, 2) // Coordinator + 1 device
	assert.Len(t, topo.Links, 1)

	// Verify Coordinator presence
	foundCoord := false
	for _, n := range topo.Nodes {
		if n.Type == mesh.NodeCoordinator {
			foundCoord = true
			break
		}
	}
	assert.True(t, foundCoord)

	// Verify links have valid LQI and Quality
	for _, l := range topo.Links {
		assert.NotEmpty(t, l.Source)
		assert.NotEmpty(t, l.Target)
		assert.True(t, l.LQI >= 0)
		assert.NotEmpty(t, l.Quality)
	}
}
