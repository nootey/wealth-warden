package ws

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

const maxConnsPerUser = 5

var _ Broadcaster = (*Hub)(nil)

type Hub struct {
	logger *zap.Logger
	mu     sync.RWMutex
	users  map[int64][]*Client
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		logger: logger,
		users:  make(map[int64][]*Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, clients := range h.users {
		for _, c := range clients {
			c.close()
		}
	}
	h.users = make(map[int64][]*Client)
}

func (h *Hub) Serve(ctx context.Context, c *Client) {
	h.register(c)
	defer h.unregister(c)
	c.serve(ctx)
}

func (h *Hub) Send(userID int64, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.users[userID] {
		select {
		case c.send <- event:
		default:
			h.logger.Warn("ws send buffer full, dropping event",
				zap.Int64("user_id", userID),
				zap.String("event_type", event.Type),
			)
		}
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.users[c.userID]
	if len(clients) >= maxConnsPerUser {
		oldest := clients[0]
		clients = clients[1:]
		oldest.close()
	}
	h.users[c.userID] = append(clients, c)
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.users[c.userID]
	for i, existing := range clients {
		if existing == c {
			h.users[c.userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.users[c.userID]) == 0 {
		delete(h.users, c.userID)
	}
	c.close()
}
