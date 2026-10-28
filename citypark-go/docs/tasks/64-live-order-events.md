# Task 64: Live order events over WebSocket

## Goal
An admin connects to `GET /orders/live` with a WebSocket and gets every order event as JSON as it happens: an online
order created (not paid yet), a cash order created, an online order paid, an order confirmed. The admin's live orders
page (citypark-admin task 22) is built on it.

## Depends on
29, 63.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/service.go` (`Notifier`, `Create`), `internal/app/iiko_orders.go`
(`Notifiers`), `internal/auth/middleware.go` (`RequireRole`, `Allows`).

## Background
Decided by the owner (2026-10-28):
- Browsers cannot set `Authorization` on a WebSocket, so the JWT travels in the subprotocol header:
  the browser calls `new WebSocket(url, ['bearer', token])`, which sends `Sec-WebSocket-Protocol: bearer, <jwt>`.
  The server must answer with the subprotocol `bearer`, or the browser closes the connection. The token never goes
  into a URL, so it is not in access logs.
- Unpaid online orders appear too: a new `OrderPending` event. Telegram and iiko ignore it.
- No replay: a client that reconnects reloads the list with `GET /orders`. Events are not stored.
- The origin check is off (`InsecureSkipVerify`): the token is a header, not a cookie, so another site cannot ride an
  admin's session (no cross-site WebSocket hijacking). CORS is already `*`.

## Files to create
- `internal/live/hub.go`, `internal/live/handler.go`, `internal/live/live_test.go`

## Files to change
- `go.mod`, `go.sum`: add `github.com/coder/websocket` (latest v1).
- `docs/tasks/CONVENTIONS.md`: add `github.com/coder/websocket` to "Allowed dependencies".
- `internal/order/service.go`: `OrderPending` in `Notifier`; call it in `Create` (Spec 1).
- `internal/order/service_test.go`: `fakeNotifier` gets `pending []Order` and the method; tests from "Tests to write".
- `internal/app/notifier.go` (`TelegramNotifier`), `internal/app/iiko_orders.go` (`IikoOrders`, `Notifiers`),
  `internal/app/iiko_orders_test.go` (`recordingNotifier`): add `OrderPending`.
- `internal/logging/middleware.go`: `statusWriter` gets `Unwrap() http.ResponseWriter` (returns the wrapped writer),
  so `http.ResponseController` and the WebSocket library can hijack the connection through the logging middleware.
  Add a test that a request through `Requests` can be hijacked (`http.NewResponseController(w).Hijack()` returns no
  error under `httptest.NewServer`).
- `internal/app/app.go`: `hub := live.NewHub(log)`; `Notifiers{notifier, iikoOrders, hub}`;
  `live.NewHandler(hub, tokens, log).Routes(r)`; `App` gets `Hub *live.Hub`.
- `cmd/server/main.go`: call `a.Hub.Close()` right **before** `srv.Shutdown` (Shutdown does not close hijacked
  connections and would wait for them until the timeout).
- `docs/API.md`, `internal/apidocs/openapi.yaml` (Spec 4).

## Spec

### 1. `order.Notifier`
```go
type Notifier interface {
	OrderPending(ctx context.Context, o Order)   // an online order was created and waits for payment
	OrderCreated(ctx context.Context, o Order)   // a cash order was placed
	OrderPaid(ctx context.Context, o Order)      // an online order was paid
	OrderConfirmed(ctx context.Context, o Order) // an admin confirmed an order
}
```
In `Create`, online branch: after `SetStatus(..., StatusPending)` succeeds, if the order was **newly saved** (not
`found`), set `o.Status = StatusPending` and call `s.notifier.OrderPending(ctx, o)`. A reused duplicate does not fire.
`TelegramNotifier` and `IikoOrders` implement it as a no-op; `Notifiers` forwards it like the others.

### 2. `internal/live/hub.go`
```go
type Event struct {
	Type  string       `json:"type"` // "hello" | "order.pending" | "order.created" | "order.paid" | "order.confirmed"
	Order *order.Order `json:"order,omitempty"`
}

type Hub struct{ /* mu sync.Mutex; subs map[chan Event]struct{}; closed bool; log *slog.Logger */ }

