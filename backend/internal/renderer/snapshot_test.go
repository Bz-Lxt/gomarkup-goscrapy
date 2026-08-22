package renderer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goscrapy/internal/timeutil"
)

func TestStorePutGetRoundTrip(t *testing.T) {
	s := NewStore(time.Minute)
	rec := &Record{ID: "snap_1", CreatedAt: timeutil.Now()}
	s.Put(rec)
	got, ok := s.Get("snap_1")
	if !ok {
		t.Fatal("expected record to be found")
	}
	if got.ID != rec.ID {
		t.Fatalf("got %q want %q", got.ID, rec.ID)
	}
}

func TestStoreConcurrentPutGet(t *testing.T) {
	// Regression test for the "concurrent map iteration and map write"
	// panic observed under load when many snapshots are created and read
	// at the same time. Run with -race to detect the data race; without
	// the fix the map write in Put happens after Unlock and races with
	// both gcLocked's iteration and concurrent Get's read.
	s := NewStore(time.Minute)

	const writers = 32
	const readers = 32
	const iterations = 200

	var wg sync.WaitGroup
	var puts atomic.Int64
	var gets atomic.Int64

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := fmt.Sprintf("snap_%d_%d", g, i)
				s.Put(&Record{ID: id, CreatedAt: timeutil.Now()})
				puts.Add(1)
				// also read it back immediately to mimic the
				// "create snapshot then fetch image right away" load pattern.
				if _, ok := s.Get(id); !ok {
					t.Errorf("Get(%q) = false, want true", id)
				}
			}
		}(w)
	}

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := fmt.Sprintf("snap_%d_%d", g%writers, i%iterations)
				if _, ok := s.Get(id); ok {
					gets.Add(1)
				}
			}
		}(r)
	}

	wg.Wait()

	if got := puts.Load(); got != int64(writers*iterations) {
		t.Fatalf("puts = %d, want %d", got, writers*iterations)
	}
}

func TestStoreTTLExpiry(t *testing.T) {
	s := NewStore(20 * time.Millisecond)
	rec := &Record{ID: "snap_ttl", CreatedAt: timeutil.Now()}
	s.Put(rec)
	if _, ok := s.Get("snap_ttl"); !ok {
		t.Fatal("expected record before TTL")
	}
	// Force expiry by backdating the record's CreatedAt via a new Put
	// with an old timestamp; then trigger GC through another Put.
	old := timeutil.Now().Add(-time.Hour)
	s.Put(&Record{ID: "snap_ttl", CreatedAt: old})
	if _, ok := s.Get("snap_ttl"); ok {
		t.Fatal("expected record to be expired")
	}
}

func TestStoreSnapshotIDs(t *testing.T) {
	s := NewStore(time.Minute)
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("snap_%d", i)
		s.Put(&Record{ID: id, CreatedAt: timeutil.Now()})
	}
	ids := s.Snapshot()
	if len(ids) != 5 {
		t.Fatalf("Snapshot() returned %d ids, want 5", len(ids))
	}
}
