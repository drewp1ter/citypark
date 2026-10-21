# Task 26: Order service — create

## Goal
The business logic of placing an order: checks, saving, notifying staff (cash) or signing the payment form (online).

## Depends on
16, 25b.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/types.go`, `internal/order/store.go`, `internal/catalog/types.go`.
Signatures you call (look them up only if needed): `user.NewStore(tx).UpsertCustomer(ctx, name, phone)`, `address.NewStore(tx).FindOrCreate(ctx, city, address)` and `.LinkUser(ctx, userID, addressID)`, `deliveryprice.NewStore(db).ByName(ctx, name)`, `catalog.NewStore(tx).ProductsByIDs(ctx, ids)`, `validate.NormalizePhone`.

## Files to create
- `internal/order/service.go`
- `internal/order/service_test.go`

## Spec

```go
package order

// Notifier tells staff about orders. Implementations must not block for long and must not fail the request.
type Notifier interface {
    OrderCreated(ctx context.Context, o Order)   // a cash order was placed
    OrderPaid(ctx context.Context, o Order)      // an online order was paid
    OrderConfirmed(ctx context.Context, o Order) // an admin confirmed an order
}

type PaymentRequest struct {
    OrderID int64
    Amount  money.Kopecks
}

// PaymentForm is what the frontend needs to open the Moneta payment form.
type PaymentForm struct {
    Amount    string // rubles with two decimals, "790.00"
    Signature string
}

// PaymentCallback is what Moneta sends after a payment (used in task 30).
type PaymentCallback struct {
    TransactionID string // our order id, as text
    OperationID   string // Moneta's id of the payment
    Amount        string
    Signature     string
}

// Payments signs payment forms and checks payment callbacks. It makes no network calls.
type Payments interface {
    Form(r PaymentRequest) PaymentForm
    Verify(c PaymentCallback) bool
}

type CreateInput struct {
    Phone       string         `json:"phone" validate:"required,ruphone"`
    Name        string         `json:"name" validate:"required,max=80"`
    Comment     string         `json:"comment" validate:"max=8192"`
    PaymentType string         `json:"paymentType" validate:"required,oneof=cash online"`
    Address     *address.Input `json:"address" validate:"omitempty"`
    Items       []ItemInput    `json:"items" validate:"required,min=1,dive"`
}

// CreateResult: a cash order fills RedirectURL; an online order fills Amount and MntSignature.
type CreateResult struct {
    OrderID      int64  `json:"orderId"`
    RedirectURL  string `json:"redirectUrl,omitempty"`
    Amount       string `json:"amount,omitempty"`
    MntSignature string `json:"mntSignature,omitempty"`
}

type Options struct {
    ThankYouCash string
}

type Service struct { /* db *sqlx.DB, clock clock.Clock, notifier Notifier, payments Payments, opts Options */ }

func NewService(db *sqlx.DB, c clock.Clock, n Notifier, p Payments, opts Options) *Service

