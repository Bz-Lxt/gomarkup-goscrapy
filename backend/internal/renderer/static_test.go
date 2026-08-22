package renderer

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

// TestCaptureFromHTML_OWNSPngData is a regression test for the buffer-pool
// aliasing bug. The fetched HTML byte slice is handed to CaptureFromHTML
// and, after the snapshot is produced, the caller resets it and reuses the
// backing array (mimicking a buffer pool). The returned PNG must be
// unaffected.
//
// The input is padded so len(body) >= len(PNG); only then does the buggy
// "append(body[:0], pngBytes...)" reuse body's backing array instead of
// allocating a fresh one. A pooled download buffer is normally much larger
// than the ~7 KiB PNG, which is why the bug only manifests in production
// (buffer reuse) and not with tiny inputs.
func TestCaptureFromHTML_OWNSPngData(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><title>x</title></head>
<body><header class="nav"><a href="/">AURORA</a></header>
<main><h1>全部商品</h1>
<section class="grid">
<article class="product-card" id="p-1">
  <h2 class="title">Aurora Headphones</h2>
  <div class="price">¥1299</div>
  <div class="sku">SKU-1001</div>
</article>
<article class="product-card" id="p-2">
  <h2 class="title">Nautilus Watch</h2>
  <div class="price">¥2480</div>
  <div class="sku">SKU-1002</div>
</article>
</section></main></body></html>`)
	// Pad to 32 KiB so body capacity comfortably exceeds the PNG size; this
	// is the condition under which aliasing actually bites.
	html = append(html, []byte("<!-- "+strings.Repeat("x", 32*1024)+" -->")...)

	cap, err := CaptureFromHTML("http://example.com/list.html", html)
	if err != nil {
		t.Fatalf("CaptureFromHTML: %v", err)
	}
	pngSnapshot := append([]byte(nil), cap.PNG...) // independent copy of the bytes as returned

	if len(cap.PNG) == 0 {
		t.Fatal("expected non-empty PNG")
	}
	// Sanity: with the padding above the buggy append(body[:0], ...) would
	// alias body's array, so the guard below would catch it.
	if len(html) < len(cap.PNG) {
		t.Fatalf("test setup error: body %d smaller than PNG %d", len(html), len(cap.PNG))
	}

	// Simulate the buffer pool: caller resets the input slice to zero length
	// (keeping capacity) and then reuses the backing array for new content.
	for i := range html {
		html[i] = 0
	}
	html = append(html[:0], []byte("REUSED BUFFER FOR NEXT DOWNLOAD")...)

	// The PNG produced earlier must still decode and be byte-identical to
	// what was captured before the input buffer was mutated.
	if !bytes.Equal(cap.PNG, pngSnapshot) {
		t.Fatalf("PNG data was corrupted after input buffer reuse: got len=%d want len=%d",
			len(cap.PNG), len(pngSnapshot))
	}
	if _, err := png.Decode(bytes.NewReader(cap.PNG)); err != nil {
		t.Fatalf("stored PNG no longer decodes after buffer reuse: %v", err)
	}
}

// TestRenderPNG_OwnsSlice ensures RenderPNG returns bytes independent of any
// caller-provided buffer; a guard against reintroducing an aliasing trick.
func TestRenderPNG_OwnsSlice(t *testing.T) {
	png1, err := RenderPNG(nil)
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if len(png1) == 0 {
		t.Fatal("expected non-empty PNG output")
	}
	snapshot := append([]byte(nil), png1...)
	// Re-render; the previous result must not be retroactively mutated.
	_, _ = RenderPNG(nil)
	if !bytes.Equal(png1, snapshot) {
		t.Fatal("first RenderPNG result changed after a second call")
	}
	if _, err := png.Decode(bytes.NewReader(png1)); err != nil {
		t.Fatalf("PNG does not decode: %v", err)
	}
}
