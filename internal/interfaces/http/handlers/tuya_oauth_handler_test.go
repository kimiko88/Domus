package handlers

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/plugins/tuya_cloud"
	_ "modernc.org/sqlite"
)

type tuyaOAuthTestTransport func(*http.Request) (*http.Response,error)
func (f tuyaOAuthTestTransport) RoundTrip(r *http.Request) (*http.Response,error) { return f(r) }

func TestWriteOAuthSuccessPage(t *testing.T) {
	rec := httptest.NewRecorder()
	writeOAuthSuccessPage(rec)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want %d", rec.Code, 200)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") {
		t.Fatalf("Content-Security-Policy is missing restrictive default: %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Autorizzazione Smart Life completata",
		"account Smart Life è stato collegato",
		"connessione Tailscale",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("success page does not contain %q", want)
		}
	}
	if strings.Contains(body, "window.location") || strings.Contains(body, "http-equiv=\"refresh\"") {
		t.Fatal("success page must not redirect the browser")
	}
}


func TestIsTuyaTokenRejection(t *testing.T) {
	for _,code:=range []string{"1010","1011","1012"} {
		if !isTuyaTokenRejection(&tuya_cloud.OAuthAPIError{HTTPStatus:http.StatusOK,Code:code,Message:"token invalid"}) {
			t.Errorf("code %s should trigger a token refresh",code)
		}
	}
	for _,msg:=range []string{"permission deny","token scope not allow","scope not allowed"} {
		if isTuyaTokenRejection(&tuya_cloud.OAuthAPIError{HTTPStatus:http.StatusOK,Code:"1010",Message:msg}) {
			t.Errorf("authorization error %q must not trigger a token refresh",msg)
		}
	}
	if isTuyaTokenRejection(&tuya_cloud.OAuthAPIError{HTTPStatus:http.StatusOK,Code:"1106",Message:"permission deny"}) {
		t.Fatal("permission errors must not trigger a token refresh")
	}
	if isTuyaTokenRejection(io.EOF) {
		t.Fatal("transport errors must not trigger a token refresh")
	}
}

func TestRefreshRejectedTokenRotatesRefreshTokenOnlyOnce(t *testing.T) {
	db,err:=sql.Open("sqlite",":memory:")
	if err!=nil { t.Fatal(err) }
	defer db.Close()
	cipher,err:=crypto.NewEnvelopeCipher([]byte("test master key"))
	if err!=nil { t.Fatal(err) }

	var refreshCalls atomic.Int32
	httpClient:=&http.Client{Transport:tuyaOAuthTestTransport(func(r *http.Request)(*http.Response,error) {
		if r.URL.Path!="/v1.0/token/refresh-old" { t.Fatalf("unexpected refresh path: %s",r.URL.Path) }
		refreshCalls.Add(1)
		body:=`{"success":true,"result":{"uid":"user-1","access_token":"access-new","refresh_token":"refresh-new","expire_time":7200}}`
		return &http.Response{StatusCode:http.StatusOK,Body:io.NopCloser(strings.NewReader(body)),Header:make(http.Header)},nil
	})}
	client,err:=tuya_cloud.NewOAuthClient(tuya_cloud.OAuthConfig{
		ClientID:"app-id",Secret:"app-secret",Identifier:"app.bundle",
		BaseURL:"https://openapi.tuyaeu.com",HTTPClient:httpClient,
	})
	if err!=nil { t.Fatal(err) }
	h,err:=NewTuyaOAuthHandler(db,cipher,client,"app-id","https://example.com/authorize","https://example.com/callback")
	if err!=nil { t.Fatal(err) }
	old:=tuya_cloud.OAuthToken{UID:"user-1",AccessToken:"access-old",RefreshToken:"refresh-old",ExpireTime:7200}
	if err=h.storeToken(context.Background(),"domus-user",old);err!=nil { t.Fatal(err) }

	first,err:=h.refreshRejectedToken(context.Background(),"domus-user",old)
	if err!=nil { t.Fatal(err) }
	if first.AccessToken!="access-new" || first.RefreshToken!="refresh-new" { t.Fatalf("unexpected refreshed token: %#v",first) }

	// A concurrent request that observed the same rejected access token must
	// reuse the rotated credentials, not consume refresh-old again.
	second,err:=h.refreshRejectedToken(context.Background(),"domus-user",old)
	if err!=nil { t.Fatal(err) }
	if second.AccessToken!="access-new" || second.RefreshToken!="refresh-new" { t.Fatalf("unexpected reused token: %#v",second) }
	if got:=refreshCalls.Load();got!=1 { t.Fatalf("refresh calls = %d, want exactly 1",got) }

	var accessEnc,refreshEnc string
	if err=db.QueryRow(`SELECT access_token_enc,refresh_token_enc FROM tuya_user_oauth_connections WHERE user_id=?`,"domus-user").Scan(&accessEnc,&refreshEnc);err!=nil { t.Fatal(err) }
	storedAccess,err:=cipher.Decrypt(accessEnc);if err!=nil { t.Fatal(err) }
	storedRefresh,err:=cipher.Decrypt(refreshEnc);if err!=nil { t.Fatal(err) }
	if storedAccess!="access-new" || storedRefresh!="refresh-new" { t.Fatal("rotated credentials were not persisted") }
}
