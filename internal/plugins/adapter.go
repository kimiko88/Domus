// Package plugins defines common interfaces and types for hardware and protocol adapters.
package plugins

import "context"

// Adapter defines the lifecycle and capabilities required for any integration adapter.
type Adapter interface {
	// ID returns the unique identifier of the integration instance.
	ID() string
	// Type returns the protocol/vendor identifier (e.g. "mqtt_generic", "zigbee2mqtt").
	Type() string
	// Start boots up network listeners, background polls, or subscriptions.
	Start(ctx context.Context) error
	// Stop gracefully disconnects and frees adapter resources.
	Stop(ctx context.Context) error
	// HealthCheck returns nil if the adapter connection is active.
	HealthCheck(ctx context.Context) error
}
