package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// Topics clients can subscribe to (docs/03-api.md §10).
const (
	TopicJobs     = "jobs"
	TopicProgress = "progress"
	TopicLibrary  = "library"
)

const (
	wsWriteWait  = 10 * time.Second
	wsPongWait   = 60 * time.Second
	wsPingPeriod = 30 * time.Second
	wsSendBuffer = 32
)

// Hub is the WebSocket fan-out: clients subscribe to topics and receive JSON event
// frames. One multiplexed socket per client (docs/01-architecture.md §7).
type Hub struct {
	logger  *slog.Logger
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

// NewHub creates an empty hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{logger: logger, clients: make(map[*wsClient]struct{})}
}

type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
	topics map[string]bool
}

// outbound is a server->client event frame.
type outbound struct {
	Type  string `json:"type"`
	Topic string `json:"topic"`
	Data  any    `json:"data,omitempty"`
}

// inbound is a client->server control frame.
type inbound struct {
	Type   string   `json:"type"`
	Topics []string `json:"topics,omitempty"`
}

var wsUpgrader = websocket.Upgrader{
	// Loopback/LAN only by design; auth is enforced by the token middleware before the
	// upgrade. Origin is therefore not used as a security boundary here.
	CheckOrigin: func(*http.Request) bool { return true },
}

func (h *Hub) handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return // Upgrade already wrote an error response.
		}
		c := &wsClient{conn: conn, send: make(chan []byte, wsSendBuffer), topics: make(map[string]bool)}
		h.add(c)
		go h.writePump(c)
		h.readPump(c)
	}
}

func (h *Hub) add(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *wsClient) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// Broadcast sends an event to every client subscribed to topic. Slow clients (full
// buffer) are dropped rather than blocking the broadcaster.
func (h *Hub) Broadcast(topic, eventType string, data any) {
	payload, err := json.Marshal(outbound{Type: eventType, Topic: topic, Data: data})
	if err != nil {
		h.logger.Error("ws marshal", "err", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.mu.Lock()
		subscribed := c.topics[topic]
		c.mu.Unlock()
		if !subscribed {
			continue
		}
		select {
		case c.send <- payload:
		default:
			h.logger.Warn("ws client send buffer full; dropping event", "topic", topic)
		}
	}
}

// BroadcastJob publishes a job update on the jobs topic.
func (h *Hub) BroadcastJob(j domain.Job) {
	h.Broadcast(TopicJobs, "job."+string(j.State), toJobDTO(j))
}

// BroadcastProgress publishes a progress update on the progress topic.
func (h *Hub) BroadcastProgress(p domain.Progress) {
	h.Broadcast(TopicProgress, "progress.updated", toProgressDTO(p))
}

func (h *Hub) readPump(c *wsClient) {
	defer func() {
		h.remove(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var in inbound
		if json.Unmarshal(msg, &in) != nil {
			continue
		}
		switch in.Type {
		case "subscribe":
			c.mu.Lock()
			for _, t := range in.Topics {
				c.topics[t] = true
			}
			c.mu.Unlock()
		case "unsubscribe":
			c.mu.Lock()
			for _, t := range in.Topics {
				delete(c.topics, t)
			}
			c.mu.Unlock()
		case "ping":
			// Application-level ping; reply on the same socket.
			select {
			case c.send <- mustJSON(outbound{Type: "pong"}):
			default:
			}
		}
	}
}

func (h *Hub) writePump(c *wsClient) {
	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
