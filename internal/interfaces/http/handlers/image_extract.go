package handlers

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os/exec"
	"time"
)

const (
	maxReceiptImageBytes = 4 << 20
	maxReceiptImagePixels = 24_000_000
	maxReceiptImageText = 64 << 10
)

var errReceiptImageTextTooLarge = errors.New("image OCR text exceeds the 64 KiB limit")

// extractReceiptImageText validates the actual image format and dimensions before
// passing bounded input to Tesseract. It uses stdin (no user-controlled paths),
// never invokes a shell, and caps both runtime and output.
func extractReceiptImageText(parent context.Context, data []byte) (string, error) {
	if len(data) == 0 || len(data) > maxReceiptImageBytes {
		return "", errors.New("image must be 4 MB or smaller")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return "", errors.New("unsupported or invalid image: use JPEG or PNG")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxReceiptImagePixels {
		return "", errors.New("image dimensions exceed the 24 megapixel safety limit")
	}

	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", "ita+eng", "--psm", "6")
	cmd.Stdin = bytes.NewReader(data)
	out := &cappedBuffer{limit: maxReceiptImageText}
	cmd.Stdout, cmd.Stderr = out, io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil { return "", ctx.Err() }
		if errors.Is(err, errPDFTextTooLarge) { return "", errReceiptImageTextTooLarge }
		if _, lookErr := exec.LookPath("tesseract"); lookErr != nil {
			return "", errors.New("image OCR is unavailable: install tesseract-ocr and ita/eng language data")
		}
		return "", errors.New("unable to recognize text in this image")
	}
	text := out.String()
	if text == "" { return "", errors.New("OCR could not recognize text in this image") }
	return text, nil
}
