package schedule

import (
	"fmt"
	"strings"
	"time"
)

type TimeSlot struct {
	Start string  `json:"start"` // "06:30"
	End   string  `json:"end"`   // "08:00"
	Value float64 `json:"value"` // e.g. 20.5 (temperature in °C)
}

type WeeklyGrid map[string][]TimeSlot // "monday", "tuesday", etc.

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
