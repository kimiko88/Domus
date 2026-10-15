// Package events defines the domain event types and the event bus contract.
// All domain events are value objects: immutable after creation.
package events

import "time"

// EventType identifies the kind of domain event.
type EventType string

const (
	// SystemStarted fires once when the application finishes startup.
	SystemStarted EventType = "system.started"

	// DeviceDiscovered fires when an adapter finds a new device.
	DeviceDiscovered EventType = "device.discovered"
	// DeviceOnline fires when a previously-offline device comes back online.
	DeviceOnline EventType = "device.online"
	// DeviceOffline fires when a device stops responding.
	DeviceOffline EventType = "device.offline"

	// IntegrationStateChanged fires when an integration adapter connects, disconnects, or changes health state.
	IntegrationStateChanged EventType = "integration.state_changed"

	// EntityStateChanged fires every time an entity's state or availability changes.
	EntityStateChanged EventType = "entity.state_changed"

	// CommandRequested fires when a user or automation requests a command on an entity.
	CommandRequested EventType = "command.requested"
	// CommandSucceeded fires when the adapter confirms a command was accepted.
	CommandSucceeded EventType = "command.succeeded"
	// CommandFailed fires when a command could not be executed.
	CommandFailed EventType = "command.failed"

	// AutomationTriggered fires when an automation's trigger condition is met.
	AutomationTriggered EventType = "automation.triggered"
	// AutomationRunCompleted fires when an automation run finishes successfully.
	AutomationRunCompleted EventType = "automation.run_completed"
	// AutomationRunFailed fires when an automation run encounters a fatal error.
	AutomationRunFailed EventType = "automation.run_failed"

	// SceneActivated fires when a scene is applied.
	SceneActivated EventType = "scene.activated"

	// NotificationCreated fires when an internal notification is generated.
	NotificationCreated EventType = "notification.created"

	// CameraDetected fires when an object (person, vehicle, animal, package) is detected by Frigate/camera vision.
	CameraDetected EventType = "camera.detected"

	// TagScanned fires when an NFC tag or QR code is scanned.
	TagScanned EventType = "tag.scanned"

	// CalendarEventStart fires when a scheduled calendar event begins.
	CalendarEventStart EventType = "calendar.event_start"

	// CalendarEventEnd fires when a scheduled calendar event ends.
	CalendarEventEnd EventType = "calendar.event_end"

	// MediaStateChanged fires when a media player state, volume, or track changes.
	MediaStateChanged EventType = "media.state_changed"

	// OTAStateChanged fires when a device firmware upgrade state updates.
	OTAStateChanged EventType = "ota.state_changed"

	// AlarmTriggered fires when the security alarm enters triggered state.
	AlarmTriggered EventType = "alarm.triggered"

	// AlarmDisarmed fires when the security alarm is successfully disarmed.
	AlarmDisarmed EventType = "alarm.disarmed"

	// WebhookReceived fires when an inbound webhook is executed.
	WebhookReceived EventType = "webhook.received"

	// PresenceUpdated fires when a person's spatial zone or GPS presence state changes.
	PresenceUpdated EventType = "presence.updated"
)

// Event is the common envelope for all domain events.
// Handlers must not mutate the event after receiving it.
type Event struct {
	// ID is a unique identifier for this event occurrence (UUID recommended).
	ID string `json:"id"`

	// Type identifies the kind of event.
	Type EventType `json:"type"`

	// Payload carries event-specific data. Handlers should type-assert to the
	// expected payload struct. nil is valid for events without extra data.
	Payload any `json:"payload,omitempty"`

	// OccurredAt is the wall-clock time the event happened.
	OccurredAt time.Time `json:"occurred_at"`

	// Source is the adapter or component that produced the event.
	// Examples: "mqtt_generic", "zigbee2mqtt", "user:550e8400-...", "system".
	Source string `json:"source,omitempty"`
}
