package chore

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChore_Validate(t *testing.T) {
	t.Run("valid chore with defaults applied", func(t *testing.T) {
		c := &Chore{
			ID:    "chore-1",
			Title: "Innaffiare le piante",
		}
		require.NoError(t, c.Validate())
		assert.Equal(t, PriorityMedium, c.Priority)
		assert.Equal(t, RecurrenceNone, c.RecurrenceType)
		assert.Equal(t, StatusPending, c.Status)
	})

	t.Run("valid chore preserves explicit values", func(t *testing.T) {
		due := time.Now().Add(24 * time.Hour)
		c := &Chore{
			ID:             "chore-2",
			Title:          "Pulizia filtro cappa",
			Priority:       PriorityUrgent,
			RecurrenceType: RecurrenceMonthly,
			Status:         StatusOverdue,
			DueDate:        &due,
		}
		require.NoError(t, c.Validate())
		assert.Equal(t, PriorityUrgent, c.Priority)
		assert.Equal(t, RecurrenceMonthly, c.RecurrenceType)
		assert.Equal(t, StatusOverdue, c.Status)
	})

	t.Run("empty title", func(t *testing.T) {
		c := &Chore{Title: "   "}
		assert.ErrorIs(t, c.Validate(), ErrInvalidTitle)
	})

	t.Run("title exceeds 150 chars", func(t *testing.T) {
		c := &Chore{Title: strings.Repeat("X", 151)}
		assert.ErrorIs(t, c.Validate(), ErrInvalidTitle)
	})

	t.Run("json serialization", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		c := &Chore{
			ID:              "ch-100",
			Title:           "Lavatrice",
			Priority:        PriorityHigh,
			Status:          StatusCompleted,
			LastCompletedAt: &now,
			LastCompletedBy: "admin",
		}
		data, err := json.Marshal(c)
		require.NoError(t, err)

		var unmarshaled Chore
		require.NoError(t, json.Unmarshal(data, &unmarshaled))
		assert.Equal(t, c.Title, unmarshaled.Title)
		assert.Equal(t, PriorityHigh, unmarshaled.Priority)
		assert.Equal(t, "admin", unmarshaled.LastCompletedBy)
	})
}

func TestChore_NextDueDate(t *testing.T) {
	base := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	t.Run("recurrence daily", func(t *testing.T) {
		c := &Chore{RecurrenceType: RecurrenceDaily}
		next := c.NextDueDate(base)
		require.NotNil(t, next)
		assert.Equal(t, time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC), *next)
	})

	t.Run("recurrence weekly", func(t *testing.T) {
		c := &Chore{RecurrenceType: RecurrenceWeekly}
		next := c.NextDueDate(base)
		require.NotNil(t, next)
		assert.Equal(t, time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), *next)
	})

	t.Run("recurrence monthly", func(t *testing.T) {
		c := &Chore{RecurrenceType: RecurrenceMonthly}
		next := c.NextDueDate(base)
		require.NotNil(t, next)
		assert.Equal(t, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC), *next)
	})

	t.Run("recurrence none or telemetry returns nil", func(t *testing.T) {
		c1 := &Chore{RecurrenceType: RecurrenceNone}
		assert.Nil(t, c1.NextDueDate(base))

		c2 := &Chore{RecurrenceType: RecurrenceTelemetry}
		assert.Nil(t, c2.NextDueDate(base))

		c3 := &Chore{RecurrenceType: RecurrenceType("unknown")}
		assert.Nil(t, c3.NextDueDate(base))
	})
}
