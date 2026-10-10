package tag_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/tag"
	domainTag "github.com/domus-hub/domus/internal/domain/tag"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagService_RegisterAndScan(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := tag.NewMemoryRepository()
	svc := tag.NewService(repo, bus)
	ctx := context.Background()

	tRecord, err := svc.RegisterTag(ctx, "Comodino Notte", "04:A2:3B:5C:88", domainTag.ActionTypeScene, "scene_goodnight")
	require.NoError(t, err)
	assert.NotEmpty(t, tRecord.ID)
	assert.Equal(t, "04:A2:3B:5C:88", tRecord.TagID)

	res, err := svc.ScanTag(ctx, "04:A2:3B:5C:88", "user_123")
	require.NoError(t, err)
	assert.True(t, res.ActionExecuted)
	assert.Equal(t, "scene_goodnight", res.Tag.ActionTargetID)
	assert.NotNil(t, res.Tag.LastScannedAt)
}
