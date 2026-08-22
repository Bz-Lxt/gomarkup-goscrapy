package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"goscrapy/internal/logger"
	"goscrapy/internal/model"
	"goscrapy/internal/timeutil"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
	last    []byte
}

func NewHub() *Hub {
	return &Hub{clients: map[*client]struct{}{}}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Named("ws").Warn("upgrade failed", zap.Error(err))
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 8)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	if h.last != nil {
		select {
		case c.send <- h.last:
		default:
		}
	}
	h.mu.Unlock()
	go c.write()
	c.read()
	h.remove(c)
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
	_ = c.conn.Close()
}

func (c *client) read() {
	defer c.conn.Close()
	_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) write() {
	tick := time.NewTicker(40 * time.Second)
	defer tick.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-tick.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type metricsMsg struct {
	Type  string             `json:"type"`
	TS    string             `json:"ts"`
	Nodes []model.WorkerNode `json:"nodes"`
}

func (h *Hub) BroadcastMetrics(nodes []model.WorkerNode) {
	if nodes == nil {
		nodes = []model.WorkerNode{}
	}
	view := make([]model.WorkerNode, len(nodes))
	copy(view, nodes)
	payload := metricsMsg{Type: "metrics", TS: timeutil.Format(timeutil.Now()), Nodes: view}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.last = b
	targets := make([]chan []byte, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c.send)
	}
	h.mu.Unlock()
	for _, send := range targets {
		select {
		case send <- b:
		default:
		}
	}
}

func (h *Hub) Clients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
