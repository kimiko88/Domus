package open_display

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallActionInjectsDiscoveredTransportAndAuthenticates(t *testing.T) {
	const address = "AA:BB:CC:DD:EE:FF"
	const token = "unit-test-bridge-secret-that-is-long-enough"
	var received bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Method != http.MethodPost {
			t.Errorf("method=%s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/devices/"+address+"/actions/led" {
			t.Errorf("path=%q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("bridge bearer token was not forwarded")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode bridge payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["transport"] != "ble" || payload["mac"] != address {
			t.Errorf("discovered transport not injected: %#v", payload)
		}
		if payload["brightness"] != float64(8) {
			t.Errorf("user action payload not forwarded: %#v", payload)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	adapter := &Adapter{
		cfg: Config{Token: token},
		baseURL: server.URL,
		client: server.Client(),
		known: map[string]DeviceInfo{
			address: {Name: "Kitchen display", Address: address, MAC: address, Transport: "ble", Online: true},
		},
	}
	if err := adapter.CallAction(context.Background(), address, "led", json.RawMessage(`{"brightness":8}`)); err != nil {
		t.Fatalf("CallAction returned error: %v", err)
	}
	if !received {
		t.Fatal("bridge was not called")
	}
}

func TestCallActionRejectsUnknownOfflineAndReservedTransport(t *testing.T) {
	adapter := &Adapter{known: map[string]DeviceInfo{
		"AA:BB:CC:DD:EE:01": {Address: "AA:BB:CC:DD:EE:01", Transport: "ble", Online: false},
	}}
	if err := adapter.CallAction(context.Background(), "AA:BB:CC:DD:EE:FF", "shell", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected unknown action to be rejected")
	}
	if err := adapter.CallAction(context.Background(), "AA:BB:CC:DD:EE:01", "led", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected offline device to be rejected")
	}
	if err := adapter.CallAction(context.Background(), "AA:BB:CC:DD:EE:01", "led", json.RawMessage(`{"transport":"wifi"}`)); err == nil {
		t.Fatal("expected transport override to be rejected")
	}
	if err := adapter.CallAction(context.Background(), "AA:BB:CC:DD:EE:01", "led", json.RawMessage(`not-json`)); err == nil {
		t.Fatal("expected malformed JSON to be rejected")
	}
	if err := adapter.CallAction(context.Background(), "AA:BB:CC:DD:EE:01", "led", json.RawMessage(`[]`)); err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("expected JSON object validation error, got %v", err)
	}
}

func TestCallActionReturnsSanitizedBridgeFailure(t *testing.T) {
	const address = "AA:BB:CC:DD:EE:02"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid LED setting"))
	}))
	defer server.Close()
	adapter := &Adapter{
		cfg: Config{Token: "unit-test-bridge-secret-that-is-long-enough"},
		baseURL: server.URL, client: server.Client(),
		known: map[string]DeviceInfo{address: {Address: address, Transport: "ble", Online: true}},
	}
	err := adapter.CallAction(context.Background(), address, "led", json.RawMessage(`{"brightness":8}`))
	if err == nil || !strings.Contains(err.Error(), "HTTP 400") {
		t.Fatalf("expected bridge HTTP failure, got %v", err)
	}
}
