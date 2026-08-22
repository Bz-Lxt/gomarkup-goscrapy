package metrics_test

import (
	"testing"
	"time"

	"goscrapy/internal/metrics"
)

func TestRingWindowIsSnapshot(t *testing.T) {
	ring := metrics.NewRing(8)
	want := metrics.Point{
		At:          time.Unix(1700000000, 0),
		CPU:         12.5,
		MemoryMB:    256,
		PagesPerMin: 42,
		FailRate:    0.1,
	}
	ring.Add(want)

	window := ring.Window()
	if len(window) != 1 {
		t.Fatalf("Window() returned %d points, want 1", len(window))
	}
	window[0].CPU = 99
	window[0].PagesPerMin = 0

	got, ok := ring.Latest()
	if !ok {
		t.Fatal("Latest() reported an empty ring after Add")
	}
	if got != want {
		t.Fatalf("mutating Window() result changed stored sample: got %+v, want %+v", got, want)
	}
}
