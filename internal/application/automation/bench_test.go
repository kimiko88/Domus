package automation

import (
	"context"
	"testing"
	"time"

	domainauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
)

func BenchmarkConditionEvaluation_Simple(b *testing.B) {
	eval := NewEvaluator()
	repo := &mockEntityRepoForFuzz{
		entities: map[string]*entity.Entity{
			"sensor.temp": {
				ID:        "e-1",
				EntityKey: "sensor.temp",
				State:     "22.5",
			},
		},
	}

	above := 20.0
	below := 30.0
	cond := domainauto.Condition{
		Condition: "numeric_state",
		EntityKey: "sensor.temp",
		Above:     &above,
		Below:     &below,
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = eval.EvaluateCondition(ctx, cond, repo)
	}
}

func BenchmarkConditionEvaluation_Composite(b *testing.B) {
	eval := NewEvaluator()
	repo := &mockEntityRepoForFuzz{
		entities: map[string]*entity.Entity{
			"sensor.temp":  {EntityKey: "sensor.temp", State: "22.5"},
			"switch.light": {EntityKey: "switch.light", State: "ON"},
		},
	}

	above := 20.0
	cond := domainauto.Condition{
		Condition: "and",
		Conditions: []domainauto.Condition{
			{Condition: "state", EntityKey: "switch.light", State: "ON"},
			{Condition: "numeric_state", EntityKey: "sensor.temp", Above: &above},
		},
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = eval.EvaluateCondition(ctx, cond, repo)
	}
}

func BenchmarkSunEngineCalculate(b *testing.B) {
	engine := NewSunEngine(45.4642, 9.1900)
	tVal := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Calculate(tVal)
	}
}
