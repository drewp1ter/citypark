# Task 31: Application wiring

## Goal
Connect everything: adapters between `order` and the Telegram client / Moneta signer, one function that builds the full router, and the real `main`.

## Depends on
02, 04, 14, 16, 18, 19, 22, 23, 24, 27, 29, 30.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/config/config.go`, `internal/order/service.go`, `cmd/server/main.go`.
For every other package you only need its constructor; find it with `grep -rn "^func New" internal/`.

## Files to create
- `internal/app/notifier.go`
- `internal/app/payments.go`
- `internal/app/app.go`
- `internal/app/app_test.go`

## Files to change
- `cmd/server/main.go` (rewrite)

## Spec

### `notifier.go` — implements `order.Notifier`
```go
type MessageSender interface {
    SendMessage(ctx context.Context, chatID, text string) error
}

type TelegramNotifier struct { /* sender MessageSender, chatCreated, chatConfirmed string, log *slog.Logger, wg sync.WaitGroup */ }

func NewTelegramNotifier(sender MessageSender, chatCreated, chatConfirmed string, log *slog.Logger) *TelegramNotifier

// Wait blocks until all sends that were started have finished. Used on shutdown and in tests.
func (n *TelegramNotifier) Wait()
```

| Method | Message 1: `telegram.FormatOrder(...)` to | Message 2: the customer's phone to |
|---|---|---|
| `OrderCreated` | `chatCreated` | `chatCreated` |
| `OrderPaid` | `chatConfirmed` | `chatCreated` |
| `OrderConfirmed` | `chatConfirmed` | `chatCreated` |

- `OrderCreated` does nothing unless `o.PaymentType == order.PaymentCash`.
- Each method returns at once and sends in a goroutine (`wg.Add(1)` before starting it), using `context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)`.
- A failed send is logged with `log.Error` (message `telegram send failed`, attributes `order_id`, `error`) and the second message is not sent. Nothing is returned to the caller.
- If the chat id for message 1 is empty, log a warning and send nothing.
- Convert `order.Order` to `telegram.OrderMessage`: `Address` is `city + ", " + address` or `""` for pickup; `Total` is `o.Total`, `Delivery` is `o.DeliveryPrice`; items map `CategoryName`, `Name`, `Amount`.

### `payments.go` — implements `order.Payments`
```go
type MonetaPayments struct { /* signer *moneta.Signer */ }
func NewMonetaPayments(signer *moneta.Signer) *MonetaPayments
```
- `Form`: `amount := moneta.FormatAmount(r.Amount)`; return `order.PaymentForm{Amount: amount, Signature: signer.RequestSignature(r.OrderID, amount)}`.
- `Verify`: `signer.VerifyCallback(c.TransactionID, c.OperationID, c.Amount, c.Signature)`.

### `app.go`
```go
type App struct {
    Handler  http.Handler
    Notifier *TelegramNotifier
}

