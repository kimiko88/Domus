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
