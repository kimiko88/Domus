package tuya_cloud

import (
 "context"
 "errors"
 "io"
 "net/http"
 "strings"
 "testing"
)

type oauthRoundTripper func(*http.Request)(*http.Response,error)
func (f oauthRoundTripper) RoundTrip(r *http.Request)(*http.Response,error){return f(r)}

func TestNewOAuthClientRequiresRegionalHTTPS(t *testing.T) {
 for _,base:=range []string{"http://openapi.tuyaeu.com","https://evil.example","https://user:pass@openapi.tuyaeu.com","https://openapi.tuyaeu.com/path"} {
  if _,err:=NewOAuthClient(OAuthConfig{ClientID:"id",Secret:"secret",Identifier:"app.bundle",BaseURL:base});err==nil { t.Fatalf("expected %q to be rejected",base) }
 }
}

func TestOAuthExchangeUsesAuthorizationCodeMode(t *testing.T) {
 var request *http.Request
 httpClient:=&http.Client{Transport:oauthRoundTripper(func(r *http.Request)(*http.Response,error){
  request=r
  return &http.Response{StatusCode:http.StatusOK,Body:io.NopCloser(strings.NewReader(`{"success":true,"result":{"uid":"user-1","access_token":"access","refresh_token":"refresh","expire_time":7200}}`)),Header:make(http.Header)},nil
 })}
 c,err:=NewOAuthClient(OAuthConfig{ClientID:"app-id",Secret:"app-secret",Identifier:"app.bundle",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:httpClient})
 if err!=nil {t.Fatal(err)}
 token,err:=c.ExchangeCode(context.Background(),"auth-code")
 if err!=nil {t.Fatal(err)}
 if token.UID!="user-1" || token.AccessToken!="access" || token.RefreshToken!="refresh" {t.Fatalf("unexpected token result: %#v",token)}
 if request==nil || request.URL.Query().Get("grant_type")!="2" || request.URL.Query().Get("code")!="auth-code" {t.Fatalf("unexpected OAuth request: %#v",request)}
 for _,header:=range []string{"client_id","identifier","t","nonce","sign","sign_method"} { if request.Header.Get(header)=="" {t.Errorf("missing %s header",header)} }
}

func TestOAuthDevicesPreservesProviderErrorDetails(t *testing.T) {
 httpClient:=&http.Client{Transport:oauthRoundTripper(func(r *http.Request)(*http.Response,error){
  if r.URL.Path!="/v1.0/users/user-1/devices" { t.Fatalf("unexpected path: %s",r.URL.Path) }
  if r.Header.Get("access_token")!="access" { t.Fatal("missing access token header") }
  return &http.Response{StatusCode:http.StatusOK,Body:io.NopCloser(strings.NewReader(`{"success":false,"code":1106,"msg":"permission deny","result":null}`)),Header:make(http.Header)},nil
 })}
 c,err:=NewOAuthClient(OAuthConfig{ClientID:"app-id",Secret:"app-secret",Identifier:"app.bundle",BaseURL:"https://openapi.tuyaeu.com",HTTPClient:httpClient})
 if err!=nil { t.Fatal(err) }
 _,err=c.Devices(context.Background(),OAuthToken{UID:"user-1",AccessToken:"access"})
 var providerErr *OAuthAPIError
 if !errors.As(err,&providerErr) { t.Fatalf("expected OAuthAPIError, got %T: %v",err,err) }
 if providerErr.HTTPStatus!=http.StatusOK || providerErr.Code!="1106" || providerErr.Message!="permission deny" {
  t.Fatalf("unexpected provider diagnostics: %#v",providerErr)
 }
}

func TestSafeMessageNormalizesProviderDiagnostics(t *testing.T) {
	got := safeMessage(" permission\n deny\t" + strings.Repeat("x", 200))
	if strings.ContainsAny(got, "\n\r\t") {
		t.Fatalf("provider message contains control characters: %q", got)
	}
	if len([]rune(got)) > 180 {
		t.Fatalf("provider message length = %d, want at most 180 runes", len([]rune(got)))
	}
}
