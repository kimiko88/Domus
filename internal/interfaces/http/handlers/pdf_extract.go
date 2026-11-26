package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	maxReceiptPDFBytes = 3 << 20
	maxReceiptPDFText = 64 << 10
)

var errPDFTextTooLarge = errors.New("PDF text exceeds the 64 KiB extraction limit")

// Serialize OCR workloads to keep CPU and memory use bounded on Raspberry Pi.
var receiptOCRSlots = make(chan struct{}, 1)

func acquireReceiptOCR(ctx context.Context) (func(), error) {
	select {
	case receiptOCRSlots <- struct{}{}:
		return func() { <-receiptOCRSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type cappedBuffer struct { bytes.Buffer; limit int }
func (b *cappedBuffer) Write(p []byte)(int,error) {
	if b.Len()+len(p)>b.limit { remaining:=b.limit-b.Len(); if remaining>0 { _,_=b.Buffer.Write(p[:remaining]) }; return remaining,errPDFTextTooLarge }
	return b.Buffer.Write(p)
}

// extractPDFText invokes Poppler without a shell and uses a private, short-lived workspace for OCR;
// and limits both processing time and the number of pages examined.
func extractPDFText(parent context.Context, pdf []byte) (string, error) {
	if len(pdf) < 5 || len(pdf) > maxReceiptPDFBytes || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return "", errors.New("file is not a valid PDF or exceeds the 3 MiB limit")
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	release, err := acquireReceiptOCR(ctx)
	if err != nil { return "", err }
	defer release()

	for _, binary := range []string{"pdfinfo", "pdftotext"} {
		if _, err := exec.LookPath(binary); err != nil {
			return "", errors.New("PDF extraction is unavailable: install poppler-utils on the Domus host")
		}
	}
	dir, err := os.MkdirTemp("", "domus-pdf-")
	if err != nil { return "", errors.New("unable to allocate temporary PDF workspace") }
	defer os.RemoveAll(dir)
	pdfPath := filepath.Join(dir, "document.pdf")
	if err := os.WriteFile(pdfPath, pdf, 0600); err != nil {
		return "", errors.New("unable to stage PDF for extraction")
	}

	// Inspect the entire document before extracting any page. Do not silently
	// import a partial receipt when the PDF is longer than the supported limit.
	info := exec.CommandContext(ctx, "pdfinfo", pdfPath)
	infoOut := &cappedBuffer{limit: 32 << 10}
	info.Stdout, info.Stderr = infoOut, io.Discard
	if err := info.Run(); err != nil {
		if ctx.Err() != nil { return "", ctx.Err() }
		if errors.Is(err, errPDFTextTooLarge) { return "", errors.New("unable to inspect PDF metadata") }
		return "", errors.New("unable to inspect this PDF; it may be encrypted or damaged")
	}
	pageCount, err := parsePDFPageCount(infoOut.String())
	if err != nil { return "", err }
	if pageCount > 20 {
		return "", fmt.Errorf("PDF contains %d pages; the limit is 20 pages. Split the document and retry", pageCount)
	}

	// Prefer the embedded text layer; it is more accurate than OCR.
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", "-f", "1", "-l", strconv.Itoa(pageCount), pdfPath, "-")
	out := &cappedBuffer{limit: maxReceiptPDFText}
	cmd.Stdout, cmd.Stderr = out, io.Discard
	if err := cmd.Run(); err == nil {
		if text := strings.TrimSpace(out.String()); text != "" { return text, nil }
	} else {
		if ctx.Err() != nil { return "", ctx.Err() }
		if errors.Is(err, errPDFTextTooLarge) { return "", errPDFTextTooLarge }
		return "", errors.New("unable to read this PDF; it may be encrypted or damaged")
	}

	// Scanned PDFs are rasterized one page at a time to bound temporary disk use.
	for _, binary := range []string{"pdftoppm", "tesseract"} {
		if _, err := exec.LookPath(binary); err != nil {
			return "", errors.New("PDF OCR is unavailable: install poppler-utils, tesseract-ocr and ita/eng language data")
		}
	}
	var text strings.Builder
	for page := 1; page <= pageCount; page++ {
		prefix := filepath.Join(dir, "page")
		render := exec.CommandContext(ctx, "pdftoppm", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), "-singlefile", "-scale-to", "1400", "-png", pdfPath, prefix)
		render.Stdout, render.Stderr = io.Discard, io.Discard
		if err := render.Run(); err != nil {
			if ctx.Err() != nil { return "", ctx.Err() }
			return "", errors.New("unable to render scanned PDF page")
		}
		imagePath := prefix + ".png"
		stat, err := os.Stat(imagePath)
		if err != nil || stat.Size() > 8<<20 { return "", errors.New("rendered PDF page exceeds the 8 MiB safety limit") }
		ocr := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", "ita+eng", "--psm", "6")
		pageText := &cappedBuffer{limit: maxReceiptPDFText - text.Len()}
		ocr.Stdout, ocr.Stderr = pageText, io.Discard
		if err := ocr.Run(); err != nil {
			if ctx.Err() != nil { return "", ctx.Err() }
			if errors.Is(err, errPDFTextTooLarge) { return "", errPDFTextTooLarge }
			return "", errors.New("OCR failed while reading a PDF page")
		}
		text.WriteString(pageText.String())
		if text.Len() < maxReceiptPDFText { text.WriteByte('\n') }
		_ = os.Remove(imagePath)
	}
	result := strings.TrimSpace(text.String())
	if result == "" { return "", errors.New("OCR could not recognize text in this PDF") }
	return result, nil
}

func parsePDFPageCount(info string) (int, error) {
	for _, line := range strings.Split(info, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.TrimSuffix(fields[0], ":") != "Pages" { continue }
		count, err := strconv.Atoi(fields[1])
		if err != nil || count < 1 { return 0, errors.New("PDF contains no readable pages") }
		return count, nil
	}
	return 0, errors.New("PDF page count could not be determined")
}
