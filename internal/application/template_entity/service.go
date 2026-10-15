// Package template_entity provides the runtime evaluation engine and service for dynamic template entities.
package template_entity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/domus-hub/domus/internal/domain/entity"
	domte "github.com/domus-hub/domus/internal/domain/template_entity"
	"github.com/domus-hub/domus/internal/domain/events"
)

var statesRegex = regexp.MustCompile(`states\(['"]([^'"]+)['"]\)`)

// Service manages template entities, evaluates expressions, and publishes updates on the event bus.
type Service struct {
	teRepo     domte.Repository
	entityRepo entity.Repository
	bus        events.Bus
	mu         sync.RWMutex
	unsub      func()
}

// NewService creates a new template entity runtime Service.
func NewService(
	teRepo domte.Repository,
	entityRepo entity.Repository,
	bus events.Bus,
) *Service {
	return &Service{
		teRepo:     teRepo,
		entityRepo: entityRepo,
		bus:        bus,
	}
}

// Start begins listening to the event bus for state changes of source entities.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initial evaluation and sync of all template entities into entity catalog
	templates, err := s.teRepo.List(ctx)
	if err == nil {
		for _, te := range templates {
			_ = s.evaluateAndPublish(ctx, te)
		}
	}

	if s.bus != nil {
		s.unsub = s.bus.Subscribe(events.EntityStateChanged, s.handleEntityStateChanged)
	}
	slog.Info("template entity engine started")
	return nil
}

// Stop cleanly unregisters event subscriptions.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unsub != nil {
		s.unsub()
		s.unsub = nil
	}
	slog.Info("template entity engine stopped")
}

func (s *Service) handleEntityStateChanged(ctx context.Context, evt events.Event) error {
	payload, ok := evt.Payload.(map[string]any)
	if !ok {
		return nil
	}

	var changedEntityID string
	if id, ok := payload["entity_id"].(string); ok && id != "" {
		changedEntityID = id
	} else if key, ok := payload["entity_key"].(string); ok && key != "" {
		changedEntityID = key
	}
	if changedEntityID == "" {
		return nil
	}

	templates, err := s.teRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, te := range templates {
		isSource := false
		for _, src := range te.SourceEntityIDs {
			if src == changedEntityID {
				isSource = true
				break
			}
		}
		if isSource {
			go func(t *domte.TemplateEntity) {
				_ = s.evaluateAndPublish(context.Background(), t)
			}(te)
		}
	}
	return nil
}

// CreateTemplateEntity validates, extracts source entity dependencies, and stores a new template entity.
func (s *Service) CreateTemplateEntity(ctx context.Context, te *domte.TemplateEntity) error {
	if err := te.Validate(); err != nil {
		return err
	}

	// Automatically extract referenced source entities from expression if not explicitly provided
	if len(te.SourceEntityIDs) == 0 {
		te.SourceEntityIDs = ExtractSourceEntityIDs(te.StateTemplate)
	}

	if err := s.teRepo.Create(ctx, te); err != nil {
		return err
	}

	// Register or update virtual entity in entity.Repository
	return s.evaluateAndPublish(ctx, te)
}

// UpdateTemplateEntity updates an existing template entity and re-evaluates its state.
func (s *Service) UpdateTemplateEntity(ctx context.Context, te *domte.TemplateEntity) error {
	if err := te.Validate(); err != nil {
		return err
	}
	if len(te.SourceEntityIDs) == 0 {
		te.SourceEntityIDs = ExtractSourceEntityIDs(te.StateTemplate)
	}

	if err := s.teRepo.Update(ctx, te); err != nil {
		return err
	}
	return s.evaluateAndPublish(ctx, te)
}

// DeleteTemplateEntity removes a template entity and cleans up the catalog.
func (s *Service) DeleteTemplateEntity(ctx context.Context, id string) error {
	te, err := s.teRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.teRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Remove from entity repository if present
	_ = s.entityRepo.Delete(ctx, te.EntityID)
	return nil
}

// Evaluate evaluates a template expression string against current repository states.
func (s *Service) Evaluate(ctx context.Context, tmpl string) (string, error) {
	stateLookup := func(id string) string {
		ent, err := s.entityRepo.GetByID(ctx, id)
		if err != nil || ent == nil {
			return "0"
		}
		return ent.State
	}
	return EvaluateExpression(tmpl, stateLookup)
}

