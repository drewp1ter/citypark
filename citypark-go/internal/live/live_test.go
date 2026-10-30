package live

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/order"
)

func discardLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func recv(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e, ok := <-ch:
		if !ok {
			t.Fatal("channel closed, want an event")
		}
		return e
	case <-time.After(time.Second):
		t.Fatal("no event within 1s")
	}
	return Event{}
}

func TestHubFanOut(t *testing.T) {
	h := NewHub(discardLog())
	a, _ := h.Subscribe()
	b, _ := h.Subscribe()

	h.OrderPending(context.Background(), order.Order{ID: 7})

	for _, ch := range []<-chan Event{a, b} {
		e := recv(t, ch)
		if e.Type != "order.pending" || e.Order == nil || e.Order.ID != 7 {
			t.Errorf("event = %+v, want order.pending for order 7", e)
		}
	}
}

func TestHubEventTypes(t *testing.T) {
	h := NewHub(discardLog())
	ch, _ := h.Subscribe()
	ctx, o := context.Background(), order.Order{ID: 1}

	h.OrderPending(ctx, o)
	h.OrderPaid(ctx, o)

	for _, want := range []string{"order.pending", "order.paid"} {
		if got := recv(t, ch).Type; got != want {
			t.Errorf("type = %q, want %q", got, want)
		}
	}
}

func TestHubUnsubscribe(t *testing.T) {
	h := NewHub(discardLog())
	ch, unsubscribe := h.Subscribe()
	unsubscribe()
	unsubscribe() // must not panic

	h.OrderPaid(context.Background(), order.Order{ID: 1})

	if e, ok := <-ch; ok {
		t.Errorf("got %+v after unsubscribe, want a closed channel", e)
	}
	if n := h.subscribers(); n != 0 {
		t.Errorf("subscribers = %d, want 0", n)
	}
}

func TestHubSlowSubscriberDropped(t *testing.T) {
	h := NewHub(discardLog())
	slow, unsubscribeSlow := h.Subscribe()
	fast, _ := h.Subscribe()

	got := make(chan int)
	go func() {
		n := 0
		for range fast {
			n++
			if n == 17 {
				break
			}
		}
		got <- n
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 17; i++ {
			h.OrderPending(context.Background(), order.Order{ID: int64(i)})
			// Give the fast reader time so its own buffer never overflows.
			time.Sleep(time.Millisecond)
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}

	if n := <-got; n != 17 {
		t.Errorf("fast subscriber got %d events, want 17", n)
	}
	n := 0
	for range slow {
		n++
	}
	if n != subscriberBuffer {
		t.Errorf("slow subscriber holds %d events, want %d", n, subscriberBuffer)
	}
	unsubscribeSlow() // already dropped; must not panic
}

func TestHubClose(t *testing.T) {
	h := NewHub(discardLog())
	a, _ := h.Subscribe()
	b, _ := h.Subscribe()

	h.Close()

	for _, ch := range []<-chan Event{a, b} {
		if _, ok := <-ch; ok {
			t.Error("channel still open after Close")
		}
	}
	late, unsubscribe := h.Subscribe()
	unsubscribe()
	if _, ok := <-late; ok {
		t.Error("Subscribe after Close returned an open channel")
	}
}

type liveEnv struct {
	hub    *Hub
	tokens *auth.Tokens
	url    string // ws://host/orders/live
}

func newLiveEnv(t *testing.T) *liveEnv {
	t.Helper()
	e := &liveEnv{
		hub:    NewHub(discardLog()),
		tokens: auth.NewTokens("secret", time.Hour, clock.Fixed(time.Now())),
	}
	r := chi.NewRouter()
	NewHandler(e.hub, e.tokens, discardLog()).Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	e.url = "ws" + strings.TrimPrefix(srv.URL, "http") + "/orders/live"
	return e
}

func (e *liveEnv) token(t *testing.T, role string) string {
	t.Helper()
	tok, err := e.tokens.Issue(1, role)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return tok
}

func (e *liveEnv) dial(t *testing.T, protocols ...string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	return websocket.Dial(ctx, e.url, &websocket.DialOptions{Subprotocols: protocols})
}

func readText(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("message type = %v, want text", typ)
	}
	return strings.TrimSpace(string(data)) // wsjson ends a message with a newline
}

