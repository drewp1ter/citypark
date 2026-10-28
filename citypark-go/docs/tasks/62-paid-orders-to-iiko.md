# Task 62: Paid online orders go to iiko

## Goal
When Moneta confirms the payment of an online order, the order is created in iiko in the background (task 60's
`CreateDelivery`) and iiko's order id is saved on it. If iiko does not create it, staff get a Telegram alert and enter
it by hand. Cash orders are not sent.

## Depends on
30, 60, 61.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/app/notifier.go`, `internal/app/app.go`, `internal/order/store.go`
(`selectOrders`, `orderRow`, `loadItems`, `SetStatus`).

## Background
The owner's decisions (2026-10-28):
- Our address is free text, but iiko's legacy address needs a separate street and house. So the iiko address is a fixed
  placeholder from config (the restaurant: Белореченск, Гоголя, 61), as the old site did. The real address goes into the
  comment and `deliveryPoint.comment`.
- The delivery fee is written only in the comment. The external payment is the items total (`o.Total`).
- The order id is a GUID made **once** per order, so a retry can never create a second iiko order.

Tasks 59 and 61 are done, so `Order` has `DeliveryPrice` and `DeliveryTime`.

## Files to create
- `migrations/00016_order_iiko.sql`
- `internal/app/iiko_orders.go`
- `internal/app/iiko_orders_test.go`

## Files to change
- `internal/order/types.go`, `internal/order/store.go`, `internal/order/store_test.go`
- `internal/config/config.go`, `internal/config/config_test.go`
- `internal/app/app.go`, `internal/app/app_test.go` (if it builds `App`), `cmd/server/main.go`
- `.env.example` (under `# iiko`), `README.md` (env table and a short «Orders in iiko» paragraph)

## Spec

### Migration
```sql
-- +goose Up
ALTER TABLE orders ADD COLUMN iiko_order_id text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE orders DROP COLUMN iiko_order_id;
```

### order
- `Item.IikoID string \`db:"iiko_id" json:"-"\``. `loadItems` selects `COALESCE(p.iiko_id, '') AS iiko_id`.
- `Order.IikoOrderID string \`json:"-"\``. `selectOrders` and `orderRow` read `o.iiko_order_id`.
- ```go
  // SetIikoOrderID saves the id of the order in iiko. It returns store.ErrNotFound when the order does not exist.
  func (s *Store) SetIikoOrderID(ctx context.Context, id int64, iikoID string) error
  ```
  `UPDATE orders SET iiko_order_id = $2, updated_at = now() WHERE id = $1`. No rows affected means `store.ErrNotFound`.

### config
`Iiko` gets these fields (env variable and default):

| Field | Env | Default |
|---|---|---|
| `TerminalGroupID` | `IIKO_TERMINAL_GROUP_ID` | `""` |
| `PaymentTypeID` | `IIKO_PAYMENT_TYPE_ID` | `""` |
| `PaymentTypeKind` | `IIKO_PAYMENT_TYPE_KIND` | `Card` |
| `AddressCity` | `IIKO_ADDRESS_CITY` | `Белореченск` |
| `AddressStreet` | `IIKO_ADDRESS_STREET` | `Гоголя` |
| `AddressHouse` | `IIKO_ADDRESS_HOUSE` | `61` |

README: the ids come from iiko with a token from `v2/access_token`: `POST /api/1/terminal_groups` and
`POST /api/1/payment_types` with `{"organizationIds": ["<IIKO_ORGANIZATION_ID>"]}`. Take the payment type the restaurant
uses for online payments. `IIKO_PAYMENT_TYPE_KIND` is its `paymentTypeKind`. Without both ids, orders are not sent.

