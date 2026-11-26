package handlers

import (
	"context"
	"strings"
	"testing"
)

func TestExtractPDFTextRejectsNonPDF(t *testing.T) {
	_,err:=extractPDFText(context.Background(),[]byte("not a PDF"))
	if err==nil || !strings.Contains(err.Error(),"valid PDF") { t.Fatalf("expected invalid PDF error, got %v",err) }
}
func TestCappedBufferStopsLargeOutput(t *testing.T) {
	b:=&cappedBuffer{limit:4}
	if _,err:=b.Write([]byte("1234")); err!=nil { t.Fatal(err) }
	if _,err:=b.Write([]byte("5")); err!=errPDFTextTooLarge { t.Fatalf("expected output limit error, got %v",err) }
	if b.String()!="1234" { t.Fatalf("unexpected retained text %q",b.String()) }
}


func TestParsePDFPageCount(t *testing.T) {
	count, err := parsePDFPageCount("Title: Receipt\nPages:          2\nPage size: 595 x 842 pts")
	if err != nil {
		t.Fatalf("unexpected page-count error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 pages, got %d", count)
	}
}

func TestParsePDFPageCountRejectsMissingOrInvalidMetadata(t *testing.T) {
	for _, raw := range []string{"Title: Receipt", "Pages: 0", "Pages: many"} {
		if _, err := parsePDFPageCount(raw); err == nil {
			t.Errorf("expected page-count metadata %q to be rejected", raw)
		}
	}
}
