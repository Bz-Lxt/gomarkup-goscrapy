package ratelimit

import (
	"context"
	"sync"
	"time"
)

type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	rate     float64
	capacity float64
	last     time.Time
}

func newBucket(qps float64) *tokenBucket {
	if qps <= 0 {
		qps = 2
	}
	now := time.Now()
	return &tokenBucket{tokens: qps, rate: qps, capacity: qps * 2, last: now}
}

func (b *tokenBucket) setRate(qps float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if qps <= 0 {
		qps = 0.2
	}
	b.rate = qps
	if b.capacity < qps*2 {
		b.capacity = qps * 2
	}
}

func (b *tokenBucket) wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(b.last).Seconds()
		b.last = now
		b.tokens += elapsed * b.rate
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		need := (1 - b.tokens) / b.rate
		d := time.Duration(need * float64(time.Second))
		if d < 5*time.Millisecond {
			d = 5 * time.Millisecond
		}
		timer := time.NewTimer(d)
		defer func() {
			timer.Stop()
			b.mu.Unlock()
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (b *tokenBucket) rateLocked() float64 {
	return b.rate
}

// Limiter is a per-domain token bucket collection.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	defaultQ float64
}

func NewLimiter(defaultQPS float64) *Limiter {
	if defaultQPS <= 0 {
		defaultQPS = 2
	}
	return &Limiter{buckets: map[string]*tokenBucket{}, defaultQ: defaultQPS}
}

func (l *Limiter) Wait(ctx context.Context, domain string, qps float64) error {
	if domain == "" {
		domain = "default"
	}
	if qps <= 0 {
		qps = l.defaultQ
	}
	b := l.bucket(domain, qps)
	return b.wait(ctx)
}

func (l *Limiter) SetQPS(domain string, qps float64) {
	b := l.bucket(domain, qps)
	b.setRate(qps)
}

func (l *Limiter) CurrentQPS(domain string) float64 {
	l.mu.Lock()
	b := l.buckets[domain]
	l.mu.Unlock()
	if b == nil {
		return l.defaultQ
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rate
}

func (l *Limiter) bucket(domain string, qps float64) *tokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[domain]
	if !ok {
		b = newBucket(qps)
		l.buckets[domain] = b
	}
	return b
}
