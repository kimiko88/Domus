package auth_test

import (
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
)

func TestTOTPGenerationAndVerification(t *testing.T) {
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}
	if len(secret) < 16 {
		t.Fatalf("secret too short: %s", secret)
	}

	now := time.Now().UTC()
	passcode := auth.GenerateTOTPPasscode(secret, now)
	if len(passcode) != 6 {
		t.Fatalf("expected 6-digit passcode, got %s", passcode)
	}

	// Verify with exact timestamp
	if !auth.VerifyTOTP(secret, passcode, now) {
		t.Errorf("expected passcode %s to verify successfully at %v", passcode, now)
	}

	// Verify within 1-step window (30s past or future)
	past := now.Add(-25 * time.Second)
	if !auth.VerifyTOTP(secret, passcode, past) {
		t.Errorf("expected passcode to verify within 30s window")
	}

	// Invalid passcode should fail
	if auth.VerifyTOTP(secret, "999999", now) && passcode != "999999" {
		t.Errorf("invalid passcode unexpectedly succeeded")
	}
}

func TestBackupCodesAndQRCode(t *testing.T) {
	codes := auth.GenerateBackupCodes(8)
	if len(codes) != 8 {
		t.Fatalf("expected 8 backup codes, got %d", len(codes))
	}
	for _, c := range codes {
		if len(c) != 9 || c[4] != '-' {
			t.Errorf("unexpected backup code format: %s", c)
		}
	}

	qrURL := auth.BuildQRCodeURL("admin", "JBSWY3DPEHPK3PXP", "Domus")
	if qrURL == "" || qrURL[:8] != "otpauth:" {
		t.Errorf("unexpected otpauth QR url: %s", qrURL)
	}
}
