package tuya_local

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
)

func TestDeviceClientMergesPartialDPSReports(t *testing.T) {
	client := &DeviceClient{}
	waiter := &commandWaiter{
		sequence: 11,
		command:  CmdControl,
		expected: map[string]any{"2": 215, "4": 1},
		ack: make(chan error, 1),
		reported: make(chan map[string]any, 1),
		confirmed: make(chan map[string]any, 1),
		observed: make(map[string]any),
	}
	client.pending = waiter

	first, _ := json.Marshal(map[string]any{"dps": map[string]any{"2": 215}})
	client.handleResponsePacket(&Packet{Sequence: 100, Command: CmdStatus, Payload: first})
	select {
	case <-waiter.confirmed:
		t.Fatal("a partial DPS report must not confirm the complete command")
	default:
	}

	second, _ := json.Marshal(map[string]any{"dps": map[string]any{"4": 1}})
	client.handleResponsePacket(&Packet{Sequence: 101, Command: CmdStatus, Payload: second})

	select {
	case state := <-waiter.confirmed:
		if state["2"] != float64(215) || state["4"] != float64(1) {
			t.Fatalf("merged confirmed DPS = %+v", state)
		}
	case <-time.After(time.Second):
		t.Fatal("expected merged partial DPS reports to confirm the command")
	}

	latest := client.LatestDPS()
	if latest["2"] != float64(215) || latest["4"] != float64(1) {
		t.Fatalf("latest DPS snapshot = %+v", latest)
	}
}

func TestSameDPSValueAcceptsNumericStrings(t *testing.T) {
	if !sameDPSValue(215, "215.0") {
		t.Fatal("numeric DPS strings should match numeric command values")
	}
	if sameDPSValue(215, "214") {
		t.Fatal("different numeric DPS values must not match")
	}
}

func TestRequestStatusSendsVersionSpecificQuery(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := &DeviceClient{
		LocalKey: []byte("0123456789abcdef"),
		Version:  "3.3",
		conn:     clientConn,
		stopCh:   make(chan struct{}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.requestStatus(ctx)
	}()

	frame := make([]byte, 24)
	if _, err := io.ReadFull(serverConn, frame); err != nil {
		t.Fatalf("read status query frame: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("request status: %v", err)
	}
	packet, err := DecodePacket(frame, client.LocalKey)
	if err != nil {
		t.Fatalf("decode status query: %v", err)
	}
	if packet.Command != CmdDpQuery {
		t.Fatalf("status query command = %#x, want %#x", packet.Command, CmdDpQuery)
	}
}
