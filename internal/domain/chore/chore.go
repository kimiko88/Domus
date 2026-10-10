// Package chore defines the domain model, priority, recurrence, and repository interface for household chores.
package chore

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidTitle  = errors.New("chore: title cannot be empty and must be under 150 characters")
	ErrChoreNotFound = errors.New("chore: not found")
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
	RecurrenceRule  string         `json:"recurrence_rule,omitempty"`
	Status          Status         `json:"status"`
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
	return nil
}

// NextDueDate calculates the next occurrence after completion for recurring chores.
func (c *Chore) NextDueDate(from time.Time) *time.Time {
	var next time.Time
	switch c.RecurrenceType {
	case RecurrenceDaily:
		next = from.AddDate(0, 0, 1)
	case RecurrenceWeekly:
		next = from.AddDate(0, 0, 7)
	case RecurrenceMonthly:
		next = from.AddDate(0, 1, 0)
	default:
		return nil
	}
	return &next
}

// Repository defines storage operations for domestic chores.
type Repository interface {
	Create(ctx context.Context, chore *Chore) error
	GetByID(ctx context.Context, id string) (*Chore, error)
	List(ctx context.Context, filter Filter) ([]*Chore, error)
	Update(ctx context.Context, chore *Chore) error
	Delete(ctx context.Context, id string) error
	GetPendingDueBefore(ctx context.Context, cutoff time.Time) ([]*Chore, error)
}

// Filter specifies query criteria for listing chores.
type Filter struct {
	Status         Status
	AssignedUserID string
	AreaID         string
}
