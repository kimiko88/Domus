// Package alarm provides application services for Home Security Alarm Control Panel.
package alarm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/alarm"
	"github.com/domus-hub/domus/internal/domain/events"
)

// SirenController sends turn_on and turn_off commands to siren entities.
type SirenController interface {
	ControlEntity(ctx context.Context, entityKey, service string) error
}

// Service manages security alarm states, delays, PIN validation and sensor monitoring.
type PINHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, encodedHash string) (bool, error)
}

type pinAttemptState struct {
	failures    int
	blockedUntil time.Time
}

type Service struct {
	db        *sql.DB
	bus       events.Bus
	sirenCtrl SirenController
	pinHasher PINHasher
	pinAttempts map[string]pinAttemptState
	mu        sync.RWMutex

	// Active timers
	pendingTimer *time.Timer
}

// NewService creates a new alarm service instance.
func NewService(db *sql.DB, bus events.Bus, sirenCtrl SirenController) *Service {
	s := &Service{
		db:        db,
		bus:       bus,
		sirenCtrl: sirenCtrl,
		pinAttempts: make(map[string]pinAttemptState),
	}
	s.listenSensorEvents()
	return s
}

// WithPINHasher configures Argon2id hashing and verification for alarm PINs.
func (s *Service) WithPINHasher(hasher PINHasher) *Service {
	s.pinHasher = hasher
	return s
}

// GetPanel retrieves the current alarm panel state.
func (s *Service) GetPanel(ctx context.Context) (*alarm.ControlPanel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getPanelUnsafe(ctx)
}

// Arm arms the system into armed_home, armed_away, or armed_night.
func (s *Service) Arm(ctx context.Context, targetState alarm.State, pin, user string) (*alarm.ControlPanel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := s.getPanelUnsafe(ctx)
	if err != nil {
		return nil, err
	}

	if targetState != alarm.StateArmedHome && targetState != alarm.StateArmedAway && targetState != alarm.StateArmedNight {
		return nil, alarm.ErrInvalidMode
	}

	if err := s.verifyPIN(cp, pin, user); err != nil {
		return nil, err
	}

	// Cancel any active pending timer
	if s.pendingTimer != nil {
		s.pendingTimer.Stop()
		s.pendingTimer = nil
	}

	now := time.Now().UTC()
	cp.State = targetState
	cp.LastChangedBy = user
	cp.UpdatedAt = now

	if err := s.persistPanelUnsafe(ctx, cp); err != nil {
		return nil, err
	}

	s.publishEvent("alarm.state_changed", map[string]any{
		"state":     string(cp.State),
		"user":      user,
		"timestamp": now.Format(time.RFC3339),
	})

	return cp, nil
}