func (e *liveEnv) waitSubscribers(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if e.hub.subscribers() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("subscribers = %d, want %d", e.hub.subscribers(), want)
}

func TestLiveRejectsBadAuth(t *testing.T) {
	e := newLiveEnv(t)
	tests := []struct {
		name      string
		protocols []string
		status    int
	}{
		{"no subprotocol", nil, http.StatusUnauthorized},
		{"garbage token", []string{"bearer", "garbage"}, http.StatusUnauthorized},
		{"only bearer", []string{"bearer"}, http.StatusUnauthorized},
		{"wrong scheme", []string{"basic", e.token(t, auth.RoleAdmin)}, http.StatusUnauthorized},
		{"user role", []string{"bearer", e.token(t, auth.RoleUser)}, http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, resp, err := e.dial(t, tc.protocols...)
			if err == nil {
				_ = conn.CloseNow()
				t.Fatal("dial succeeded, want a refused handshake")
			}
			if resp == nil || resp.StatusCode != tc.status {
				t.Fatalf("response = %+v, want status %d", resp, tc.status)
			}
		})
	}
	if n := e.hub.subscribers(); n != 0 {
		t.Errorf("subscribers = %d, want 0", n)
	}
}

func TestLiveStreamsEvents(t *testing.T) {
	e := newLiveEnv(t)
	conn, _, err := e.dial(t, "bearer", e.token(t, auth.RoleAdmin))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	if got := conn.Subprotocol(); got != "bearer" {
		t.Errorf("Subprotocol = %q, want bearer", got)
	}
	if got := readText(t, conn); got != `{"type":"hello"}` {
		t.Errorf("first message = %s, want the hello", got)
	}

	e.hub.OrderPaid(context.Background(), order.Order{ID: 42, Status: order.StatusPaid})

	var msg struct {
		Type  string `json:"type"`
		Order struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
	}
	if err := json.Unmarshal([]byte(readText(t, conn)), &msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.Type != "order.paid" || msg.Order.ID != 42 || msg.Order.Status != order.StatusPaid {
		t.Errorf("message = %+v, want order.paid for order 42", msg)
	}
}

func TestLiveUnsubscribesOnClientClose(t *testing.T) {
	e := newLiveEnv(t)
	conn, _, err := e.dial(t, "bearer", e.token(t, auth.RoleAdmin))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	readText(t, conn) // hello
	e.waitSubscribers(t, 1)

	if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Logf("Close: %v", err)
	}
	e.waitSubscribers(t, 0)
}

func TestLiveHubCloseTellsClientToReconnect(t *testing.T) {
	e := newLiveEnv(t)
	conn, _, err := e.dial(t, "bearer", e.token(t, auth.RoleAdmin))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()
	readText(t, conn) // hello

	e.hub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, err = conn.Read(ctx)
	if got := websocket.CloseStatus(err); got != websocket.StatusTryAgainLater {
		t.Errorf("close status = %d (err %v), want %d", got, err, websocket.StatusTryAgainLater)
	}
}

func TestTokenParsing(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    string
		ok      bool
	}{
		{"one header", []string{"bearer, abc"}, "abc", true},
		{"no space", []string{"bearer,abc"}, "abc", true},
		{"two headers", []string{"bearer", "abc"}, "abc", true},
		{"missing", nil, "", false},
		{"three values", []string{"bearer, abc, def"}, "", false},
		{"empty token", []string{"bearer, "}, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/orders/live", nil)
			for _, h := range tc.headers {
				r.Header.Add("Sec-WebSocket-Protocol", h)
			}
			got, ok := token(r)
			if got != tc.want || ok != tc.ok {
				t.Errorf("token = %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
