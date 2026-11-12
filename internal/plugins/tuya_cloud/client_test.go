package tuya_cloud

import (
	"net/http"
	"testing"
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
