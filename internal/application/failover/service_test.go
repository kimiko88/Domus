package failover

import (
	"context"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/events"
)

type mockFailoverBus struct {
	events []events.Event
}

func (m *mockFailoverBus) Publish(ctx context.Context, e events.Event) error {
	m.events = append(m.events, e)
	return nil
}
func (m *mockFailoverBus) Subscribe(t events.EventType, h events.Handler) func() { return func() {} }
func (m *mockFailoverBus) SubscribeAll(h events.Handler) func()                  { return func() {} }
func (m *mockFailoverBus) Close() error                                          { return nil }

func TestFailoverService_HeartbeatAndPromotion(t *testing.T) {
	bus := &mockFailoverBus{}
	svc := NewService("hub_primary", RoleLeader, "192.168.1.101:8080", bus)

	status := svc.GetStatus()
	if status.Role != RoleLeader {
		t.Errorf("expected role leader, got %s", status.Role)
	}

	// 1. Inbound Heartbeat from standby
	resp := svc.ReceiveHeartbeat(HeartbeatPacket{
		SenderNodeID:      "hub_secondary",
		SenderRole:        RoleStandby,
		WALSequenceNumber: 2000000,
		SentAt:            time.Now().UTC(),
	})
	if resp.SenderNodeID != "hub_primary" {
		t.Errorf("expected sender node hub_primary, got %s", resp.SenderNodeID)
	}

	// 2. Standby node test & Auto-failover
	standby := NewService("hub_secondary", RoleStandby, "192.168.1.100:8080", bus)
	if standby.GetStatus().Role != RoleStandby {
		t.Errorf("expected standby role")
	}

	// Simulate missed ticks
	standby.CheckHeartbeatTick()
	if standby.GetStatus().Role != RoleStandby {
		t.Errorf("still standby after 1 tick")
	}

	// Second missed tick triggers promotion
	standby.CheckHeartbeatTick()
	if standby.GetStatus().Role != RoleLeader {
		t.Errorf("expected promotion to leader after 2 missed ticks, got %s", standby.GetStatus().Role)
	}
}
