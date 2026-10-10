package matter

import (
	"sync"
	"time"
)

// MatterEndpoint represents a hardware endpoint on a Matter node.
type MatterEndpoint struct {
	ID         uint16            `json:"id"`
	DeviceType uint32            `json:"device_type"`
	Clusters   map[uint32]bool   `json:"clusters"`
}

// MatterNode represents an operational commissioned Matter device.
type MatterNode struct {
	mu           sync.RWMutex
	NodeID       uint64                    `json:"node_id"`
	VendorID     uint16                    `json:"vendor_id"`
	ProductID    uint16                    `json:"product_id"`
	Name         string                    `json:"name"`
	IPAddress    string                    `json:"ip_address"`
	Port         int                       `json:"port"`
	DeviceType   uint32                    `json:"device_type"`
	Endpoints    map[uint16]*MatterEndpoint`json:"endpoints"`
	Online       bool                      `json:"online"`
	LastSeenAt   time.Time                 `json:"last_seen_at"`

	// Cluster attributes state cache
	OnOff        bool                      `json:"on_off"`
	Level        uint8                     `json:"level"`         // 0-254
	Temperature  float64                   `json:"temperature"`   // °C
	Humidity     float64                   `json:"humidity"`      // %
	TargetTemp   float64                   `json:"target_temp"`   // °C
	HVACMode     string                    `json:"hvac_mode"`     // off, cool, heat
	Occupancy    bool                      `json:"occupancy"`
	PowerWatts   float64                   `json:"power_watts"`
}

// NewMatterNode constructs a new operational Matter node.
func NewMatterNode(nodeID uint64, name string, ip string, port int, devType uint32) *MatterNode {
	if port <= 0 {
		port = 5540 // Standard Matter UDP port
	}
	n := &MatterNode{
		NodeID:     nodeID,
		Name:       name,
		IPAddress:  ip,
		Port:       port,
		DeviceType: devType,
		Endpoints:  make(map[uint16]*MatterEndpoint),
		Online:     true,
		LastSeenAt: time.Now().UTC(),
		HVACMode:   "off",
	}

	// Default endpoint 1
	n.Endpoints[1] = &MatterEndpoint{
		ID:         1,
		DeviceType: devType,
		Clusters:   make(map[uint32]bool),
	}

	return n
}

// SetOnOff updates the on/off cluster state.
func (n *MatterNode) SetOnOff(on bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.OnOff = on
	n.LastSeenAt = time.Now().UTC()
}

// SetLevel updates level control cluster state.
func (n *MatterNode) SetLevel(lvl uint8) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Level = lvl
	if lvl > 0 {
		n.OnOff = true
	} else {
		n.OnOff = false
	}
	n.LastSeenAt = time.Now().UTC()
}

// SetSensors updates environmental sensor clusters.
func (n *MatterNode) SetSensors(temp, hum float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Temperature = temp
	n.Humidity = hum
	n.LastSeenAt = time.Now().UTC()
}
