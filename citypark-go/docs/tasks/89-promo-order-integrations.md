# Task 89: Promo RMS and staff integration

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
- `internal/config/config.go` and its tests
- order creation service and application wiring for the configuration guard
- `.env.example` and `README.md`

## Spec
Reconcile discounted orders in iiko and staff notifications.
Read docs/tasks/IIKO_PROMOCODES.md produced by task84 and implement its RMS mapping. Add optional
IIKO_PROMO_DISCOUNT_TYPE_ID (validate UUID when present). Missing configuration prevents new promo orders
before order writes/payment creation: return 503 with Russian message «Заказы с промокодом временно недоступны.».
No-code orders remain usable; existing pending-order retries retain their original snapshots. Document the new
error in API.md/OpenAPI. Already-paid dispatch failures use existing alerts/retry rules, never undiscounted fallback.

Preserve original prices and quantities. Assign globally unique position UUIDs once when constructing an outgoing
order and reuse them throughout its retries. For positive saved discount, send exactly one RMS entry with
configured discountTypeId, saved total discount in rubles and selectivePositions from saved eligibility. Include
eligible lines with zero allocated discount; exclude ineligible lines. For zero total discount omit the RMS entry.
Never recompute eligibility, prices or discounts from live catalog/campaigns. iiko may allocate the discount
differently across eligible lines; require the exact order discount and food payment, not identical line amounts.
Food payment equals discounted food total; delivery keeps existing comment treatment. Include recorded promo
code, discount and payable total in staff Telegram messages. Keep retry IDs/idempotency and diagnostic privacy.

Document restaurant setup: real RMS ID, assign-amount/FlexibleSum, selective application, payment compatibility,
both fulfillment modes and absence of settings that change the required totals. Keep campaigns disabled until
explicitly authorized deployment acceptance confirms the resulting iiko totals. Do not send live orders/messages.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestPromoIikoPayload: pickup/delivery fixtures, binding cap, overlapping targets and uneven quantities → exact total
  discount/payment; original prices/quantities unchanged; only saved eligible positions selected, including zero
  allocation lines. Zero total discount omits RMS. Do not require identical iiko/backend line allocations.
- TestPromoIikoRetry: stable external and position IDs throughout retries; original snapshots after campaign edits.
- TestPromoStaffMessage: code, discount, food and delivery totals agree with persisted order; no-code text unchanged.
- TestPromoConfiguration: missing/invalid ID, new promo order rejection before writes, no-code compatibility,
  and existing pending retries retaining their snapshots.
- TestPromoIntegrationFailure: existing failure/retry behavior preserved; no silent undiscounted fallback.
Use httptest for iiko/Telegram and existing integration helpers.

## API documentation
Document the promo configuration 503 error for POST /orders in API.md and OpenAPI; keep existing wire shapes.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
