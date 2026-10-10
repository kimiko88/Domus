package automation

import (
	"context"
	"testing"
	"time"

	domauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
)

type mockEntityRepo struct {
	entities map[string]*entity.Entity
}

func (m *mockEntityRepo) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	if e, ok := m.entities[key]; ok {
		return e, nil
	}
	return nil, entity.ErrEntityNotFound
}

func (m *mockEntityRepo) Create(ctx context.Context, e *entity.Entity) error          { return nil }
func (m *mockEntityRepo) GetByID(ctx context.Context, id string) (*entity.Entity, error) { return nil, nil }
func (m *mockEntityRepo) List(ctx context.Context, f entity.ListFilter) ([]*entity.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepo) Update(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepo) Delete(ctx context.Context, id string) error        { return nil }
func (m *mockEntityRepo) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	return nil
}

func TestSunEngine_AstronomicalCalculations(t *testing.T) {
	// Rome: Lat 41.9028, Lon 12.4964
	sun := NewSunEngine(41.9028, 12.4964)

	// Summer Solstice noon: June 21 at 12:00 UTC
	noonSummer := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	pos := sun.Calculate(noonSummer)

	if pos.Elevation < 60.0 {
		t.Errorf("expected high solar elevation in Rome during summer noon (>60°), got %.2f°", pos.Elevation)
	}
	if !pos.IsDay {
		t.Errorf("expected daylight at solar noon")
	}
	if pos.Sunrise.IsZero() || pos.Sunset.IsZero() {
		t.Errorf("expected valid sunrise and sunset times")
	}
}

func TestSimulator_DryRunTrace(t *testing.T) {
	repo := &mockEntityRepo{
		entities: map[string]*entity.Entity{
			"sensor.humidity": {
				ID:    "1",
				State: "68.5",
			},
			"binary_sensor.window": {
				ID:    "2",
				State: "off",
			},
		},
	}

	sim := NewSimulator(repo, nil)
	ctx := context.Background()

	sixtyFive := 65.0
	rule := &domauto.Automation{
		ID:   "auto_dehumidifier",
		Name: "Accendi deumidificatore",
		Conditions: []domauto.Condition{
			{
				Condition: "and",
				Conditions: []domauto.Condition{
					{Condition: "numeric_state", EntityKey: "sensor.humidity", Above: &sixtyFive},
					{Condition: "state", EntityKey: "binary_sensor.window", State: "off"},
				},
			},
		},
		Actions: []domauto.Action{
			{Type: "service", Service: "turn_on", TargetEntityKey: "switch.dehumidifier"},
		},
	}

	result, err := sim.DryRun(ctx, rule, time.Now())
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	if !result.Passed {
		t.Errorf("expected automation conditions to pass, got failed")
	}
	if len(result.Actions) != 1 {
		t.Errorf("expected 1 action that would execute, got %d", len(result.Actions))
	}
	if len(result.Traces) != 1 || !result.Traces[0].Passed {
		t.Errorf("expected root AND trace passed")
	}
}

func TestBlueprints_CatalogAndInstantiate(t *testing.T) {
	bp, found := GetBlueprint("window_climate_saver")
	if !found {
		t.Fatalf("expected window_climate_saver in catalog")
	}

	// Missing params should fail
	_, err := bp.Factory(map[string]string{})
	if err == nil {
		t.Errorf("expected error on missing parameters")
	}

	// Valid params should produce valid automation
	auto, err := bp.Factory(map[string]string{
		"window_sensor":  "binary_sensor.kitchen_window",
		"climate_entity": "climate.living_room_ac",
	})
	if err != nil {
		t.Fatalf("instantiate failed: %v", err)
	}

	if auto.Triggers[0].EntityKey != "binary_sensor.kitchen_window" {
		t.Errorf("unexpected trigger entity: %s", auto.Triggers[0].EntityKey)
	}
	if auto.Actions[0].TargetEntityKey != "climate.living_room_ac" {
		t.Errorf("unexpected action target: %s", auto.Actions[0].TargetEntityKey)
	}
}
