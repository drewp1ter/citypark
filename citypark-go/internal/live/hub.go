// Package live streams order events to admins over WebSocket.
package live

import (
	"context"
	"log/slog"
	"sync"

	"github.com/drewp1ter/citypark-go/internal/order"
)

// subscriberBuffer is how many events a subscriber may fall behind before it is dropped.
const subscriberBuffer = 16

// Event is one message sent to a client.
type Event struct {
	Type  string       `json:"type"` // "hello" | "order.pending" | "order.created" | "order.paid" | "order.confirmed"
	Order *order.Order `json:"order,omitempty"`
}

// Hub fans order events out to subscribers. It lives in memory, so it serves one server instance.
type Hub struct {
	mu     sync.Mutex
	subs   map[chan Event]struct{}
	closed bool
	log    *slog.Logger
}

var _ order.Notifier = (*Hub)(nil)

func NewHub(log *slog.Logger) *Hub {
	return &Hub{subs: make(map[chan Event]struct{}), log: log}
}

// Subscribe registers a subscriber. The channel is closed when unsubscribe is called, when the subscriber is too slow,
// or on Close. unsubscribe is safe to call more than once.
func (h *Hub) Subscribe() (events <-chan Event, unsubscribe func()) {
	ch := make(chan Event, subscriberBuffer)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		close(ch)
		return ch, func() {}
	}
	h.subs[ch] = struct{}{}
	return ch, func() { h.remove(ch) }
}

// remove closes ch unless the hub already did.
func (h *Hub) remove(ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
}

// Close closes every subscriber. Later Subscribe calls get a closed channel.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for ch := range h.subs {
		delete(h.subs, ch)
		close(ch)
	}
}

// publish never blocks: a subscriber with a full buffer is dropped, and its client reconnects.
func (h *Hub) publish(typ string, o order.Order) {
	e := Event{Type: typ, Order: &o}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- e:
		default:
			delete(h.subs, ch)
			close(ch)
			h.log.Warn("live subscriber dropped", "event", typ, "order_id", o.ID)
		}
	}
}

func (h *Hub) OrderPending(_ context.Context, o order.Order)   { h.publish("order.pending", o) }
func (h *Hub) OrderCreated(_ context.Context, o order.Order)   { h.publish("order.created", o) }
func (h *Hub) OrderPaid(_ context.Context, o order.Order)      { h.publish("order.paid", o) }
func (h *Hub) OrderConfirmed(_ context.Context, o order.Order) { h.publish("order.confirmed", o) }

// subscribers is the number of registered subscribers. Used in tests.
func (h *Hub) subscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