// Disarm disarms the security system with PIN verification and stops sirens.
func (s *Service) Disarm(ctx context.Context, pin, user string) (*alarm.ControlPanel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := s.getPanelUnsafe(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.verifyPIN(cp, pin, user); err != nil {
		return nil, err
	}

	if s.pendingTimer != nil {
		s.pendingTimer.Stop()
		s.pendingTimer = nil
	}

	now := time.Now().UTC()
	cp.State = alarm.StateDisarmed
	cp.LastChangedBy = user
	cp.UpdatedAt = now

	// Turn off all sirens
	if s.sirenCtrl != nil {
		for _, siren := range cp.SirenEntities {
			_ = s.sirenCtrl.ControlEntity(ctx, siren, "turn_off")
		}
	}

	if err := s.persistPanelUnsafe(ctx, cp); err != nil {
		return nil, err
	}

	s.publishEvent("alarm.disarmed", map[string]any{
		"state":     string(cp.State),
		"user":      user,
		"timestamp": now.Format(time.RFC3339),
	})

	return cp, nil
}

// Trigger immediately fires the alarm siren and records trigger reason.
func (s *Service) Trigger(ctx context.Context, reason string) (*alarm.ControlPanel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := s.getPanelUnsafe(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	cp.State = alarm.StateTriggered
	cp.LastTriggeredReason = reason
	cp.TriggeredAt = &now
	cp.UpdatedAt = now

	if s.sirenCtrl != nil {
		for _, siren := range cp.SirenEntities {
			_ = s.sirenCtrl.ControlEntity(ctx, siren, "turn_on")
		}
	}

	if err := s.persistPanelUnsafe(ctx, cp); err != nil {
		return nil, err
	}

	s.publishEvent("alarm.triggered", map[string]any{
		"state":     string(cp.State),
		"reason":    reason,
		"timestamp": now.Format(time.RFC3339),
	})

	return cp, nil
}

// HandleSensorEvent is called when a sensor entity reports state changes.
func (s *Service) HandleSensorEvent(ctx context.Context, entityKey string, detected bool) {
	if !detected {
		return
	}

	s.mu.Lock()
	cp, err := s.getPanelUnsafe(ctx)
	if err != nil || !cp.IsArmed() {
		s.mu.Unlock()
		return
	}

	isPerimeter := containsString(cp.PerimeterSensors, entityKey)
	isInterior := containsString(cp.InteriorSensors, entityKey)

	shouldTrigger := false
	if (cp.State == alarm.StateArmedHome || cp.State == alarm.StateArmedNight) && isPerimeter {
		shouldTrigger = true
	} else if cp.State == alarm.StateArmedAway && (isPerimeter || isInterior) {
		shouldTrigger = true
	}

	if !shouldTrigger {
		s.mu.Unlock()
		return
	}

	// If entry delay is configured, enter pending state
	if cp.EntryDelaySeconds > 0 && cp.State != alarm.StatePending {
		cp.State = alarm.StatePending
		endsAt := time.Now().UTC().Add(time.Duration(cp.EntryDelaySeconds) * time.Second)
		cp.PendingEndsAt = &endsAt
		_ = s.persistPanelUnsafe(ctx, cp)

		s.publishEvent("alarm.pending_entry", map[string]any{
			"sensor":        entityKey,
			"delay_seconds": cp.EntryDelaySeconds,
			"ends_at":       endsAt.Format(time.RFC3339),
		})

		s.pendingTimer = time.AfterFunc(time.Duration(cp.EntryDelaySeconds)*time.Second, func() {
			_, _ = s.Trigger(context.Background(), "Intrusione rilevata dal sensore: "+entityKey)
		})

		s.mu.Unlock()
		return
	}

	s.mu.Unlock()
	_, _ = s.Trigger(ctx, "Intrusione rilevata dal sensore: "+entityKey)
}

// UpdateConfig updates alarm configuration.
func (s *Service) UpdateConfig(ctx context.Context, entryDelay, exitDelay int, perimeter, interior, sirens []string, newPin string) (*alarm.ControlPanel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := s.getPanelUnsafe(ctx)
	if err != nil {
		return nil, err
	}

	if entryDelay >= 0 {
		cp.EntryDelaySeconds = entryDelay
	}
	if exitDelay >= 0 {
		cp.ExitDelaySeconds = exitDelay
	}
	if perimeter != nil {
		cp.PerimeterSensors = perimeter
	}
	if interior != nil {
		cp.InteriorSensors = interior
	}
	if sirens != nil {
		cp.SirenEntities = sirens
	}
	if newPin != "" {
		if len(newPin) < 6 || len(newPin) > 12 {
			return nil, fmt.Errorf("alarm: PIN must contain between 6 and 12 digits")
		}
		for _, digit := range newPin {
			if digit < '0' || digit > '9' {
				return nil, fmt.Errorf("alarm: PIN must contain only digits")
			}
		}
		if s.pinHasher == nil {
			return nil, fmt.Errorf("alarm: PIN hashing is unavailable")
		}
		hashed, err := s.pinHasher.HashPassword(newPin)
		if err != nil {
			return nil, fmt.Errorf("alarm: failed to hash PIN: %w", err)
		}
		cp.PinHash = hashed
	}
	cp.UpdatedAt = time.Now().UTC()

	if err := s.persistPanelUnsafe(ctx, cp); err != nil {
		return nil, err
	}

	return cp, nil
}

func (s *Service) verifyPIN(cp *alarm.ControlPanel, pin, actor string) error {
	if cp.PinHash == "" {
		return alarm.ErrPinNotConfigured
	}
	if s.pinHasher == nil {
		return fmt.Errorf("alarm: PIN verifier is unavailable")
	}
	key := strings.TrimSpace(actor)
	if key == "" {
		key = "anonymous"
	}
	now := time.Now().UTC()
	attempt := s.pinAttempts[key]
	if attempt.blockedUntil.After(now) {
		return alarm.ErrPinRateLimited
	}
	valid, err := s.pinHasher.VerifyPassword(strings.TrimSpace(pin), cp.PinHash)
	if err != nil {
		return fmt.Errorf("alarm: PIN verifier rejected stored hash: %w", err)
	}
	if !valid {
		attempt.failures++
		if attempt.failures >= 5 {
			attempt.blockedUntil = now.Add(5 * time.Minute)
		}
		s.pinAttempts[key] = attempt
		return alarm.ErrInvalidPin
	}
	delete(s.pinAttempts, key)
	return nil
}

func (s *Service) getPanelUnsafe(ctx context.Context) (*alarm.ControlPanel, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, state, pin_hash, entry_delay_seconds, exit_delay_seconds, trigger_delay_seconds,
		       perimeter_sensors_json, interior_sensors_json, siren_entities_json,
		       last_changed_by, last_triggered_reason, triggered_at, updated_at
		FROM alarm_control_panels
		WHERE id = 'alarm_main'
	`)

	var cp alarm.ControlPanel
	var pinHash, lastChangedBy, lastTriggeredReason sql.NullString
	var perimJSON, interiorJSON, sirenJSON string
	var triggeredAt sql.NullTime

	err := row.Scan(
		&cp.ID, &cp.Name, &cp.State, &pinHash, &cp.EntryDelaySeconds, &cp.ExitDelaySeconds, &cp.TriggerDelaySeconds,
		&perimJSON, &interiorJSON, &sirenJSON,
		&lastChangedBy, &lastTriggeredReason, &triggeredAt, &cp.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		cp = alarm.ControlPanel{
			ID:                "alarm_main",
			Name:              "Allarme Domus Casa",
			State:             alarm.StateDisarmed,
			PinHash:           "",
			EntryDelaySeconds: 30,
			ExitDelaySeconds:  60,
			PerimeterSensors:  []string{},
			InteriorSensors:   []string{},
			SirenEntities:     []string{},
			UpdatedAt:         time.Now().UTC(),
		}
		_, insertErr := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO alarm_control_panels (
				id, name, state, pin_hash, entry_delay_seconds, exit_delay_seconds,
				perimeter_sensors_json, interior_sensors_json, siren_entities_json
			) VALUES ('alarm_main', 'Allarme Domus Casa', 'disarmed', NULL, 30, 60, '[]', '[]', '[]')
		`)
		if insertErr != nil {
			return nil, fmt.Errorf("alarm: initialize panel: %w", insertErr)
		}
		return &cp, nil
	} else if err != nil {
		return nil, err
	}

	cp.PinHash = pinHash.String
	_ = json.Unmarshal([]byte(perimJSON), &cp.PerimeterSensors)
	_ = json.Unmarshal([]byte(interiorJSON), &cp.InteriorSensors)
	_ = json.Unmarshal([]byte(sirenJSON), &cp.SirenEntities)

	if lastChangedBy.Valid {
		cp.LastChangedBy = lastChangedBy.String
	}
	if lastTriggeredReason.Valid {
		cp.LastTriggeredReason = lastTriggeredReason.String
	}
	if triggeredAt.Valid {
		t := triggeredAt.Time
		cp.TriggeredAt = &t
	}

	// Migrate legacy plaintext PINs in place. The historical default 1234 is
	// discarded rather than carried forward as a usable security credential.
	if cp.PinHash == "1234" {
		cp.PinHash = ""
		if _, err := s.db.ExecContext(ctx, "UPDATE alarm_control_panels SET pin_hash = NULL WHERE id = 'alarm_main'"); err != nil {
			return nil, fmt.Errorf("alarm: clear insecure default PIN: %w", err)
		}
	} else if cp.PinHash != "" && !strings.HasPrefix(cp.PinHash, "$argon2id$") {
		if s.pinHasher == nil {
			return nil, fmt.Errorf("alarm: legacy PIN needs migration but Argon2id is unavailable")
		}
		encoded, err := s.pinHasher.HashPassword(strings.TrimSpace(cp.PinHash))
		if err != nil {
			return nil, fmt.Errorf("alarm: migrate legacy PIN: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, "UPDATE alarm_control_panels SET pin_hash = ? WHERE id = 'alarm_main'", encoded); err != nil {
			return nil, fmt.Errorf("alarm: persist migrated PIN: %w", err)
		}
		cp.PinHash = encoded
	}

	return &cp, nil
}

func (s *Service) persistPanelUnsafe(ctx context.Context, cp *alarm.ControlPanel) error {
	perimJSON, _ := json.Marshal(cp.PerimeterSensors)
	interiorJSON, _ := json.Marshal(cp.InteriorSensors)
	sirenJSON, _ := json.Marshal(cp.SirenEntities)

	var trigTime any = nil
	if cp.TriggeredAt != nil {
		trigTime = *cp.TriggeredAt
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alarm_control_panels (
			id, name, state, pin_hash, entry_delay_seconds, exit_delay_seconds, trigger_delay_seconds,
			perimeter_sensors_json, interior_sensors_json, siren_entities_json,
			last_changed_by, last_triggered_reason, triggered_at, updated_at
		) VALUES ('alarm_main', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			state = excluded.state,
			pin_hash = excluded.pin_hash,
			entry_delay_seconds = excluded.entry_delay_seconds,
			exit_delay_seconds = excluded.exit_delay_seconds,
			trigger_delay_seconds = excluded.trigger_delay_seconds,
			perimeter_sensors_json = excluded.perimeter_sensors_json,
			interior_sensors_json = excluded.interior_sensors_json,
			siren_entities_json = excluded.siren_entities_json,
			last_changed_by = excluded.last_changed_by,
			last_triggered_reason = excluded.last_triggered_reason,
			triggered_at = excluded.triggered_at,
			updated_at = excluded.updated_at
	`, cp.Name, cp.State, cp.PinHash, cp.EntryDelaySeconds, cp.ExitDelaySeconds, cp.TriggerDelaySeconds,
		string(perimJSON), string(interiorJSON), string(sirenJSON),
		cp.LastChangedBy, cp.LastTriggeredReason, trigTime, cp.UpdatedAt)

	return err
}

func (s *Service) publishEvent(eventType string, data any) {
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), events.Event{
			Type:       events.EventType(eventType),
			OccurredAt: time.Now().UTC(),
			Payload:    data,
		})
	}
}

func (s *Service) listenSensorEvents() {
	if s.bus == nil {
		return
	}
	s.bus.Subscribe(events.EntityStateChanged, func(ctx context.Context, ev events.Event) error {
		payloadMap, ok := ev.Payload.(map[string]any)
		if !ok {
			return nil
		}
		entityKey, _ := payloadMap["entity_key"].(string)
		stateStr := fmt.Sprintf("%v", payloadMap["state"])
		detected := stateStr == "true" || stateStr == "ON" || stateStr == "on" || stateStr == "open"
		s.HandleSensorEvent(ctx, entityKey, detected)
		return nil
	})
}

func containsString(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
