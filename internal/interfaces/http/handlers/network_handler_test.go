package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/network"
	"github.com/domus-hub/domus/internal/interfaces/http/handlers"
)

type mockARPReaderForHandler struct {
	hosts []network.DiscoveredHost
}

func (m *mockARPReaderForHandler) ReadEntries(ctx context.Context) ([]network.DiscoveredHost, error) {
	return m.hosts, nil
}

func TestNetworkHandler_ListAndTriggerScan(t *testing.T) {
	scanner := network.NewScanner(nil, network.Config{
		ARPReader: &mockARPReaderForHandler{
			hosts: []network.DiscoveredHost{
				{IP: "192.168.1.100", MAC: "34:94:54:11:22:33"}, // Shelly
			},
		},
	})

	h := handlers.NewNetworkHandler(scanner)

	// 1. Initial list before scan
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/network/devices", nil)
	wList := httptest.NewRecorder()
	h.ListDiscovered(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wList.Code)
	}

	// 2. Trigger Scan
	reqScan := httptest.NewRequest(http.MethodPost, "/api/v1/network/scan", nil)
	wScan := httptest.NewRecorder()
	h.TriggerScan(wScan, reqScan)
	if wScan.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for scan, got %d", wScan.Code)
	}

	var scanResp struct {
		Count int                      `json:"count"`
		Hosts []network.DiscoveredHost `json:"hosts"`
	}
	if err := json.NewDecoder(wScan.Body).Decode(&scanResp); err != nil {
		t.Fatalf("failed to decode scan response: %v", err)
	}
	if scanResp.Count != 1 {
		t.Errorf("expected 1 discovered host, got %d", scanResp.Count)
	}
	if scanResp.Hosts[0].Vendor != "Shelly (Allterco)" {
		t.Errorf("expected Shelly (Allterco), got %q", scanResp.Hosts[0].Vendor)
	}

	// 3. Probe Host
	body, _ := json.Marshal(map[string]string{"ip": "192.168.1.100"})
	reqProbe := httptest.NewRequest(http.MethodPost, "/api/v1/network/probe", bytes.NewReader(body))
	wProbe := httptest.NewRecorder()
	h.ProbeHost(wProbe, reqProbe)
	if wProbe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for probe, got %d", wProbe.Code)
	}
}
