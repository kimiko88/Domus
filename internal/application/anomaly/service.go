package anomaly

import (
	"context"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Service provides a unified facade for household anomaly detection.
type Service struct {
	mu           sync.RWMutex
	bus          events.Bus
	waterMeters  map[string]*WaterLeakDetector
	appliances   map[string]*ApplianceHealthMonitor
	latestAlerts []any
}

// NewService instantiates the predictive maintenance service.
func NewService(bus events.Bus) *Service {
	return &Service{
		bus:          bus,
		waterMeters:  make(map[string]*WaterLeakDetector),
		appliances:   make(map[string]*ApplianceHealthMonitor),
		latestAlerts: make([]any, 0, 50),
	}
}

// RegisterWaterMeter enrolls a water meter entity for micro-leak monitoring.
func (s *Service) RegisterWaterMeter(meterID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waterMeters[meterID] = NewWaterLeakDetector(meterID)
}

// RegisterAppliance enrolls an electrical appliance for failure prediction.
func (s *Service) RegisterAppliance(id string, kind ApplianceType, nominalWatts float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appliances[id] = NewApplianceHealthMonitor(id, kind, nominalWatts)
}

// CheckWaterFlow ingests a flow telemetry point.
func (s *Service) CheckWaterFlow(meterID string, flowLMin float64, now time.Time) *MicroLeakAlert {
	s.mu.Lock()
	m, ok := s.waterMeters[meterID]
	if !ok {
		m = NewWaterLeakDetector(meterID)
		s.waterMeters[meterID] = m
	}
	s.mu.Unlock()

	isLeak, alert := m.IngestFlowRate(flowLMin, now)
	if isLeak && alert != nil {
		s.recordAlert(alert)
		if s.bus != nil {
			go s.publishEvent("anomaly.water_leak", alert)
		}
		return alert
	}
	return nil
}

// CheckApplianceTemp ingests internal compartment temperature.
func (s *Service) CheckApplianceTemp(id string, tempC float64, now time.Time) *ApplianceHealthAlert {
	s.mu.RLock()
	m, ok := s.appliances[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	isAlert, alert := m.IngestTemperature(tempC, now)
	if isAlert && alert != nil {
		s.recordAlert(alert)
		if s.bus != nil {
			go s.publishEvent("anomaly.appliance_temp", alert)
		}
		return alert
	}
	return nil
}

// CheckAppliancePower ingests active power wattage telemetry.
func (s *Service) CheckAppliancePower(id string, watts float64, now time.Time) *ApplianceHealthAlert {
	s.mu.RLock()
	m, ok := s.appliances[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	isAlert, alert := m.IngestPowerWatts(watts, now)
	if isAlert && alert != nil {
		s.recordAlert(alert)
		if s.bus != nil {
			go s.publishEvent("anomaly.appliance_power", alert)
		}
		return alert
	}
	return nil
}

// AssessSurfaceMold calculates condensation margin for an entity zone.
func (s *Service) AssessSurfaceMold(ambientTempC, rhPct, surfaceTempC float64, now time.Time) MoldAssessment {
	res := AssessMoldRisk(ambientTempC, rhPct, surfaceTempC, now)
	if res.RiskLevel == MoldRiskHigh || res.RiskLevel == MoldRiskCritical {
		s.recordAlert(res)
		if s.bus != nil {
			go s.publishEvent("anomaly.mold_risk", res)
		}
	}
	return res
}

func (s *Service) recordAlert(a any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.latestAlerts) >= 100 {
		s.latestAlerts = s.latestAlerts[1:]
	}
	s.latestAlerts = append(s.latestAlerts, a)
}

func (s *Service) publishEvent(eventType string, payload any) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.bus.Publish(ctx, events.Event{
		ID:         uuid.NewString(),
		Type:       events.NotificationCreated,
		OccurredAt: time.Now(),
		Source:     "anomaly.detector",
		Payload: map[string]any{
			"category": eventType,
			"data":     payload,
		},
	})
}
