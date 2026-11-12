package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"os/exec"
	"strings"
	"time"
)

const (
	maxReceiptPDFBytes = 3 << 20
	maxReceiptPDFText = 64 << 10
)

var errPDFTextTooLarge = errors.New("PDF text exceeds the 64 KiB extraction limit")

type cappedBuffer struct { bytes.Buffer; limit int }
func (b *cappedBuffer) Write(p []byte)(int,error) {
	if b.Len()+len(p)>b.limit { remaining:=b.limit-b.Len(); if remaining>0 { _,_=b.Buffer.Write(p[:remaining]) }; return 0,errPDFTextTooLarge }
	return b.Buffer.Write(p)
}

// extractPDFText invokes Poppler without a shell, writes no uploaded data to disk,
// and limits both processing time and the number of pages examined.
func extractPDFText(ctx context.Context, pdf []byte) (string, error) {
	if len(pdf) < 5 || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return "", errors.New("file is not a valid PDF")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// First use the embedded text layer; this is much faster and more accurate
	// than OCR for digitally generated invoices.
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", "-f", "1", "-l", "20", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out := &cappedBuffer{limit: maxReceiptPDFText}
	cmd.Stdout = out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err == nil {
		text := strings.TrimSpace(out.String())
		if text != "" { return text, nil }
	} else {
		if ctx.Err() != nil { return "", ctx.Err() }
		if _, lookErr := exec.LookPath("pdftotext"); lookErr != nil {
			return "", errors.New("PDF extraction is unavailable: install poppler-utils on the Domus host")
		}
		if errors.Is(err, errPDFTextTooLarge) { return "", errPDFTextTooLarge }
		return "", errors.New("unable to read this PDF; it may be encrypted or damaged")
	}

	// Scanned PDFs have no text layer: rasterize at a bounded resolution and
	// OCR each of at most 20 pages with Italian and English language models.
	if _, err := exec.LookPath("pdftoppm"); err != nil { return "", errors.New("PDF OCR is unavailable: pdftoppm was not found") }
	if _, err := exec.LookPath("tesseract"); err != nil { return "", errors.New("PDF OCR is unavailable: install tesseract-ocr and ita/eng language data") }
	dir, err := os.MkdirTemp("", "domus-pdf-")
	if err != nil { return "", errors.New("unable to allocate temporary PDF workspace") }
	defer os.RemoveAll(dir)
	pdfPath := filepath.Join(dir, "document.pdf")
	if err := os.WriteFile(pdfPath, pdf, 0600); err != nil { return "", errors.New("unable to stage PDF for OCR") }
	prefix := filepath.Join(dir, "page")
	render := exec.CommandContext(ctx, "pdftoppm", "-f", "1", "-l", "20", "-scale-to", "1800", "-png", pdfPath, prefix)
	render.Stdout = io.Discard
	render.Stderr = io.Discard
	if err := render.Run(); err != nil {
		if ctx.Err() != nil { return "", ctx.Err() }
		return "", errors.New("unable to render scanned PDF pages")
	}
	pages, err := filepath.Glob(prefix + "-*.png")
	if err != nil || len(pages) == 0 { return "", errors.New("PDF contains no readable pages") }
	var text strings.Builder
	for _, page := range pages {
		ocr := exec.CommandContext(ctx, "tesseract", page, "stdout", "-l", "ita+eng", "--psm", "6")
		pageText := &cappedBuffer{limit: maxReceiptPDFText - text.Len()}
		ocr.Stdout = pageText
		ocr.Stderr = io.Discard
		if err := ocr.Run(); err != nil {
			if ctx.Err() != nil { return "", ctx.Err() }
			if errors.Is(err, errPDFTextTooLarge) { return "", errPDFTextTooLarge }
			return "", errors.New("OCR failed while reading a PDF page")
		}
		text.WriteString(pageText.String())
		text.WriteByte('\n')
	}
	result := strings.TrimSpace(text.String())
	if result == "" { return "", errors.New("OCR could not recognize text in this PDF") }
	return result, nil
}
