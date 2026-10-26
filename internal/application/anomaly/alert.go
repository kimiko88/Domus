package anomaly

import (
	"context"
	"time"
)

// AnomalyAlert represents an anomaly incident recorded by the predictive engine.
type AnomalyAlert struct {
	ID           string         `json:"id"`
	Category     string         `json:"category"` // "water_leak", "appliance_temp", "appliance_power", "mold_risk"
	Severity     string         `json:"severity"` // "info", "warning", "critical"
	EntityID     string         `json:"entity_id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Acknowledged bool           `json:"acknowledged"`
	CreatedAt    time.Time      `json:"created_at"`
	Data         map[string]any `json:"data,omitempty"`
}

// Repository defines SQLite operations for predictive maintenance alerts and appliance telemetry.
type Repository interface {
	SaveAlert(ctx context.Context, alert *AnomalyAlert) error
	ListAlerts(ctx context.Context, acknowledged *bool, limit int) ([]*AnomalyAlert, error)
	AcknowledgeAlert(ctx context.Context, id string) error
	SaveWaterMeter(ctx context.Context, meterID string, flowLMin float64, lastSeen time.Time) error
	GetWaterMeter(ctx context.Context, meterID string) (float64, time.Time, error)
	SaveAppliance(ctx context.Context, id, kind string, nominalWatts, lastWatts, lastTemp float64, lastSeen time.Time) error
	ListAppliances(ctx context.Context) ([]map[string]any, error)
}