// Preview evaluates a template expression with optionally supplied mock or live states.
func (s *Service) Preview(ctx context.Context, tmpl string, mockStates map[string]string) (string, error) {
	stateLookup := func(id string) string {
		if val, exists := mockStates[id]; exists {
			return val
		}
		ent, err := s.entityRepo.GetByID(ctx, id)
		if err != nil || ent == nil {
			return "0"
		}
		return ent.State
	}
	return EvaluateExpression(tmpl, stateLookup)
}

func (s *Service) evaluateAndPublish(ctx context.Context, te *domte.TemplateEntity) error {
	stateResult, err := s.Evaluate(ctx, te.StateTemplate)
	if err != nil {
		stateResult = "unavailable"
	}

	// Prepare attributes
	attrs := map[string]any{
		"friendly_name": te.Name,
		"template_id":   te.ID,
		"is_template":   true,
	}
	if te.UnitOfMeasurement != "" {
		attrs["unit_of_measurement"] = te.UnitOfMeasurement
	}
	if te.DeviceClass != "" {
		attrs["device_class"] = te.DeviceClass
	}
	if te.Icon != "" {
		attrs["icon"] = te.Icon
	}

	if te.AttributesTemplate != "" && te.AttributesTemplate != "{}" {
		var extraAttrs map[string]any
		if err := json.Unmarshal([]byte(te.AttributesTemplate), &extraAttrs); err == nil {
			for k, v := range extraAttrs {
				attrs[k] = v
			}
		}
	}

	attrsBytes, _ := json.Marshal(attrs)
	attrsStr := string(attrsBytes)

	ent, err := s.entityRepo.GetByID(ctx, te.EntityID)
	now := time.Now().UTC()
	if err != nil || ent == nil {
		entityKey := te.Domain + "." + strings.ReplaceAll(strings.ToLower(te.Name), " ", "_")
		if strings.Contains(te.EntityID, ".") {
			entityKey = te.EntityID
		}
		newEntity := &entity.Entity{
			ID:                te.EntityID,
			EntityKey:         entityKey,
			Domain:            te.Domain,
			Name:              te.Name,
			State:             stateResult,
			Attributes:        attrsStr,
			UnitOfMeasurement: te.UnitOfMeasurement,
			Icon:              te.Icon,
			Available:         true,
			LastChangedAt:     now,
			LastUpdatedAt:     now,
		}
		_ = s.entityRepo.Create(ctx, newEntity)
	} else {
		oldState := ent.State
		ent.State = stateResult
		ent.Attributes = attrsStr
		if te.UnitOfMeasurement != "" {
			ent.UnitOfMeasurement = te.UnitOfMeasurement
		}
		if te.Icon != "" {
			ent.Icon = te.Icon
		}
		ent.LastUpdatedAt = now
		if oldState != stateResult {
			ent.LastChangedAt = now
		}
		_ = s.entityRepo.Update(ctx, ent)
	}

	// Broadcast state changed event
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.Event{
			ID:         uuid.NewString(),
			Type:       events.EntityStateChanged,
			OccurredAt: now,
			Source:     "template_entity",
			Payload: map[string]any{
				"entity_id":  te.EntityID,
				"state":      stateResult,
				"attributes": attrsStr,
				"available":  true,
			},
		})
	}

	return nil
}

// ExtractSourceEntityIDs parses an expression string and extracts all states('entity_id') references.
func ExtractSourceEntityIDs(expr string) []string {
	matches := statesRegex.FindAllStringSubmatch(expr, -1)
	var ids []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}
	return ids
}

