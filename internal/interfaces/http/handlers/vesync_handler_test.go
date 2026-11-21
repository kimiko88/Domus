package handlers

import (
	"errors"
	"strings"
	"testing"
)

func TestSafeVeSyncErrorRedactsPossibleCredentials(t *testing.T) {
	for _, message := range []string{
		"token invalid for account", "incorrect password for account", "ACCOUNTID invalid",
	} {
		got := safeVeSyncError(errors.New(message))
		if strings.Contains(strings.ToLower(got), "token") || strings.Contains(strings.ToLower(got), "password") || strings.Contains(strings.ToLower(got), "accountid") {
			t.Errorf("credential-bearing error was not redacted: %q", got)
		}
	}
}

func TestNewVeSyncHandlerRequiresEncryptedStorage(t *testing.T) {
	if _, err := NewVeSyncHandler(nil, nil); err == nil {
		t.Fatal("VeSync handler must require an integration repository and encryption")
	}
}