// Create places an order. in must already be validated.
func (s *Service) Create(ctx context.Context, in CreateInput) (CreateResult, error)
```

### `Create` steps
1. **Normalise.** Phone through `validate.NormalizePhone`; trim `Name` and `Comment`. Merge items with the same `ProductID` by adding their amounts. Compute `ItemsHash`.
2. **Delivery price.** With an address: `deliveryprice.ByName(address.City)`; `store.ErrNotFound` → `httpx.BadRequest("Указано не существующее имя населенного пункта.")`. Without an address the delivery price is 0.
3. **Reuse (online only).** `FindOpenDuplicate(phone, address, comment, hash)`. If found, load it with `ByID` and jump to step 6. This stops a customer who presses "pay" twice from creating two orders.
4. **Save**, inside `store.InTx`, with stores built from the `tx`:
   1. `catalog.ProductsByIDs(ids)`. If fewer products come back than requested → `httpx.BadRequest("Продукт не найден.")`.
   2. Sale-window checks with `now := s.clock.Now()`, product by product in the order of the request; the first failure wins. Checks per product, in this order:

      | Check | Error (status 503) |
      |---|---|
      | `!p.CategoryWindow().OpenOnDay(now)` | `Возможность заказа "{category name}" доступна только в следующие дни недели: {category DaysString}` |
      | `!p.CategoryWindow().OpenAtTime(now)` | `Время заказа "{category name}" ограничено с {category StartString} до {category EndString}.` |
      | `!p.Window().OpenOnDay(now)` | `Возможность заказа "{product name}" доступна только в следующие дни недели: {product DaysString}` |
      | `!p.Window().OpenAtTime(now)` | `Время заказа "{product name}" ограничено с {product StartString} до {product EndString}.` |

   3. `user.UpsertCustomer(name, phone)`.
   4. With an address: `address.FindOrCreate`, then `LinkUser(customer.ID, address.ID)`.
   5. `Insert` with `OfferedPrice` = the product's current `Price`.
5. **Load** the saved order with `ByID` (after the transaction).
6. **Cash:** `s.notifier.OrderCreated(ctx, order)`; return `{order.ID, opts.ThankYouCash}`.
7. **Online:**
   - `form := s.payments.Form(PaymentRequest{OrderID: order.ID, Amount: order.Total + order.DeliveryPrice})`.
   - If the order's status is `new`: `SetStatus(order.ID, StatusPending)`. A reused order (step 3) may already be `pending`; leave it.
   - Return `{OrderID: order.ID, Amount: form.Amount, MntSignature: form.Signature}`.

   The frontend opens the Moneta payment form with the order id, the amount and the signature. The customer pays there; the order becomes `paid` in task 30.

All expected failures are `*httpx.Error`. Other errors are returned unchanged.

## Tests to write
`db := testutil.NewDB(t)`. Fakes in the test file:
```go
type fakeNotifier struct{ created, paid, confirmed []Order }
type fakePayments struct{ requests []PaymentRequest; form PaymentForm; valid bool }
```
`Form` records the request and returns `form`; `Verify` returns `valid` (task 30 uses it).

Seed: a category `Мангал` with products at 59900 and 10000 kopecks; a delivery price `Сочи` = 15000. Clock: Friday 2024-05-24 15:00.

- `TestCreateCash`: with address → result has the order id and the cash URL; one order with status `new`, `delivery_price_kopecks = 15000`, items with snapshot prices; the result has empty `Amount` and `MntSignature`; the notifier got `OrderCreated` once with that order; payments not called; `user_addresses` has a link.
- `TestCreatePickup`: no address → `address_id` NULL, delivery 0.
- `TestCreateExistingCustomer`: an existing user with the phone and another name → no new user, the name is updated.
- `TestCreateMergesDuplicateItems`: the same product twice with amounts 1 and 2 → one item with amount 3.
- `TestCreateUnknownCity`: 400 with the exact message; no order.
- `TestCreateUnknownProduct`: product id 999 → 400 `Продукт не найден.`; no order and no new user (rollback).
- `TestCreateDeletedProduct`: 400.
- `TestCreateCategoryClosedDay`: category `sale_days = '1111100'`, clock on Saturday 2024-05-25 15:00 → 503 with `Возможность заказа "Мангал" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт`.
- `TestCreateCategoryClosedTime`: category window 9:00–21:00, clock Friday 05:00 → 503 `Время заказа "Мангал" ограничено с 9:00 до 21:00.`
- `TestCreateProductClosedDay` and `TestCreateProductClosedTime`: the same with the window on the product; the message carries the product's name and the product's own hours.
- `TestCreateOnline`: payments returns `PaymentForm{Amount: "749.00", Signature: "sig"}` → the result has the order id, `Amount == "749.00"`, `MntSignature == "sig"` and an empty `RedirectURL`; order status `pending`, `payment_id` empty; the request has the order id and amount = total + delivery (74900 for one product at 59900 with delivery 15000); the notifier was not called.
- `TestCreateOnlinePickup`: no address → the request amount is the total alone.
- `TestCreateOnlineReuse`: the same input twice → one order row, the same `OrderID`, payments called twice with the same `OrderID` and amount; the order is still `pending`.
- `TestCreateOnlineNotReusedForCash`: the same input as cash twice → two orders.
- `TestCreateResultJSON`: marshal a cash result → keys are exactly `orderId, redirectUrl`; marshal an online result → keys are exactly `orderId, amount, mntSignature`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
HTTP (task 27). The payment callback (task 30). Real Telegram calls and real Moneta signatures (task 31).
