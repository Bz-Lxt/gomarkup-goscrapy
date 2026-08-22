package ratelimit

import (
	"sync"
	"time"
)

type sample struct {
	status int
	lat    time.Duration
	at     time.Time
}

// Adaptive watches recent 429/403 and latency, then nudges per-domain QPS.
type Adaptive struct {
	mu      sync.Mutex
	window  time.Duration
	samples map[string][]sample
	baseQPS map[string]float64
	curQPS  map[string]float64
	limiter *Limiter
}

func NewAdaptive(l *Limiter) *Adaptive {
	return &Adaptive{
		window:  30 * time.Second,
		samples: map[string][]sample{},
		baseQPS: map[string]float64{},
		curQPS:  map[string]float64{},
		limiter: l,
	}
}

func (a *Adaptive) Observe(domain string, status int, lat time.Duration, baseQPS float64) float64 {
	if domain == "" {
		domain = "default"
	}
	if baseQPS <= 0 {
		baseQPS = 2
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.baseQPS[domain] = baseQPS
	list := append(a.samples[domain], sample{status: status, lat: lat, at: now})
	cut := now.Add(-a.window)
	kept := list[:0]
	for _, s := range list {
		if s.at.After(cut) {
			kept = append(kept, s)
		}
	}
	a.samples[domain] = kept

	var blocked, total int
	var sum time.Duration
	for _, s := range kept {
		total++
		sum += s.lat
		if s.status == 429 || s.status == 403 {
			blocked++
		}
	}
	cur := a.curQPS[domain]
	if cur <= 0 {
		cur = baseQPS
	}
	if total > 0 {
		ratio := float64(blocked) / float64(total)
		avg := sum / time.Duration(total)
		switch {
		case ratio >= 0.2:
			cur = cur * 0.5
		case ratio >= 0.05:
			cur = cur * 0.75
		case avg > 1500*time.Millisecond:
			cur = cur * 0.85
		case ratio == 0 && avg < 400*time.Millisecond:
			cur = cur * 1.08
		}
	}
	if cur < 0.2 {
		cur = 0.2
	}
	if cur > baseQPS {
		cur = baseQPS
	}
	a.curQPS[domain] = cur
	if a.limiter != nil {
		a.limiter.SetQPS(domain, cur)
	}
	return cur
}

func (a *Adaptive) QPS(domain string) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if q, ok := a.curQPS[domain]; ok && q > 0 {
		return q
	}
	if q, ok := a.baseQPS[domain]; ok && q > 0 {
		return q
	}
	return 2
}

func (a *Adaptive) ApplyCommand(domain string, qps float64) {
	a.mu.Lock()
	a.curQPS[domain] = qps
	a.mu.Unlock()
	if a.limiter != nil {
		a.limiter.SetQPS(domain, qps)
	}
}
