// Package automation provides the evaluation and execution runtime for automation rules.
package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
	"github.com/domus-hub/domus/internal/domain/events"
)

// Evaluator checks trigger activations and condition requirements.
type Evaluator struct{}

// NewEvaluator constructs a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateTrigger checks if an incoming domain event matches the given trigger rule.
func (e *Evaluator) EvaluateTrigger(trig automation.Trigger, evt events.Event) bool {
	switch trig.Platform {
	case "event":
		return string(evt.Type) == trig.EventType

	case "state":
		if evt.Type != events.EntityStateChanged {
			return false
		}
		payloadMap, ok := evt.Payload.(map[string]any)
		if !ok {
			return false
		}
		key, _ := payloadMap["entity_key"].(string)
		if key != trig.EntityKey {
			return false
		}

		stateRaw := payloadMap["state"]
		currVal := extractStateString(stateRaw)
		if trig.From != "" {
			oldRaw, exists := payloadMap["old_state"]
			if !exists || !strings.EqualFold(extractStateString(oldRaw), trig.From) {
				return false
			}
		}
		if trig.To == "" {
			return true
		}
		return strings.EqualFold(currVal, trig.To)

	case "numeric_state":
		if evt.Type != events.EntityStateChanged {
			return false
		}
		payloadMap, ok := evt.Payload.(map[string]any)
		if !ok {
			return false
		}
		key, _ := payloadMap["entity_key"].(string)
		if key != trig.EntityKey {
			return false
		}

		value := payloadMap["state"]
		if trig.Attribute != "" {
			attributes := decodeAttributes(payloadMap["attributes"])
			var found bool
			value, found = attributes[trig.Attribute]
			if !found {
				return false
			}
		}
		val, ok := extractNumericValue(value)
		if !ok {
			return false
		}

		if trig.Above != nil && val <= *trig.Above {
			return false
		}
		if trig.Below != nil && val >= *trig.Below {
			return false
		}
		// Numeric-state triggers fire on entry into the configured range.
		if oldRaw, exists := payloadMap["old_state"]; exists {
			oldValue := oldRaw
			if trig.Attribute != "" {
				oldAttributes := decodeAttributes(payloadMap["old_attributes"])
				var found bool
				oldValue, found = oldAttributes[trig.Attribute]
				if !found {
					oldValue = nil
				}
			}
			if oldVal, ok := extractNumericValue(oldValue); ok && numericValueMatchesTrigger(oldVal, trig) {
				return false
			}
		}
		return true

	case "geofence":
		if string(evt.Type) != "presence.state_changed" {
			return false
		}
		geoPayload, ok := evt.Payload.(map[string]any)
		if !ok {
			return false
		}
		state, _ := geoPayload["state"].(string)
		switch strings.ToLower(strings.TrimSpace(trig.EventType)) {
		case "enter_home":
			return state == "just_arrived" || state == "home"
		case "leave_home":
			return state == "just_left" || state == "away"
		default:
			return false
		}

	case "time":
		return e.EvaluateTimeTrigger(trig, time.Now())

	default:
		return false
	}
}

// EvaluateTimeCondition evaluates a local wall-clock window. Windows spanning
// midnight (for example 22:00-06:00) are treated as an OR across midnight.
// Bounds are inclusive and both HH:MM and HH:MM:SS are accepted.
func (e *Evaluator) EvaluateTimeCondition(cond automation.Condition, now time.Time) bool {
	if cond.After == "" && cond.Before == "" {
		return true
	}
	parse := func(value string) (int, bool) {
		if value == "" {
			return 0, true
		}
		parsed, err := time.ParseInLocation("15:04:05", strings.TrimSpace(value), now.Location())
		if err != nil {
			parsed, err = time.ParseInLocation("15:04", strings.TrimSpace(value), now.Location())
		}
		if err != nil {
			return 0, false
		}
		return parsed.Hour()*3600 + parsed.Minute()*60 + parsed.Second(), true
	}
	after, ok := parse(cond.After)
	if !ok {
		return false
	}
	before, ok := parse(cond.Before)
	if !ok {
		return false
	}
	current := now.Hour()*3600 + now.Minute()*60 + now.Second()
	if cond.After == "" {
		return current <= before
	}
	if cond.Before == "" {
		return current >= after
	}
	if after <= before {
		return current >= after && current <= before
	}
	return current >= after || current <= before
}

