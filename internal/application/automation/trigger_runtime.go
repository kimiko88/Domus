package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
)

func triggerIdentifier(trigger domauto.Trigger, index int) string {
	if strings.TrimSpace(trigger.ID) != "" {
		return trigger.ID
	}
	return fmt.Sprintf("trigger-%d", index)
}

// seedEntityObservations initializes transition tracking from persisted states so
// a "from" trigger can work immediately after a hub restart.
func (e *Engine) seedEntityObservations(ctx context.Context) {
	if e.entityRepo == nil {
		return
	}
	entities, err := e.entityRepo.List(ctx, entity.ListFilter{})
	if err != nil {
		slog.Warn("automation: unable to seed entity trigger states", "error", err)
		return
	}
	e.triggerMu.Lock()
	defer e.triggerMu.Unlock()
	for _, item := range entities {
		if item == nil || strings.TrimSpace(item.EntityKey) == "" {
			continue
		}
		e.entityEvents[item.EntityKey] = events.Event{
			Type: events.EntityStateChanged,
			OccurredAt: time.Now().UTC(),
			Source: "automation.state_seed",
			Payload: map[string]any{
				"entity_key": item.EntityKey,
				"state": item.State,
				"attributes": item.Attributes,
			},
		}
	}
}

// trackEntityStateEvent attaches the previous state to incoming events. Adapter
// event payloads remain immutable outside the automation engine.
func (e *Engine) trackEntityStateEvent(evt events.Event) events.Event {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return evt
	}
	entityKey, _ := payload["entity_key"].(string)
	if strings.TrimSpace(entityKey) == "" {
		return evt
	}
	enriched := make(map[string]any, len(payload)+2)
	for key, value := range payload {
		enriched[key] = value
	}

	e.triggerMu.Lock()
	if previous, exists := e.entityEvents[entityKey]; exists {
		if _, supplied := enriched["old_state"]; !supplied {
			if oldPayload, ok := previous.Payload.(map[string]any); ok {
				if oldState, exists := oldPayload["state"]; exists {
					enriched["old_state"] = oldState
				}
				if oldAttributes, exists := oldPayload["attributes"]; exists {
					enriched["old_attributes"] = oldAttributes
				}
			}
		}
	}
	updated := evt
	updated.Payload = enriched
	e.entityEvents[entityKey] = updated
	e.triggerMu.Unlock()

	e.cancelInvalidPendingTriggers(entityKey, updated)
	return updated
}

// scheduleDelayedAutomationTrigger fires a state trigger only after its target
// condition has remained true for the configured duration.
func (e *Engine) scheduleDelayedAutomationTrigger(
	fallback context.Context,
	rule *domauto.Automation,
	trigger domauto.Trigger,
	triggerID string,
	evt events.Event,
) {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return
	}
	entityKey, _ := payload["entity_key"].(string)
	if entityKey == "" {
		return
	}
	key := rule.ID + ":" + triggerID

	parent := e.executionContext(fallback)
	timerCtx, cancel := context.WithCancel(parent)
	pending := &pendingAutomationTrigger{
		cancel: cancel,
		entityKey: entityKey,
		trigger: trigger,
		rule: rule,
		triggerID: triggerID,
		event: evt,
	}
	e.triggerMu.Lock()
	if existing := e.pendingTriggers[key]; existing != nil {
		e.triggerMu.Unlock()
		cancel()
		return
	}
	e.pendingTriggers[key] = pending
	e.triggerMu.Unlock()

	go func() {
		timer := time.NewTimer(time.Duration(trigger.ForSeconds) * time.Second)
		defer timer.Stop()
		select {
		case <-timerCtx.Done():
			return
		case <-timer.C:
		}

		e.triggerMu.Lock()
		if e.pendingTriggers[key] != pending {
			e.triggerMu.Unlock()
			return
		}
		delete(e.pendingTriggers, key)
		latest, exists := e.entityEvents[entityKey]
		e.triggerMu.Unlock()
		cancel()
		if !exists || !e.triggerStillMatches(trigger, latest) {
			return
		}

		e.mu.Lock()
		running := e.running
		executionCtx := e.executionCtx
		e.mu.Unlock()
		if !running || executionCtx == nil || executionCtx.Err() != nil {
			return
		}
		triggerContext, _ := json.Marshal(map[string]any{
			"trigger": trigger,
			"event": pending.event,
			"for_seconds": trigger.ForSeconds,
			"confirmed_at": time.Now().UTC().Format(time.RFC3339Nano),
		})
		go e.launchRuleExecution(executionCtx, rule, triggerID, string(triggerContext))
	}()
}

func (e *Engine) triggerStillMatches(trigger domauto.Trigger, evt events.Event) bool {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return false
	}
	current := make(map[string]any, len(payload))
	for key, value := range payload {
		if key != "old_state" && key != "old_attributes" {
			current[key] = value
		}
	}
	currentEvent := evt
	currentEvent.Payload = current
	trigger.From = ""
	trigger.ForSeconds = 0
	return e.evaluator.EvaluateTrigger(trigger, currentEvent)
}

func (e *Engine) cancelInvalidPendingTriggers(entityKey string, current events.Event) {
	e.triggerMu.Lock()
	defer e.triggerMu.Unlock()
	for key, pending := range e.pendingTriggers {
		if pending == nil || pending.entityKey != entityKey {
			continue
		}
		if !e.triggerStillMatches(pending.trigger, current) {
			pending.cancel()
			delete(e.pendingTriggers, key)
		}
	}
}

func (e *Engine) cancelPendingAutomationTriggers() {
	e.triggerMu.Lock()
	defer e.triggerMu.Unlock()
	for key, pending := range e.pendingTriggers {
		if pending != nil && pending.cancel != nil {
			pending.cancel()
		}
		delete(e.pendingTriggers, key)
	}
}

func (e *Engine) isSolarTriggerDue(trigger domauto.Trigger, now time.Time) bool {
	e.mu.Lock()
	sun := e.sunEngine
	e.mu.Unlock()
	if sun == nil {
		return false
	}
	position := sun.Calculate(now)
	var eventTime time.Time
	switch strings.ToLower(strings.TrimSpace(trigger.EventType)) {
	case "sunrise":
		eventTime = position.Sunrise
	case "sunset":
		eventTime = position.Sunset
	default:
		return false
	}
	dayLength := position.Sunset.Sub(position.Sunrise)
	if eventTime.IsZero() || dayLength <= 0 || dayLength >= 24*time.Hour {
		// SunEngine returns an empty or full-day interval when there is no
		// distinct sunrise/sunset (polar night/day). Do not fire fake events.
		return false
	}
	if len(trigger.Days) > 0 {
		weekday := (int(now.Weekday())+6)%7 + 1
		matched := false
		for _, day := range trigger.Days {
			if day == weekday {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	localEvent := eventTime.In(now.Location())
	return localEvent.Year() == now.Year() &&
		localEvent.YearDay() == now.YearDay() &&
		localEvent.Hour() == now.Hour() &&
		localEvent.Minute() == now.Minute()
}
