# Task 100: MONETA CHECK diagnostics and Sentry reports

## Goal
Every MONETA CHECK request writes one log line with the facts needed to find out why it failed, and every failed CHECK sends one Sentry event with a fixed reason code. No customer data, signature or secret reaches the log line or Sentry.

## Depends on
Task 93 (Sentry), task 99 (UUID transaction id).

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/SENTRY.md`, `internal/monitoring/sentry.go`, `internal/order/handler.go` (`monetaCheck`), `internal/order/service.go` (`PaymentCheck`).

## Files to change
- `internal/order/service.go`: `PaymentCheck` returns a `CheckOutcome`; reason constants.
- `internal/order/handler.go`: `HandlerOptions.OnCheckFailure`; `monetaCheck` logs and reports.
- `internal/monitoring/sentry.go`: `CaptureCheckFailure`; `scrub` keeps the reason.
- `internal/app/app.go`: pass `monitoring.CaptureCheckFailure` in `order.HandlerOptions`.
- Existing tests that call `PaymentCheck` (`internal/order/check_test.go`, `payment_uuid_test.go`): use `outcome.Response`.
- `docs/SENTRY.md`: add the "MONETA CHECK failures" paragraph from "Documentation".

## Spec

Reasons (fixed set, English snake_case; the same strings are the Sentry tag value and the log field):
```go
const (
	CheckBadRequest        = "bad_request"          // form not parsed, duplicate MNT_ field, or MNT_COMMAND is not CHECK; HTTP 400
	CheckBadSignature      = "bad_signature"        // HTTP 403
	CheckInternalError     = "internal_error"       // any other error from PaymentCheck; HTTP 500
	CheckInvalidID         = "invalid_transaction_id" // code 500, id is not a canonical UUID (for example an old number)
	CheckOrderNotFound     = "order_not_found"      // code 500, no order has this UUID
	CheckOrderCanceled     = "order_canceled"       // code 500
	CheckNotPayable        = "order_not_payable"    // code 500, status is not new/pending, or payment_id is set but code was not 200 (cannot happen today; keep as the default branch)
	CheckRetryNotAllowed   = "retry_not_allowed"    // code 500, new/pending order but PaymentRetry.Allowed is false (another day, outside payment hours)
	CheckAmountMismatch    = "amount_mismatch"      // code 500, payable order but cb.Amount differs from the order amount
)
```
Codes `100`, `200` and `402` are successes: reason is empty, nothing is reported.

```go
// CheckOutcome is the result of PaymentCheck. Reason is empty for result codes 100, 200 and 402.
type CheckOutcome struct {
	Response      moneta.CheckResponse
	Reason        string // one of the Check* constants, or ""
	OrderID       int64  // 0 when no order was found
	OrderStatus   string // "" when no order was found
	RetryReason   string // PaymentRetry.Reason when Reason is CheckRetryNotAllowed, else ""
	ExpectedAmount string // amount we computed, "" when no order was found
}
func (s *Service) PaymentCheck(ctx context.Context, cb PaymentCallback) (CheckOutcome, error)

type HandlerOptions struct {
	// ...existing fields
	OnCheckFailure func(ctx context.Context, reason string) // nil = no reporting
}
```
`PaymentCheck` keeps its logic and its HTTP behaviour. It only adds the outcome. A bad signature still returns the same 403 `*httpx.Error`; the handler maps it to `CheckBadSignature`.

`monetaCheck` after the response is decided, always writes one line with `slog.Default()`:
```
slog.Info("moneta check", "transaction_id", cb.TransactionID, "http_status", 200, "result_code", "500",
    "reason", "order_not_found", "order_id", 0, "order_status", "", "amount_sent", cb.Amount,
    "amount_expected", "", "retry_reason", "")