func NewHub(log *slog.Logger) *Hub
// Subscribe registers a subscriber. The channel is closed when unsubscribe is called, when the subscriber is too slow,
// or on Close. unsubscribe is safe to call more than once.
func (h *Hub) Subscribe() (events <-chan Event, unsubscribe func())
func (h *Hub) Close() // closes every subscriber; later Subscribe returns an already closed channel
// order.Notifier: each method publishes Event{Type: "order.<x>", Order: &o}.
```
- Each subscriber channel has a buffer of 16. Publishing never blocks: if a subscriber's buffer is full, it is
  removed and its channel closed, and the hub logs `live subscriber dropped` at Warn.
- `var _ order.Notifier = (*Hub)(nil)`.

### 3. `internal/live/handler.go`
```go
func NewHandler(hub *Hub, tokens *auth.Tokens, log *slog.Logger) *Handler
func (h *Handler) Routes(r chi.Router) // GET /orders/live
```
| Method | Path | Role | Behaviour |
|---|---|---|---|
| GET | `/orders/live` | admin (own check) | WebSocket; events as text JSON messages |

`GET /orders/live` does not use `RequireRole` (it reads the header itself) and is a plain `http.HandlerFunc`, not
`httpx.Handle`, because after the upgrade there is no HTTP response to write. Steps:
1. Split `r.Header.Values("Sec-WebSocket-Protocol")` on commas, trim spaces. It must be exactly two values,
   `bearer` and a token; else `httpx.WriteError` 401 `Требуется авторизация.` (use the same message and helper
   `RequireRole` uses for a missing token).
2. `tokens.Parse(token)`; an error → 401 with the same message as `RequireRole` for a bad token. Role not allowed
   for `admin` (`auth.Allows`) → 403 with `RequireRole`'s message.
3. `websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"bearer"}, InsecureSkipVerify: true})`.
   On error, log and return (Accept has already written the response).
4. `events, unsubscribe := hub.Subscribe()`; `defer unsubscribe()`. Use `ctx := conn.CloseRead(r.Context())`
   (the client sends nothing; this handles its close frames).
5. Write `{"type":"hello"}` (`wsjson.Write` with a 5 s timeout context).
6. Loop: on an event write it (5 s timeout); on a 30 s ticker `conn.Ping` (10 s timeout); on `ctx.Done()` return.
   Any write or ping error → return. A closed `events` channel → `conn.Close(websocket.StatusTryAgainLater,
   "reconnect")` and return. On a normal return `conn.Close(websocket.StatusNormalClosure, "")`.

Chi matches the static `/orders/live` before `/orders/{id}`, so `GET /orders/7` still goes to the order handler.

### 4. Docs
- `docs/API.md`, under Orders, a section `#### GET /orders/live (WebSocket)`: role admin; the handshake
  (`new WebSocket('ws://host/api/v1/orders/live', ['bearer', token])`); 401/403 before the upgrade with the exact
  messages; the message list with one example each, copied from the test output; that `order` has the same shape as
  `GET /orders/{id}`; pings every 30 s; close code 1013 means reconnect; there is no replay, so a client reloads the
  list after each `hello`; a browser cannot read the 401 of a failed handshake, so a client should make a normal
  authorised request after a failed connect to find out whether the token expired. Proxies must pass the `Upgrade`
  and `Connection` headers and keep idle connections for more than 30 s.
- `internal/apidocs/openapi.yaml`: path `/orders/live` `get` with `security` bearer, a `101` response
  (`description: WebSocket upgrade; messages are LiveOrderEvent`), `401`, `403`; component schema `LiveOrderEvent`
  (`type` enum of the five values, required; `order` → the existing order schema, optional). Reference
  `LiveOrderEvent` from the 101 response `content: application/json` so the admin's `api:gen` produces the type.

## Tests to write
In `internal/live/live_test.go` (no database needed for hub tests):
- `TestHubFanOut`: two subscribers; `OrderCreated` → both get `order.created` with the order id.
- `TestHubUnsubscribe`: after `unsubscribe` the channel is closed and gets nothing; calling it twice does not panic.
- `TestHubSlowSubscriberDropped`: one subscriber reads nothing; publish 17 events → the call returns (no block), its
  channel holds 16 events and is then closed; a second, reading subscriber got all 17.
- `TestHubClose`: `Close` closes all channels; `Subscribe` afterwards returns a closed channel.
- Handler, through `httptest.NewServer(chi router with Routes)` and `websocket.Dial(ctx, "ws://...",
  &websocket.DialOptions{Subprotocols: []string{"bearer", token}})`; tokens from `auth.NewTokens` as other tests do:
  - no subprotocol → dial fails, response status 401; `bearer, garbage` → 401; a `user` token → 403.
  - admin token → `conn.Subprotocol() == "bearer"`; first message `{"type":"hello"}`; then `hub.OrderPaid(o)` →
    `{"type":"order.paid","order":{"id":...}}`.
  - after the client closes, the hub has no subscribers (poll a test-only count, or `Subscribe`/publish and check).
  - `hub.Close()` while connected → the client's read returns close status 1013.
- `internal/order/service_test.go`:
  - `TestCreateOnlinePendingEvent`: online create → `pending` has one order, its `Status` is `pending`; `created` empty.
  - `TestCreateOnlineDuplicateNoEvent`: the same online input twice → `pending` still has one order.
  - `TestCreateCash`: also assert `pending` is empty.
- A route test in the app or order package: with both handlers mounted, `GET /orders/7` with an admin token still
  returns the order (or 404 for a missing one), not a WebSocket error.

## Done when
CONVENTIONS "Done means" passes, with coverage for `internal/live`, `internal/order`, `internal/logging`.
With the dev server running and `TOKEN` from `POST /auth/sign-in` as the dev admin:
`websocat -H "Sec-WebSocket-Protocol: bearer, $TOKEN" ws://localhost:3000/api/v1/orders/live` prints
`{"type":"hello"}`; a cash `POST /orders` in another terminal prints an `order.created` line (if `websocat` is not
installed, skip this check and say so in the commit message).

## Out of scope
- The admin page (citypark-admin task 22).
- Storing or replaying events, events for item changes or deletion, events for customers (role `user`).
- Several server instances (the hub is in memory; one instance is deployed).
