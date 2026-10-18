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
	profiles    map[string]*domclimate.ThermalProfile
}

func newMockPredictiveRepo() *mockPredictiveRepo {
	return &mockPredictiveRepo{
		zones:       make(map[string]*domclimate.PredictiveZoneConfig),
		evaluations: make(map[string][]*domclimate.PredictiveEvaluation),
		profiles:    make(map[string]*domclimate.ThermalProfile),
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
	delete(m.profiles, id)
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

func (m *mockPredictiveRepo) SaveProfile(ctx context.Context, profile *domclimate.ThermalProfile) error {
	m.profiles[profile.ZoneID] = profile
	return nil
}

func (m *mockPredictiveRepo) GetProfile(ctx context.Context, zoneID string) (*domclimate.ThermalProfile, error) {
	p, exists := m.profiles[zoneID]
	if !exists {
		return nil, domclimate.ErrThermalProfileNotFound
	}
	return p, nil
}

func (m *mockPredictiveRepo) ListProfiles(ctx context.Context) ([]*domclimate.ThermalProfile, error) {
	var list []*domclimate.ThermalProfile
	for _, p := range m.profiles {
		list = append(list, p)
	}
	return list, nil
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

func TestPredictiveService_ThermalProfileAndOptimalStartStop(t *testing.T) {
	ctx := context.Background()
	repo := newMockPredictiveRepo()
	bus := &mockEventBus{}
	weatherCli := appweather.NewClient()

	svc := appclimate.NewPredictiveService(repo, nil, weatherCli, bus, nil)

	zone := &domclimate.PredictiveZoneConfig{
		ID:                  "zone_floor_test",
		Name:                "Camera Matrimoniale",
		TRVEntityKey:        "climate.valvola_camera",
		TempSensorKey:       "sensor.camera_temp",
		TargetTemperature:   21.0,
		Orientation:         domclimate.OrientationSouth,
		WindowAreaSqm:       5.0,
		EmitterType:         domclimate.EmitterFloorRadiant,
		ThermalInertiaHours: 4.0,
		Enabled:             true,
	}
	if err := svc.SaveZone(ctx, zone); err != nil {
		t.Fatalf("failed to save zone: %v", err)
	}

	// 1. Get initial profile (should create default for Radiant Floor: 0.6 °C/h, 0.15 °C/h cool)
	profile, err := svc.GetZoneProfile(ctx, "zone_floor_test")
	if err != nil {
		t.Fatalf("failed to get zone profile: %v", err)
	}
	if profile.ZoneID != "zone_floor_test" || profile.HeatRatePerHr != 0.6 || profile.CoolRatePerHr != 0.15 {
		t.Errorf("unexpected default profile: %+v", profile)
	}

	// 2. Train from heating cycle (60 mins, rose from 19.0 to 20.2 -> observed 1.2 °C/h)
	trainedProfile, err := svc.TrainFromCycle(ctx, domclimate.ThermalCycleObservation{
		ZoneID:          "zone_floor_test",
		CycleType:       "heating",
		StartTemp:       19.0,
		EndTemp:         20.2,
		DurationMinutes: 60,
		ObservedAt:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to train from cycle: %v", err)
	}
	if trainedProfile.SamplesCount != 1 {
		t.Errorf("expected 1 sample, got %d", trainedProfile.SamplesCount)
	}
	if trainedProfile.HeatRatePerHr <= 0.6 {
		t.Errorf("expected heat rate to increase from 0.6, got %f", trainedProfile.HeatRatePerHr)
	}

	// 3. Optimal Start (target time in 3 hours)
	targetTime := time.Now().UTC().Add(3 * time.Hour)
	preheatRec, err := svc.CalculateOptimalStart(ctx, "zone_floor_test", targetTime, 21.0)
	if err != nil {
		t.Fatalf("failed to calculate optimal start: %v", err)
	}
	if preheatRec.PreheatMinutes <= 0 {
		t.Errorf("expected positive preheat minutes, got %d", preheatRec.PreheatMinutes)
	}
	if preheatRec.StartTime.After(targetTime) {
		t.Errorf("start time must be before target time")
	}
	if preheatRec.Reason == "" {
		t.Errorf("expected non-empty reason")
	}

	// 4. Optimal Coast-Down (departure time in 4 hours)
	departureTime := time.Now().UTC().Add(4 * time.Hour)
	coastRec, err := svc.CalculateCoastDown(ctx, "zone_floor_test", departureTime)
	if err != nil {
		t.Fatalf("failed to calculate coast down: %v", err)
	}
	if coastRec.CoastDownMinutes < 15 || coastRec.CoastDownMinutes > 120 {
		t.Errorf("expected coast down between 15 and 120 mins, got %d", coastRec.CoastDownMinutes)
	}
	if coastRec.EnergySavedKwh <= 0 {
		t.Errorf("expected positive energy saved, got %f", coastRec.EnergySavedKwh)
	}
	if coastRec.StopTime.After(departureTime) {
		t.Errorf("stop time must be before departure time")
	}
}

