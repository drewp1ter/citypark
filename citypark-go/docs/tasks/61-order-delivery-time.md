# Task 61: Order delivery time

## Goal
`POST /orders` accepts an optional `deliveryTime` (`"HH:MM"`, the «Ко времени…» choice on the website). It is saved on
the order, returned by the order read endpoints and shown in the Telegram message. Today the website writes it into
the comment; citypark-modern task 28 switches to the field, and task 62 copies it to iiko.

## Depends on
27.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/service.go` (`CreateInput`, `Create`, `saveParams`, `save`),
`internal/order/store.go` (`InsertParams`, `Insert`, `selectOrders`, `orderRow`, `selectDuplicate*`,
`FindOpenDuplicate`), `internal/telegram/message.go`.

## Files to create
- `migrations/00015_order_delivery_time.sql`

## Files to change
- `internal/validate/validate.go`, `internal/validate/validate_test.go`: the `hhmm` tag.
- `internal/order/types.go`, `internal/order/service.go`, `internal/order/store.go` and their tests.
- `internal/telegram/message.go`, `internal/telegram/telegram_test.go`: the «Ко времени» line.
- `internal/app/notifier.go` (`orderMessage` copies the field), `internal/app/app_test.go` if it checks the message.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec

### Migration
```sql
-- +goose Up
ALTER TABLE orders ADD COLUMN delivery_time text NOT NULL DEFAULT ''
    CHECK (delivery_time ~ '^(([01][0-9]|2[0-3]):[0-5][0-9])?$');

-- +goose Down
ALTER TABLE orders DROP COLUMN delivery_time;
```

### validate
A new tag `hhmm`: the value matches `^([01][0-9]|2[0-3]):[0-5][0-9]$`. It is registered next to `ruphone`. Its message
in `message()` is `Укажите время в формате ЧЧ:ММ`. With `omitempty`, an empty string passes.

### order
- `CreateInput.DeliveryTime string \`json:"deliveryTime" validate:"omitempty,hhmm"\``. Empty means «как можно скорее».
- `saveParams.deliveryTime`, `InsertParams.DeliveryTime` and the `Insert` statement write `delivery_time`.
- `Order.DeliveryTime string \`json:"deliveryTime"\``. `selectOrders` and `orderRow` read `o.delivery_time`.
- Duplicate lookup: both `selectDuplicatePickup` and `selectDuplicateDelivery` get `AND o.delivery_time = $N` (the next
  free placeholder). `FindOpenDuplicate(ctx, phone, addr, comment, deliveryTime, itemsHash)` gets the new parameter
  after `comment`. `Create` passes `in.DeliveryTime`. The same cart for another time is a new order.
- No check against working hours. The website already allows only 10:00–23:00.

### telegram
`OrderMessage.DeliveryTime string // "" = as soon as possible`. In `FormatOrder`, right after the «Оплата» line, write
`line("Ко времени", m.DeliveryTime)` only when it is not empty. `orderMessage` in `internal/app/notifier.go` copies
`o.DeliveryTime`.

## Tests to write
- `validate_test.go` (table, a struct with `T string \`validate:"omitempty,hhmm"\``):
  - `""`, `"09:30"`, `"00:00"` and `"23:59"` pass;
  - `"24:00"`, `"9:30"`, `"18-30"`, `"18:60"` and `"18:30:00"` give `errors.t = "Укажите время в формате ЧЧ:ММ"`.
- `service_test.go`:
  - `TestCreateDeliveryTime`: create with `DeliveryTime: "18:30"`, then `ByID` gives `DeliveryTime == "18:30"`.
    Without it, `""`.
  - `TestCreateDuplicateOtherTime`: two identical online orders, the second with another `DeliveryTime`, give two
    different ids. Same time gives the same id (as today).
- `create_handler_test.go`: `deliveryTime: "25:00"` gives 400 with `errors.deliveryTime = "Укажите время в формате ЧЧ:ММ"`.
- `store_test.go`: `Insert` with `DeliveryTime` reads back from `ByID` and `List`. Every existing `FindOpenDuplicate`
  call gets the new argument.
- `telegram_test.go`: `FormatOrder` with `DeliveryTime: "18:30"` has the line `Ко времени:` padded like the others,
  followed by `18:30`, right after «Оплата». Without it the line is missing (the existing golden text stays unchanged).

## API documentation
- `#### POST /orders`: the request field `deliveryTime` (string, optional, `HH:MM`, empty = «как можно скорее»), with
  the error `400 errors.deliveryTime: «Укажите время в формате ЧЧ:ММ»`. Note: «Identical orders for different times are
  separate orders».
- Every order response (`GET /orders/{id}`, the lists, the admin endpoints) has `deliveryTime` (`""` when not set).
- The same in `openapi.yaml` (`CreateOrderInput`, `Order`).

## Done when
CONVENTIONS "Done means" pass, coverage of `validate`, `order`, `telegram` and `app` is at least 80%, and `docs/API.md`
is updated.

## Out of scope
- Sending the time to iiko (task 62).
- The website (citypark-modern task 28) and the admin panel.
- Dates: a delivery time is always today.
