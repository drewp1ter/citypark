# Task 27: Create-order endpoint and working hours

## Goal
`POST /orders` (public) and a middleware that rejects orders outside the restaurant's working hours.

## Depends on
26.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/service.go`, `internal/salewindow/salewindow.go`.

## Files to create
- `internal/order/handler.go`
- `internal/order/workinghours.go`
- `internal/order/create_handler_test.go`

## Spec

### `workinghours.go`
```go
// WorkingHours rejects requests outside [start, end). start and end look like "9:00".
// If either is empty or cannot be parsed, every request passes.
func WorkingHours(c clock.Clock, start, end string) func(http.Handler) http.Handler
```
- Parse both with `salewindow.ParseClock` once, when the middleware is built.
- Per request: `salewindow.Window{Start: &s, End: &e}.OpenAtTime(c.Now())`.
- Closed → `httpx.WriteError` with status 503 and message
  `"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с " + start + " до " + end`
  (start and end exactly as configured; `\n` is a real newline).

### `handler.go`
```go
type HandlerOptions struct {
    WorkingHoursStart string
    WorkingHoursEnd   string
    DefaultPageSize   int
}

type Handler struct { /* svc *Service, tokens *auth.Tokens, opts HandlerOptions */ }

func NewHandler(svc *Service, tokens *auth.Tokens, opts HandlerOptions) *Handler
func (h *Handler) Routes(r chi.Router)
```
`tokens` and `DefaultPageSize` are not used yet; tasks 28–30 add routes to this handler and need them. Keep the constructor signature as written.

`Routes` registers:
```go
r.With(WorkingHours(h.svc.clock, h.opts.WorkingHoursStart, h.opts.WorkingHoursEnd)).
    Post("/orders", httpx.Handle(h.create))
```

`create`:
1. `httpx.DecodeJSON` into `CreateInput`.
2. `validate.Struct`.
3. `h.svc.Create`.
4. 201 with the `CreateResult`.

Request example:
```json
{
  "phone": "+79991234567",
  "name": "Иван",
  "comment": "Без лука",
  "paymentType": "cash",
  "address": { "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [{ "productId": 1, "amount": 2 }]
}
```
Response for `"paymentType": "cash"`: `201 {"orderId": 1, "redirectUrl": "http://localhost:3001/thankyou"}`. The frontend sends the customer to `redirectUrl`.

Response for `"paymentType": "online"`: `201 {"orderId": 1, "amount": "790.00", "mntSignature": "8c65fcd4afe3dc4cb915e04aaa273574"}`. The frontend opens the Moneta payment form with these three values (`MNT_TRANSACTION_ID`, `MNT_AMOUNT`, `MNT_SIGNATURE`).

## Tests to write
`db := testutil.NewDB(t)`; fakes for `Notifier` and `Payments` as in `service_test.go` (reuse them: same package). Seed a category with three products and a delivery price `Сочи`. Default clock: Friday 2024-05-24 15:00; working hours `9:00`–`22:00`.

- `TestCreateOrder`: full body with address → 201, body has exactly the keys `orderId` and `redirectUrl`; one order in the database.
- `TestCreateOrderPickup`: no `address` → 201.
- `TestCreateOrderNight`: clock at 05:00 → 503 with body
  `{"statusCode":503,"message":"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с 9:00 до 22:00"}` and no order.
- `TestCreateOrderNoWorkingHours`: options with empty start → 201 at 05:00.
- `TestCreateOrderCategoryWeekend`: category with `sale_days = '1111100'`, clock on Saturday 15:00 → 503, message starts with `Возможность заказа`.
- `TestCreateOrderProductWeekend`: the same with the window on a product → 503.
- `TestCreateOrderBadAmount`: an item with `amount: 0` → 400 with `errors` key `items[0].amount`; no order.
- `TestCreateOrderBadProduct`: `productId: 999` → 400 `{"statusCode":400,"message":"Продукт не найден."}`.
- `TestCreateOrderValidation`: `{}` → 400 with `errors` keys `phone`, `name`, `paymentType`, `items`.
- `TestCreateOrderBadPaymentType`: `paymentType: "card"` → 400 with `errors.paymentType`.
- `TestCreateOrderBadAddress`: `address: {"city": ""}` → 400 with `errors` keys `address.city` and `address.address`.
- `TestCreateOrderBadJSON`: body `{bad` → 400 `Некорректный JSON`.
- `TestCreateOrderOnline`: fake payments returns `PaymentForm{Amount: "790.00", Signature: "sig"}` → 201 and the body has exactly the keys `orderId`, `amount` (`790.00`) and `mntSignature` (`sig`).
- `TestWorkingHoursBoundaries`: table test on the middleware alone with a dummy handler: 08:59 → 503, 09:00 → 200, 21:59 → 200, 22:00 → 503; unparsable start (`"abc"`) → 200.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Reading and changing orders (tasks 28, 29). The payment callback (task 30).
