package mesh

import (
	"context"
	"fmt"
	"strings"
)

type NodeType string

const (
	NodeCoordinator NodeType = "coordinator"
	NodeRouter      NodeType = "router"
	NodeEndDevice   NodeType = "end_device"
)

type MeshNode struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol string   `json:"protocol"` // "zigbee", "zwave", "thread"
	Type     NodeType `json:"type"`
	LQI      int      `json:"lqi"` // 0 to 255
	Battery  *int     `json:"battery,omitempty"`
	ParentID *string  `json:"parent_id,omitempty"`
}

type LinkQuality string

const (
	QualityExcellent LinkQuality = "excellent" // LQI >= 180
	QualityGood      LinkQuality = "good"      // 100 <= LQI < 180
	QualityWeak      LinkQuality = "weak"      // 30 <= LQI < 100
	QualityCritical  LinkQuality = "critical"  // LQI < 30
)

type MeshLink struct {
	Source  string      `json:"source"`
	Target  string      `json:"target"`
	LQI     int         `json:"lqi"`
	Quality LinkQuality `json:"quality"`
}

type MeshTopology struct {
	Protocol string     `json:"protocol"`
	Nodes    []MeshNode `json:"nodes"`
	Links    []MeshLink `json:"links"`
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetTopology returns the active RF mesh topology for Zigbee, Z-Wave, or Thread.
func (s *Service) GetTopology(ctx context.Context, protocol string) (*MeshTopology, error) {
	proto := strings.ToLower(protocol)
	if proto == "" {
		proto = "zigbee"
	}

	topo := &MeshTopology{
		Protocol: proto,
		Nodes:    make([]MeshNode, 0),
		Links:    make([]MeshLink, 0),
	}

	coordID := fmt.Sprintf("%s_coordinator", proto)
	topo.Nodes = append(topo.Nodes, MeshNode{
		ID:       coordID,
		Name:     fmt.Sprintf("Coordinator %s USB", strings.ToUpper(proto)),
		Protocol: proto,
		Type:     NodeCoordinator,
		LQI:      255,
	})

	// Add sample mesh nodes (Routers and EndDevices) for demonstration / telemetry
	r1ID := fmt.Sprintf("%s_router_salotto", proto)
	r2ID := fmt.Sprintf("%s_router_cucina", proto)
	ed1ID := fmt.Sprintf("%s_sensor_balcone", proto)
	ed2ID := fmt.Sprintf("%s_sensor_camera", proto)

	battery85 := 85
	battery92 := 92

	topo.Nodes = append(topo.Nodes,
		MeshNode{
			ID:       r1ID,
			Name:     "Presa Smart Salotto (Router)",
			Protocol: proto,
			Type:     NodeRouter,
			LQI:      215,
			ParentID: &coordID,
		},
		MeshNode{
			ID:       r2ID,
			Name:     "Relè Incasso Cucina (Router)",
			Protocol: proto,
			Type:     NodeRouter,
			LQI:      160,
			ParentID: &coordID,
		},
		MeshNode{
			ID:       ed1ID,
			Name:     "Sensore Meteo Balcone",
			Protocol: proto,
			Type:     NodeEndDevice,
			LQI:      75,
			Battery:  &battery85,
			ParentID: &r1ID,
		},
		MeshNode{
			ID:       ed2ID,
			Name:     "Sensore Presenza Camera",
			Protocol: proto,
			Type:     NodeEndDevice,
			LQI:      190,
			Battery:  &battery92,
			ParentID: &r2ID,
		},
	)

	// Add links with calculated quality
	addLink := func(src, dst string, lqi int) {
		q := QualityCritical
		if lqi >= 180 {
			q = QualityExcellent
		} else if lqi >= 100 {
			q = QualityGood
		} else if lqi >= 30 {
			q = QualityWeak
		}
		topo.Links = append(topo.Links, MeshLink{
			Source:  src,
			Target:  dst,
			LQI:     lqi,
			Quality: q,
		})
	}

	addLink(coordID, r1ID, 215)
	addLink(coordID, r2ID, 160)
	addLink(r1ID, r2ID, 110)
	addLink(r1ID, ed1ID, 75)
	addLink(r2ID, ed2ID, 190)

	return topo, nil
}
