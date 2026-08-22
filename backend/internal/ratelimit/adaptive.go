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
	mu       sync.Mutex
	window   time.Duration
	samples  map[string][]sample
	baseQPS  map[string]float64
	curQPS   map[string]float64
	cmdEpoch map[string]uint64 // bumped on every ApplyCommand
	limiter  *Limiter
}

func NewAdaptive(l *Limiter) *Adaptive {
	return &Adaptive{
		window:   30 * time.Second,
		samples:  map[string][]sample{},
		baseQPS:  map[string]float64{},
		curQPS:   map[string]float64{},
		cmdEpoch: map[string]uint64{},
		limiter:  l,
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
	// If a control-plane command has been applied for this domain
	// (cmdEpoch > 0), use the command's QPS as the authoritative ceiling
	// and don't let it be overwritten by the rule's default QPS.  This
	// ensures the command's intent survives subsequent Observe calls.
	ceiling := baseQPS
	if a.cmdEpoch[domain] > 0 {
		ceiling = a.baseQPS[domain]
	} else {
		a.baseQPS[domain] = baseQPS
	}
	old := a.samples[domain]
	list := make([]sample, 0, len(old)+1)
	list = append(list, old...)
	list = append(list, sample{status: status, lat: lat, at: now})
	cut := now.Add(-a.window)
	kept := make([]sample, 0, len(list))
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
		cur = ceiling
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
	if cur > ceiling {
		cur = ceiling
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

// ApplyCommand atomically applies a control-plane QPS directive.  It acquires
// a.mu so that it cannot race with a concurrent Observe call: the two
// operations are fully serialised, guaranteeing that whichever acquires the
// lock last leaves the canonical value.  Because ApplyCommand updates both
// curQPS and the underlying Limiter while holding the same lock, QPS() and
// Limiter.CurrentQPS() always observe a consistent value.
func (a *Adaptive) ApplyCommand(domain string, qps float64) {
	if domain == "" {
		domain = "default"
	}
	if qps <= 0 {
		qps = 0.2
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Bump the command epoch so that Observe knows a control-plane
	// directive has been issued and treats a.baseQPS[domain] as the
	// authoritative ceiling rather than overwriting it with rule.QPS.
	a.cmdEpoch[domain]++
	// Update the Limiter first so that CurrentQPS() sees the new value as
	// early as possible, then update curQPS so QPS() returns the command
	// value the moment the lock is released.
	if a.limiter != nil {
		a.limiter.SetQPS(domain, qps)
	}
	a.curQPS[domain] = qps
	a.baseQPS[domain] = qps
}
