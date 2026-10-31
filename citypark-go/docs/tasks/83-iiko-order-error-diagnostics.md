# Task 83: iiko order error diagnostics

## Goal
Expose command exception messages and request identifiers when paid order creation fails in iiko. Preserve pickup and courier behavior.

## Depends on
62, 79. Independent of the unfinished MONETA task 82; explicitly requested as a separate fix.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/delivery.go`, `internal/app/iiko_orders.go`, `internal/iiko/delivery_test.go`.

## Files to create
- `docs/tasks/83-iiko-order-error-diagnostics.md`

## Files to change
- `internal/iiko/delivery.go`: command exception fallback and structured request context.
- `internal/iiko/delivery_test.go`: exception shapes and diagnostic context tests.
- `internal/app/iiko_orders.go`: log each failed attempt with local and iiko order IDs.
- `internal/app/iiko_orders_test.go`: verify failed-attempt logging.
- `docs/tasks/PROGRESS.md`: record this task and validation results.

## Spec
- Keep `DeliveryByClient` without a delivery point for pickup, and `DeliveryByCourier` with an address for delivery.
- For command state `Error`, prefer nonblank `errorReason`, then a string `exception` or its string `message` (including `Message`), then `unknown error`. Trim surrounding whitespace. Unexpected JSON exception structures must not fail decoding the command response.
- Do not dump arbitrary exception properties, credentials, or request payloads. Extract only the error message for staff alerts.
- Wrap create/status failures with private request context implementing `error`, `Unwrap`, and `slog.LogValuer`. Preserve the error text and `errors.Is`/`errors.As` behavior. Structured logs expose `message`, `endpoint`, `correlation_id`, and the submitted `iiko_order_id`; the correlation ID is empty before a valid create response.
- Log each failed attempt with the local `order_id`, submitted `iiko_order_id`, and structured error. Preserve retry classification, pending handling, and Telegram messages.
- No endpoint, schema, or public API changes.

## Tests to write
- `TestCreateDeliveryPollException`: reason precedence, object/string messages, capitalized property, null/empty/unexpected values, and exclusion of unrelated exception fields.
- `TestCreateDeliveryErrorDiagnostics`: creation rejection, command rejection, and decode failure expose the stage and identifiers without dumping private payloads.
- `TestIikoOrdersLogsFailedAttempt`: permanent rejection logs local and submitted IDs and sends no retry.
- Existing pickup/courier serialization, error matching, retries, and notification tests must continue to pass.

## API documentation
No HTTP endpoint behavior changes; `docs/API.md` is unaffected. Upstream schema: https://api-ru.iiko.services/api-docs/docs (`ErrorCommandStatus.exception` is unconstrained JSON; `errorReason` is nullable).

## Done when
All CONVENTIONS checks pass, including the serial full suite and at least 80% coverage in `internal/iiko` and `internal/app`, using `TEST_DB_DSN` from `.env.development`. Do not start a database. If it is unavailable, record the blocker without ticking or committing.

## Out of scope
Changing delivery types or address configuration, repairing the underlying rejection without its details, replaying order 16, changing payments, or completing task 82.