// EvaluateTimeTrigger matches a daily local-wall-clock time trigger. Both HH:MM and
// HH:MM:SS are accepted; minute precision remains active for the whole matching minute.
func (e *Evaluator) EvaluateTimeTrigger(trig automation.Trigger, now time.Time) bool {
	if trig.Platform != "time" || strings.TrimSpace(trig.At) == "" {
		return false
	}
	scheduled, err := time.ParseInLocation("15:04:05", strings.TrimSpace(trig.At), now.Location())
	hasSeconds := true
	if err != nil {
		scheduled, err = time.ParseInLocation("15:04", strings.TrimSpace(trig.At), now.Location())
		hasSeconds = false
	}
	if err != nil {
		return false
	}
	if now.Hour() != scheduled.Hour() || now.Minute() != scheduled.Minute() {
		return false
	}
	if len(trig.Days) > 0 {
		// ISO-8601 weekday numbering: Monday=1 ... Sunday=7.
		weekday := (int(now.Weekday())+6)%7 + 1
		matched := false
		for _, day := range trig.Days {
			if day < 1 || day > 7 {
				return false
			}
			if day == weekday {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	return !hasSeconds || now.Second() == scheduled.Second()
}

// EvaluateCondition evaluates whether a condition rule holds given current system state.
func (e *Evaluator) EvaluateCondition(
	ctx context.Context,
	cond automation.Condition,
	entityRepo entity.Repository,
) (bool, error) {
	return e.EvaluateConditionWithTrigger(ctx, cond, entityRepo, "")
}

// EvaluateConditionWithTrigger evaluates a condition rule with active trigger context.
func (e *Evaluator) EvaluateConditionWithTrigger(
	ctx context.Context,
	cond automation.Condition,
	entityRepo entity.Repository,
	triggerID string,
) (bool, error) {
	switch cond.Condition {
	case "trigger":
		if cond.TriggerID != "" {
			return cond.TriggerID == triggerID, nil
		}
		if cond.State != "" {
			return cond.State == triggerID, nil
		}
		return false, nil

	case "and":
		for _, sub := range cond.Conditions {
			ok, err := e.EvaluateConditionWithTrigger(ctx, sub, entityRepo, triggerID)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil

	case "or":
		for _, sub := range cond.Conditions {
			ok, err := e.EvaluateConditionWithTrigger(ctx, sub, entityRepo, triggerID)
			if err == nil && ok {
				return true, nil
			}
		}
		return false, nil

	case "not":
		for _, sub := range cond.Conditions {
			ok, err := e.EvaluateConditionWithTrigger(ctx, sub, entityRepo, triggerID)
			if err != nil {
				return false, err
			}
			if ok {
				return false, nil
			}
		}
		return true, nil

	case "state":
		ent, err := entityRepo.GetByKey(ctx, cond.EntityKey)
		if err != nil {
			return false, err
		}
		currVal := extractStateString(ent.State)
		return strings.EqualFold(currVal, cond.State), nil

	case "numeric_state":
		ent, err := entityRepo.GetByKey(ctx, cond.EntityKey)
		if err != nil {
			return false, err
		}
		val, ok := extractEntityNumericValue(ent, cond.Attribute)
		if !ok {
			return false, nil
		}
		if cond.Above != nil && val <= *cond.Above {
			return false, nil
		}
		if cond.Below != nil && val >= *cond.Below {
			return false, nil
		}
		return true, nil

	case "time":
		return e.EvaluateTimeCondition(cond, time.Now()), nil

	default:
		return false, fmt.Errorf("unsupported automation condition %q", cond.Condition)
	}
}

func extractStateString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		// Could be a JSON string like {"state":"ON"} or raw "ON"
		if strings.HasPrefix(val, "{") {
			var m map[string]any
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				if s, ok := m["state"]; ok {
					return fmtString(s)
				}
				if valVal, ok := m["value"]; ok {
					return fmtString(valVal)
				}
			}
		}
		return val
	case map[string]any:
		if s, ok := val["state"]; ok {
			return fmtString(s)
		}
		if valVal, ok := val["value"]; ok {
			return fmtString(valVal)
		}
	}
	return fmtString(v)
}

func extractEntityNumericValue(ent *entity.Entity, attribute string) (float64, bool) {
	if ent == nil {
		return 0, false
	}
	if attribute != "" && ent.Attributes != "" {
		var attributes map[string]any
		if err := json.Unmarshal([]byte(ent.Attributes), &attributes); err == nil {
			if value, exists := attributes[attribute]; exists {
				return extractNumericValue(value)
			}
		}
		return 0, false
	}
	return extractNumericValue(ent.State)
}

func decodeAttributes(value any) map[string]any {
	switch raw := value.(type) {
	case map[string]any:
		return raw
	case string:
		var attrs map[string]any
		_ = json.Unmarshal([]byte(raw), &attrs)
		return attrs
	default:
		return nil
	}
}

func numericValueMatchesTrigger(value float64, trig automation.Trigger) bool {
	if trig.Above != nil && value <= *trig.Above {
		return false
	}
	if trig.Below != nil && value >= *trig.Below {
		return false
	}
	return true
}

func extractNumericValue(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case string:
		if strings.HasPrefix(val, "{") {
			var m map[string]any
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				if numVal, ok := extractNumericValue(m["value"]); ok {
					return numVal, true
				}
				if numVal, ok := extractNumericValue(m["state"]); ok {
					return numVal, true
				}
			}
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	case map[string]any:
		if numVal, ok := extractNumericValue(val["value"]); ok {
			return numVal, true
		}
		if numVal, ok := extractNumericValue(val["state"]); ok {
			return numVal, true
		}
	}
	return 0, false
}

func fmtString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case bool:
		return strconv.FormatBool(val)
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
