package proxy_test

import (
	"fmt"
	"sync"
	"testing"

	"goscrapy/internal/proxy"
)

func TestPoolEvictionsConcurrentWithHealthReports(t *testing.T) {
	const rounds = 200
	p := proxy.New("real", []string{"http://proxy.example:8080"})
	ready := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			p.Reset([]string{"http://proxy.example:8080"})
			for failure := 0; failure < 3; failure++ {
				if i == 0 && failure == 2 {
					close(ready)
				}
				p.Report("http://proxy.example:8080", false, fmt.Sprintf("dial failure %d", i))
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-ready
		for i := 0; i < rounds*20; i++ {
			_ = p.Evictions()
		}
	}()
	wg.Wait()

	if got := p.Evictions(); got != rounds {
		t.Fatalf("evictions = %d, want %d", got, rounds)
	}
}
