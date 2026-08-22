package renderer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goscrapy/internal/fetcher"
)

// TestCaptureFallsBackOnCDPFailure reproduces the scenario where the CDP
// renderer is pointed at a node that does NOT expose DevTools, so
// resolveDebuggerWS fails with "parse debugger version". The static
// fallback fetches the page successfully (HTTP 200); the service must
// return the generated snapshot instead of leaking the stale CDP error.
func TestCaptureFallsBackOnCDPFailure(t *testing.T) {
	// Target page that the fetcher will capture via the static path.
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>
			<h1>Product</h1>
			<article class="product-card">
				<h2 class="title">Aurora Headphones</h2>
				<div class="price">1299</div>
				<div class="sku">SKU-1001</div>
				<a class="product-link" href="/p-1">view</a>
			</article>
		</body></html>`))
	}))
	defer page.Close()

	// "Compatibility node" that does NOT have DevTools enabled: its
	// /json/version endpoint returns a 200 with non-JSON HTML (access log),
	// which triggers "parse debugger version" in resolveDebuggerWS.
	compat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>OK</body></html>"))
	}))
	defer compat.Close()

	fetch := fetcher.New(nil, 5*time.Second)
	svc := NewService(compat.URL, fetch)

	rec, err := svc.Capture(context.Background(), page.URL)
	if err != nil {
		t.Fatalf("expected snapshot from static fallback, got error: %v", err)
	}
	if rec == nil {
		t.Fatal("expected non-nil record")
	}
	if rec.ID == "" || !strings.HasPrefix(rec.ID, "snap_") {
		t.Fatalf("expected generated snapshot_id, got %q", rec.ID)
	}
	if rec.Source != "static" {
		t.Fatalf("expected source=static, got %q", rec.Source)
	}
	if rec.PNG == nil {
		t.Fatal("expected non-nil PNG from static render")
	}
	if rec.URL == "" || !strings.HasPrefix(rec.URL, page.URL) {
		t.Fatalf("expected url prefix %s, got %q", page.URL, rec.URL)
	}
}

// TestCaptureCDPErrorDoesNotLeak verifies that the service stores and returns
// the fallback snapshot when CaptureCDP returns an error but CaptureStatic
// succeeds. This is a regression guard for the stale-error bug where the
// method returned the prior CDP error instead of the fallback record.
func TestCaptureCDPErrorDoesNotLeak(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><h1>Fallback OK</h1></body></html>`))
	}))
	defer page.Close()

	// Renderer endpoint that is not a valid DevTools endpoint at all.
	badRenderer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer badRenderer.Close()

	fetch := fetcher.New(nil, 5*time.Second)
	svc := NewService(badRenderer.URL, fetch)

	rec, err := svc.Capture(context.Background(), page.URL)
	if err != nil {
		t.Fatalf("stale CDP error leaked to caller: %v", err)
	}
	if rec == nil || rec.ID == "" {
		t.Fatal("expected snapshot record with id from fallback")
	}
	if rec.Source != "static" {
		t.Fatalf("expected source=static, got %q", rec.Source)
	}

	// Verify the record is retrievable from the store.
	got, ok := svc.Get(rec.ID)
	if !ok {
		t.Fatal("expected record to be stored")
	}
	if got.ID != rec.ID {
		t.Fatalf("store mismatch: %s != %s", got.ID, rec.ID)
	}
}

// TestCaptureStaticOnly verifies the path when no renderer is configured:
// the static fetch path produces a snapshot directly.
func TestCaptureStaticOnly(t *testing.T) {
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><h1>Static Page</h1></body></html>`))
	}))
	defer page.Close()

	fetch := fetcher.New(nil, 5*time.Second)
	svc := NewService("", fetch) // no renderer configured

	rec, err := svc.Capture(context.Background(), page.URL)
	if err != nil {
		t.Fatalf("static-only capture failed: %v", err)
	}
	if rec == nil || rec.ID == "" {
		t.Fatal("expected snapshot record")
	}
	if rec.Source != "static" {
		t.Fatalf("expected source=static, got %q", rec.Source)
	}
}
