package climate_test

import (
	"context"
	"testing"
	"time"

	appclimate "github.com/domus-hub/domus/internal/application/climate"
	appweather "github.com/domus-hub/domus/internal/application/weather"
	domclimate "github.com/domus-hub/domus/internal/domain/climate"
	"github.com/domus-hub/domus/internal/domain/events"
)

type mockPredictiveRepo struct {
	zones       map[string]*domclimate.PredictiveZoneConfig
	evaluations map[string][]*domclimate.PredictiveEvaluation
}

func newMockPredictiveRepo() *mockPredictiveRepo {
	return &mockPredictiveRepo{
		zones:       make(map[string]*domclimate.PredictiveZoneConfig),
		evaluations: make(map[string][]*domclimate.PredictiveEvaluation),
	}
}

func (m *mockPredictiveRepo) Save(ctx context.Context, cfg *domclimate.PredictiveZoneConfig) error {
	if cfg.ID == "" {
		cfg.ID = "pzone_mock"
	}
	m.zones[cfg.ID] = cfg
	return nil
}

func (m *mockPredictiveRepo) GetByID(ctx context.Context, id string) (*domclimate.PredictiveZoneConfig, error) {
	z, exists := m.zones[id]
	if !exists {
		return nil, domclimate.ErrPredictiveZoneNotFound
	}
	return z, nil
}

func (m *mockPredictiveRepo) List(ctx context.Context) ([]*domclimate.PredictiveZoneConfig, error) {
	var list []*domclimate.PredictiveZoneConfig
	for _, z := range m.zones {
		list = append(list, z)
	}
	return list, nil
}

func (m *mockPredictiveRepo) Delete(ctx context.Context, id string) error {
	delete(m.zones, id)
	return nil
}

func (m *mockPredictiveRepo) SaveEvaluation(ctx context.Context, eval *domclimate.PredictiveEvaluation) error {
	m.evaluations[eval.ZoneID] = append(m.evaluations[eval.ZoneID], eval)
	return nil
}

func (m *mockPredictiveRepo) GetLatestEvaluations(ctx context.Context) ([]*domclimate.PredictiveEvaluation, error) {
	var res []*domclimate.PredictiveEvaluation
	for _, list := range m.evaluations {
		if len(list) > 0 {
			res = append(res, list[len(list)-1])
		}
	}
	return res, nil
}

func (m *mockPredictiveRepo) GetZoneEvaluations(ctx context.Context, zoneID string, limit int) ([]*domclimate.PredictiveEvaluation, error) {
	return m.evaluations[zoneID], nil
}

type mockEventBus struct {
	published []events.Event
}

func (b *mockEventBus) Publish(ctx context.Context, event events.Event) error {
	b.published = append(b.published, event)
	return nil
}

func (b *mockEventBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockEventBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestPredictiveService_LifecycleAndEvaluation(t *testing.T) {
	ctx := context.Background()
	repo := newMockPredictiveRepo()
	bus := &mockEventBus{}
	weatherCli := appweather.NewClient()

	svc := appclimate.NewPredictiveService(repo, nil, weatherCli, bus, nil)

	// 1. Save Zone
	zone := &domclimate.PredictiveZoneConfig{
		ID:                   "zone_test_live",
		Name:                 "Studio Termico",
		TRVEntityKey:         "climate.termostato_studio",
		TempSensorKey:        "sensor.studio_temp",
		PressureSensorKey:    "sensor.studio_pressione",
		TargetTemperature:    21.0,
		Orientation:          domclimate.OrientationSouth,
		WindowAreaSqm:        3.5,
		EmitterType:          domclimate.EmitterRadiator,
		ThermalInertiaHours:  2.0,
		PreheatEnabled:       true,
		SolarCutoffEnabled:   true,
		PressureBoostEnabled: true,
		Enabled:              true,
	}

	if err := svc.SaveZone(ctx, zone); err != nil {
		t.Fatalf("failed to save zone: %v", err)
	}

	// 2. Add pressure samples to simulate a cold front drop (-2.0 hPa/h)
	now := time.Now().UTC()
	svc.RecordPressureSample("sensor.studio_pressione", 1015.0, now.Add(-2*time.Hour))
	svc.RecordPressureSample("sensor.studio_pressione", 1011.0, now)

	trend := svc.CalculatePressureTrend("sensor.studio_pressione")
	if trend != -2.0 {
		t.Errorf("expected pressure trend -2.0, got %f", trend)
	}

	// 3. Evaluate Zone
	eval, err := svc.EvaluateZone(ctx, "zone_test_live")
	if err != nil {
		t.Fatalf("failed to evaluate zone: %v", err)
	}

	if eval.ZoneID != "zone_test_live" {
		t.Errorf("expected zone ID zone_test_live, got %s", eval.ZoneID)
	}
	if !eval.ColdFrontDetected {
		t.Errorf("expected cold front detected due to -2.0 trend")
	}

	// Verify command was published on event bus to adjust TRV
	if len(bus.published) == 0 {
		t.Errorf("expected command event dispatched on bus")
	} else {
		lastEvt := bus.published[len(bus.published)-1]
		if lastEvt.Type != events.CommandRequested {
			t.Errorf("expected CommandRequested, got %s", lastEvt.Type)
		}
	}

	// 4. Status Check
	status, err := svc.GetDashboardStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get dashboard status: %v", err)
	}
	if status.TotalZones != 1 || status.EnabledZones != 1 {
		t.Errorf("unexpected status zone count: %+v", status)
	}
	if !status.ColdFrontActive {
		t.Errorf("expected ColdFrontActive to be true")
	}
}
