package tuya_cloud

import (
	"net/http"
	"testing"
	"context"
	"io"
	"bytes"
	"strings"
)

func TestNewRequiresRegionalHTTPSHost(t *testing.T) {
	for _, base := range []string{"http://openapi.tuyaeu.com", "https://evil.example", "https://user:pass@openapi.tuyaeu.com", "https://openapi.tuyaeu.com:444"} {
		if _, err := New(Config{ClientID:"id",Secret:"secret",BaseURL:base}); err == nil {
			t.Fatalf("expected %q to be rejected",base)
		}
	}
	if _,err:=New(Config{ClientID:"id",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:&http.Client{}}); err!=nil { t.Fatalf("valid regional endpoint rejected: %v",err) }
}
func TestSendCommandsValidatesBeforeNetwork(t *testing.T) {
	c,err:=New(Config{ClientID:"id",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com"})
	if err!=nil { t.Fatal(err) }
	if err:=c.SendCommands(t.Context(),"../other",[]Command{{Code:"switch",Value:true}}); err==nil { t.Fatal("expected invalid device ID") }
	if err:=c.SendCommands(t.Context(),"device",nil); err==nil { t.Fatal("expected empty command list rejection") }
	if err:=c.SendCommands(t.Context(),"device",[]Command{{Code:"../admin",Value:true}}); err==nil { t.Fatal("expected invalid command code") }
}
func TestSignatureIsUppercaseHex(t *testing.T) {
	c,_:=New(Config{ClientID:"client",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com"})
	got:=c.signature(http.MethodGet,"/v1.0/token?grant_type=1",nil,"1700000000000","",true)
	if len(got)!=64 { t.Fatalf("signature length = %d, want 64",len(got)) }
	for _,ch:=range got { if (ch<'0'||ch>'9')&&(ch<'A'||ch>'F') { t.Fatalf("signature is not uppercase hex: %q",got) } }
}

type roundTripFunc func(*http.Request) (*http.Response,error)
func (f roundTripFunc) RoundTrip(r *http.Request)(*http.Response,error) { return f(r) }

func TestDevicesObtainsAndCachesProjectToken(t *testing.T) {
	calls:=0
	transport:=roundTripFunc(func(r *http.Request)(*http.Response,error) {
		calls++
		if r.Header.Get("client_id")!="client-id" || r.Header.Get("sign_method")!="HMAC-SHA256" || len(r.Header.Get("sign"))!=64 {
			t.Fatalf("missing or malformed Tuya signing headers")
		}
		body:=`{"success":true,"result":{"access_token":"access-token","expire_time":7200}}`
		if strings.HasPrefix(r.URL.Path,"/v1.0/token") {
			if r.Header.Get("access_token")!="" { t.Fatal("token endpoint must not carry an access token") }
		} else {
			if r.Header.Get("access_token")!="access-token" { t.Fatalf("business request has wrong token: %q",r.Header.Get("access_token")) }
			body=`{"success":true,"result":{"devices":[{"id":"device-1","name":"Lamp","online":true}]}}`
		}
		return &http.Response{StatusCode:200,Header:make(http.Header),Body:io.NopCloser(strings.NewReader(body)),Request:r},nil
	})
	client,err:=New(Config{ClientID:"client-id",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:&http.Client{Transport:transport}})
	if err!=nil { t.Fatal(err) }
	for i:=0;i<2;i++ {
		devices,err:=client.Devices(context.Background())
		if err!=nil { t.Fatal(err) }
		if len(devices)!=1 || devices[0].ID!="device-1" { t.Fatalf("unexpected devices: %#v",devices) }
	}
	if calls!=3 { t.Fatalf("request count = %d, want one token request and two list requests",calls) }
}

func TestSendCommandsAllowsOnlyDeclaredFunctions(t *testing.T) {
	commandsSent:=0
	transport:=roundTripFunc(func(r *http.Request)(*http.Response,error) {
		payload:=`{"success":true,"result":{"access_token":"token","expire_time":3600}}`
		switch r.URL.Path {
		case "/v1.0/token":
		case "/v1.0/iot-03/devices/device-1/functions":
			payload=`{"success":true,"result":{"functions":[{"code":"switch_led","type":"Boolean"}]}}`
		case "/v1.0/iot-03/devices/device-1/commands":
			commandsSent++
			body,_:=io.ReadAll(r.Body)
			if !bytes.Contains(body,[]byte(`"code":"switch_led"`)) { t.Fatalf("unexpected command body: %s",body) }
			payload=`{"success":true,"result":true}`
		default:
			t.Fatalf("unexpected Tuya API path %s",r.URL.Path)
		}
		return &http.Response{StatusCode:200,Header:make(http.Header),Body:io.NopCloser(strings.NewReader(payload)),Request:r},nil
	})
	client,err:=New(Config{ClientID:"client-id",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:&http.Client{Transport:transport}})
	if err!=nil { t.Fatal(err) }
	if err:=client.SendCommands(context.Background(),"device-1",[]Command{{Code:"switch_led",Value:true}}); err!=nil { t.Fatal(err) }
	if err:=client.SendCommands(context.Background(),"device-1",[]Command{{Code:"battery",Value:true}}); err==nil { t.Fatal("read-only/unknown datapoint must be rejected") }
	if commandsSent!=1 { t.Fatalf("sent %d commands, want exactly one",commandsSent) }
}

func TestTokenRequestDoesNotFollowRedirects(t *testing.T) {
	calls:=0
	transport:=roundTripFunc(func(r *http.Request)(*http.Response,error) {
		calls++
		return &http.Response{StatusCode:http.StatusFound,Header:http.Header{"Location":[]string{"https://attacker.example/collect"}},Body:io.NopCloser(strings.NewReader("")),Request:r},nil
	})
	client,err:=New(Config{ClientID:"client-id",Secret:"secret",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:&http.Client{Transport:transport}})
	if err!=nil { t.Fatal(err) }
	if _,err:=client.accessToken(context.Background()); err==nil { t.Fatal("expected redirect response to fail token acquisition") }
	if calls!=1 { t.Fatalf("redirect target was contacted: %d requests",calls) }
}

func TestGetDeviceRetrievesLocalKey(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"token","expire_time":3600}}`
		if strings.HasPrefix(r.URL.Path, "/v1.0/devices/dev-123") {
			body = `{"success":true,"result":{"id":"dev-123","name":"Smart Plug","local_key":"test_local_key_16","ip":"192.168.1.120","category":"cz","online":true}}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client, err := New(Config{ClientID: "client-id", Secret: "secret", BaseURL: "https://openapi.tuyaeu.com", HTTPClient: &http.Client{Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	dev, err := client.GetDevice(context.Background(), "dev-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dev.LocalKey != "test_local_key_16" || dev.IP != "192.168.1.120" || dev.Category != "cz" {
		t.Fatalf("unexpected device details: %+v", dev)
	}
}

func TestDevicesByUID_RetrievesSmartLifeDevicesAndLocalKey(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{"access_token":"token","expire_time":3600}}`
		if r.URL.Path == "/v1.0/users/eu_user_12345/devices" {
			body = `{"success":true,"result":[{"id":"dev-smartlife-1","name":"Smart Life Bulb","local_key":"localkey12345678","ip":"192.168.1.42","category":"dj","online":true}]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client, err := New(Config{
		ClientID:   "client-id",
		Secret:     "secret",
		BaseURL:    "https://openapi.tuyaeu.com",
		UID:        "eu_user_12345",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}

	devices, err := client.Devices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].ID != "dev-smartlife-1" || devices[0].LocalKey != "localkey12345678" || devices[0].IP != "192.168.1.42" {
		t.Fatalf("unexpected device data: %+v", devices[0])
	}
}