// New builds all stores, services and handlers and returns the router.
func New(cfg config.Config, db *sqlx.DB, log *slog.Logger, c clock.Clock) *App
```
Router:
```go
r := chi.NewRouter()
r.Use(logging.Recover(log), logging.Requests(log), cors)
r.Route("/api/v1", func(r chi.Router) { /* every package's Routes(r) */ })
```
- `cors`: a small middleware in this file. It sets `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Headers: Authorization, Content-Type, X-Update-Token`, `Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`, and answers `OPTIONS` requests with 204.
- Mount: `health`, `user.AuthHandler`, `user.Handler`, `address`, `deliveryprice`, `catalog`, `iiko`, `order`.
- Build the clients from `cfg`: `iiko.NewClient(cfg.Iiko.BaseURL, ...)`, `telegram.NewClient(cfg.Telegram.BaseURL, cfg.Telegram.Token, nil)`.
- `moneta.New(cfg.Moneta.ID, cfg.Moneta.Secret, cfg.Moneta.TestMode)`, wrapped in `NewMonetaPayments`.
- If `cfg.Moneta.ID` or `cfg.Moneta.Secret` is empty, log one warning (`moneta is not configured`): online payments cannot work.
- `auth.NewTokens(cfg.JWT.Secret, cfg.JWT.ExpiresIn, c)`.
- `order.NewService(db, c, notifier, payments, order.Options{ThankYouCash: cfg.ThankYouPageCash})`.
- `order.NewHandler(svc, tokens, order.HandlerOptions{cfg.WorkingHours.Start, cfg.WorkingHours.End, cfg.DefaultPageSize})`.

### `cmd/server/main.go`
1. `config.Load(os.LookupEnv)`; on error print it and exit 1.
2. `log := logging.New(os.Stdout, cfg.Env, cfg.Debug)`; `slog.SetDefault(log)`.
3. `store.Open(ctx, cfg.DatabaseURL)`; `store.Migrate`.
4. `user.EnsureAdmin` with `cfg.Admin`; log when it created the account.
5. `app.New(cfg, db, log, clock.System(cfg.Timezone))`.
6. `http.Server{Addr: ":<port>", Handler: a.Handler, ReadHeaderTimeout: 10s}`.
7. Graceful shutdown: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`; on signal call `srv.Shutdown` with a 20 second timeout, then `a.Notifier.Wait()`, then `db.Close()`.

Put the steps in `func run() error` and keep `main` to `if err := run(); err != nil { ...; os.Exit(1) }`.

## Tests to write
`app_test.go`, with `db := testutil.NewDB(t)` where a database is needed.
- `TestNotifierCash`: a fake `MessageSender` that records calls; `OrderCreated` with a cash order, then `Wait()` → two sends: the formatted text to `chatCreated`, then the phone to `chatCreated`.
- `TestNotifierCreatedOnlineIgnored`: online order → no sends.
- `TestNotifierPaid` and `TestNotifierConfirmed`: text to `chatConfirmed`, phone to `chatCreated`.
- `TestNotifierSendError`: the sender fails → one attempt, no panic, an error line in the log buffer.
- `TestPaymentsAdapter`: with `moneta.New("100", "secret", false)`: `Form` for order 15 and 79000 kopecks → `{Amount: "790.00", Signature: "8c65fcd4afe3dc4cb915e04aaa273574"}`; `Verify` is true for `{"15", "555", "790.00", "998b6268fe3a6c7f4eeb8ccec65a9566"}` and false when the amount is `790.01`.
- `TestAppSmoke`: `app.New` with a config from `config.Load` on an empty map, pointing the two base URLs (iiko, Telegram) at one `httptest.NewServer`:
  - `GET /api/v1/health` → 200;
  - `GET /api/v1/categories` → 200 `[]`;
  - `GET /api/v1/user` → 401;
  - `OPTIONS /api/v1/orders` → 204 with the CORS headers;
  - `GET /nope` → 404.
- `TestAppOrderFlow`: seed a category, a product and a delivery price; set working hours to empty in the config; `POST /api/v1/orders` (cash) → 201; after `a.Notifier.Wait()` the fake Telegram server received two `sendMessage` requests.
- `TestAppOnlineFlow`: config with `MONETA_ID=100`, `MONETA_SECRET=secret`, empty working hours; seed a product at 79000 kopecks; `POST /api/v1/orders` (online, no address) → 201 with `amount` `790.00`; compute the callback signature in the test as the MD5 of `"100" + orderId + "555" + "790.00" + "RUB" + "0" + "secret"`; `POST /api/v1/webhooks/moneta` with the form fields → 200 `SUCCESS`; `GET` of the order from the database shows status `paid`; after `Wait()` the fake Telegram server received two `sendMessage` requests. The same callback with a wrong signature → 403 `FAIL`.

## Done when
The commands in CONVENTIONS "Done means" pass, and `go run ./cmd/server` starts against `make db-up` and answers `curl localhost:3000/api/v1/health`.

## Out of scope
Seed data (task 32), Docker (task 33).
