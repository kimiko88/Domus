package tunnel

import (
	"context"
	"strings"
	"testing"
)

func TestWireGuard_KeyPairGeneration(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate WireGuard keypair: %v", err)
	}
	if priv == "" || pub == "" {
		t.Fatal("expected non-empty base64 keys")
	}
	if priv == pub {
		t.Fatal("private and public keys must not be identical")
	}
}

func TestTunnelService_EnrollAndListPeers(t *testing.T) {
	svc, err := NewTunnelService("domus.example.com", 51820)
	if err != nil {
		t.Fatalf("failed to construct TunnelService: %v", err)
	}

	serverCfg := svc.GetServerConfig()
	if serverCfg.ListenPort != 51820 {
		t.Errorf("expected port 51820, got %d", serverCfg.ListenPort)
	}
	if serverCfg.PublicKey == "" {
		t.Fatal("expected non-empty server public key")
	}

	// Enroll Peer 1 (iOS iPhone)
	peer1, err := svc.EnrollPeer(context.Background(), "iPhone 16 Pro")
	if err != nil {
		t.Fatalf("failed to enroll peer 1: %v", err)
	}
	if peer1.ClientIP != "10.14.0.2" {
		t.Errorf("expected IP 10.14.0.2, got %s", peer1.ClientIP)
	}
	if !strings.Contains(peer1.ConfigText, "[Interface]") || !strings.Contains(peer1.ConfigText, "[Peer]") {
		t.Errorf("config text does not contain required wireguard headers: %s", peer1.ConfigText)
	}
	if !strings.HasPrefix(peer1.QRCodeDataURI, "data:image/svg+xml;base64,") {
		t.Errorf("expected SVG QR code data URI, got: %s", peer1.QRCodeDataURI)
	}

	// Enroll Peer 2 (Android Pixel)
	peer2, err := svc.EnrollPeer(context.Background(), "Pixel 9")
	if err != nil {
		t.Fatalf("failed to enroll peer 2: %v", err)
	}
	if peer2.ClientIP != "10.14.0.3" {
		t.Errorf("expected IP 10.14.0.3, got %s", peer2.ClientIP)
	}

	peers := svc.ListPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 enrolled peers, got %d", len(peers))
	}

	// Test Remove Peer
	removed := svc.RemovePeer(peer1.PeerID)
	if !removed {
		t.Error("expected peer1 removal to return true")
	}
	if len(svc.ListPeers()) != 1 {
		t.Fatalf("expected 1 remaining peer, got %d", len(svc.ListPeers()))
	}
}

func TestTunnelService_TailscaleIntegration(t *testing.T) {
	svc, err := NewTunnelService("domus.local", 51820)
	if err != nil {
		t.Fatalf("failed to construct TunnelService: %v", err)
	}

	initial := svc.GetTailscaleStatus()
	if initial.Connected {
		t.Error("expected initial tailscale to be disconnected")
	}

	status := svc.ConnectTailscale("tskey-auth-1234567890")
	if !status.Connected || status.IP == "" {
		t.Errorf("expected connected tailscale status with IP, got %+v", status)
	}
}
