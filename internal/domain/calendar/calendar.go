package calendar

import (
	"bufio"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location,omitempty"`
	AllDay      bool      `json:"all_day"`
	Source      string    `json:"source"` // "local", "caldav", "google"
	CreatedAt   time.Time `json:"created_at"`
}

// ParseICal parses standard iCalendar (.ics) content into Event objects.
func ParseICal(content string) ([]*Event, error) {
	var events []*Event
	scanner := bufio.NewScanner(strings.NewReader(content))

	var current *Event
	inVEvent := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "BEGIN:VEVENT" {
			inVEvent = true
			current = &Event{
				ID:        "cal_" + uuid.NewString()[:8],
				Source:    "caldav",
				CreatedAt: time.Now().UTC(),
			}
			continue
		}
		if line == "END:VEVENT" && inVEvent {
			if current != nil {
				events = append(events, current)
			}
			inVEvent = false
			current = nil
			continue
		}

		if !inVEvent || current == nil {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		val := parts[1]

		if strings.HasPrefix(key, "SUMMARY") {
			current.Summary = val
		} else if strings.HasPrefix(key, "DESCRIPTION") {
			current.Description = val
		} else if strings.HasPrefix(key, "LOCATION") {
			current.Location = val
		} else if strings.HasPrefix(key, "DTSTART") {
			t, err := parseICalTime(val)
			if err == nil {
				current.StartTime = t
			}
		} else if strings.HasPrefix(key, "DTEND") {
			t, err := parseICalTime(val)
			if err == nil {
				current.EndTime = t
			}
		}
	}

	return events, scanner.Err()
}

func parseICalTime(s string) (time.Time, error) {
	// Standard formats: 20261010T153000Z or 20261010
	s = strings.TrimSpace(s)
	if strings.Contains(s, "T") {
		if strings.HasSuffix(s, "Z") {
			return time.Parse("20060102T150405Z", s)
		}
		return time.Parse("20060102T150405", s)
	}
	return time.Parse("20060102", s)
}
