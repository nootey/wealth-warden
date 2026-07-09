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
)

type Client struct {
	userID int64
	conn   *websocket.Conn
	send   chan Event
	done   chan struct{}
	once   sync.Once
}

func NewClient(userID int64, conn *websocket.Conn) *Client {
	return &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan Event, sendBuffer),
		done:   make(chan struct{}),
	}
}

func (c *Client) close() {
	c.once.Do(func() { close(c.done) })
}

func (c *Client) serve(ctx context.Context) {
	defer func() { _ = c.conn.Close(websocket.StatusNormalClosure, "") }()

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
