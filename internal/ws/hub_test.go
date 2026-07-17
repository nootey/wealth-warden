package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
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

func (h *Hub) connCount(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID])
}

func (h *Hub) userCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users)
}

// serveHub exposes Serve over a real upgrade, so tests observe the register ->
// serve -> unregister lifecycle through an actual connection rather than the map.
func serveHub(t *testing.T, h *Hub, userID int64, sessionID string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		h.Serve(r.Context(), NewClient(userID, sessionID, conn))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dialHub(t *testing.T, ctx context.Context, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

// TestSendDropsWhenBufferFull pins the non-blocking contract: a client that
// never drains must not stall the caller, which is a job goroutine.
func TestSendDropsWhenBufferFull(t *testing.T) {
	h := newTestHub()
	c := NewClient(1, "", nil)
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
	first, second := NewClient(1, "", nil), NewClient(1, "", nil)
	h.register(first)
	h.register(second)
	h.register(NewClient(2, "", nil))

	h.Send(1, Event{Type: TypeReportCompleted})

	if len(first.send) != 1 || len(second.send) != 1 {
		t.Fatalf("got %d and %d events, want 1 each", len(first.send), len(second.send))
	}
}

func TestCloseSessionTargetsOnlyThatSession(t *testing.T) {
	h := newTestHub()
	revokedA := NewClient(1, "revoked", nil)
	revokedB := NewClient(1, "revoked", nil)
	kept := NewClient(1, "kept", nil)
	otherUser := NewClient(2, "revoked", nil)
	for _, c := range []*Client{revokedA, revokedB, kept, otherUser} {
		h.register(c)
	}

	h.CloseSession(1, "revoked")

	if !isClosed(revokedA.done) || !isClosed(revokedB.done) {
		t.Fatal("revoked session's clients were not closed")
	}
	if isClosed(kept.done) {
		t.Fatal("other session's client was closed")
	}
	if isClosed(otherUser.done) {
		t.Fatal("other user's client was closed")
	}
}

func TestCloseUserSparesOtherUsers(t *testing.T) {
	h := newTestHub()
	first := NewClient(1, "a", nil)
	second := NewClient(1, "b", nil)
	otherUser := NewClient(2, "c", nil)
	for _, c := range []*Client{first, second, otherUser} {
		h.register(c)
	}

	h.CloseUser(1)

	if !isClosed(first.done) || !isClosed(second.done) {
		t.Fatal("user's clients were not closed")
	}
	if isClosed(otherUser.done) {
		t.Fatal("other user's client was closed")
	}
}

func TestRegisterEvictsOldestBeyondCap(t *testing.T) {
	h := newTestHub()

	clients := make([]*Client, 0, maxConnsPerUser+1)
	for range maxConnsPerUser + 1 {
		c := NewClient(1, "", nil)
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
	first, second := NewClient(1, "", nil), NewClient(1, "", nil)
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

// TestServeUnregistersOnDisconnect guards the leak: a dropped connection must
// leave no entry behind, or the fan-out in Send grows without bound.
func TestServeUnregistersOnDisconnect(t *testing.T) {
	h := newTestHub()
	srv := serveHub(t, h, 1, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialHub(t, ctx, srv)
	waitFor(t, func() bool { return h.connCount(1) == 1 }, "client never registered")

	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatalf("close: %v", err)
	}

	waitFor(t, func() bool { return h.userCount() == 0 }, "hub retained client after disconnect")
}

// TestEvictedConnectionIsTerminated pins that closing done actually unblocks
// serve and closes the socket, rather than only flipping a channel.
func TestEvictedConnectionIsTerminated(t *testing.T) {
	h := newTestHub()
	srv := serveHub(t, h, 1, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	oldest := dialHub(t, ctx, srv)
	waitFor(t, func() bool { return h.connCount(1) == 1 }, "first client never registered")

	for range maxConnsPerUser {
		dialHub(t, ctx, srv)
	}
	waitFor(t, func() bool { return h.connCount(1) == maxConnsPerUser }, "hub exceeded connection cap")

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	// CloseStatus is -1 for a read that timed out, which is what a still-open
	// connection looks like; only a close frame proves the server hung up.
	_, _, err := oldest.Read(readCtx)
	if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("evicted connection not closed by server, read error = %v", err)
	}
}

// TestCloseSessionSendsRevokedCode pins the wire contract: the web client
// decides between "log out" and "reconnect" purely from the close code.
func TestCloseSessionSendsRevokedCode(t *testing.T) {
	h := newTestHub()
	srv := serveHub(t, h, 1, "revoked")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialHub(t, ctx, srv)
	waitFor(t, func() bool { return h.connCount(1) == 1 }, "client never registered")

	h.CloseSession(1, "revoked")

	readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
	defer readCancel()
	_, _, err := conn.Read(readCtx)
	if websocket.CloseStatus(err) != statusSessionRevoked {
		t.Fatalf("close status = %v, want %v", websocket.CloseStatus(err), statusSessionRevoked)
	}
}

// TestConcurrentSendAndChurn exercises Send against registration churn; meaningful
// only under -race, where it pins the mutex discipline the non-blocking Send relies on.
func TestConcurrentSendAndChurn(t *testing.T) {
	h := newTestHub()

	const (
		workers    = 8
		iterations = 200
	)

	var churn sync.WaitGroup
	for range workers {
		churn.Add(1)
		go func() {
			defer churn.Done()
			for range iterations {
				c := NewClient(1, "", nil)
				h.register(c)
				h.unregister(c)
			}
		}()
	}

	stop := make(chan struct{})
	var senders sync.WaitGroup
	for range workers {
		senders.Add(1)
		go func() {
			defer senders.Done()
			for {
				select {
				case <-stop:
					return
				default:
					h.Send(1, Event{Type: TypeReportCompleted})
				}
			}
		}()
	}

	churn.Wait()
	close(stop)
	senders.Wait()

	if got := h.userCount(); got != 0 {
		t.Fatalf("users retained after churn = %d, want 0", got)
	}
}
