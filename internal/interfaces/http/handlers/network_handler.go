// Package handlers provides HTTP REST handlers for network discovery and device fingerprinting.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/domus-hub/domus/internal/infrastructure/network"
)

// NetworkHandler manages API endpoints for network device discovery and fingerprinting.
type NetworkHandler struct {
	scanner *network.Scanner
}

// NewNetworkHandler creates a NetworkHandler instance.
func NewNetworkHandler(scanner *network.Scanner) *NetworkHandler {
	return &NetworkHandler{scanner: scanner}
}

// ListDiscovered returns all currently discovered network devices with vendor fingerprints.
func (h *NetworkHandler) ListDiscovered(w http.ResponseWriter, r *http.Request) {
	if h.scanner == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	hosts := h.scanner.ListHosts()
	writeJSON(w, http.StatusOK, hosts)
}

// TriggerScan runs an immediate active/passive ARP scan and returns discovered devices.
func (h *NetworkHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	if h.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "network scanner is not initialized")
		return
	}

	hosts, err := h.scanner.ScanARP(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scan_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(hosts),
		"hosts": hosts,
	})
}

// ProbeHost requests an in-depth TCP port probe on a specific host to refine its fingerprint.
func (h *NetworkHandler) ProbeHost(w http.ResponseWriter, r *http.Request) {
	if h.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "network scanner is not initialized")
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IP == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "valid IP address required in request body")
		return
	}

	openPorts, err := h.scanner.ProbeHostPorts(r.Context(), req.IP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "probe_failed", err.Error())
		return
	}

	host, _ := h.scanner.GetHost(req.IP)
	writeJSON(w, http.StatusOK, map[string]any{
		"ip":         req.IP,
		"open_ports": openPorts,
		"host":       host,
	})
}
