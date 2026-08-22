package metrics

import (
	"sync"
	"time"
)

type Point struct {
	At          time.Time
	CPU         float64
	MemoryMB    float64
	PagesPerMin float64
	FailRate    float64
}

type Ring struct {
	mu    sync.Mutex
	max   int
	items []Point
}

func NewRing(max int) *Ring {
	if max < 8 {
		max = 60
	}
	return &Ring{max: max, items: make([]Point, 0, max)}
}

func (r *Ring) Add(p Point) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.items) >= r.max {
		copy(r.items, r.items[1:])
		r.items[len(r.items)-1] = p
		return
	}
	r.items = append(r.items, p)
}

func (r *Ring) Latest() (Point, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.items) == 0 {
		return Point{}, false
	}
	return r.items[len(r.items)-1], true
}

func (r *Ring) Window() []Point {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items
}

func (c *Collector) Record(ring *Ring) Snapshot {
	s := c.Snapshot()
	if ring != nil {
		ring.Add(Point{
			At:          time.Now(),
			CPU:         s.CPU,
			MemoryMB:    s.MemoryMB,
			PagesPerMin: s.PagesPerMin,
			FailRate:    s.FailRate,
		})
	}
	return s
}
