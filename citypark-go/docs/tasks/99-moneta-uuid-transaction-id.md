# Task 99: UUID as the MONETA transaction id

## Goal
MONETA gets a random UUID as `MNT_TRANSACTION_ID` instead of the order's database id, so the order number is no longer guessable and never reaches the payment provider or the success URL. `POST /orders` and the rest of the API still use the numeric `orderId`.

## Depends on
Nothing.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/moneta/moneta.go`, `internal/order/service.go`, `internal/order/payment_retry.go`.

## Files to create
- `migrations/00034_order_payment_uuid.sql`

## Files to change
- `docs/design/schema.sql`: add `payment_uuid` to `orders` (same definition as the migration).
- `internal/moneta/moneta.go`: `RequestSignature` and `PaymentURL` take `transactionID string` instead of `orderID int64`; remove the `strconv` use that becomes unused.
- `internal/app/payments.go`: pass `r.TransactionID`.
- `internal/order/types.go`, `internal/order/store.go`: `Order` gets `PaymentUUID string`; add `ByPaymentUUID`.
- `internal/order/service.go`: `PaymentRequest`, `PaymentConfirmed`, `PaymentCheck`, `Create`.
- `internal/order/payment_retry.go`: sign the saved UUID.
- Existing tests in `internal/moneta`, `internal/order`, `internal/app` that build a payment form or a callback with a numeric `MNT_TRANSACTION_ID`: use the order's `PaymentUUID`.
- `docs/API.md` and `internal/apidocs/openapi.yaml`: see "API documentation".

## Spec

Migration (goose, `-- +goose Up` / `Down` like the other files). PostgreSQL 17 has `gen_random_uuid()` built in, so the default also fills existing rows:
```sql
ALTER TABLE orders ADD COLUMN payment_uuid uuid NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX orders_payment_uuid_idx ON orders (payment_uuid);
```
Down drops the index and the column.

Go:
```go
type PaymentRequest struct {
	TransactionID string // the order's PaymentUUID; replaces OrderID
	Amount        money.Kopecks
}
func (s *Signer) RequestSignature(transactionID, amount string) string
func (s *Signer) PaymentURL(base, transactionID, amount, successURL, failURL string) string
// order.Store
func (s *Store) ByPaymentUUID(ctx context.Context, id string) (Order, error) // store.ErrNotFound when missing
```
- `orderRow` and `selectOrders` read `o.payment_uuid::text`. The text is lowercase with dashes. The column is never written by Go: the database default creates it on `INSERT`, and the order is read back after creation as today.
- The signature string is unchanged apart from the id: `shopID + transactionID + amount + Currency + testMode + secret`.
- Validate a callback id with a helper `parseTransactionID(s string) (string, bool)` in `internal/order`: it must match the canonical form `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` after `strings.ToLower`. Return the lowercase value. Use `regexp` or a manual check; add no dependency.
- **Old numeric ids are rejected.** A payment started before this deploy carries a number such as `15`; there is no fallback, the number does not match the helper.
- `PaymentConfirmed`: invalid id → the same `httpx.BadRequest("Некорректное уведомление")` as now; the signature is checked before the order is loaded, as now; then `ByPaymentUUID`. The rest (amount check, idempotence, canceled order) is unchanged and keeps working on `o.ID`.
- `PaymentCheck`: signature check first, as now. Invalid id or unknown UUID → code `"500"` with empty amount, exactly like an unknown number does today. The response echoes `cb.TransactionID` unchanged.
- `Create` and `RetryPayment` call `Form(PaymentRequest{TransactionID: o.PaymentUUID, ...})`. A retry therefore signs the same transaction id every time.
- Logs that print `cb.TransactionID` keep the key `order_id` only where they print a real order id; where they print the callback value, rename the key to `transaction_id`.

## Tests to write
- `TestMonetaPaymentURLUsesTransactionID` (`internal/moneta`): a fixed shop id, secret and UUID → the URL has `MNT_TRANSACTION_ID=<uuid>` and `MNT_SIGNATURE` equals the md5 of the documented concatenation, computed in the test.
- `TestParseTransactionID`: table: valid lowercase; valid uppercase (returned lowercase); `""`; `"15"`; UUID without dashes; one character too long; a non-hex character → ok only for the first two.
- `TestCreateOrderUsesPaymentUUID`: create an order with the fake payments → the request has `TransactionID` equal to the row's `payment_uuid`, and the id is not the number of `orderId`.
- `TestOrderPaymentUUIDsAreUnique`: two orders → different `PaymentUUID`, both match the canonical form.
- `TestPaymentConfirmedByUUID`: signed callback with the order's UUID → order is `paid`, `payment_id` stored.
- `TestPaymentConfirmedRejectsNumericID`: signed callback with `MNT_TRANSACTION_ID=<orders.id>` → 400 `Некорректное уведомление`, order unchanged.
- `TestPaymentConfirmedUnknownUUID`: signed callback with a random valid UUID → 404 `Заказ не найден`.
- `TestPaymentCheckByUUID`: new order with a UUID → code `100`; paid order → `200`; canceled → `500`; numeric id and unknown UUID → `500` with empty amount; bad signature → 403.
- `TestRetryPaymentSignsSameUUID`: call twice → both `redirectUrl` contain the same `MNT_TRANSACTION_ID`, equal to the order's `payment_uuid`.
- `TestMigrationBackfillsPaymentUUID` is not needed (the column has a default); the unique index is covered by the uniqueness test.

Use `testutil.NewDB`, `testutil.DoJSON` and the existing payment fakes in `internal/order`. Update existing tests rather than deleting them.

## API documentation
In `docs/API.md` and `openapi.yaml`:
- `MNT_TRANSACTION_ID` in the payment-notification table: "our payment UUID of the order (lowercase, with dashes)", not a whole number.
- The two `redirectUrl` examples (`POST /orders` and the payment-retry endpoint): take new ones from test output.
- The note after `POST /orders`: after paying, MONETA appends `MNT_TRANSACTION_ID` (now a UUID) to the success URL; `orderId` in the response stays the numeric id; the frontend must not read the order number from `MNT_TRANSACTION_ID`.
- Add a one-line deploy note: payment links created before the deploy fail at the callback, because old numeric ids are rejected.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/moneta`, `internal/order` and `internal/app` is at least 80%, and `grep -rn 'strconv.ParseInt(cb.TransactionID' internal` prints nothing.

## Out of scope
- Showing the UUID to customers, staff or in the admin panel; no JSON field is added.
- Changing `orderId` anywhere else, or the `payment_id` column (MONETA's operation id).
- Any fallback for numeric transaction ids.
- The website (`citypark-modern`): its payment-return pages match `MNT_TRANSACTION_ID` against the order number, so it needs website task 60. Deploy both together. Do not touch `../citypark-modern/`.
- `citypark-admin`: it does not read `MNT_TRANSACTION_ID`.
