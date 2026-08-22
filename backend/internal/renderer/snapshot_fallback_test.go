package renderer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"goscrapy/internal/fetcher"
	"goscrapy/internal/renderer"
)

func TestCaptureFallsBackAfterRendererProbeError(t *testing.T) {
	var pageRequests atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("renderer is restarting"))
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, _ *http.Request) {
		pageRequests.Add(1)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body><article><h1>fallback page</h1></article></body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := renderer.NewService(srv.URL, fetcher.New(nil, time.Second))
	rec, err := svc.Capture(context.Background(), srv.URL+"/page")
	if err != nil {
		t.Fatalf("Capture returned an error after the static fallback fetched the page: %v", err)
	}
	if rec == nil || rec.ID == "" {
		t.Fatal("Capture did not return a stored snapshot")
	}
	if rec.Source != "static" {
		t.Fatalf("snapshot source = %q, want static", rec.Source)
	}
	if len(rec.PNG) == 0 {
		t.Fatal("static fallback returned an empty image")
	}
	if got := pageRequests.Load(); got != 1 {
		t.Fatalf("page requests = %d, want 1", got)
	}
}
