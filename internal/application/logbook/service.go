// Package logbook provides service and event bus listeners for human-readable home activity logs.
package logbook

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
	domlog "github.com/domus-hub/domus/internal/domain/logbook"
	"github.com/google/uuid"
)

// Service translates low-level events into human-readable timeline entries and queries them.
type Service struct {
	repo       domlog.Repository
	entityRepo entity.Repository
	bus        events.Bus
	unsubs     []func()
	mu         sync.Mutex
}

// NewService creates a new logbook Service.
func NewService(repo domlog.Repository, entityRepo entity.Repository, bus events.Bus) *Service {
	return &Service{
		repo:       repo,
		entityRepo: entityRepo,
		bus:        bus,
	}
}

// Start begins subscribing to domain events to generate activity entries.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bus == nil {
		return nil
	}

	u1 := s.bus.Subscribe(events.EntityStateChanged, s.handleEntityStateChanged)
	u2 := s.bus.Subscribe(events.AutomationTriggered, s.handleAutomationTriggered)
	u3 := s.bus.Subscribe(events.AlarmTriggered, s.handleAlarmTriggered)
	u4 := s.bus.Subscribe(events.PresenceUpdated, s.handlePresenceUpdated)

	s.unsubs = append(s.unsubs, u1, u2, u3, u4)
	slog.Info("logbook service started and listening for domain events")
	return nil
}

// Stop cleanly unregisters all event bus listeners.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.unsubs {
		if u != nil {
			u()
		}
	}
	s.unsubs = nil
}

// Query retrieves logbook entries matching the filter.
func (s *Service) Query(ctx context.Context, filter domlog.Filter) ([]*domlog.Entry, error) {
	return s.repo.List(ctx, filter)
}

// LogEntry explicitly records a log entry.
func (s *Service) LogEntry(ctx context.Context, entry *domlog.Entry) error {
	return s.repo.Save(ctx, entry)
}

func (s *Service) handleEntityStateChanged(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}

	var entityKey string
	if k, ok := payload["entity_key"].(string); ok && k != "" {
		entityKey = k
	} else if id, ok := payload["entity_id"].(string); ok {
		entityKey = id
	}
	if entityKey == "" {
		return nil
	}

	stateStr := fmt.Sprintf("%v", payload["state"])
	domain := ""
	name := entityKey
	if parts := strings.SplitN(entityKey, ".", 2); len(parts) == 2 {
		domain = parts[0]
		name = parts[1]
	}

	if s.entityRepo != nil {
		if ent, err := s.entityRepo.GetByKey(ctx, entityKey); err == nil && ent != nil {
			name = ent.Name
			domain = ent.Domain
		}
	}

	// Format human message
	message := formatEntityMessage(name, domain, stateStr)

	entry := &domlog.Entry{
		ID:          uuid.NewString(),
		EntityID:    entityKey,
		EntityName:  name,
		Domain:      domain,
		State:       stateStr,
		Message:     message,
		ContextType: "system",
		Timestamp:   evt.OccurredAt,
	}

	return s.repo.Save(ctx, entry)
}

func (s *Service) handleAutomationTriggered(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}
	name, _ := payload["name"].(string)
	autoID, _ := payload["automation_id"].(string)
	if name == "" {
		name = autoID
	}

	return s.repo.Save(ctx, &domlog.Entry{
		ID:          uuid.NewString(),
		EntityID:    fmt.Sprintf("automation.%s", autoID),
		EntityName:  name,
		Domain:      "automation",
		State:       "triggered",
		Message:     fmt.Sprintf("Automazione '%s' attivata", name),
		ContextType: "automation",
		ContextID:   autoID,
		ContextName: name,
		Timestamp:   evt.OccurredAt,
	})
}

func (s *Service) handleAlarmTriggered(ctx context.Context, evt events.Event) error {
	return s.repo.Save(ctx, &domlog.Entry{
		ID:          uuid.NewString(),
		EntityID:    "alarm_control_panel.security_alarm",
		EntityName:  "Allarme Sicurezza",
		Domain:      "alarm_control_panel",
		State:       "triggered",
		Message:     "Allarme sicurezza intrusione attivato!",
		ContextType: "system",
		Timestamp:   evt.OccurredAt,
	})
}

func (s *Service) handlePresenceUpdated(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}
	name, _ := payload["name"].(string)
	state, _ := payload["state"].(string)

	msg := fmt.Sprintf("%s è fuori casa", name)
	if state == "home" {
		msg = fmt.Sprintf("%s è arrivato a casa", name)
	} else if state != "not_home" && state != "" {
		msg = fmt.Sprintf("%s è arrivato in zona '%s'", name, state)
	}

	return s.repo.Save(ctx, &domlog.Entry{
		ID:          uuid.NewString(),
		EntityID:    fmt.Sprintf("person.%s", strings.ToLower(name)),
		EntityName:  name,
		Domain:      "person",
		State:       state,
		Message:     msg,
		ContextType: "user",
		ContextName: name,
		Timestamp:   evt.OccurredAt,
	})
}

func formatEntityMessage(name, domain, state string) string {
	switch domain {
	case "light":
		if state == "on" || strings.Contains(state, `"on":true`) {
			return fmt.Sprintf("Luce '%s' accesa", name)
		}
		return fmt.Sprintf("Luce '%s' spenta", name)
	case "switch":
		if state == "on" || strings.Contains(state, `"on":true`) {
			return fmt.Sprintf("Interruttore '%s' attivato", name)
		}
		return fmt.Sprintf("Interruttore '%s' disattivato", name)
	case "binary_sensor":
		if state == "on" || strings.Contains(state, `"state":true`) {
			return fmt.Sprintf("Sensore '%s' rilevato", name)
		}
		return fmt.Sprintf("Sensore '%s' tornato a riposo", name)
	case "lock":
		if state == "locked" {
			return fmt.Sprintf("Serratura '%s' bloccata", name)
		}
		return fmt.Sprintf("Serratura '%s' sbloccata", name)
	default:
		return fmt.Sprintf("%s: stato cambiato in %s", name, state)
	}
}
