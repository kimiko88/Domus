package mesh_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/mesh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeshService_GetTopology(t *testing.T) {
	svc := mesh.NewService()
	ctx := context.Background()

	topo, err := svc.GetTopology(ctx, "zigbee")
	require.NoError(t, err)
	assert.Equal(t, "zigbee", topo.Protocol)
	assert.NotEmpty(t, topo.Nodes)
	assert.NotEmpty(t, topo.Links)

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
