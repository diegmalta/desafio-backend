package wsbus

import (
	"sync"

	"github.com/google/uuid"
)

// Hub tracks WebSocket clients per citizen_id (in-process only).
type Hub struct {
	mu        sync.RWMutex
	byCitizen map[uuid.UUID]map[*Client]struct{}
	closed    bool
}

func NewHub() *Hub {
	return &Hub{byCitizen: make(map[uuid.UUID]map[*Client]struct{})}
}

// Register adds a client; must be called before Run.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	m := h.byCitizen[c.citizenID]
	if m == nil {
		m = make(map[*Client]struct{})
		h.byCitizen[c.citizenID] = m
	}
	m[c] = struct{}{}
}

// Unregister removes a client (safe if already removed).
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m, ok := h.byCitizen[c.citizenID]
	if !ok {
		return
	}
	delete(m, c)
	if len(m) == 0 {
		delete(h.byCitizen, c.citizenID)
	}
}

// Dispatch sends payload to all connections for the citizen (JSON bytes).
// Slow clients are dropped: full buffer closes the connection.
func (h *Hub) Dispatch(citizenID uuid.UUID, payload []byte) {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return
	}
	m := h.byCitizen[citizenID]
	clients := make([]*Client, 0, len(m))
	for c := range m {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- append([]byte(nil), payload...):
		default:
			c.shutdown()
		}
	}
}

// Close shuts down all clients and prevents new registrations.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	var all []*Client
	for _, m := range h.byCitizen {
		for c := range m {
			all = append(all, c)
		}
	}
	h.byCitizen = make(map[uuid.UUID]map[*Client]struct{})
	h.mu.Unlock()

	for _, c := range all {
		c.shutdown()
	}
}
