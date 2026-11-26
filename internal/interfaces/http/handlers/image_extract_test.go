package handlers

import (
	"bytes"
	"context"
	"image"
	"image/gif"
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
	if err == nil || !strings.Contains(err.Error(), "4 MB") {
		t.Fatalf("expected image size error, got %v", err)
	}
}


func TestExtractReceiptImageTextRejectsFormatsNotNormalizedByTheScanner(t *testing.T) {
	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := gif.Encode(&encoded, img, nil); err != nil {
		t.Fatalf("encode GIF: %v", err)
	}
	_, err := extractReceiptImageText(context.Background(), encoded.Bytes())
	if err == nil || !strings.Contains(err.Error(), "use JPEG or PNG") {
		t.Fatalf("expected non-JPEG/PNG image to be rejected, got %v", err)
	}
}
