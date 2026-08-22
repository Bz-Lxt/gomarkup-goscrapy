package ws_test

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"goscrapy/internal/model"
	"goscrapy/internal/ws"
)

func TestHubBroadcastDuringDisconnect(t *testing.T) {
	hub := ws.NewHub()
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	const clients = 80
	conns := make([]*websocket.Conn, 0, clients)
	for i := 0; i < clients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial websocket %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	stop := make(chan struct{})
	var broadcasts sync.WaitGroup
	broadcasts.Add(1)
	go func() {
		defer broadcasts.Done()
		for {
			select {
			case <-stop:
				return
			default:
				hub.BroadcastMetrics([]model.WorkerNode{{ID: "worker-1"}})
			}
		}
	}()

	var closes sync.WaitGroup
	for _, conn := range conns {
		closes.Add(1)
		go func(conn *websocket.Conn) {
			defer closes.Done()
			_ = conn.Close()
		}(conn)
	}
	closes.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for hub.Clients() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(stop)
	broadcasts.Wait()
	if got := hub.Clients(); got != 0 {
		t.Fatalf("connected clients after all peers disconnected: %d", got)
	}

	hub.BroadcastMetrics([]model.WorkerNode{{ID: "worker-2"}})
}
