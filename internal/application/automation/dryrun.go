// Package automation provides the dry-run what-if simulator and condition execution tracer.
package automation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	domauto "github.com/domus-hub/domus/internal/domain/automation"
	"github.com/domus-hub/domus/internal/domain/entity"
)

// ConditionTrace represents the outcome and evaluation reasoning for a single condition rule.
type ConditionTrace struct {
	ConditionType string           `json:"condition_type"`
	EntityKey     string           `json:"entity_key,omitempty"`
	ActualValue   any              `json:"actual_value,omitempty"`
	ExpectedValue any              `json:"expected_value,omitempty"`
	Passed        bool             `json:"passed"`
	Reason        string           `json:"reason"`
	SubTraces     []ConditionTrace `json:"sub_traces,omitempty"`
}

// DryRunResult provides the simulated execution outcome of an automation.
type DryRunResult struct {
	AutomationID string           `json:"automation_id"`
	Passed       bool             `json:"passed"`
	EvaluatedAt  time.Time        `json:"evaluated_at"`
	Traces       []ConditionTrace `json:"traces"`
	Actions      []domauto.Action `json:"actions_would_execute"`
}

// Simulator executes What-If evaluations against live or simulated entity states.
type Simulator struct {
	entities entity.Repository
	sun      *SunEngine
}

// NewSimulator creates an automation dry-run simulator.
func NewSimulator(entities entity.Repository, sun *SunEngine) *Simulator {
	return &Simulator{
		entities: entities,
		sun:      sun,
	}
}

// DryRun evaluates the automation rules without triggering real device side effects.
func (s *Simulator) DryRun(ctx context.Context, a *domauto.Automation, evalTime time.Time) (*DryRunResult, error) {
	result := &DryRunResult{
		AutomationID: a.ID,
		EvaluatedAt:  evalTime,
		Traces:       make([]ConditionTrace, 0, len(a.Conditions)),
	}

	allPassed := true
	for _, cond := range a.Conditions {
		trace := s.evaluateCondition(ctx, cond, evalTime)
		result.Traces = append(result.Traces, trace)
		if !trace.Passed {
			allPassed = false
		}
	}

	result.Passed = allPassed
	if allPassed {
		result.Actions = a.Actions
	}

	return result, nil
}

func (s *Simulator) evaluateCondition(ctx context.Context, c domauto.Condition, now time.Time) ConditionTrace {
	trace := ConditionTrace{
		ConditionType: c.Condition,
		EntityKey:     c.EntityKey,
	}

	switch c.Condition {
	case "and":
		passed := true
		for _, sub := range c.Conditions {
			subTrace := s.evaluateCondition(ctx, sub, now)
			trace.SubTraces = append(trace.SubTraces, subTrace)
			if !subTrace.Passed {
				passed = false
			}
		}
		trace.Passed = passed
		trace.Reason = fmt.Sprintf("AND block: %d of %d passed", countPassed(trace.SubTraces), len(trace.SubTraces))
		return trace

	case "or":
		passed := false
		for _, sub := range c.Conditions {
			subTrace := s.evaluateCondition(ctx, sub, now)
			trace.SubTraces = append(trace.SubTraces, subTrace)
			if subTrace.Passed {
				passed = true
			}
		}
		trace.Passed = passed
		trace.Reason = fmt.Sprintf("OR block: %d of %d passed", countPassed(trace.SubTraces), len(trace.SubTraces))
		return trace

	case "not":
		passed := true
		for _, sub := range c.Conditions {
			subTrace := s.evaluateCondition(ctx, sub, now)
			trace.SubTraces = append(trace.SubTraces, subTrace)
			if subTrace.Passed {
				passed = false
			}
		}
		trace.Passed = passed
		trace.Reason = "NOT block evaluation"
		return trace

	case "state":
		trace.ExpectedValue = c.State
		ent, err := s.getEntity(ctx, c.EntityKey)
		if err != nil {
			trace.Passed = false
			trace.Reason = "Entity not found or repository error: " + err.Error()
			return trace
		}
		trace.ActualValue = ent.State
		trace.Passed = ent.State == c.State
		trace.Reason = fmt.Sprintf("Expected state '%s', actual state is '%s'", c.State, ent.State)
		return trace

	case "numeric_state":
		ent, err := s.getEntity(ctx, c.EntityKey)
		if err != nil {
			trace.Passed = false
			trace.Reason = "Entity not found"
			return trace
		}
		num, err := strconv.ParseFloat(ent.State, 64)
		if err != nil {
			trace.Passed = false
			trace.Reason = "State is not a valid number: " + ent.State
			return trace
		}
		trace.ActualValue = num

		passed := true
		reason := fmt.Sprintf("Actual value: %.2f", num)
		if c.Above != nil {
			trace.ExpectedValue = fmt.Sprintf("> %.2f", *c.Above)
			if num <= *c.Above {
				passed = false
				reason += fmt.Sprintf(" (failed: not above %.2f)", *c.Above)
			}
		}
		if c.Below != nil {
			if num >= *c.Below {
				passed = false
				reason += fmt.Sprintf(" (failed: not below %.2f)", *c.Below)
			}
		}
		trace.Passed = passed
		trace.Reason = reason
		return trace

	case "time":
		// Time of day condition
		curTimeStr := now.Format("15:04:05")
		trace.ActualValue = curTimeStr
		passed := true
		if c.After != "" && curTimeStr < c.After {
			passed = false
		}
		if c.Before != "" && curTimeStr > c.Before {
			passed = false
		}
		trace.Passed = passed
		trace.Reason = fmt.Sprintf("Current time %s (after: %s, before: %s)", curTimeStr, c.After, c.Before)
		return trace

	default:
		trace.Passed = true
		trace.Reason = "Condition type passed by default: " + c.Condition
		return trace
	}
}

func (s *Simulator) getEntity(ctx context.Context, key string) (*entity.Entity, error) {
	if s.entities == nil {
		return nil, fmt.Errorf("entity repo not configured")
	}
	return s.entities.GetByKey(ctx, key)
}

func countPassed(traces []ConditionTrace) int {
	cnt := 0
	for _, t := range traces {
		if t.Passed {
			cnt++
		}
	}
	return cnt
}
