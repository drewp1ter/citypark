# Task 79: Automatic paid orders and one staff Telegram chat

## Goal
Remove manual order confirmation. A verified successful payment automatically submits the order to iiko and notifies one staff Telegram chat, without a staff approval step.

## Depends on
62, 64, 65, 78. Required before citypark-admin task 24.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/order/service.go`
- `internal/app/notifier.go`
- `internal/app/iiko_orders.go`

## Files to create
None required; add focused test files if needed.

## Files to change
- `internal/order/service.go`, `handler.go`, and their tests: remove manual confirmation and its notifier contract.
- `internal/app/notifier.go`, `iiko_orders.go`, `app.go`, and their tests: one staff chat and no confirmation fan-out.
- `internal/config/config.go`, `config_test.go`: one staff chat setting.
- `internal/live/hub.go`, `live_test.go`: remove confirmation events.
- `internal/apidocs/openapi.yaml`, `docs/API.md`: update HTTP and WebSocket contracts.
- `.env.example`, `README.md`: configuration and deployment instructions; update other current deployment documentation referencing the removed settings.

## Spec
The owner explicitly chose to remove manual confirmation throughout the API and admin, preserve historical confirmed orders, and send order details plus a separate phone message to one chat.

### Payment and iiko
- Keep the existing verified Moneta callback as the trigger. Preserve signature and amount checks, atomic MarkPaid behavior, and duplicate-callback protection.
- Retain status `paid` after successful payment. Do not automatically set `confirmed`: payment success is not confirmation of iiko acceptance.
- Preserve automatic iiko submission, existing retry policy, stable iiko ID within retries, and handling of uncertain creation results. Telegram order notification runs independently of iiko success.
- Preserve failure and uncertain-result alerts, sending them to the single staff chat.
- Unpaid orders must not be sent to iiko or announced in Telegram. Existing handling of callbacks for canceled and historical confirmed orders stays unchanged.

### Remove manual confirmation
- Remove `POST /orders/{id}/confirm`, its handler, and `Service.Confirm`; the removed route must return 404 for an authenticated admin.
- Remove `OrderConfirmed` from the notifier interface, implementations, fan-out, and test fakes.
- Remove `order.confirmed` from live events and the documented event enum. Keep pending and paid events.
- Preserve the `confirmed` status in storage and read/filter contracts for historical records. Do not rewrite data or introduce a database migration.
- Tests asserting the superseded confirmation behavior must be replaced with removal and historical-read regression tests; this task explicitly authorizes those changes.

### Single staff chat
Use these interfaces:
```go
type Telegram struct { BaseURL, Token, StaffChatID string }
func NewTelegramNotifier(sender MessageSender, staffChatID string, log *slog.Logger) *TelegramNotifier
```
- Read `TELEGRAM_STAFF_CHAT_ID`; stop reading `TELEGRAM_CHAT_ID_FOR_ORDER_CREATED` and `TELEGRAM_CHAT_ID_FOR_ORDER_CONFIRMED`. No legacy fallback.
- After a newly recorded successful payment, send the existing formatted order, then its phone number as a separate message, both to StaffChatID.
- Preserve asynchronous sending, timeout, shutdown waiting, and error logging. If the first send fails, do not send the phone message. If the phone send fails, log the failure. Empty StaffChatID skips notifications with the existing warning behavior.
- Use StaffChatID for iiko alerts too. Customer Telegram authentication and account pairing are unrelated and must remain unchanged.
- Document deployment: set the new variable to the chosen existing staff chat before deploying, remove the old variables, and restart the backend. Do not edit real environment files or contact Telegram during implementation.

## Tests to write
Use existing database helpers and fake external services; no real payment or Telegram messages.
- `TestPaidOrderAutomaticDispatch`: valid signed callback marks the order paid, submits to fake iiko, and sends order details then phone to the same chat without a confirmation request.
- `TestDuplicatePaymentDispatch`: repeated callback does not repeat submission or messages.
- Preserve tests for invalid signatures, incorrect amounts, unpaid orders, canceled orders, and historical confirmed callbacks.
- `TestConfirmRouteRemoved`: authenticated admin POST to the old endpoint returns 404 and causes no side effects.
- `TestHistoricalConfirmedOrderReadable`: historical confirmed orders retain their status and can be read and filtered.
- Update notifier tests for the single chat, empty chat, first-send failure, and second-send failure.
- Update config tests to verify the new setting and that old settings do not populate StaffChatID.
- Update iiko tests to verify failure/uncertain-result alerts use the single chat and existing retry protections still hold.
- Update live event tests to cover pending and paid without confirmation events.

## API documentation
Remove the confirmation endpoint from `docs/API.md` and OpenAPI. Remove `order.confirmed` from WebSocket documentation. Explain automatic dispatch on payment and mark `confirmed` as a historical read/filter value. Keep order response shapes otherwise unchanged.

## Done when
Run all CONVENTIONS checks, loading TEST_DB_DSN from `.env.development` before database tests:
```bash
gofmt -l .
go build ./...
go vet ./...
go test -p 1 ./...
go test -p 1 -cover ./internal/order/... ./internal/app/... ./internal/config/... ./internal/live/...
```
Every changed internal package has at least 80% statement coverage. Do not start the database; stop and ask the owner if it is unavailable. Tick only this task when complete and commit `task 79: automatic paid orders and one staff Telegram chat`.

## Out of scope
Admin changes belong to citypark-admin task 24. No public website changes, durable notification queue, new retry policy, historical status migration, or changes to customer Telegram accounts. Historical task files remain unchanged.
