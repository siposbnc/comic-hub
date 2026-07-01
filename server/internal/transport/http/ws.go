package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/presence"
)

// Topics clients can subscribe to (docs/03-api.md §10).
const (
	TopicJobs      = "jobs"
	TopicProgress  = "progress"
	TopicLibrary   = "library"
	TopicBookmarks = "bookmarks"
	TopicPresence  = "presence"
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
	conn *websocket.Conn
	send chan []byte
	// user is the connection's authenticated identity (the implicit owner in embedded /
	// auth-off mode), fixed at upgrade. Per-user topics (progress, bookmarks) deliver
	// only to that user's sockets; presence applies the user's content ceiling.
	user   domain.User
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
		// Resolve identity before the upgrade; absent (embedded / auth-off) = the
		// unrestricted implicit owner, matching currentUserID's fallback.
		user, ok := userFromContext(r.Context())
		if !ok {
			user = domain.User{ID: domain.OwnerUserID}
		}
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return // Upgrade already wrote an error response.
		}
		c := &wsClient{conn: conn, send: make(chan []byte, wsSendBuffer), user: user, topics: make(map[string]bool)}
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
	h.broadcastWhere(topic, eventType, data, nil)
}

// broadcastWhere sends an event to every subscribed client whose connection identity
// passes allow (nil = everyone). This is how per-user topics and content-ceiling
// filtering work without per-client payloads.
func (h *Hub) broadcastWhere(topic, eventType string, data any, allow func(viewer domain.User) bool) {
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
		if !subscribed || (allow != nil && !allow(c.user)) {
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

// BroadcastProgress publishes a progress update on the progress topic — only to the
// owning user's connections (cross-device sync is per-user; other members' reading
// activity is presence's job, which applies content ceilings).
func (h *Hub) BroadcastProgress(p domain.Progress) {
	h.broadcastWhere(TopicProgress, "progress.updated", toProgressDTO(p),
		func(viewer domain.User) bool { return viewer.ID == p.UserID })
}

// BroadcastBookmarks signals that one of the user's books' bookmarks changed
// (added/edited/removed), so that user's other devices refresh the list.
func (h *Hub) BroadcastBookmarks(userID, bookID string) {
	h.broadcastWhere(TopicBookmarks, "bookmarks.updated", map[string]string{"bookId": bookID},
		func(viewer domain.User) bool { return viewer.ID == userID })
}

// BroadcastPresence publishes a presence change ("now reading"). Entries for a book
// above a viewer's content ceiling are withheld from that viewer — restricted users
// never learn of over-rated content, mirroring browse filtering.
func (h *Hub) BroadcastPresence(e presence.Entry, active bool) {
	eventType := "presence.updated"
	var data any = e
	if !active {
		eventType = "presence.cleared"
		data = map[string]string{"userId": e.UserID}
	}
	h.broadcastWhere(TopicPresence, eventType, data,
		func(viewer domain.User) bool { return access.Allowed(viewer.AgeRatingMax, e.AgeRating) })
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
