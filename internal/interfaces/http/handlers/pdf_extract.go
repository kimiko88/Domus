package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

const (
	maxReceiptPDFBytes = 10 << 20
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
func extractPDFText(ctx context.Context,pdf []byte)(string,error) {
	if len(pdf)<5 || !bytes.HasPrefix(pdf,[]byte("%PDF-")) { return "",errors.New("file is not a valid PDF") }
	ctx,cancel:=context.WithTimeout(ctx,20*time.Second); defer cancel()
	cmd:=exec.CommandContext(ctx,"pdftotext","-layout","-enc","UTF-8","-f","1","-l","20","-","-")
	cmd.Stdin=bytes.NewReader(pdf)
	out:=&cappedBuffer{limit:maxReceiptPDFText}
	cmd.Stdout=out
	cmd.Stderr=io.Discard
	if err:=cmd.Run(); err!=nil {
		if errors.Is(err,errPDFTextTooLarge) { return "",errPDFTextTooLarge }
		if ctx.Err()!=nil { return "",ctx.Err() }
		if _,lookErr:=exec.LookPath("pdftotext"); lookErr!=nil { return "",errors.New("PDF extraction is unavailable: install poppler-utils on the Domus host") }
		return "",errors.New("unable to extract text from this PDF; it may be encrypted, damaged, or image-only")
	}
	text:=bytes.TrimSpace(out.Bytes())
	if len(text)==0 { return "",errors.New("PDF contains no selectable text; scanned PDFs require OCR before import") }
	return string(text),nil
}
