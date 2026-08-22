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
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
	closed bool
}

// trySend performs a non-blocking send that is safe to call concurrently with
// shutdown: the send and the close are serialized by c.mu, so a broadcast
// racing with a disconnect can never hit a "send on closed channel" panic.
func (c *client) trySend(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.send <- b:
	default:
	}
}

// shutdown closes the send channel exactly once and is mutually exclusive with
// trySend, guaranteeing no in-flight broadcast send can observe a closed chan.
func (c *client) shutdown() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
	c.mu.Unlock()
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
		c.trySend(h.last) // safe even if a concurrent disconnect is racing
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
	c.shutdown() // closes c.send exactly once, mutually exclusive with trySend
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
	targets := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.Unlock()
	for _, c := range targets {
		c.trySend(b) // safe vs. concurrent shutdown: no send-on-closed
	}
}

func (h *Hub) Clients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