// EvaluateExpression evaluates an expression with states('...') replacements, arithmetic, and logic.
func EvaluateExpression(expr string, stateLookup func(string) string) (string, error) {
	expr = strings.TrimSpace(expr)
	// Strip {{ and }} if present (Jinja2 style)
	expr = strings.TrimPrefix(expr, "{{")
	expr = strings.TrimSuffix(expr, "}}")
	expr = strings.TrimSpace(expr)

	// Replace all states('entity_id') with their actual state values
	replaced := statesRegex.ReplaceAllStringFunc(expr, func(m string) string {
		match := statesRegex.FindStringSubmatch(m)
		if len(match) > 1 {
			return stateLookup(match[1])
		}
		return "0"
	})

	// 0. Function: round(inner_math_expr, decimals)
	if strings.HasPrefix(replaced, "round(") && strings.HasSuffix(replaced, ")") {
		inner := strings.TrimSuffix(strings.TrimPrefix(replaced, "round("), ")")
		lastComma := strings.LastIndex(inner, ",")
		decimals := 2
		mathPart := inner
		if lastComma != -1 {
			mathPart = strings.TrimSpace(inner[:lastComma])
			decStr := strings.TrimSpace(inner[lastComma+1:])
			if d, err := strconv.Atoi(decStr); err == nil {
				decimals = d
			}
		}
		val, err := evaluateMath(mathPart)
		if err == nil {
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, val), nil
		}
	}

	// 1. Ternary condition: condition ? true_val : false_val
	if strings.Contains(replaced, "?") && strings.Contains(replaced, ":") {
		parts := strings.SplitN(replaced, "?", 2)
		condPart := strings.TrimSpace(parts[0])
		subParts := strings.SplitN(parts[1], ":", 2)
		trueVal := strings.Trim(strings.TrimSpace(subParts[0]), `"'`)
		falseVal := strings.Trim(strings.TrimSpace(subParts[1]), `"'`)

		isTrue, err := evaluateBoolean(condPart)
		if err == nil {
			if isTrue {
				return trueVal, nil
			}
			return falseVal, nil
		}
	}

	// 2. Comparison evaluation: >, <, >=, <=, ==, !=
	if isComparison(replaced) {
		res, err := evaluateBoolean(replaced)
		if err == nil {
			if res {
				return "on", nil
			}
			return "off", nil
		}
	}

	// 3. Mathematical arithmetic evaluation: +, -, *, /
	if isArithmetic(replaced) {
		val, err := evaluateMath(replaced)
		if err == nil {
			if math.Trunc(val) == val {
				return fmt.Sprintf("%.0f", val), nil
			}
			return fmt.Sprintf("%.2f", val), nil
		}
	}

	// Return raw string if no evaluation was needed
	return strings.Trim(replaced, `"'`), nil
}

func isComparison(s string) bool {
	return strings.Contains(s, "==") || strings.Contains(s, "!=") ||
		strings.Contains(s, ">=") || strings.Contains(s, "<=") ||
		strings.Contains(s, ">") || strings.Contains(s, "<")
}

func isArithmetic(s string) bool {
	return strings.Contains(s, "+") || strings.Contains(s, "-") ||
		strings.Contains(s, "*") || strings.Contains(s, "/")
}

func evaluateBoolean(s string) (bool, error) {
	s = strings.TrimSpace(s)
	// Boolean literal check
	if strings.EqualFold(s, "true") || strings.EqualFold(s, "on") {
		return true, nil
	}
	if strings.EqualFold(s, "false") || strings.EqualFold(s, "off") {
		return false, nil
	}

	ops := []string{">=", "<=", "==", "!=", ">", "<"}
	for _, op := range ops {
		if idx := strings.Index(s, op); idx != -1 {
			leftStr := strings.Trim(strings.TrimSpace(s[:idx]), `"'`)
			rightStr := strings.Trim(strings.TrimSpace(s[idx+len(op):]), `"'`)

			leftNum, leftErr := strconv.ParseFloat(leftStr, 64)
			rightNum, rightErr := strconv.ParseFloat(rightStr, 64)

			if leftErr == nil && rightErr == nil {
				switch op {
				case ">":
					return leftNum > rightNum, nil
				case "<":
					return leftNum < rightNum, nil
				case ">=":
					return leftNum >= rightNum, nil
				case "<=":
					return leftNum <= rightNum, nil
				case "==":
					return leftNum == rightNum, nil
				case "!=":
					return leftNum != rightNum, nil
				}
			} else {
				// String comparison
				switch op {
				case "==":
					return leftStr == rightStr, nil
				case "!=":
					return leftStr != rightStr, nil
				}
			}
		}
	}
	return false, fmt.Errorf("unable to evaluate boolean expression: %s", s)
}

func evaluateMath(s string) (float64, error) {
	// Simple left-to-right parser for binary math operations
	s = strings.TrimSpace(s)
	for _, op := range []string{"+", "-", "*", "/"} {
		if idx := strings.Index(s, op); idx != -1 {
			left, err1 := strconv.ParseFloat(strings.TrimSpace(s[:idx]), 64)
			right, err2 := strconv.ParseFloat(strings.TrimSpace(s[idx+1:]), 64)
			if err1 == nil && err2 == nil {
				switch op {
				case "+":
					return left + right, nil
				case "-":
					return left - right, nil
				case "*":
					return left * right, nil
				case "/":
					if right == 0 {
						return 0, fmt.Errorf("division by zero")
					}
					return left / right, nil
				}
			}
		}
	}
	return strconv.ParseFloat(s, 64)
}
