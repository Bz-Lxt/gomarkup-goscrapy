package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchPreservesRedirectLimitError(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+r.URL.Path+"/next", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	client := New(nil, 2*time.Second)
	_, err := client.Fetch(context.Background(), server.URL+"/start", false)
	if err == nil {
		t.Fatal("Fetch returned nil error for an unbounded redirect chain")
	}
	if !strings.Contains(err.Error(), "too many redirects") {
		t.Fatalf("Fetch error = %q, want redirect limit error", err)
	}
}
