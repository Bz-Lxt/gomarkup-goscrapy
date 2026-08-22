package worker

import (
	"sync"
	"time"
)

type Counters struct {
	mu       sync.Mutex
	pages    int64
	fails    int64
	results  int64
	enqueued int64
	started  time.Time
}

func newCounters() *Counters {
	return &Counters{started: time.Now()}
}

func (c *Counters) Add(pages, fails, results, enqueued int64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.pages += pages
	c.fails += fails
	c.results += results
	c.enqueued += enqueued
	c.mu.Unlock()
}

func (c *Counters) Snapshot() (pages, fails, results, enqueued int64, ppm float64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pages, fails, results, enqueued = c.pages, c.fails, c.results, c.enqueued
	elapsed := time.Since(c.started).Minutes()
	if elapsed <= 0 {
		elapsed = 1.0 / 60.0
	}
	ppm = float64(pages) / elapsed
	return
}
