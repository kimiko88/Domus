package mesh

import (
	"context"
	"fmt"
	"strings"

	"github.com/domus-hub/domus/internal/domain/device"
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

type Service struct {
	devRepo device.Repository
}

func NewService(devRepo ...device.Repository) *Service {
	var r device.Repository
	if len(devRepo) > 0 {
		r = devRepo[0]
	}
	return &Service{devRepo: r}
}

// GetTopology returns the RF mesh topology for Zigbee, Z-Wave, or Thread based on database devices.
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

	if s.devRepo == nil {
		return topo, nil
	}

	devices, err := s.devRepo.List(ctx, device.ListFilter{})
	if err != nil {
		return nil, fmt.Errorf("mesh: failed to list devices: %w", err)
	}

	var matchedDevices []*device.Device
	for _, d := range devices {
		intID := strings.ToLower(d.IntegrationID)
		model := strings.ToLower(d.Model)
		meta := strings.ToLower(d.Metadata)
		if strings.Contains(intID, proto) || strings.Contains(model, proto) || strings.Contains(meta, proto) {
			matchedDevices = append(matchedDevices, d)
		}
	}

	// If no devices match this protocol in the database, return empty topology
	if len(matchedDevices) == 0 {
		return topo, nil
	}

	coordID := fmt.Sprintf("%s_coordinator", proto)
	topo.Nodes = append(topo.Nodes, MeshNode{
		ID:       coordID,
		Name:     fmt.Sprintf("Coordinator %s USB", strings.ToUpper(proto)),
		Protocol: proto,
		Type:     NodeCoordinator,
		LQI:      255,
	})

	for _, d := range matchedDevices {
		nodeType := NodeEndDevice
		nameLow := strings.ToLower(d.Name)
		modelLow := strings.ToLower(d.Model)
		if strings.Contains(modelLow, "router") || strings.Contains(nameLow, "presa") || strings.Contains(nameLow, "rele") || strings.Contains(nameLow, "relè") || strings.Contains(nameLow, "plug") {
			nodeType = NodeRouter
		}

		lqi := 200
		if !d.Online {
			lqi = 0
		}

		pID := coordID
		topo.Nodes = append(topo.Nodes, MeshNode{
			ID:       d.ID,
			Name:     d.Name,
			Protocol: proto,
			Type:     nodeType,
			LQI:      lqi,
			ParentID: &pID,
		})

		q := QualityGood
		if lqi >= 180 {
			q = QualityExcellent
		} else if lqi < 30 {
			q = QualityCritical
		}

		topo.Links = append(topo.Links, MeshLink{
			Source:  coordID,
			Target:  d.ID,
			LQI:     lqi,
			Quality: q,
		})
	}

	return topo, nil
}
