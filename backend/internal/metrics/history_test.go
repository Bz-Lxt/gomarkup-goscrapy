package metrics

import (
	"testing"
)

func TestRingWindowIsImmutable(t *testing.T) {
	r := NewRing(8)
	r.Add(Point{CPU: 12.3, PagesPerMin: 50, MemoryMB: 100, FailRate: 0.1})
	r.Add(Point{CPU: 45.6, PagesPerMin: 80, MemoryMB: 200, FailRate: 0.2})

	view := r.Window()
	if len(view) != 2 {
		t.Fatalf("len=%d", len(view))
	}

	// Caller performs in-place normalization on the returned array.
	for i := range view {
		view[i].CPU = 0
		view[i].PagesPerMin = 0
	}

	// The collector's stored data must be untouched.
	latest, ok := r.Latest()
	if !ok {
		t.Fatal("no latest")
	}
	if latest.CPU != 45.6 || latest.PagesPerMin != 80 {
		t.Fatalf("latest corrupted: %+v", latest)
	}

	view2 := r.Window()
	if view2[1].CPU != 45.6 || view2[1].PagesPerMin != 80 {
		t.Fatalf("window corrupted: %+v", view2[1])
	}
}
