package media_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/media"
	domainMedia "github.com/domus-hub/domus/internal/domain/media"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMediaService_RegisterAndBroadcast(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	repo := media.NewMemoryRepository()
	svc := media.NewService(repo, bus)
	ctx := context.Background()

	p1, err := svc.RegisterPlayer(ctx, "Speaker Salotto", domainMedia.PlatformCast, nil)
	require.NoError(t, err)

	p2, err := svc.RegisterPlayer(ctx, "Speaker Cucina", domainMedia.PlatformAirPlay, nil)
	require.NoError(t, err)

	// Set volume
	updated, err := svc.SetVolume(ctx, p1.ID, 0.75)
	require.NoError(t, err)
	assert.Equal(t, 0.75, updated.Volume)

	// Broadcast doorbell alert to all speakers
	count, err := svc.BroadcastAnnouncement(ctx, &domainMedia.BroadcastAnnouncement{
		AudioURL: "/tts/doorbell.wav",
		Message:  "Qualcuno ha suonato al campanello!",
		Volume:   0.9,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	refreshed, _ := repo.GetPlayer(ctx, p2.ID)
	assert.Equal(t, "Qualcuno ha suonato al campanello!", refreshed.CurrentTitle)
}
