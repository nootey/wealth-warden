package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wealth-warden/internal/ws"
	"wealth-warden/pkg/config"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TestWebsocketHandler_PushesEventToConnectedUser exercises the upgrade through
// gin: a hijacked writer, the hub registration, and the JSON frame on the wire.
func TestWebsocketHandler_PushesEventToConnectedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := ws.NewHub(zap.NewNop())
	handler := NewWebsocketHandler(hub, &config.Config{})

	router := gin.New()
	router.GET("/api/ws", func(c *gin.Context) {
		c.Set("user_id", int64(7))
		handler.connect(c)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	// Registration completes on the server goroutine after the handshake returns,
	// so resend until the first frame lands. Events are droppable by design.
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				hub.Send(7, ws.Event{Type: ws.TypeReportCompleted, Payload: ws.ReportPayload{ReportID: 42}})
			}
		}
	}()

	var got ws.Event
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		t.Fatalf("read: %v", err)
	}

	if got.Type != ws.TypeReportCompleted {
		t.Fatalf("event type = %q, want %q", got.Type, ws.TypeReportCompleted)
	}
	payload, ok := got.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload = %T, want object", got.Payload)
	}
	if id, _ := payload["report_id"].(float64); id != 42 {
		t.Fatalf("report_id = %v, want 42", payload["report_id"])
	}
}

func TestWebsocketHandler_RejectsForeignOriginInRelease(t *testing.T) {
	cfg := &config.Config{Release: true}
	cfg.CORS.AllowedOrigins = []string{"https://app.example.com"}

	opts := acceptOptions(cfg)
	if opts.InsecureSkipVerify {
		t.Fatal("origin verification disabled in release")
	}
	if len(opts.OriginPatterns) != 1 || opts.OriginPatterns[0] != "app.example.com" {
		t.Fatalf("origin patterns = %v, want [app.example.com]", opts.OriginPatterns)
	}
}
