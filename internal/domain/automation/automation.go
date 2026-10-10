// Package automation defines the automation domain models, DSL, and repository interface.
package automation

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAutomationNotFound = errors.New("automation: not found")
	ErrInvalidName        = errors.New("automation: name cannot be empty")
	ErrMissingTriggers    = errors.New("automation: at least one trigger is required")
	ErrMissingActions     = errors.New("automation: at least one action is required")
)

const (
	RunStatusRunning   = "running"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusSkipped   = "skipped"
)

// Trigger defines a condition source that causes an automation to evaluate.
type Trigger struct {
	ID         string   `json:"id,omitempty"` // Unique trigger identifier (e.g. "movimento", "timeout")
	Platform   string   `json:"platform"`     // "state", "numeric_state", "time", "event"
	EntityKey  string   `json:"entity_key,omitempty"`
	From       string   `json:"from,omitempty"`
	To         string   `json:"to,omitempty"`
	Above      *float64 `json:"above,omitempty"`
	Below      *float64 `json:"below,omitempty"`
	At         string   `json:"at,omitempty"` // "07:30:00"
	EventType  string   `json:"event_type,omitempty"`
	ForSeconds int      `json:"for_seconds,omitempty"`
}

// Condition defines a filter that must evaluate to true for actions to execute.
type Condition struct {
	Condition  string      `json:"condition"` // "state", "numeric_state", "time", "and", "or", "not", "trigger"
	TriggerID  string      `json:"trigger_id,omitempty"`
	EntityKey  string      `json:"entity_key,omitempty"`
	State      string      `json:"state,omitempty"`
	Above      *float64    `json:"above,omitempty"`
	Below      *float64    `json:"below,omitempty"`
	After      string      `json:"after,omitempty"`  // "HH:MM:SS"
	Before     string      `json:"before,omitempty"` // "HH:MM:SS"
	Conditions []Condition `json:"conditions,omitempty"`
}

// ChooseBranch defines a conditional execution branch within a choose action.
type ChooseBranch struct {
	Conditions []Condition `json:"conditions"`
	Sequence   []Action    `json:"sequence"`
}

// Action defines an operation performed when triggers and conditions pass.
type Action struct {
	Type            string         `json:"type"`              // "service", "delay", "event", "choose"
	Service         string         `json:"service,omitempty"` // "turn_on", "turn_off", "toggle"
	TargetEntityKey string         `json:"target_entity_key,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	DelayMs         int            `json:"delay_ms,omitempty"`
	EventType       string         `json:"event_type,omitempty"`
	EventPayload    map[string]any `json:"event_payload,omitempty"`
	Choose          []ChooseBranch `json:"choose,omitempty"`
	Default         []Action       `json:"default,omitempty"`
}

// Automation represents an event-driven automation rule.
type Automation struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	Enabled         bool        `json:"enabled"`
	Triggers        []Trigger   `json:"triggers"`
	Conditions      []Condition `json:"conditions"`
	Actions         []Action    `json:"actions"`
	ExecutionMode   string      `json:"execution_mode"` // "single", "restart", "queued", "parallel"
	CooldownSeconds int         `json:"cooldown_seconds"`
	LastRunAt       *time.Time  `json:"last_run_at,omitempty"`
	LastStatus      string      `json:"last_status,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// Validate checks basic invariants for the automation rule.
func (a *Automation) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return ErrInvalidName
	}
	if len(a.Triggers) == 0 {
		return ErrMissingTriggers
	}
	if len(a.Actions) == 0 {
		return ErrMissingActions
	}
	if a.ExecutionMode == "" {
		a.ExecutionMode = "single"
	}
	if a.CooldownSeconds < 0 {
		a.CooldownSeconds = 0
	}
	return nil
}

// AutomationRun records a single execution of an automation.
type AutomationRun struct {
	ID             string     `json:"id"`
	AutomationID   string     `json:"automation_id"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	Status         string     `json:"status"`          // "running", "completed", "failed", "skipped"
	TriggerContext string     `json:"trigger_context"` // JSON string
	Log            string     `json:"log"`
	DurationMs     int64      `json:"duration_ms"`
}

// Complete marks the run as finished.
func (r *AutomationRun) Complete(finishedAt time.Time, log string) {
	r.FinishedAt = &finishedAt
	r.Status = RunStatusCompleted
	r.Log = log
	r.DurationMs = finishedAt.Sub(r.StartedAt).Milliseconds()
}

// Fail marks the run as failed.
func (r *AutomationRun) Fail(finishedAt time.Time, errMsg string) {
	r.FinishedAt = &finishedAt
	r.Status = RunStatusFailed
	r.Log = errMsg
	r.DurationMs = finishedAt.Sub(r.StartedAt).Milliseconds()
}