### app (`iiko_orders.go`)
```go
// DeliveryCreator sends a delivery to iiko. *iiko.Client implements it.
type DeliveryCreator interface {
	CreateDelivery(ctx context.Context, terminalGroupID string, o iiko.DeliveryOrder) (iiko.DeliveryResult, error)
}

type IikoOrderSettings struct {
	TerminalGroupID, PaymentTypeID, PaymentTypeKind string
	City, Street, House                             string          // placeholder iiko address
	Delays                                          []time.Duration // waits before each retry
}

// IikoOrders implements order.Notifier: a paid order is created in iiko in a goroutine.
type IikoOrders struct { /* creator, db, alerts, alertChat, settings, log, wg */ }

var _ order.Notifier = (*IikoOrders)(nil)

func NewIikoOrders(c DeliveryCreator, db *sqlx.DB, alerts MessageSender, alertChat string, s IikoOrderSettings, log *slog.Logger) *IikoOrders

// Wait blocks until all started sends have finished. Used on shutdown and in tests.
func (n *IikoOrders) Wait()

func (n *IikoOrders) OrderCreated(ctx context.Context, o order.Order)   {} // cash orders are not sent
func (n *IikoOrders) OrderConfirmed(ctx context.Context, o order.Order) {}
func (n *IikoOrders) OrderPaid(ctx context.Context, o order.Order)

// Notifiers sends each event to every notifier, in order.
type Notifiers []order.Notifier // implements order.Notifier

// deliveryOrder builds the iiko order. id is the iiko order GUID.
func deliveryOrder(o order.Order, s IikoOrderSettings, id string) (iiko.DeliveryOrder, error)

// newUUID returns a random version 4 UUID made with crypto/rand.
func newUUID() string
```

`NewIikoOrders`: when `TerminalGroupID` or `PaymentTypeID` is empty, log Warn `iiko orders are not configured` once.
`OrderPaid` then does nothing.

`OrderPaid` starts a goroutine (`wg.Add` before it, `context.WithoutCancel(ctx)`) and returns at once. The goroutine:
1. `id := newUUID()` once.
2. `d, err := deliveryOrder(o, s, id)`. On error, alert (below) and stop.
3. Attempt `i` from 0 to `len(Delays)`. Before attempt `i > 0`, wait `Delays[i-1]`. Each attempt runs
   `CreateDelivery(ctx with a 2-minute timeout, s.TerminalGroupID, d)` with the same `d`.
   - On success: `order.NewStore(db).SetIikoOrderID(ctx, o.ID, res.ID)` (log Error if it fails), log Info
     `order created in iiko` with `order_id` and `iiko_id`, then stop.
   - `errors.Is(err, iiko.ErrDeliveryPending)`: send the pending alert and stop. Do not retry, because iiko may still
     create it.
   - `iiko.Permanent(err)`, or this was the last attempt: send the failure alert and stop.
   - Otherwise log Warn `iiko delivery failed, will retry` and go on.
4. Alerts go to `alertChat` through `alerts.SendMessage`. A failed send or an empty `alertChat` is only logged. Texts,
   verbatim:
   - failure: `Заказ №<id> не создан в iiko: <err>. Внесите его в iikoFront вручную.`
   - pending: `Заказ №<id> отправлен в iiko, но создание не подтверждено. Проверьте его в iikoFront.`

`deliveryOrder`:
- `ID: id`, `ExternalNumber: strconv.FormatInt(o.ID, 10)`, `Phone: o.Customer.Phone`,
  `Customer: {Type: "one-time", Name: o.Customer.Name}`.
- `o.Address == nil` (pickup): `OrderServiceType: iiko.ServiceDeliveryByClient`, `DeliveryPoint: nil`. Otherwise
  `iiko.ServiceDeliveryByCourier` and
  `&DeliveryPoint{Address: {Type: "legacy", Street: {Name: s.Street, City: s.City}, House: s.House}, Comment: "<city>, <address>"}`.
- `Comment`: these lines joined with `"\n"`, in this order, each only when it applies:
  1. `Заказ с сайта №<id>, оплачен онлайн` (always)
  2. `Ко времени: <o.DeliveryTime>`, when it is not empty
  3. `Адрес: <city>, <address>`, when there is an address
  4. `Доставка: <o.DeliveryPrice.String()> ₽ (оплачена)`, when `DeliveryPrice > 0`
  5. `Комментарий: <o.Comment>`, when it is not empty
- `Items`: one per `o.Items`:
  `{Type: "Product", ProductID: it.IikoID, Amount: float64(it.Amount), Price: it.OfferedPrice.Rubles()}`. An item with
  an empty `IikoID` makes the function return `fmt.Errorf("product %d has no iiko id", it.ProductID)`.
