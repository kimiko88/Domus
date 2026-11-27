// Package chore defines the domain model, priority, recurrence, and repository interface for household chores.
package chore

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTitle  = errors.New("chore: title cannot be empty and must be under 150 characters")
	ErrChoreNotFound            = errors.New("chore: not found")
	ErrInvalidRecurrenceInterval = errors.New("chore: recurrence interval must be between 1 and 999")
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

type RecurrenceType string

const (
	RecurrenceNone      RecurrenceType = "none"
	RecurrenceDaily     RecurrenceType = "daily"
	RecurrenceWeekly    RecurrenceType = "weekly"
	RecurrenceMonthly   RecurrenceType = "monthly"
	RecurrenceTelemetry RecurrenceType = "telemetry"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusOverdue   Status = "overdue"
)

// Chore represents a domestic chore or maintenance task.
type Chore struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	AreaID          string         `json:"area_id,omitempty"`
	AssignedUserID  string         `json:"assigned_user_id,omitempty"`
	Priority        Priority       `json:"priority"`
	DueDate         *time.Time     `json:"due_date,omitempty"`
	RecurrenceType  RecurrenceType `json:"recurrence_type"`
	RecurrenceRule     string         `json:"recurrence_rule,omitempty"`
	RecurrenceParentID string         `json:"recurrence_parent_id,omitempty"`
	Status             Status         `json:"status"`
	LastCompletedAt *time.Time     `json:"last_completed_at,omitempty"`
	LastCompletedBy string         `json:"last_completed_by,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Validate checks the chore properties.
func (c *Chore) Validate() error {
	trimmed := strings.TrimSpace(c.Title)
	if len(trimmed) == 0 || len(trimmed) > 150 {
		return ErrInvalidTitle
	}
	if c.Priority == "" {
		c.Priority = PriorityMedium
	}
	if c.RecurrenceType == "" {
		c.RecurrenceType = RecurrenceNone
	}
	if c.Status == "" {
		c.Status = StatusPending
	}
	if _, err := parseRecurrenceInterval(c.RecurrenceRule); err != nil {
		return err
	}
	return nil
}

// RecurrenceInterval returns the configured recurrence interval, defaulting to one
// for chores created before interval support was introduced.
func (c *Chore) RecurrenceInterval() int {
	interval, err := parseRecurrenceInterval(c.RecurrenceRule)
	if err != nil {
		return 1
	}
	return interval
}

func parseRecurrenceInterval(rule string) (int, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return 1, nil
	}
	// Older clients stored the interval as a plain number.
	if value, err := strconv.Atoi(rule); err == nil {
		if value < 1 || value > 999 {
			return 0, ErrInvalidRecurrenceInterval
		}
		return value, nil
	}

	normalized := strings.TrimPrefix(strings.ToUpper(rule), "RRULE:")
	for _, part := range strings.Split(normalized, ";") {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) != 2 || !strings.EqualFold(strings.TrimSpace(keyValue[0]), "INTERVAL") {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(keyValue[1]))
		if err != nil || value < 1 || value > 999 {
			return 0, ErrInvalidRecurrenceInterval
		}
		return value, nil
	}
	return 1, nil
}

// NextDueDate calculates the next occurrence after completion for recurring chores.
func (c *Chore) NextDueDate(from time.Time) *time.Time {
	interval := c.RecurrenceInterval()
	var next time.Time
	switch c.RecurrenceType {
	case RecurrenceDaily:
		next = from.AddDate(0, 0, interval)
	case RecurrenceWeekly:
		next = from.AddDate(0, 0, 7*interval)
	case RecurrenceMonthly:
		next = addMonthsClamped(from, interval)
	default:
		return nil
	}
	return &next
}

// addMonthsClamped preserves the day when possible and clamps to the last day
// of shorter months (for example, January 31st becomes February 28th or 29th).
func addMonthsClamped(from time.Time, months int) time.Time {
	targetMonth := time.Date(
		from.Year(), from.Month()+time.Month(months), 1,
		from.Hour(), from.Minute(), from.Second(), from.Nanosecond(), from.Location(),
	)
	lastDay := time.Date(targetMonth.Year(), targetMonth.Month()+1, 0, 0, 0, 0, 0, from.Location()).Day()
	day := from.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(
		targetMonth.Year(), targetMonth.Month(), day,
		from.Hour(), from.Minute(), from.Second(), from.Nanosecond(), from.Location(),
	)
}

// Repository defines storage operations for domestic chores.
type Repository interface {
	Create(ctx context.Context, chore *Chore) error
	GetByID(ctx context.Context, id string) (*Chore, error)
	List(ctx context.Context, filter Filter) ([]*Chore, error)
	Update(ctx context.Context, chore *Chore) error
	ReopenCompleted(ctx context.Context, chore *Chore) error
	Delete(ctx context.Context, id string) error
	GetPendingDueBefore(ctx context.Context, cutoff time.Time) ([]*Chore, error)
}

// Filter specifies query criteria for listing chores.
type Filter struct {
	Status         Status
	AssignedUserID string
	AreaID         string
}
