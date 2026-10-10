// Package automation provides the evaluation and execution runtime for automation rules.
package automation

import (
	"context"
	"encoding/json"
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

		if trig.To == "" {
			return true // Fires on any change if 'To' not specified
		}

		stateRaw := payloadMap["state"]
		currVal := extractStateString(stateRaw)
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

		val, ok := extractNumericValue(payloadMap["state"])
		if !ok {
			return false
		}

		if trig.Above != nil && val <= *trig.Above {
			return false
		}
		if trig.Below != nil && val >= *trig.Below {
			return false
		}
		return true

	default:
		return false
	}
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
		val, ok := extractNumericValue(ent.State)
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
		nowStr := time.Now().Format("15:04:05")
		if cond.After != "" && nowStr < cond.After {
			return false, nil
		}
		if cond.Before != "" && nowStr > cond.Before {
			return false, nil
		}
		return true, nil

	default:
		return true, nil
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

func extractNumericValue(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		if strings.HasPrefix(val, "{") {
			var m map[string]any
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				if numVal, ok := m["value"].(float64); ok {
					return numVal, true
				}
			}
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	case map[string]any:
		if numVal, ok := val["value"].(float64); ok {
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
