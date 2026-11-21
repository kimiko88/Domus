package imou_life

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidAPIMethod(t *testing.T) {
	valid := []string{"getStreamUrl", "v2/device.detail.page", "v2/device.get"}
	for _, method := range valid {
		if !validAPIMethod(method) { t.Errorf("expected %q to be allowed", method) }
	}
	invalid := []string{"", "/v2/device.detail.page", "v2//device", "../device", "v2/../device", "v2/device?x=1", "https://example.com"}
	for _, method := range invalid {
		if validAPIMethod(method) { t.Errorf("expected %q to be rejected", method) }
	}
}

func TestListDevicesUsesSignedFixedHostAndDoesNotExposeProviderSecrets(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"code":"0","data":{"count":1,"deviceList":[{"deviceId":"dev-1","deviceName":"Ingresso","channelList":[{"channelId":"0","channelName":"Canale"}],"playToken":"must-not-be-exposed","encryptedInfo":"must-not-be-exposed"}]}}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{AccessKey:"app-key",SecretKey:"app-secret",DataCenter:"eu"}, server.Client())
	if err != nil { t.Fatal(err) }
	client.baseURL = server.URL
	devices, count, err := client.ListDevices(context.Background(), "Ot_test-access-token", 1, 50)
	if err != nil { t.Fatalf("ListDevices: %v", err) }
	if gotPath != "/openapi/v2/device.detail.page" { t.Fatalf("unexpected API path %q", gotPath) }
	if count != 1 || len(devices) != 1 || devices[0].DeviceID != "dev-1" { t.Fatalf("unexpected device response: %+v (count %d)", devices, count) }
}
