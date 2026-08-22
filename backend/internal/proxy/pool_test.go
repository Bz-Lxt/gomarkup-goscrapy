package proxy

import (
	"testing"
)

func TestRoundRobin(t *testing.T) {
	p := New("mock", nil)
	if p.Size() != 3 {
		t.Fatalf("size=%d", p.Size())
	}
	seen := map[string]int{}
	for i := 0; i < 9; i++ {
		u := p.Next()
		if u == "" {
			t.Fatal("empty proxy")
		}
		seen[u]++
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(seen))
	}
	for u, n := range seen {
		if n != 3 {
			t.Fatalf("%s hit %d", u, n)
		}
	}
}

func TestFailEviction(t *testing.T) {
	p := New("mock", nil)
	target := p.Next()
	for i := 0; i < 3; i++ {
		p.Report(target, false, "timeout")
	}
	snap := p.Snapshot()
	evicted := false
	for _, v := range snap {
		if v.URL == target && v.Evicted {
			evicted = true
		}
	}
	if !evicted {
		t.Fatal("expected eviction after 3 failures")
	}
	for i := 0; i < 20; i++ {
		if p.Next() == target {
			t.Fatal("evicted proxy still returned")
		}
	}
}

func TestReviveWhenAllDead(t *testing.T) {
	p := New("mock", nil)
	urls := []string{
		"http://mock-proxy-1:8000",
		"http://mock-proxy-2:8000",
		"http://mock-proxy-3:8000",
	}
	for _, u := range urls {
		for i := 0; i < 3; i++ {
			p.Report(u, false, "fail")
		}
	}
	got := p.Next()
	if got == "" {
		t.Fatal("pool should revive when all evicted")
	}
}

func TestRealMode(t *testing.T) {
	p := New("real", []string{"http://a:1", "http://b:2"})
	if p.Size() != 2 {
		t.Fatalf("real size=%d", p.Size())
	}
	if p.Mode() != "real" {
		t.Fatalf("mode=%s", p.Mode())
	}
}
