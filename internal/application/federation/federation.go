// Package federation coordinates multi-hub deployments (Parent/Satellite model)
// linked over encrypted WireGuard tunnels or mTLS endpoints.
package federation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
	"github.com/google/uuid"
)

// HubRole declares the topological function of this hub instance.
type HubRole string

const (
	RoleParent     HubRole = "parent"    // Main residence hub that aggregates all remote entities
	RoleSatellite  HubRole = "satellite" // Remote outbuilding or vacation home hub
	RoleStandalone HubRole = "standalone"
)

// NodeStatus defines the connectivity health of a federated satellite.
type NodeStatus string

const (
	StatusOnline  NodeStatus = "online"
	StatusOffline NodeStatus = "offline"
	StatusSyncing NodeStatus = "syncing"
)

// SatelliteNode represents an enrolled remote Domus appliance.
type SatelliteNode struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Endpoint      string     `json:"endpoint"` // e.g. "https://10.0.8.2:8443" via WireGuard
	AuthToken     string     `json:"auth_token,omitempty"`
	Status        NodeStatus `json:"status"`
	EntitiesCount int        `json:"entities_count"`
	PingLatencyMs float64    `json:"ping_latency_ms"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
}

// RemoteEntity represents an entity hosted on a satellite hub and mirrored onto the parent.
type RemoteEntity struct {
	GlobalID    string         `json:"global_id"` // Format: "<satellite_id>:<local_id>"
	SatelliteID string         `json:"satellite_id"`
	LocalID     string         `json:"local_id"`
	Domain      string         `json:"domain"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes,omitempty"`
	LastUpdated time.Time      `json:"last_updated"`
}

// CommandProxy is an abstraction to transmit command requests across the WireGuard/mTLS link.
type CommandProxy interface {
	ForwardCommand(ctx context.Context, satelliteID, localEntityID, command string, payload any) error
}

// Service manages the federation lifecycle, mirroring, and command routing.
type Service struct {
	mu           sync.RWMutex
	role         HubRole
	satellites   map[string]*SatelliteNode
	entities     map[string]RemoteEntity // Key: GlobalID
	bus          events.Bus
	commandProxy CommandProxy
}

// Config sets up federation parameters.
type Config struct {
	Role         HubRole
	CommandProxy CommandProxy
}

// NewService instantiates a Federation Service.
func NewService(bus events.Bus, cfg Config) *Service {
	role := cfg.Role
	if role == "" {
		role = RoleStandalone
	}

	return &Service{
		role:         role,
		satellites:   make(map[string]*SatelliteNode),
		entities:     make(map[string]RemoteEntity),
		bus:          bus,
		commandProxy: cfg.CommandProxy,
	}
}

// Role returns current hub topology role.
func (s *Service) Role() HubRole {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.role
}

// RegisterSatellite enrolls or updates a satellite node on the parent hub.
func (s *Service) RegisterSatellite(node SatelliteNode) error {
	if node.ID == "" {
		return errors.New("satellite ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	node.Status = StatusOnline
	node.LastHeartbeat = time.Now()
	nodeCopy := node
	s.satellites[node.ID] = &nodeCopy

	slog.Info("federated satellite enrolled",
		"satellite_id", node.ID,
		"name", node.Name,
		"endpoint", node.Endpoint,
	)
	return nil
}

// UpdateHeartbeat records a periodic ping from a satellite node.
func (s *Service) UpdateHeartbeat(satelliteID string, latencyMs float64, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sat, ok := s.satellites[satelliteID]; ok {
		sat.Status = StatusOnline
		sat.LastHeartbeat = now
		sat.PingLatencyMs = latencyMs
	}
}

// SyncEntities batch mirrors entities from a satellite node.
func (s *Service) SyncEntities(satelliteID string, remoteEntities []RemoteEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sat, ok := s.satellites[satelliteID]
	if !ok {
		return fmt.Errorf("satellite %q is not registered", satelliteID)
	}

	for _, e := range remoteEntities {
		globalID := fmt.Sprintf("%s:%s", satelliteID, e.LocalID)
		e.GlobalID = globalID
		e.SatelliteID = satelliteID
		if e.LastUpdated.IsZero() {
			e.LastUpdated = time.Now()
		}
		s.entities[globalID] = e
	}

	sat.EntitiesCount = len(remoteEntities)
	sat.Status = StatusOnline

	slog.Info("synced federated entities from satellite",
		"satellite_id", satelliteID,
		"count", len(remoteEntities),
	)
	return nil
}

// IngestStateChange processes a real-time state update from a satellite and broadcasts it to the parent bus.
func (s *Service) IngestStateChange(satelliteID, localEntityID, newState string, attrs map[string]any, now time.Time) {
	globalID := fmt.Sprintf("%s:%s", satelliteID, localEntityID)

	s.mu.Lock()
	ent, exists := s.entities[globalID]
	if !exists {
		// Auto-register discovered remote entity
		parts := strings.Split(localEntityID, ".")
		domain := "unknown"
		if len(parts) > 0 {
			domain = parts[0]
		}
		ent = RemoteEntity{
			GlobalID:    globalID,
			SatelliteID: satelliteID,
			LocalID:     localEntityID,
			Domain:      domain,
		}
	}
	ent.State = newState
	ent.Attributes = attrs
	ent.LastUpdated = now
	s.entities[globalID] = ent
	s.mu.Unlock()

	// Broadcast domain event on parent hub
	if s.bus != nil {
		go func(e RemoteEntity) {
			pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = s.bus.Publish(pubCtx, events.Event{
				ID:         uuid.NewString(),
				Type:       events.EntityStateChanged,
				OccurredAt: now,
				Source:     "federation." + satelliteID,
				Payload: map[string]any{
					"entity_id":  e.GlobalID,
					"state":      e.State,
					"attributes": e.Attributes,
					"federated":  true,
					"satellite":  satelliteID,
				},
			})
		}(ent)
	}
}

// DispatchCommand forwards an entity command across the federation tunnel to the satellite.
func (s *Service) DispatchCommand(ctx context.Context, globalID, command string, payload any) error {
	parts := strings.SplitN(globalID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid federated entity ID format %q (expected <satellite_id>:<local_id>)", globalID)
	}

	satelliteID := parts[0]
	localID := parts[1]

	s.mu.RLock()
	sat, exists := s.satellites[satelliteID]
	proxy := s.commandProxy
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("satellite %q not found", satelliteID)
	}
	if sat.Status != StatusOnline {
		return fmt.Errorf("satellite %q is currently %s", satelliteID, sat.Status)
	}
	if proxy == nil {
		return errors.New("no command proxy configured to transmit federated commands")
	}

	slog.Info("forwarding command to federated satellite",
		"satellite_id", satelliteID,
		"local_id", localID,
		"command", command,
	)

	return proxy.ForwardCommand(ctx, satelliteID, localID, command, payload)
}

// ListSatellites returns all registered satellite hubs.
func (s *Service) ListSatellites() []SatelliteNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SatelliteNode, 0, len(s.satellites))
	for _, sat := range s.satellites {
		result = append(result, *sat)
	}
	return result
}

// ListEntities returns all mirrored remote entities.
func (s *Service) ListEntities() []RemoteEntity {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RemoteEntity, 0, len(s.entities))
	for _, ent := range s.entities {
		result = append(result, ent)
	}
	return result
}

// GetEntity retrieves a single federated entity by global ID.
func (s *Service) GetEntity(globalID string) (RemoteEntity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ent, ok := s.entities[globalID]
	return ent, ok
}
