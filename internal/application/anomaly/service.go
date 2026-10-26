package anomaly

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// Service provides a unified facade for household anomaly detection.
type Service struct {
	mu           sync.RWMutex
	bus          events.Bus
	repo         Repository
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

// WithRepository connects SQLite database persistence for alerts and appliance states.
func (s *Service) WithRepository(repo Repository) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo = repo
	return s
}

// ListAlerts returns recorded anomaly alerts, optionally filtered by acknowledged state.
func (s *Service) ListAlerts(ctx context.Context, acknowledged *bool, limit int) ([]*AnomalyAlert, error) {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo != nil {
		return repo.ListAlerts(ctx, acknowledged, limit)
	}

	return []*AnomalyAlert{}, nil
}

// AcknowledgeAlert marks an alert as acknowledged.
func (s *Service) AcknowledgeAlert(ctx context.Context, id string) error {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo != nil {
		return repo.AcknowledgeAlert(ctx, id)
	}
	return nil
}

// ListAppliances returns all monitored appliances with live health metrics.
func (s *Service) ListAppliances(ctx context.Context) ([]map[string]any, error) {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo != nil {
		return repo.ListAppliances(ctx)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]map[string]any, 0, len(s.appliances))
	for id, a := range s.appliances {
		res = append(res, map[string]any{
			"id":            id,
			"kind":          a.kind,
			"nominal_watts": a.nominalWatts,
		})
	}
	return res, nil
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
	if len(s.latestAlerts) >= 100 {
		s.latestAlerts = s.latestAlerts[1:]
	}
	s.latestAlerts = append(s.latestAlerts, a)
	repo := s.repo
	s.mu.Unlock()

	if repo != nil {
		alert := &AnomalyAlert{
			ID:        uuid.NewString(),
			CreatedAt: time.Now().UTC(),
		}

		switch v := a.(type) {
		case *MicroLeakAlert:
			alert.Category = "water_leak"
			alert.Severity = v.Severity
			alert.EntityID = v.MeterEntityID
			alert.Title = "Perdita Idrica Rilevata"
			alert.Description = fmt.Sprintf("Flusso continuo non-stop %.2f L/min per oltre %d minuti: %s", v.ContinuousFlowLMin, int(v.Duration.Minutes()), v.Reason)
			alert.Data = map[string]any{"flow_l_min": v.ContinuousFlowLMin, "duration_s": v.Duration.Seconds(), "total_liters": v.TotalLeakedLiters}
		case *ApplianceHealthAlert:
			alert.Category = "appliance"
			alert.Severity = v.Severity
			alert.EntityID = v.ApplianceID
			alert.Title = fmt.Sprintf("Anomalia Elettrodomestico: %s", v.Reason)
			alert.Description = fmt.Sprintf("Valore osservato: %.2f (atteso: %s)", v.ObservedValue, v.ExpectedRange)
			alert.Data = map[string]any{"reason": v.Reason, "observed": v.ObservedValue, "expected": v.ExpectedRange}
		case MoldAssessment:
			alert.Category = "mold_risk"
			alert.Severity = string(v.RiskLevel)
			alert.EntityID = "climate_zone"
			alert.Title = "Rischio Condensazione e Formazione Muffa"
			alert.Description = fmt.Sprintf("Delta rugiada %.1f°C, Umidità Superficie %.1f%%", v.DeltaToDewC, v.SurfaceRHPct)
			alert.Data = map[string]any{"delta_to_dew_c": v.DeltaToDewC, "surface_rh_pct": v.SurfaceRHPct}
		default:
			alert.Category = "generic"
			alert.Severity = "warning"
			alert.EntityID = "system"
			alert.Title = "Anomalia di Sistema"
			alert.Description = "Rilevata potenziale anomalia nei parametri di esercizio"
		}

		_ = repo.SaveAlert(context.Background(), alert)
	}
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
