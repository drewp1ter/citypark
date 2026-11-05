# Task 95: Retry payment for existing unpaid orders

## Goal
Let an authenticated customer obtain a payment link for an existing unpaid order from website or Telegram history. Enforce same-day payment and restaurant opening hours on retries and Moneta pre-payment checks, while retaining valid confirmations for money already collected.

## Depends on
Backend tasks 88–90 (saved promo pricing and payments) and 94 (current completed baseline).

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/order/service.go`
- `internal/order/handler.go`
- `internal/order/workinghours.go`

## Files to create
- `internal/order/payment_retry.go` — eligibility and retry service methods.
- `internal/order/payment_retry_test.go` — service, HTTP and eligibility tests.

## Files to change
- `internal/order/service.go` — inject opening hours and apply eligibility to payment checks.
- `internal/order/handler.go` — authenticated retry route and customer-read eligibility fields.
- `internal/order/check_test.go` — expiry, hours and payment-check regressions.
- `internal/order/read_handler_test.go` — customer response contract.
- `internal/order/callback_test.go` — delayed confirmation regression.
- `internal/app/app.go` — wire the existing hours source into the order service.
- `internal/app/app_test.go` — verify production wiring.
- `docs/API.md`, `internal/apidocs/openapi.yaml` — document new route and customer fields.
- Existing order/API contract tests where exact response shapes or service setup require updates.
- `docs/tasks/PROGRESS.md` — tick only after implementation acceptance passes.

## Spec
### Service and eligibility
Use the existing database store, payment signer, injected clock and working-hours source; no migration or new SQL is required. Add:

```go
type PaymentRetryEligibility struct {
    Allowed bool   `json:"allowed"`
    Reason  string `json:"reason"`
}
func (s *Service) SetPaymentHours(hours HoursSource)
func (s *Service) RetryPayment(ctx context.Context, orderID, accountID int64) (CreateResult, error)
```

Configure the hours source in app wiring before serving requests. Missing configuration or a failed hours read must fail closed with the existing internal-error response, never allow payment. Update test setups to inject hours.

Use one eligibility policy for customer reads, retry and Moneta CHECK:
1. Only `new` or `pending` with an empty payment ID qualify; otherwise reason `not_payable`.
2. Compare creation date with the current date in the injected clock's configured location (`APP_TIMEZONE`, default `Europe/Moscow`). Different dates, including future dates, produce `expired`.
3. Require the existing `Hours.OpenAt` rule, including its start-inclusive/end-exclusive boundary; otherwise reason `closed`.
4. Eligible orders return `{ "allowed": true, "reason": "" }`.

Read hours once per request that needs them, not per order in a list. Eligibility must use fresh backend state, never browser time or cached history. Preserve original order ID, total, delivery price, discounts, items, recipient and delivery details. Do not reprice, create an order, change its status, consume promo usage, or notify staff. Repeated retries return payment links for the same transaction ID and saved amount (`Total + DeliveryPrice`).

### HTTP contract
| Method | Path (relative to `/api/v1`) | Role | Success |
| --- | --- | --- | --- |
| POST | `/orders/{id}/payment-retry` | Authenticated customer, owner only | 200 |

Use the same customer access middleware as `/orders/my`, covering website and Mini App sessions. No request body is needed. Authenticate and check ownership before eligibility. Use existing path-ID validation and error envelopes; missing or foreign orders return the existing 404 without revealing ownership. No guest-cookie ownership or administrator bypass.

Example success: `{ "orderId": 123, "redirectUrl": "https://demo.moneta.ru/assistant.htm?..." }` (illustrative URL; documentation must use actual test output).

Errors:
- 404: existing invalid path-ID response (`httpx.PathID` returns 404, not 400).
- 401/403: existing missing/invalid session and wrong-role responses.
- 404: existing not-found response for missing or foreign orders.
- 409, `not_payable`: `Этот заказ уже оплачен или недоступен для оплаты.`
- 409, `expired`: `Срок оплаты заказа истёк. Оформите новый заказ.`
- 503, `closed`: `Оплата доступна только в часы работы ресторана.`
- 500: existing internal-error response for database, hours or payment-link generation failures; never return an empty redirect URL.

All responses, including failures, retain `Cache-Control: private, no-store`. There is no request-body schema or amount accepted from the customer.

Add `paymentRetry: { allowed: boolean, reason: "" | "not_payable" | "expired" | "closed" }` to each customer order returned by `GET /orders/my` and owner `GET /orders/{id}`. Use response DTOs so this transient metadata does not change persisted orders, staff events or admin list responses. Preserve existing order fields. An hours lookup failure needed to calculate eligibility fails the read with 500 rather than reporting guessed availability.

### Existing links and payment callbacks
Apply date/hours eligibility to signed Moneta CHECK requests for unpaid orders, including links issued before this feature. Ineligible checks use the existing signed non-payable code `500`; retain amount formatting, signature validation, paid-order code `200`, and eligible codes `100`/`402`. Do not emit HTTP 503 for a normal closed-hours CHECK denial. Source failures remain internal errors.

Do not add date or opening-hours restrictions to `PaymentConfirmed`: genuine delayed callbacks must retain the existing signature/amount validation, idempotence and paid-order dispatch behavior. A retry link does not reserve payment eligibility; a later CHECK can reject it. Keep repeated callbacks and concurrent retry/callback behavior safe using existing payment confirmation guards.

## Tests to write
Use `testutil.NewDB`, `testutil.DoJSON`, an injected fixed clock, fake hours and existing fake payment/notifier helpers; database tests stay serial.
- `TestPaymentRetryEligibility`: new/pending allowed; paid/confirmed/canceled/payment ID blocked; previous/future date blocked; local midnight differing from UTC; opening and closing boundaries; hours read failure.
- `TestRetryPayment`: owner receives same ID and signed saved total including delivery; promo/catalog changes do not reprice; repeated requests leave all database rows, status, promo usage and notification counts unchanged.
- `TestRetryPaymentHTTP`: 200, no/invalid session, wrong role, invalid ID, missing/foreign order, each exact denial message, database/hours failures and empty payment URL; private/no-store throughout.
- `TestCustomerOrderPaymentRetry`: list/detail metadata matches service policy, list reads hours once, existing snapshots unchanged and admin responses unaffected.
- `TestPaymentCheckRetryEligibility`: old links denied next day/after closing, eligible existing codes preserved, already-paid checks still return 200 after expiry, source failure fails closed.
- `TestPaymentRetryConfirmationRace`: payment confirmed before retry is rejected; callback after link generation works; repeated callbacks do not duplicate dispatch.
- `TestDelayedPaymentConfirmation`: valid callback after midnight/closing is still recorded using existing canceled-order handling where applicable.
- `TestPaymentRetryAppWiring`: router uses customer authentication and configured hours for retry and CHECK.

## API documentation
Add `POST /orders/{id}/payment-retry`; update customer `GET /orders/my`, `GET /orders/{id}` and Moneta CHECK documentation and OpenAPI. Explain date/timezone, saved prices, private responses, old-link expiry and delayed confirmations. Examples and existing auth/error messages must come from implementation tests.

## Done when
- All CONVENTIONS checks pass: formatting, build, vet, full serial tests and at least 80% statement coverage in each changed internal package.
- Load `TEST_DB_DSN` from `.env.development`; never start a database or Docker. If unavailable, ask the owner to start it.
- API/OpenAPI and contract tests agree; no live orders, payments, notifications or production changes are needed for acceptance.
- Tick task 95 and commit `task 95: retry payment for unpaid orders`.

## Out of scope
Website UI (website task 54), admin UI, new orders/reordering, menu repricing, schedule changes, order auto-cancellation, a new payment-attempt table, provider configuration changes and deployment.
