package renderer

import (
	"bytes"
	"testing"
)

func TestCaptureFromHTMLKeepsImageAfterInputReuse(t *testing.T) {
	body := make([]byte, 0, 1<<20)
	body = append(body, `<html><body><h1>Quarterly report</h1><p>ready</p></body></html>`...)

	capture, err := CaptureFromHTML("https://example.test/report", body)
	if err != nil {
		t.Fatalf("capture html: %v", err)
	}
	if len(capture.PNG) < 8 || !bytes.Equal(capture.PNG[:8], []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("capture did not return a PNG image")
	}
	want := append([]byte(nil), capture.PNG...)

	clear(body[:cap(body)])
	if !bytes.Equal(capture.PNG, want) {
		t.Fatal("captured image changed after the caller reused its HTML buffer")
	}
}
