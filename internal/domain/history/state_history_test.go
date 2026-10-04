package history_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/history"
)

func TestStateHistory_Validate(t *testing.T) {
	t.Run("fails when entity_id is empty", func(t *testing.T) {
		h := &history.StateHistory{
			State: `"on"`,
		}
		err := h.Validate()
		if err != history.ErrInvalidEntityID {
			t.Fatalf("expected ErrInvalidEntityID, got: %v", err)
		}
	})

	t.Run("succeeds and assigns RecordedAt if zero", func(t *testing.T) {
		h := &history.StateHistory{
			EntityID: "sensor.temp",
			State:    `{"value": 21.5}`,
		}
		err := h.Validate()
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if h.RecordedAt.IsZero() {
			t.Fatalf("expected RecordedAt to be set")
		}
	})

	t.Run("preserves existing RecordedAt", func(t *testing.T) {
		fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		h := &history.StateHistory{
			EntityID:   "sensor.temp",
			State:      `22.0`,
			RecordedAt: fixedTime,
		}
		err := h.Validate()
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if !h.RecordedAt.Equal(fixedTime) {
			t.Fatalf("expected %v, got %v", fixedTime, h.RecordedAt)
		}
	})
}
