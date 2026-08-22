package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchAndRobots(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\nDisallow: /secret\n"))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != DefaultUA {
			t.Errorf("ua=%s", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte("<html><body>hi</body></html>"))
	})
	mux.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("nope"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(nil, 3*time.Second)
	ctx := context.Background()
	res, err := c.Fetch(ctx, srv.URL+"/ok", true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 200 || !strings.Contains(string(res.Body), "hi") {
		t.Fatalf("fetch %+v", res)
	}
	blocked, err := c.Fetch(ctx, srv.URL+"/secret", true)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.RobotsSkip {
		t.Fatal("expected robots skip")
	}
}

func TestBackoff(t *testing.T) {
	if Backoff(0, time.Second) != time.Second {
		t.Fatal("base")
	}
	if Backoff(3, 200*time.Millisecond) < time.Second {
		t.Fatal("exp")
	}
	if !Retryable(429, nil) || Retryable(200, nil) {
		t.Fatal("retryable")
	}
}
