package notification_test

import (
	"context"
	"testing"

	"github.com/domus-hub/domus/internal/application/notification"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationManager_ActionableAndCritical(t *testing.T) {
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 1})
	mgr := notification.NewManager(bus)
	ctx := context.Background()

	notif := &notification.ActionableNotification{
		Title:    "Rilevato Fumo in Cucina",
		Message:  "Sensore fumo ha rilevato particolato. Disattivare carichi?",
		Critical: true,
		Volume:   1.0,
		Actions: []notification.NotificationAction{
			{ID: "disable_loads", Title: "Distacca Carichi"},
			{ID: "dismiss", Title: "Falso Allarme", Destructive: true},
		},
	}

	payload, err := mgr.CreateNotification(ctx, notif)
	require.NoError(t, err)
	assert.NotEmpty(t, notif.ID)

	// Verify APNs critical settings
	aps := payload.APNsPayload["aps"].(map[string]any)
	assert.NotNil(t, aps["sound"])
	sound := aps["sound"].(map[string]any)
	assert.Equal(t, 1, sound["critical"])

	// Verify FCM priority
	assert.Equal(t, "high", payload.FCMPayload["priority"])

	// Action response
	result, err := mgr.HandleAction(ctx, notif.ID, "disable_loads", "admin")
	require.NoError(t, err)
	assert.Equal(t, "disable_loads", result.ActionID)
	assert.Equal(t, notif.ID, result.NotificationID)
}