- `Payments`: `[{PaymentTypeKind: s.PaymentTypeKind, Sum: o.Total.Rubles(), PaymentTypeID: s.PaymentTypeID, IsProcessedExternally: true}]`.

### wiring
- `app.New`:
  - `iikoOrders := NewIikoOrders(iikoClient, db, telegramClient, cfg.Telegram.ChatIDOrderConfirmed, IikoOrderSettings{...from cfg.Iiko..., Delays: []time.Duration{30 * time.Second, 2 * time.Minute}}, log)`
  - `order.NewService(db, c, Notifiers{notifier, iikoOrders}, payments, ...)`
- `App` gets `IikoOrders *IikoOrders`. `cmd/server/main.go` calls `a.IikoOrders.Wait()` after `a.Notifier.Wait()`.
- The `order` package changes only as described above. `Service` and `PaymentConfirmed` stay unchanged:
  `OrderPaid` is already called once per paid order.

## Tests to write
`iiko_orders_test.go`. Use a fake `DeliveryCreator` that records each `DeliveryOrder` and returns scripted results, the
fake `MessageSender` from `notifier`'s tests (or a new one if there is none), `testutil.NewDB(t)` for the order row,
`Delays: []time.Duration{0, 0}`, and `Wait()` before every check.
- `TestDeliveryOrder` (table, pure):
  - delivery with time `18:30`, fee 15000 kopecks, comment `Без лука`, and two items. Check every field. The exact
    comment is
    `"Заказ с сайта №7, оплачен онлайн\nКо времени: 18:30\nАдрес: Белореченск, ул. Мира, 5\nДоставка: 150 ₽ (оплачена)\nКомментарий: Без лука"`,
    and the payment sum is `o.Total.Rubles()` (without the fee);
  - pickup with no time, no fee and no comment: `DeliveryByClient`, `DeliveryPoint == nil`, comment
    `"Заказ с сайта №7, оплачен онлайн"`;
  - an item without an iiko id gives the error `product 3 has no iiko id`.
- `TestIikoOrdersSuccess`: an order inserted through the order store, with products that have `iiko_id`. Load it with
  `ByID` and call `OrderPaid`. Expect 1 call, the saved `iiko_order_id` (`ByID(...).IikoOrderID`), no alert, and the id
  matching `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`.
- `TestIikoOrdersRetry`: `&iiko.StatusError{Status: 500}` twice, then success. Expect 3 calls with the same `ID`, the id
  saved, and no alert.
- `TestIikoOrdersGivesUp`: always a 500. Expect 3 calls and the failure alert with the error text.
- `TestIikoOrdersPermanent`: `&iiko.DeliveryError{Reason: "Product not found"}` gives 1 call and the exact alert
  `Заказ №<id> не создан в iiko: iiko: delivery not created: Product not found. Внесите его в iikoFront вручную.`
- `TestIikoOrdersPending`: `iiko.ErrDeliveryPending` gives 1 call and the pending alert.
- `TestIikoOrdersMissingIikoID`: a product without `iiko_id` gives 0 calls and the failure alert.
- `TestIikoOrdersNotConfigured`: empty `TerminalGroupID` gives 0 calls and no alert.
- `TestIikoOrdersIgnoresOtherEvents`: `OrderCreated` and `OrderConfirmed` give 0 calls.
- `TestNotifiersFanOut`: two recording notifiers. Each of the three events reaches both, in order.
- `store_test.go`:
  - `TestSetIikoOrderID`: the value reads back from `ByID`, and a missing order gives `store.ErrNotFound`;
  - `ByID` items have `IikoID` (`""` for a product without one).
- `config_test.go`: the defaults and the full-env case include the six variables.

## Done when
CONVENTIONS "Done means" pass, and coverage of `app`, `order` and `config` is at least 80%. Owner check (write it as a
note under the PROGRESS line, not done by the agent): with the real `IIKO_*` values and Moneta's demo shop, pay an order
and see it in iikoFront. Its comment should show the address, the time and the fee, and `orders.iiko_order_id` should
be filled.

## Out of scope
- Cash orders, and sending orders on admin confirm.
- Product sizes, modifiers and `completeBefore` (the time is only in the comment).
- An admin «resend to iiko» action, showing `iikoOrderId` in the API, and reading the iiko order status later.
- Putting the delivery fee into the payment or as an iiko product.
