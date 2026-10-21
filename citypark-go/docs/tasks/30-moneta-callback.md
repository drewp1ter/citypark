# Task 30: Moneta payment callback

## Goal
`POST /webhooks/moneta`: when Moneta reports a paid order, check the signature and the amount, mark the order as paid and notify staff once.

## Depends on
26. Task 27 must be done too (it creates `handler.go`).

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/service.go`, `internal/order/store.go`, `internal/order/handler.go`.

## Files to change
- `internal/order/service.go`: add `PaymentConfirmed`.
- `internal/order/handler.go`: add the route.

## Files to create
- `internal/order/callback_test.go`

## Background
After a payment Moneta calls the "Pay URL" from the shop settings with form fields:
```
MNT_ID=100&MNT_TRANSACTION_ID=15&MNT_OPERATION_ID=555&MNT_AMOUNT=790.00
&MNT_CURRENCY_CODE=RUB&MNT_TEST_MODE=0&MNT_SIGNATURE=998b6268fe3a6c7f4eeb8ccec65a9566
```
`MNT_TRANSACTION_ID` is our order id. `MNT_OPERATION_ID` is Moneta's own id of the payment.

Anyone on the internet can send such a request. It is trusted only when `MNT_SIGNATURE` is right, because the signature needs the shop secret. Moneta expects the plain text `SUCCESS` in the answer; with any other answer it repeats the call later.

## Spec

### Service
```go
// PaymentConfirmed checks a payment callback and marks the order as paid.
func (s *Service) PaymentConfirmed(ctx context.Context, cb PaymentCallback) error
```
1. `strconv.ParseInt(cb.TransactionID, 10, 64)`; on error or a value below 1 → `httpx.BadRequest("Некорректное уведомление")`.
2. `!s.payments.Verify(cb)` → `httpx.NewError(403, "Неверная подпись")`. This comes **before** the order is loaded, so an unsigned request cannot learn which orders exist.
3. `order, err := ByID(id)`; `store.ErrNotFound` → `httpx.NewError(404, "Заказ не найден")`.
4. Amount check: `s.payments.Form(PaymentRequest{order.ID, order.Total + order.DeliveryPrice}).Amount != cb.Amount` → log with `slog.Default().Error` (attributes `order_id`, `expected`, `got`) and return `httpx.BadRequest("Неверная сумма")`.
5. `changed, err := MarkPaid(order.ID, cb.OperationID)`; on error return it.
6. `changed` → reload with `ByID` and call `s.notifier.OrderPaid(ctx, order)`.
7. Not `changed` → return nil. Moneta may deliver the same callback several times; staff must be notified once. If the order's status is `canceled`, also log a warning: a canceled order was paid.

### Route
`POST /webhooks/moneta`, public, no working-hours middleware. This handler does not use `httpx.Handle`, because the answer is plain text, not JSON.

1. Build the callback from the form: `TransactionID: r.FormValue("MNT_TRANSACTION_ID")`, `OperationID: r.FormValue("MNT_OPERATION_ID")`, `Amount: r.FormValue("MNT_AMOUNT")`, `Signature: r.FormValue("MNT_SIGNATURE")`. The other `MNT_*` fields are ignored: the shop id, the currency and the test mode come from the configuration.
2. Log one line with `slog.Default().Info`: message `moneta callback`, attributes `order_id` and `operation_id`.
3. `h.svc.PaymentConfirmed(ctx, cb)` and answer with `Content-Type: text/plain; charset=utf-8`:

   | Result | Status | Body |
   |---|---|---|
   | nil | 200 | `SUCCESS` |
   | `*httpx.Error` (use `errors.As`) | the error's status | `FAIL` |
   | any other error | 500 | `FAIL` (also log it with `slog.Default().Error`) |

## Tests to write
`db := testutil.NewDB(t)`. Seed one online order (insert it with `NewStore(db).Insert`) whose total plus delivery is 790.00 rubles, and set it to `pending` with `SetStatus`. Use the fakes from `service_test.go`: `fakePayments.form = PaymentForm{Amount: "790.00"}`, `fakePayments.valid = true`.

Service:
- `TestPaymentConfirmed`: order `paid`, `payment_id = '555'`; `OrderPaid` called once with status `paid` in the passed order.
- `TestPaymentConfirmedTwice`: call twice → both nil, still one `OrderPaid` call.
- `TestPaymentConfirmedFromNew`: the order is still `new` → `paid`.
- `TestPaymentBadSignature`: `valid = false` → 403; order stays `pending`; notifier not called.
- `TestPaymentBadSignatureUnknownOrder`: `valid = false` and transaction id `999` → 403, not 404.
- `TestPaymentUnknownOrder`: transaction id `999` → 404.
- `TestPaymentBadTransactionID`: table for `""`, `abc`, `0`, `-1` → 400.
- `TestPaymentWrongAmount`: callback amount `1.00` → 400; order stays `pending`; notifier not called.
- `TestPaymentAfterConfirmed`: order status set to `confirmed` with plain SQL → nil error, status stays `confirmed`, notifier not called.
- `TestPaymentAfterCanceled`: status `canceled` → nil error, status stays `canceled`, notifier not called.

Handler (post the body with `Content-Type: application/x-www-form-urlencoded`):
- `TestCallbackSuccess`: the form from "Background" with the seeded order's id → 200, body exactly `SUCCESS`, content type `text/plain; charset=utf-8`; order is `paid`.
- `TestCallbackBadSignature`: `valid = false` → 403 `FAIL`.
- `TestCallbackUnknownOrder`: 404 `FAIL`.
- `TestCallbackEmptyBody`: 400 `FAIL`.
- `TestCallbackWrongAmount`: 400 `FAIL`.
- `TestCallbackRepeated`: post twice → 200 `SUCCESS` both times, one `OrderPaid` call.
- `TestCallbackIgnoresWorkingHours`: clock at 03:00 with working hours 9:00–22:00 → still 200.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
The Moneta Check URL and refunds. Checking the sender's IP address. Sending the paid order to iiko, e-mail and SMS.
