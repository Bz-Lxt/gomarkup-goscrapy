package bloom

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryDedup(t *testing.T) {
	f := NewMemory(1<<16, 7)
	if !f.AddIfFresh("http://mock-target/list.html") {
		t.Fatal("first insert should be fresh")
	}
	if f.AddIfFresh("http://mock-target/list.html") {
		t.Fatal("duplicate must be rejected")
	}
	if !f.Test("http://mock-target/list.html") {
		t.Fatal("expected membership")
	}
	if f.Test("http://mock-target/missing.html") {
		t.Fatal("unseen url must not match")
	}
}

func TestMemoryFalsePositiveRate(t *testing.T) {
	const n = 8000
	m := OptimalM(n, 0.01)
	k := OptimalK(m, n)
	f := NewMemory(m, k)
	for i := 0; i < n; i++ {
		f.Add(fmt.Sprintf("https://example.test/item/%d", i))
	}
	fp := 0
	trials := 8000
	for i := 0; i < trials; i++ {
		u := fmt.Sprintf("https://example.test/other/%d", i)
		if f.Test(u) {
			fp++
		}
	}
	rate := float64(fp) / float64(trials)
	if rate > 0.03 {
		t.Fatalf("false positive rate %.4f exceeds 3%% (fp=%d)", rate, fp)
	}
}

func TestRedisBloomRoundTrip(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	f := New(rdb, 1<<16, 7)
	ctx := context.Background()
	fresh, err := f.AddIfFresh(ctx, 9, "http://a/1")
	if err != nil || !fresh {
		t.Fatalf("add fresh: fresh=%v err=%v", fresh, err)
	}
	seen, err := f.Test(ctx, 9, "http://a/1")
	if err != nil || !seen {
		t.Fatalf("test: seen=%v err=%v", seen, err)
	}
	fresh, err = f.AddIfFresh(ctx, 9, "http://a/1")
	if err != nil || fresh {
		t.Fatalf("dup should not be fresh: %v %v", fresh, err)
	}
	seen, err = f.Test(ctx, 9, "http://a/2")
	if err != nil || seen {
		t.Fatalf("unseen should be false: %v %v", seen, err)
	}
}

func TestPositionsBounded(t *testing.T) {
	offs := positions([]byte("hello"), DefaultM, DefaultK)
	if len(offs) != DefaultK {
		t.Fatalf("k=%d", len(offs))
	}
	seen := map[uint64]struct{}{}
	for _, o := range offs {
		if o >= DefaultM {
			t.Fatalf("offset %d >= m", o)
		}
		seen[o] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatal("expected distinct hash slots")
	}
}

func TestOptimalParams(t *testing.T) {
	m := OptimalM(10_000_000, 0.01)
	if m < 90_000_000 || m > 100_000_000 {
		t.Fatalf("unexpected m=%d", m)
	}
	k := OptimalK(DefaultM, 10_000_000)
	if k != 7 {
		t.Fatalf("expected k=7 got %d", k)
	}
}
