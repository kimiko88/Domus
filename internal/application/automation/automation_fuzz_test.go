package automation

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	domainauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
)

type mockEntityRepoForFuzz struct {
	entities map[string]*entity.Entity
}

func (m *mockEntityRepoForFuzz) Create(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepoForFuzz) GetByID(ctx context.Context, id string) (*entity.Entity, error) {
	return nil, entity.ErrEntityNotFound
}
func (m *mockEntityRepoForFuzz) GetByKey(ctx context.Context, key string) (*entity.Entity, error) {
	if e, ok := m.entities[key]; ok {
		return e, nil
	}
	return nil, entity.ErrEntityNotFound
}
func (m *mockEntityRepoForFuzz) List(ctx context.Context, f entity.ListFilter) ([]*entity.Entity, error) {
	return nil, nil
}
func (m *mockEntityRepoForFuzz) Update(ctx context.Context, e *entity.Entity) error { return nil }
func (m *mockEntityRepoForFuzz) UpdateState(ctx context.Context, id string, stateJSON, attributesJSON string, available bool, at time.Time) error {
	return nil
}
func (m *mockEntityRepoForFuzz) Delete(ctx context.Context, id string) error { return nil }

func FuzzEvaluateCondition(f *testing.F) {
	eval := NewEvaluator()
	repo := &mockEntityRepoForFuzz{
		entities: map[string]*entity.Entity{
			"sensor.temp": {
				ID:        "e-1",
				EntityKey: "sensor.temp",
				State:     "22.5",
			},
			"switch.light": {
				ID:        "e-2",
				EntityKey: "switch.light",
				State:     "ON",
			},
		},
	}

	// Seeds
	f.Add([]byte(`{"condition":"state","entity_key":"switch.light","state":"ON"}`))
	f.Add([]byte(`{"condition":"numeric_state","entity_key":"sensor.temp","above":20,"below":30}`))
	f.Add([]byte(`{"condition":"and","conditions":[{"condition":"state","entity_key":"switch.light","state":"ON"}]}`))
	f.Add([]byte(`{"condition":"not","conditions":[]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`random corrupt non-json string`))

	ctx := context.Background()
	f.Fuzz(func(t *testing.T, data []byte) {
		var cond domainauto.Condition
		if err := json.Unmarshal(data, &cond); err == nil {
			// Evaluator must not panic on deeply nested or invalid condition shapes
			_, _ = eval.EvaluateCondition(ctx, cond, repo)
		}
	})
}

func FuzzSunEngineCalculate(f *testing.F) {
	// Seed realistic, boundary, and extreme coords
	f.Add(45.4642, 9.1900, int64(1770000000))
	f.Add(90.0, 0.0, int64(1775000000))
	f.Add(-90.0, 0.0, int64(1780000000))
	f.Add(0.0, 180.0, int64(0))
	f.Add(-45.0, -180.0, int64(2000000000))

	f.Fuzz(func(t *testing.T, lat, lon float64, unixSec int64) {
		// Clamp unixSec to reasonable years (-1000 to +3000 AD) to avoid time package overflow
		if unixSec < -62135596800 || unixSec > 32503680000 {
			return
		}
		// Bound coordinates to standard geographical range
		if lat < -90.0 || lat > 90.0 || lon < -180.0 || lon > 180.0 {
			return
		}

		engine := NewSunEngine(lat, lon)
		tVal := time.Unix(unixSec, 0).UTC()
		pos := engine.Calculate(tVal)

		// Assertions: must never return NaN
		if math.IsNaN(pos.Elevation) {
			t.Errorf("Elevation is NaN for lat=%f, lon=%f, time=%v", lat, lon, tVal)
		}
		if math.IsNaN(pos.Azimuth) {
			t.Errorf("Azimuth is NaN for lat=%f, lon=%f, time=%v", lat, lon, tVal)
		}
	})
}