```
- Level `Info` when `reason` is empty, `Warn` otherwise.
- `http_status` is the status the handler sent. For the 400, 403 and 500 paths `result_code` is `""` and `order_id` is `0`.
- Never log `MNT_SIGNATURE`, the secret or `MNT_SUBSCRIBER_ID`.
- When `reason` is not empty, call `h.opts.OnCheckFailure(r.Context(), reason)` once, if it is not nil. The call comes after the log line and must not change the HTTP response.

`monitoring.CaptureCheckFailure(ctx context.Context, reason string)`:
- Takes the hub from `sentry.GetHubFromContext(ctx)`; returns when it is nil (Sentry disabled).
- Inside `hub.WithScope`: `SetTag("moneta.check_reason", reason)`, `SetLevel(sentry.LevelWarning)`, then `hub.CaptureMessage("MONETA check failed: " + reason)`.
- Does not take the order id, amount, UUID or any other argument.

`scrub` today removes tags and replaces the message, so a message event would become "Unexpected application error" with no reason. Change it:
- Before the existing resets, read `reason := event.Tags["moneta.check_reason"]` and keep it only if it is in the allow-list of the nine constants above (copy the strings into `monitoring`; do not import `order`, to avoid an import cycle).
- After the resets, when `reason != ""` and the event has no exception: `event.Tags = map[string]string{"moneta.check_reason": reason}`, `event.Message = "MONETA check failed: " + reason`, `event.Fingerprint = []string{"moneta-check", reason}`, `event.Level = sentry.LevelWarning`. The final block that sets "Unexpected application error" must not overwrite this message.
- A tag value outside the allow-list is dropped and the event is scrubbed as before.
- Everything else stays as it is: no request data, user, breadcrumbs, contexts except the trace ids.

Result: one Sentry issue per reason (grouped by the fingerprint), each with its count and trace id. Repeated scanner traffic stays one issue per reason.

## Tests to write
- `TestPaymentCheckOutcomeReasons` (`internal/order`, table, `testutil.NewDB`): numeric id → `invalid_transaction_id`, `OrderID` 0; random UUID → `order_not_found`; canceled order → `order_canceled`, `OrderID` and `OrderStatus` set; paid order (code 200) → empty reason; new order from yesterday → `retry_not_allowed` and `RetryReason` `expired`; new order, `cb.Amount` different → `amount_mismatch`; new order, `cb.Amount` empty → code 100, empty reason; new order, matching amount → code 402, empty reason. `ExpectedAmount` equals the order amount when an order is found.
- `TestMonetaCheckLogsAndReports` (`internal/order`, through the router with `testutil.DoJSON` or a form POST): inject a fake `OnCheckFailure` that records reasons and set `slog.Default()` to a handler writing into a buffer for the test (restore it with `t.Cleanup`). Cases: unsigned request → 403, reason `bad_signature`, one call; `MNT_COMMAND` missing → 400, `bad_request`; unknown UUID → 200, `order_not_found`; payable order → 200, no call, log line level INFO with `result_code=100`. For every case check the log line has `transaction_id` and `http_status`, and does not contain the signature value.
- `TestMonetaCheckWithoutReporter`: `OnCheckFailure` is nil → the failing request still answers normally and logs.
- `TestCaptureCheckFailure` (`internal/monitoring`, fake transport like `TestMonitoring`): hub on the context, reason `order_not_found` → one event with message `MONETA check failed: order_not_found`, tag `moneta.check_reason`, fingerprint `["moneta-check","order_not_found"]`, level warning; JSON of the event contains no `Bearer`, no `@`, no UUID text. Reason `private@example.test` (not in the allow-list) → event message is `Unexpected application error` and the JSON does not contain `private@example.test`. No hub on the context → no event and no panic.
- Keep `TestMonitoring` passing unchanged.
- App wiring: one test in `internal/app` that a CHECK with a bad signature through the real router with a fake Sentry transport produces one event with the tag. If `internal/app` has no way to inject a transport, test only that `monitoring.CaptureCheckFailure` is the configured callback by running the router with `monitoring.Middleware(client)` and a fake transport.

## Documentation
Add to `docs/SENTRY.md`: "MONETA CHECK failures: every CHECK that is rejected (HTTP 400, 403, 500) or answered with result code 500 sends one warning event `MONETA check failed: <reason>`, grouped by reason. Reasons: bad_request, bad_signature, internal_error, invalid_transaction_id, order_not_found, order_canceled, order_not_payable, retry_not_allowed, amount_mismatch. The event has no order id, UUID or amount. Look in the server log for the line `moneta check` with the same `transaction_id` to see the order id, order status and amounts." No `docs/API.md` change: no endpoint changes.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/order`, `internal/monitoring` and `internal/app` is at least 80%, and `grep -rn 'PaymentCheck(' internal | grep -v _test` shows only the handler call.

## Out of scope
- Reporting successful or `200`/`402`/`100` checks.
- Reporting the payment callback (`/webhooks/moneta`) failures: a separate task if needed.
- Alert rules in Sentry, log forwarding, new environment variables.
- Changing result codes, the signature or the UUID logic from task 99.
