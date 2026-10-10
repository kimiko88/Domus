package alarm_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/domus-hub/domus/internal/application/alarm"
	domainAlarm "github.com/domus-hub/domus/internal/domain/alarm"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type mockSirenController struct {
	lastEntityKey string
	lastService   string
}

func (m *mockSirenController) ControlEntity(ctx context.Context, entityKey, service string) error {
	m.lastEntityKey = entityKey
	m.lastService = service
	return nil
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE alarm_control_panels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'disarmed',
			pin_hash TEXT,
			entry_delay_seconds INTEGER NOT NULL DEFAULT 30,
			exit_delay_seconds INTEGER NOT NULL DEFAULT 60,
			trigger_delay_seconds INTEGER NOT NULL DEFAULT 0,
			perimeter_sensors_json TEXT NOT NULL DEFAULT '[]',
			interior_sensors_json TEXT NOT NULL DEFAULT '[]',
			siren_entities_json TEXT NOT NULL DEFAULT '[]',
			last_changed_by TEXT,
			last_triggered_reason TEXT,
			triggered_at TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)
	return db
}

func TestAlarmService_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	sirenCtrl := &mockSirenController{}
	svc := alarm.NewService(db, bus, sirenCtrl)
	ctx := context.Background()

	// 1. Get initial panel
	panel, err := svc.GetPanel(ctx)
	require.NoError(t, err)
	assert.Equal(t, domainAlarm.StateDisarmed, panel.State)
	assert.Equal(t, "1234", panel.PinHash)

	// 2. Configure sensors and sirens
	panel, err = svc.UpdateConfig(ctx, 0, 0, []string{"binary_sensor.front_door"}, []string{"binary_sensor.living_motion"}, []string{"switch.siren"}, "4321")
	require.NoError(t, err)
	assert.Equal(t, "4321", panel.PinHash)
	assert.Contains(t, panel.PerimeterSensors, "binary_sensor.front_door")

	// 3. Arm with incorrect PIN should fail
	_, err = svc.Arm(ctx, domainAlarm.StateArmedHome, "wrong", "user1")
	assert.ErrorIs(t, err, domainAlarm.ErrInvalidPin)

	// 4. Arm with correct PIN
	panel, err = svc.Arm(ctx, domainAlarm.StateArmedHome, "4321", "user1")
	require.NoError(t, err)
	assert.Equal(t, domainAlarm.StateArmedHome, panel.State)
	assert.Equal(t, "user1", panel.LastChangedBy)

	// 5. Trigger perimeter sensor
	svc.HandleSensorEvent(ctx, "binary_sensor.front_door", true)

	panel, err = svc.GetPanel(ctx)
	require.NoError(t, err)
	assert.Equal(t, domainAlarm.StateTriggered, panel.State)
	assert.Equal(t, "switch.siren", sirenCtrl.lastEntityKey)
	assert.Equal(t, "turn_on", sirenCtrl.lastService)

	// 6. Disarm
	panel, err = svc.Disarm(ctx, "4321", "user1")
	require.NoError(t, err)
	assert.Equal(t, domainAlarm.StateDisarmed, panel.State)
	assert.Equal(t, "switch.siren", sirenCtrl.lastEntityKey)
	assert.Equal(t, "turn_off", sirenCtrl.lastService)
}
