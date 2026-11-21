package vesync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegionForCountry(t *testing.T) {
	for _, tc := range []struct{ country, region string }{
		{"IT", "EU"}, {"DE", "EU"}, {"US", "US"}, {"CA", "US"}, {"JP", "US"},
	} {
		if got := RegionForCountry(tc.country); got != tc.region {
			t.Errorf("RegionForCountry(%q) = %q, want %q", tc.country, got, tc.region)
		}
	}
}

func TestPasswordDigest(t *testing.T) {
	if got := passwordDigest("password"); got != "5f4dcc3b5aa765d61d8327deb882cf99" {
		t.Fatalf("unexpected MD5 digest %q", got)
	}
}

func TestAuthenticateUsesTwoStepFlowAndDoesNotSendPlainPassword(t *testing.T) {
	var paths []string
	var firstBody map[string]any
	var secondBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/globalPlatform/api/accountAuth/v1/authByPWDOrOTM":
			firstBody = body
			_, _ = w.Write([]byte(`{"code":0,"result":{"accountID":"acct-1","authorizeCode":"auth-code"}}`))
		case "/user/api/accountManage/v1/loginByAuthorizeCode4Vesync":
			secondBody = body
			_, _ = w.Write([]byte(`{"code":0,"result":{"accountID":"acct-1","countryCode":"IT","currentRegion":"EU","token":"session-token"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newClientForTest(server.Client(), server.URL)
	session, err := client.Authenticate(context.Background(), "user@example.test", "plain-secret", "IT", "Europe/Rome")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/globalPlatform/api/accountAuth/v1/authByPWDOrOTM" || paths[1] != "/user/api/accountManage/v1/loginByAuthorizeCode4Vesync" {
		t.Fatalf("unexpected login paths: %v", paths)
	}
	if firstBody["password"] != passwordDigest("plain-secret") {
		t.Fatal("authentication request must contain the provider's MD5 password digest")
	}
	if strings.Contains(strings.TrimSpace(firstBody["password"].(string)), "plain-secret") {
		t.Fatal("plain password must not be sent")
	}
	if secondBody["authorizeCode"] != "auth-code" {
		t.Fatal("second login request did not use the authorization code")
	}
	if session.Token != "session-token" || session.AccountID != "acct-1" || session.Region != "EU" {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestListDevicesSanitizesAndAddsModelCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloud/v1/deviceManaged/devices" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") != apiUserAgent {
			t.Fatalf("unexpected bypass user-agent %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(`{"code":0,"result":{"list":[{"cid":"cid-1","deviceName":"Purifier","deviceType":"Core300S","configModule":"Core300S","deviceStatus":"on","connectionStatus":"online","currentFirmVersion":"1.0"},{"deviceName":"invalid"}]}}`))
	}))
	defer server.Close()
	client := newClientForTest(server.Client(), server.URL)
	devices, err := client.ListDevices(context.Background(), Session{
		AccountID: "account", Token: "secret-token", CountryCode: "IT", Region: "EU", TimeZone: "Europe/Rome",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected one sanitized device, got %d", len(devices))
	}
	if devices[0].ProductType != "purifier" || len(devices[0].SupportedModes) != 3 || len(devices[0].FanLevels) != 3 {
		t.Fatalf("unexpected purifier capabilities: %+v", devices[0])
	}
	if strings.Contains(string(mustJSON(t, devices[0])), "secret-token") {
		t.Fatal("device view must not include account credentials")
	}
}

func TestPurifierCommandsUseAllowlistedPayloads(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloud/v2/deviceManaged/bypassV2" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":0,"result":{"code":0}}`))
	}))
	defer server.Close()
	client := newClientForTest(server.Client(), server.URL)
	session := Session{AccountID: "acct", Token: "token", CountryCode: "IT", Region: "EU", TimeZone: "Europe/Rome"}
	device := DeviceInfo{ID: "cid-1", ProductType: "purifier", ConfigModule: "Core300S", SupportedModes: []string{"sleep", "manual", "auto"}, FanLevels: []int{1, 2, 3}}
	if _, err := client.SetPurifierPower(context.Background(), session, device, true); err != nil {
		t.Fatal(err)
	}
	payload, ok := captured["payload"].(map[string]any)
	if !ok || payload["method"] != "setSwitch" {
		t.Fatalf("unexpected bypass payload: %#v", captured)
	}
	data, _ := payload["data"].(map[string]any)
	if data["enabled"] != true || data["id"] != float64(0) {
		t.Fatalf("unexpected power payload: %#v", data)
	}
	if _, err := client.SetPurifierMode(context.Background(), session, device, "manual"); err != nil {
		t.Fatalf("manual mode should map to the model's lowest supported fan level: %v", err)
	}
	payload, ok = captured["payload"].(map[string]any)
	if !ok || payload["method"] != "setLevel" {
		t.Fatalf("manual mode must map to setLevel: %#v", captured)
	}
	if _, err := client.SetPurifierMode(context.Background(), session, device, "turbo"); err == nil {
		t.Fatal("unsupported mode must be rejected before the cloud request")
	}
	if _, err := client.SetPurifierFanLevel(context.Background(), session, device, 4); err == nil {
		t.Fatal("unsupported fan level must be rejected before the cloud request")
	}
}

func TestBypassV2RequestIncludesRequiredContext(t *testing.T) {
	session := Session{AccountID: "acct", Token: "token", CountryCode: "IT", TimeZone: "Europe/Rome"}
	device := DeviceInfo{ID: "cid", ConfigModule: "Core300S"}
	got := BypassV2Request(session, device, "setSwitch", map[string]any{"enabled": true})
	for _, key := range []string{"acceptLanguage", "appVersion", "accountID", "cid", "configModule", "timeZone", "token", "userCountryCode", "configModel", "deviceId", "payload"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing BypassV2 key %s", key)
		}
	}
}

func TestClientRejectsMissingSession(t *testing.T) {
	client := NewClient(time.Second)
	if _, err := client.ListDevices(context.Background(), Session{}); err == nil {
		t.Fatal("empty session must be rejected")
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
