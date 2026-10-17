package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrInvalidSchedule  = errors.New("invalid schedule configuration")
)

type TimeSlot struct {
	Start string  `json:"start"`           // "06:30"
	End   string  `json:"end"`             // "08:00"
	Value float64 `json:"value"`           // e.g. 21.0 (temperature in °C, or 1.0 for on)
	State string  `json:"state,omitempty"` // optional semantic label: "comfort", "eco", "on"
}

type WeeklyGrid map[string][]TimeSlot // "monday", "tuesday", etc.

// Schedule represents a legacy entity schedule.
type Schedule struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	EntityKey string     `json:"entity_key"`
	Grid      WeeklyGrid `json:"grid"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (s *Schedule) GetActiveSlot(t time.Time) (*TimeSlot, bool) {
	if !s.Enabled {
		return nil, false
	}

	dayKey := strings.ToLower(t.Weekday().String())
	slots, ok := s.Grid[dayKey]
	if !ok || len(slots) == 0 {
		return nil, false
	}

	currentMinutes := t.Hour()*60 + t.Minute()

	for _, slot := range slots {
		startMin, err1 := parseTimeToMinutes(slot.Start)
		endMin, err2 := parseTimeToMinutes(slot.End)
		if err1 != nil || err2 != nil {
			continue
		}

		if currentMinutes >= startMin && currentMinutes < endMin {
			return &slot, true
		}
	}

	return nil, false
}

// WeeklySchedule represents a full 7-day x 24h cronotermostato / heating / irrigation schedule matrix.
type WeeklySchedule struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Icon             string     `json:"icon"`
	TargetEntityID   string     `json:"target_entity_id"`
	TargetService    string     `json:"target_service"`
	Grid             WeeklyGrid `json:"grid"`
	CurrentSlotState string     `json:"current_slot_state"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Repository persists WeeklySchedule entities.
type Repository interface {
	Save(ctx context.Context, s *WeeklySchedule) error
	GetByID(ctx context.Context, id string) (*WeeklySchedule, error)
	List(ctx context.Context) ([]*WeeklySchedule, error)
	Delete(ctx context.Context, id string) error
}

// EvaluateAt checks which slot is active in the weekly matrix at a given point in time.
func (s *WeeklySchedule) EvaluateAt(t time.Time) (*TimeSlot, string) {
	dayKey := strings.ToLower(t.Weekday().String())
	slots, ok := s.Grid[dayKey]
	if !ok || len(slots) == 0 {
		return nil, "off"
	}

	currentMinutes := t.Hour()*60 + t.Minute()

	for _, slot := range slots {
		startMin, err1 := parseTimeToMinutes(slot.Start)
		endMin, err2 := parseTimeToMinutes(slot.End)
		if err1 != nil || err2 != nil {
			continue
		}

		if currentMinutes >= startMin && currentMinutes < endMin {
			state := slot.State
			if state == "" {
				if slot.Value > 0 {
					state = fmt.Sprintf("%.1f", slot.Value)
				} else {
					state = "on"
				}
			}
			return &slot, state
		}
	}

	return nil, "off"
}

func parseTimeToMinutes(hhmm string) (int, error) {
	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", hhmm)
	}
	var h, m int
	_, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	if err != nil {
		return 0, err
	}
	return h*60 + m, nil
}
