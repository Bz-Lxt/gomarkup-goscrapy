package renderer_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"goscrapy/internal/renderer"
)

func TestStoreConcurrentPutAndGet(t *testing.T) {
	store := renderer.NewStore(time.Hour)
	store.Put(&renderer.Record{ID: "initial", PNG: []byte("initial"), CreatedAt: time.Now()})

	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		worker := worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 20; i++ {
				id := fmt.Sprintf("snapshot-%d-%d", worker, i)
				store.Put(&renderer.Record{ID: id, PNG: []byte(id), CreatedAt: time.Now()})
				if rec, ok := store.Get(id); ok && rec.ID != id {
					t.Errorf("Get(%q) returned snapshot %q", id, rec.ID)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
