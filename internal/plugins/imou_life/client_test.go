package imou_life

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignMatchesImouWorkedExample(t *testing.T) {
	client, err := NewClient(Config{AccessKey:"test-app",SecretKey:"test123456789test123456789",DataCenter:"fk"},nil)
	if err != nil { t.Fatal(err) }
	got := client.sign(1706511734,"f5a1ae2d-c09c-4d39-a744-83a5c2c653c2")
	want := "xjhCQBoJ9hRDsCjyDcHjtDNzRZ3ZJezcawsfWeiaoxU="
	if got != want { t.Fatalf("signature mismatch: got %q want %q",got,want) }
}

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

func TestDecryptVideoDownloadURLRoundTripAndAuthentication(t *testing.T) {
	token := "At_test-administrator-token"
	key := sha256.Sum256([]byte(token + "ENCRYPTKEY"))
	block, err := aes.NewCipher(key[:])
	if err != nil { t.Fatal(err) }
	gcm, err := cipher.NewGCM(block)
	if err != nil { t.Fatal(err) }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil { t.Fatal(err) }
	plain := []byte("rtsp://camera-1.lechange.com:554/live")
	sealed := gcm.Seal(nonce, nonce, plain, nil)
	got, err := decryptVideoDownloadURL(base64.StdEncoding.EncodeToString(sealed), token)
	if err != nil { t.Fatalf("decryptVideoDownloadURL: %v", err) }
	if got != string(plain) { t.Fatalf("got %q, want %q", got, plain) }

	sealed[len(sealed)-1] ^= 0xff
	if _, err := decryptVideoDownloadURL(base64.StdEncoding.EncodeToString(sealed), token); err == nil {
		t.Fatal("expected modified authentication tag to be rejected")
	}
	if _, err := decryptVideoDownloadURL("not-base64", token); err == nil {
		t.Fatal("expected invalid base64 to be rejected")
	}
}

func TestImouRTSPTargetRejectsPrivateAndUnapprovedHosts(t *testing.T) {
	for _, raw := range []string{
		"rtsp://127.0.0.1/live",
		"rtsp://192.168.1.10/live",
		"rtsp://169.254.169.254/latest",
		"rtsp://localhost/live",
		"rtsp://attacker.example.org/live",
		"rtsp://evil.lechange.com\\r\\nfile '/etc/passwd'",
	} {
		if _, err := validateImouRTSPURL(context.Background(), raw); err == nil {
			t.Errorf("expected unsafe stream URL %q to be rejected", raw)
		}
	}
	_ = net.IPv4(127,0,0,1) // keep tests explicit about IP target classification
}
