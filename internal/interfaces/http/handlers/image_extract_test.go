package handlers

import (
	"context"
	"strings"
	"testing"
)

func TestExtractReceiptImageTextRejectsInvalidImage(t *testing.T) {
	_, err := extractReceiptImageText(context.Background(), []byte("not an image"))
	if err == nil || !strings.Contains(err.Error(), "unsupported or invalid image") {
		t.Fatalf("expected invalid image error, got %v", err)
	}
}

func TestExtractReceiptImageTextRejectsOversizedInput(t *testing.T) {
	_, err := extractReceiptImageText(context.Background(), make([]byte, maxReceiptImageBytes+1))
	if err == nil || !strings.Contains(err.Error(), "10 MB") {
		t.Fatalf("expected image size error, got %v", err)
	}
}
