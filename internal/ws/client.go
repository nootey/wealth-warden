package ws

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	pingInterval = 30 * time.Second
	writeTimeout = 10 * time.Second
	sendBuffer   = 16
	readLimit    = 1024

	// application close code (4000-4999 range); the web client logs out when it sees it
	statusSessionRevoked websocket.StatusCode = 4001
)

type Client struct {
	userID    int64
	sessionID string
	conn      *websocket.Conn
	send      chan Event
	done      chan struct{}
	once      sync.Once

	closeCode   websocket.StatusCode
	closeReason string
}

func NewClient(userID int64, sessionID string, conn *websocket.Conn) *Client {
	return &Client{
		userID:    userID,
		sessionID: sessionID,
		conn:      conn,
		send:      make(chan Event, sendBuffer),
		done:      make(chan struct{}),
		closeCode: websocket.StatusNormalClosure,
	}
}

func (c *Client) close() {
	c.closeWith(websocket.StatusNormalClosure, "")
}

// closeWith records the close frame to send before releasing serve; the write
// happens-before serve reads the fields because it observes done closing.
func (c *Client) closeWith(code websocket.StatusCode, reason string) {
	c.once.Do(func() {
		c.closeCode = code
		c.closeReason = reason
		close(c.done)
	})
}

func (c *Client) serve(ctx context.Context) {
	defer func() { _ = c.conn.Close(c.closeCode, c.closeReason) }()

	c.conn.SetReadLimit(readLimit)
	ctx = c.conn.CloseRead(ctx)

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case event := <-c.send:
			if err := c.write(ctx, event); err != nil {
				return
			}
		case <-ticker.C:
			// Ping doubles as liveness: no pong within writeTimeout kills the connection.
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) write(ctx context.Context, event Event) error {
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return wsjson.Write(writeCtx, c.conn, event)
}
