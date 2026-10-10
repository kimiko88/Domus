package tag

import (
	"time"
)

type ActionType string

const (
	ActionTypeScene      ActionType = "scene"
	ActionTypeAutomation ActionType = "automation"
	ActionTypeService    ActionType = "service"
)

type Tag struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	TagID          string     `json:"tag_id"` // NFC serial/UID or QR code string
	ActionType     ActionType `json:"action_type"`
	ActionTargetID string     `json:"action_target_id"`
	LastScannedAt  *time.Time `json:"last_scanned_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
