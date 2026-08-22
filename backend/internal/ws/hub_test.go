package ws

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"goscrapy/internal/model"
)

// TestClientSendShutdownRace is the direct regression test for the
// "panic: send on closed channel" crash. Under the old implementation
// BroadcastMetrics sent to c.send outside any lock while remove() closed
// c.send outside the same lock, so a broadcast colliding with a disconnect
// could send on a closed channel and panic master. The send and the close
// are now serialized by the per-client mutex.
func TestClientSendShutdownRace(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		c := &client{send: make(chan []byte, 1)}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.shutdown()
		}()
		go func() {
			defer wg.Done()
			c.trySend([]byte("x"))
		}()
		wg.Wait()
	}
}

// TestBroadcastMetricsSurvivesConcurrentDisconnect exercises the full
// Hub path reported in the bug: node-metrics broadcasts racing against
// browsers refreshing/closing. Surviving clients must keep receiving
// metrics and the process must not panic.
func TestBroadcastMetricsSurvivesConcurrentDisconnect(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(h)
	defer srv.Close()
	url := "ws" + srv.URL[len("http"):] + "/"

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}

	// Open a handful of clients; some stay online the whole time.
	const total = 8
	conns := make([]*websocket.Conn, total)
	for i := range conns {
		c, _, err := dialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		conns[i] = c
	}
	// Give the server a moment to register each connection.
	time.Sleep(50 * time.Millisecond)

	nodes := []model.WorkerNode{{ID: "n1", CPU: 12.3, MemoryMB: 256}}

	var wg sync.WaitGroup
	wg.Add(2)
	// Hammer broadcasts the whole time.
	go func() {
		defer wg.Done()
		for j := 0; j < 500; j++ {
			h.BroadcastMetrics(nodes)
		}
	}()
	// Simultaneously drop half the clients (refresh / close tab).
	go func() {
		defer wg.Done()
		for i := 0; i < total; i += 2 {
			_ = conns[i].Close()
		}
	}()
	wg.Wait()

	// Surviving clients should still be able to receive a fresh broadcast.
	h.BroadcastMetrics(nodes)
	deadline := time.Now().Add(2 * time.Second)
	for i := 1; i < total; i += 2 {
		_ = conns[i].SetReadDeadline(deadline)
		if _, _, err := conns[i].ReadMessage(); err != nil {
			t.Fatalf("survivor %d stopped receiving: %v", i, err)
		}
	}
	for _, c := range conns {
		_ = c.Close()
	}
}
