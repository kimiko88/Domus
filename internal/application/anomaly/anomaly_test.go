package anomaly_test

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/anomaly"
	"github.com/domus-hub/domus/internal/domain/events"
)

type mockAnomalyBus struct {
	mu     sync.Mutex
	events []events.Event
}

func (b *mockAnomalyBus) Publish(ctx context.Context, e events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
	return nil
}

func (b *mockAnomalyBus) Subscribe(eventType events.EventType, handler events.Handler) func() {
	return func() {}
}

func (b *mockAnomalyBus) SubscribeAll(handler events.Handler) func() {
	return func() {}
}

func TestMoldAssessment_DewPointAndRisk(t *testing.T) {
	// Standard indoor conditions: 20°C, 50% RH
	// Expected dew point ~ 9.27°C
	dp := anomaly.CalculateDewPoint(20.0, 50.0)
	if math.Abs(dp-9.27) > 0.1 {
		t.Errorf("expected dew point ~9.27°C, got %.2f°C", dp)
	}

	now := time.Now()

	// 1. Warm wall surface (18°C) -> plenty of margin -> MoldRiskNone
	m1 := anomaly.AssessMoldRisk(20.0, 50.0, 18.0, now)
	if m1.RiskLevel != anomaly.MoldRiskNone {
		t.Errorf("expected MoldRiskNone for 18°C surface, got %s", m1.RiskLevel)
	}

	// 2. Cold thermal bridge surface (12.0°C) with dew point at 9.27°C
	// Surface RH is ~83% (>= 80%) -> High mold risk
	m2 := anomaly.AssessMoldRisk(20.0, 50.0, 12.0, now)
	if m2.RiskLevel != anomaly.MoldRiskHigh {
		t.Errorf("expected MoldRiskHigh for 12°C surface, got %s", m2.RiskLevel)
	}

	// 3. Ice-cold surface below dew point (8.0°C < 9.27°C) -> Critical active condensation
	m3 := anomaly.AssessMoldRisk(20.0, 50.0, 8.0, now)
	if m3.RiskLevel != anomaly.MoldRiskCritical {
		t.Errorf("expected MoldRiskCritical for 8°C surface, got %s", m3.RiskLevel)
	}
}

func TestWaterLeakDetector_ContinuousTrickle(t *testing.T) {
	detector := anomaly.NewWaterLeakDetector("sensor.main_water_meter")
	now := time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC) // Daytime 14:00

	// 1. Zero flow -> no leak
	isLeak, _ := detector.IngestFlowRate(0.0, now)
	if isLeak {
		t.Errorf("zero flow should not trigger leak")
	}

	// 2. Normal short burst (tap opened for 2 minutes)
	now = now.Add(2 * time.Minute)
	isLeak, _ = detector.IngestFlowRate(6.0, now)
	if isLeak {
		t.Errorf("short water usage should not trigger leak")
	}

	// 3. Tap closed
	now = now.Add(1 * time.Minute)
	_, _ = detector.IngestFlowRate(0.0, now)

	// 4. Continuous micro-trickle (toilet cistern leak: 0.15 L/min)
	// Daytime threshold is 75 minutes
	now = now.Add(1 * time.Minute)
	_, _ = detector.IngestFlowRate(0.15, now) // Start flow

	now = now.Add(40 * time.Minute)
	isLeak, _ = detector.IngestFlowRate(0.15, now)
	if isLeak {
		t.Errorf("at 40 min daytime, should not alarm yet (< 75 min)")
	}

	// Reach 80 minutes of continuous flow
	now = now.Add(40 * time.Minute)
	isLeak, alert := detector.IngestFlowRate(0.15, now)
	if !isLeak || alert == nil {
		t.Fatalf("expected micro-leak alert after 80 min of continuous trickle flow!")
	}
	if alert.Severity != "warning" {
		t.Errorf("expected warning severity, got %s", alert.Severity)
	}

	// 5. Test Night-time faster detection (03:00 AM)
	nightDetector := anomaly.NewWaterLeakDetector("sensor.main_water_meter")
	nightTime := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	_, _ = nightDetector.IngestFlowRate(0.2, nightTime)

	// After 35 minutes of uninterrupted night flow -> triggers alert (night threshold is 30m)
	nightTime = nightTime.Add(35 * time.Minute)
	isNightLeak, nightAlert := nightDetector.IngestFlowRate(0.2, nightTime)
	if !isNightLeak || nightAlert == nil {
		t.Fatalf("expected night-time micro leak alarm after 35 min!")
	}
	if !nightAlert.IsNightTime {
		t.Errorf("expected IsNightTime=true")
	}
}

func TestApplianceHealth_FridgeDoorAjarAndPumpAnomalies(t *testing.T) {
	bus := &mockAnomalyBus{}
	svc := anomaly.NewService(bus)

	svc.RegisterAppliance("fridge.kitchen", anomaly.ApplianceRefrigerator, 120.0)
	svc.RegisterAppliance("pump.garden", anomaly.AppliancePump, 800.0)

	now := time.Now()

	// 1. Fridge normal temp (4°C)
	alert := svc.CheckApplianceTemp("fridge.kitchen", 4.0, now)
	if alert != nil {
		t.Errorf("expected no alert for 4°C fridge")
	}

	// 2. Door left ajar -> temp rises to 12°C for 45 minutes
	_ = svc.CheckApplianceTemp("fridge.kitchen", 12.0, now)
	now = now.Add(45 * time.Minute)
	alert = svc.CheckApplianceTemp("fridge.kitchen", 12.0, now)
	if alert == nil {
		t.Fatalf("expected door ajar alert after 45 min above threshold!")
	}
	if alert.Severity != "critical" {
		t.Errorf("expected critical alert for spoiled food risk, got %s", alert.Severity)
	}

	// 3. Garden pump cavitation / dry-run (nominal 800W, drawing only 150W < 30%)
	now = now.Add(1 * time.Minute)
	pumpAlert := svc.CheckAppliancePower("pump.garden", 150.0, now)
	if pumpAlert == nil {
		t.Fatalf("expected cavitation/dry-run alert for under-load pump!")
	}

	// 4. Garden pump motor lock (nominal 800W, drawing 1300W > 135%)
	_ = svc.CheckAppliancePower("pump.garden", 1300.0, now)
	now = now.Add(20 * time.Second)
	overloadAlert := svc.CheckAppliancePower("pump.garden", 1300.0, now)
	if overloadAlert == nil {
		t.Fatalf("expected rotor overload alert!")
	}
}
