package tuya_cloud

import (
	"net/http"
	"testing"
	"context"
	"io"
	"strings"
)

func TestNewRequiresRegionalHTTPSHost(t *testing.T) {
	for _, base := range []string{"http://openapi.tuyaeu.com", "https://evil.example", "https://user:pass@openapi.tuyaeu.com"} {
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
