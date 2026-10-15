package template_entity_test

import (
	"context"
	"testing"

	appte "github.com/domus-hub/domus/internal/application/template_entity"
)

func TestExtractSourceEntityIDs(t *testing.T) {
	expr := "states('sensor.living_room_temp') > 21 ? 'warm' : states('sensor.outside_temp')"
	ids := appte.ExtractSourceEntityIDs(expr)
	if len(ids) != 2 {
		t.Fatalf("expected 2 extracted ids, got %d", len(ids))
	}
	if ids[0] != "sensor.living_room_temp" || ids[1] != "sensor.outside_temp" {
		t.Errorf("unexpected extracted ids: %v", ids)
	}
}

func TestTemplatePreview(t *testing.T) {
	svc := appte.NewService(nil, nil, nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		template   string
		mockStates map[string]string
		expected   string
	}{
		{
			name:       "arithmetic expression",
			template:   "states('sensor.solar') - states('sensor.consumption')",
			mockStates: map[string]string{"sensor.solar": "3500", "sensor.consumption": "1200"},
			expected:   "2300",
		},
		{
			name:       "ternary comparison",
			template:   "states('sensor.temp') > 25 ? 'hot' : 'comfortable'",
			mockStates: map[string]string{"sensor.temp": "28.5"},
			expected:   "hot",
		},
		{
			name:       "ternary false branch",
			template:   "states('sensor.temp') > 25 ? 'hot' : 'comfortable'",
			mockStates: map[string]string{"sensor.temp": "22.0"},
			expected:   "comfortable",
		},
		{
			name:       "round function",
			template:   "round(states('sensor.power') / 1000, 2)",
			mockStates: map[string]string{"sensor.power": "2345.67"},
			expected:   "2.35",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := svc.Preview(ctx, tc.template, tc.mockStates)
			if err != nil {
				t.Fatalf("preview failed: %v", err)
			}
			if res != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, res)
			}
		})
	}
}
