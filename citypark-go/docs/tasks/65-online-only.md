# Task 65: Online payment only

## Goal
Orders are paid only online through MONETA.RU. Cash is removed everywhere: the `paymentType` request field and order
field, the `payment_type` column, the cash branch of `Create`, the `OrderCreated` notifier method and its
`order.created` live event, the `paymentType` search filter and `THANK_YOU_PAGE_PAYMENT_CASH`.

## Depends on
64.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/service.go` (`Notifier`, `CreateInput`, `Create`),
`internal/order/store.go` (`Insert`, `Search`, `FindOpenDuplicate`), `internal/app/iiko_orders.go` (`Notifiers`).

## Background
Decided by the owner (2026-10-28): the restaurant accepts only online payment. The legacy data migration (task 35)
was skipped, so the only cash orders are in dev and seed data; dropping the column loses nothing real. The website
already sends `paymentType: "online"`; the JSON decoder ignores unknown fields, so it keeps working after this task
(citypark-modern task 31 stops sending it).

## Files to create
- `migrations/00018_drop_payment_type.sql`

## Files to change
- `internal/order/types.go`: remove `PaymentCash`, `PaymentOnline` and `Order.PaymentType`.
- `internal/order/store.go`: remove `payment_type` from the select list, `orderRow`, `InsertParams`, the `INSERT`,
  `SearchFilter.PaymentType` and its `WHERE`; remove `o.payment_type = 'online' AND` from both duplicate queries.
- `internal/order/service.go`: Spec 2.
- `internal/order/handler.go`: remove the `paymentType` query parameter of `GET /orders` (lines ~168–173).
- `internal/app/notifier.go`, `internal/app/iiko_orders.go`, `internal/live/hub.go`: remove `OrderCreated`.
  `TelegramNotifier`'s comment on `OrderPending` stays; `Event.Type` comment lists four values.
- `internal/telegram/message.go`: remove `OrderMessage.PaymentType`, the `Оплата` line and `paymentText`;
  `internal/app/notifier.go` `orderMessage` stops setting it.
- `internal/config/config.go`: remove `ThankYouPageCash`; `internal/app/app.go`: `order.Options{}` → see Spec 2.
- `internal/seed/seed.go`: seed orders without `PaymentType`, status `paid` (Spec 3).
- `.env.example`, `.env.development` (if it has the line), `README.md` (env table): remove
  `THANK_YOU_PAGE_PAYMENT_CASH`. Root `../CLAUDE.md`: the sentence "(the backend expects the frontend at `:3001`, see
  `THANK_YOU_PAGE_PAYMENT_CASH`)" → "see `MONETA_SUCCESS_URL`".
- `docs/design/schema.sql`: remove the `payment_type` line.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: Spec 4.
- Tests: see "Tests to write".

## Spec

### 1. Migration `00018_drop_payment_type.sql`
```sql
-- +goose Up
ALTER TABLE orders DROP COLUMN payment_type;

-- +goose Down
ALTER TABLE orders ADD COLUMN payment_type text NOT NULL DEFAULT 'online' CHECK (payment_type IN ('cash', 'online'));
```

### 2. `order.Service`
```go
type Notifier interface {
	OrderPending(ctx context.Context, o Order)   // an order was created and waits for payment
	OrderPaid(ctx context.Context, o Order)      // an order was paid
	OrderConfirmed(ctx context.Context, o Order) // an admin confirmed an order
}

// CreateResult: RedirectURL is the Moneta payment page.
```
- `CreateInput` loses `PaymentType`. A request that still sends `"paymentType"` is accepted (the field is ignored).
- `Options.ThankYouCash` is removed. If `Options` is then empty, remove the type and the `opts` parameter of
  `NewService`, and update every caller (`app.go`, tests).
- `Create`: the duplicate check (`FindOpenDuplicate`) runs for every order (it was online-only). Remove the cash
  branch; the rest (Moneta form, `SetStatus(..., StatusPending)`, `OrderPending` for a newly saved order) stays as is.
- `saveParams.paymentType` is removed.

### 3. Seed
The two seed orders become paid online orders: after `orders.Insert`, set status `paid` and `payment_id`
(`seed-1`, `seed-2`) with the existing `Store.MarkPaid`, so they show on the admin's lists as real paid orders.

### 4. Docs
- `POST /orders`: remove `paymentType` from the request table and examples; one response (201
  `{orderId, redirectUrl}`, `redirectUrl` is the Moneta page). Remove "for a cash order" wording.
- `GET /orders`, `GET /orders/{id}`, `GET /orders/my` and all order examples: no `paymentType`. `GET /orders` loses
  the `paymentType` query parameter.
- `GET /orders/live`: remove `order.created` from the table, the examples and the `LiveOrderEvent.type` enum.
- `openapi.yaml`: remove `paymentType` from every schema and example, and from `required` lists.
- `docs/design/api.md` is a historical design note: leave it.

## Tests to write
Change the existing tests; do not leave a test that only checks cash.
- `internal/order/service_test.go`: `input(withAddress bool)` without a payment type. Delete `TestCreateCash`,
  `TestCreateOnlineNotReusedForCash` and the cash half of `TestCreateResultJSON` (the result keys are exactly
  `orderId, redirectUrl`). Every other test that used `PaymentCash` uses the plain input; where it checked
  `RedirectURL == cashURL`, check the fake payments URL; where it expected status `new`, expect `pending`.
  Remove `cashURL`. Add `TestCreateNotifiesPending`: a new order → `pending` has one order with status `pending`.
- `internal/order/create_handler_test.go`: `orderBody(e, withAddress)`; the 201 test checks the Moneta redirect.
  Add `TestCreateIgnoresPaymentType`: a body with `"paymentType": "cash"` → 201 and an online redirect.
- `internal/order/store_test.go`, `store_payment_test.go`, `read_handler_test.go`, `admin_handler_test.go`,
  `search_test.go`: drop `PaymentType` from fixtures; delete the cash row of the duplicate table test and the
  `paymentType=cash` filter case.
- `internal/app/app_test.go`: `testOrder()` without a payment type; delete `TestNotifierCash`; tests that called
  `OrderCreated` call `OrderPaid` and expect `chatConfirmed`; the app flow tests post orders without `paymentType`
  and expect the Moneta redirect.
- `internal/live/live_test.go`: use `OrderPending` / `order.pending` instead of `OrderCreated` / `order.created`.
- `internal/telegram/telegram_test.go`: the expected text has no `Оплата:` line.
- `internal/config/config_test.go`: remove the `ThankYouPageCash` rows and env value.
- `internal/validate/validate_test.go` uses `oneof=cash online` only as a generic example: leave it.

## API documentation
`POST /orders`, `GET /orders`, `GET /orders/{id}`, `GET /orders/my`, `GET /orders/live` (Spec 4). Note for clients:
`paymentType` is gone from responses; sending it in `POST /orders` is ignored.

## Done when
CONVENTIONS "Done means" passes (`testutil.NewDB` applies the new migration to the test database), and
`grep -rni cash internal migrations/00018_drop_payment_type.sql docs/API.md .env.example README.md` finds only
`internal/validate/validate_test.go` and the migration's `Down` part.

## Out of scope
- The admin panel (citypark-admin task 23) and the website (citypark-modern task 31).
- iiko's `IIKO_PAYMENT_TYPE_ID` / `IIKO_PAYMENT_TYPE_KIND`: they describe how the paid order is booked in iiko; keep them.
- Renaming `chatCreated` / `TELEGRAM_CHAT_ID_CREATED` (it still receives the customer's phone).
- Old task files in `docs/tasks/` and `docs/design/`: they are history, do not edit them.
