package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

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
