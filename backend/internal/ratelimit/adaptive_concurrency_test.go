package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"goscrapy/internal/ratelimit"
)

func TestAdaptiveConcurrentCommandAndObservation(t *testing.T) {
	limiter := ratelimit.NewLimiter(20)
	adaptive := ratelimit.NewAdaptive(limiter)
	const domain = "news.example"

	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < 1000; i++ {
				if worker%2 == 0 {
					adaptive.ApplyCommand(domain, float64(i%9+1))
					continue
				}
				adaptive.Observe(domain, 200, 25*time.Millisecond, 20)
				_ = adaptive.QPS(domain)
			}
		}(worker)
	}
	close(start)
	wg.Wait()

	adaptive.ApplyCommand(domain, 7)
	if got := adaptive.QPS(domain); got != 7 {
		t.Fatalf("adaptive QPS after final command = %v, want 7", got)
	}
	if got := limiter.CurrentQPS(domain); got != 7 {
		t.Fatalf("limiter QPS after final command = %v, want 7", got)
	}
}
