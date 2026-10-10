package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/application/alarm"
	domainAlarm "github.com/domus-hub/domus/internal/domain/alarm"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupAlarmHandlerTest(t *testing.T) (*handlers.AlarmHandler, *sql.DB) {
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

	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 64, Workers: 2})
	svc := alarm.NewService(db, bus, nil)
	return handlers.NewAlarmHandler(svc), db
}

func TestAlarmHandler_Routes(t *testing.T) {
	h, db := setupAlarmHandlerTest(t)
	defer db.Close()

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// 1. GET /alarm
	req := httptest.NewRequest(http.MethodGet, "/alarm", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var panel domainAlarm.ControlPanel
	err := json.NewDecoder(rr.Body).Decode(&panel)
	require.NoError(t, err)
	assert.Equal(t, domainAlarm.StateDisarmed, panel.State)

	// 2. POST /alarm/arm with wrong pin
	armBody, _ := json.Marshal(map[string]string{
		"mode": "armed_home",
		"pin":  "0000",
	})
	req = httptest.NewRequest(http.MethodPost, "/alarm/arm", bytes.NewReader(armBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	// 3. POST /alarm/arm with correct pin
	armBody, _ = json.Marshal(map[string]string{
		"mode": "armed_home",
		"pin":  "1234",
	})
	req = httptest.NewRequest(http.MethodPost, "/alarm/arm", bytes.NewReader(armBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 4. POST /alarm/disarm
	disarmBody, _ := json.Marshal(map[string]string{
		"pin": "1234",
	})
	req = httptest.NewRequest(http.MethodPost, "/alarm/disarm", bytes.NewReader(disarmBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// 5. POST /alarm/trigger
	trigBody, _ := json.Marshal(map[string]string{
		"reason": "Test Panic",
	})
	req = httptest.NewRequest(http.MethodPost, "/alarm/trigger", bytes.NewReader(trigBody))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
