// Package audit defines the audit logging domain models and repository contracts.
package audit

import "time"

// ActorType identifies what entity performed an action.
type ActorType string

const (
	ActorLocalUser  ActorType = "local_user"
	ActorAPIToken   ActorType = "api_token"
	ActorAutomation ActorType = "automation"
	ActorSystem     ActorType = "system"
)

// Entry represents an immutable audit log record.
type Entry struct {
	ID         int64     `json:"id"`
	ActorType  ActorType `json:"actor_type"`
	ActorID    string    `json:"actor_id,omitempty"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type,omitempty"`
	TargetID   string    `json:"target_id,omitempty"`
	Payload    string    `json:"payload,omitempty"` // JSON string
	CreatedAt  time.Time `json:"created_at"`
}
