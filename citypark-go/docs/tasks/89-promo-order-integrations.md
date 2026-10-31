# Task 89: Promo iiko and staff integration

## Goal
Reconcile discounted orders in iiko and staff notifications.

## Depends on
84, 88

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/app/iiko_orders.go`
- `internal/app/notifier.go`

## Files to create
- `internal/app/promo_orders_test.go`

## Files to change
- `internal/iiko/delivery.go`
- `internal/app/iiko_orders.go`
- `internal/app/notifier.go`

## Spec
Reconcile discounted orders in iiko and staff notifications.
Read docs/tasks/IIKO_PROMOCODES.md produced by task84 before changing wire structs. Implement exactly its
verified mapping from saved line allocations; never recompute from live catalog/campaigns. Food payment equals
discounted food total; delivery keeps existing comment treatment. Include recorded promo code, discount and
payable total in staff Telegram messages. Keep retry IDs/idempotency and diagnostic privacy intact.
If provider configuration is required by task84, document and validate it before enabling campaigns; missing
configuration must prevent new promo orders, not silently send unreconciled orders. Do not send live messages.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestPromoIikoPayload: verified pickup/delivery fixtures, capped allocation and uneven quantities → exact reconciliation.
- TestPromoIikoRetry: stable external ID and original snapshots after campaign edits.
- TestPromoStaffMessage: code, discount, food and delivery totals agree with persisted order; no-code text unchanged.
- TestPromoIntegrationFailure: existing failure/retry behavior preserved; no silent undiscounted fallback.
Use httptest for iiko/Telegram and existing integration helpers.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
