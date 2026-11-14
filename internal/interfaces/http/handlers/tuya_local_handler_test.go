package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/plugins/tuya_local"
	"github.com/go-chi/chi/v5"
)

func TestTuyaLocalHandler_ListDevices(t *testing.T) {
	bus := &mockBusForTest{}
	adapter := tuya_local.NewAdapter("tuya_local", bus)

	adapter.RegisterDevice(tuya_local.DeviceConfig{
		DeviceID:  "local_001",
		Name:      "Living Room Plug",
		IP:        "192.168.1.155",
		LocalKey:  "1234567890123456",
		Version:   "3.3",
		Category:  tuya_local.CategoryPlug,
		EntityKey: "switch.tuya_local_001",
	})

	handler := NewTuyaLocalHandler(adapter)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListDevices status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Devices []struct {
			DeviceID  string `json:"device_id"`
			Name      string `json:"name"`
			IP        string `json:"ip"`
			Category  string `json:"category"`
			EntityKey string `json:"entity_key"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode devices: %v", err)
	}

	if len(resp.Devices) != 1 || resp.Devices[0].DeviceID != "local_001" || resp.Devices[0].IP != "192.168.1.155" {
		t.Fatalf("unexpected devices returned: %+v", resp.Devices)
	}
}
