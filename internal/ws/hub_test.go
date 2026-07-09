package ws

import (
	"testing"

	"go.uber.org/zap"
)

func newTestHub() *Hub { return NewHub(zap.NewNop()) }

func isClosed(done chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// TestSendDropsWhenBufferFull pins the non-blocking contract: a client that
// never drains must not stall the caller, which is a job goroutine.
func TestSendDropsWhenBufferFull(t *testing.T) {
	h := newTestHub()
	c := NewClient(1, nil)
	h.register(c)

	for range sendBuffer + 5 {
		h.Send(1, Event{Type: TypeReportCompleted})
	}

	if len(c.send) != sendBuffer {
		t.Fatalf("send buffer = %d, want %d", len(c.send), sendBuffer)
	}
}

func TestSendReachesAllConnectionsForUser(t *testing.T) {
	h := newTestHub()
	first, second := NewClient(1, nil), NewClient(1, nil)
	h.register(first)
	h.register(second)
	h.register(NewClient(2, nil))

	h.Send(1, Event{Type: TypeReportCompleted})

	if len(first.send) != 1 || len(second.send) != 1 {
		t.Fatalf("got %d and %d events, want 1 each", len(first.send), len(second.send))
	}
}

func TestRegisterEvictsOldestBeyondCap(t *testing.T) {
	h := newTestHub()

	clients := make([]*Client, 0, maxConnsPerUser+1)
	for range maxConnsPerUser + 1 {
		c := NewClient(1, nil)
		clients = append(clients, c)
		h.register(c)
	}

	if got := len(h.users[1]); got != maxConnsPerUser {
		t.Fatalf("connections = %d, want %d", got, maxConnsPerUser)
	}
	if !isClosed(clients[0].done) {
		t.Fatal("oldest client was not closed")
	}
	if isClosed(clients[maxConnsPerUser].done) {
		t.Fatal("newest client was closed")
	}
}

func TestUnregisterLastConnectionDeletesUser(t *testing.T) {
	h := newTestHub()
	first, second := NewClient(1, nil), NewClient(1, nil)
	h.register(first)
	h.register(second)

	h.unregister(first)
	if got := len(h.users[1]); got != 1 {
		t.Fatalf("connections = %d, want 1", got)
	}

	h.unregister(second)
	if _, ok := h.users[1]; ok {
		t.Fatal("user key retained after last connection unregistered")
	}
}
