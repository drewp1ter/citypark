# Task 88: Promo order snapshots and payments

## Goal
Persist discounts atomically and charge the recorded amount.

## Depends on
87

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/order/types.go`
- `internal/order/store.go`

## Files to create
- `migrations/00030_order_promos.sql`
- `internal/order/promo_test.go`

## Files to change
- `internal/order/types.go`
- `internal/order/store.go`
- `internal/order/service.go`
- `internal/order/handler.go`
- `internal/apidocs/openapi.yaml`
- `docs/API.md`

## Spec
Persist discounts atomically and charge the recorded amount.
Add order campaign ID/code/percentage snapshot, subtotal/discount kopecks and per-order-item discount_kopecks.
Persist per-order-item promo eligibility as well, including eligible lines with zero allocated discount.
Backfill eligibility to false for historical orders. This is an internal snapshot for RMS selectivePositions;
no public eligibility field is required. Never infer it from discount > 0 or current campaign targets.
Backfill historical subtotals from offered prices and quantities; discounts default to zero. Never modify original
OfferedPrice. Use next available migration number; reversal must not silently discard paid discount records.
Add optional promoCode and expectedPayableTotal input; require expected amount for promo orders. Reuse task87
calculation within save's transaction. Validate before insert; mismatch → 409 and no customer/order writes.
Extend duplicate lookup with normalized code while preserving all existing ownership/contact criteria. Look up
existing duplicate before current campaign validity; preserve its price snapshot on retry and retain existing
availability safety checks. Expose subtotal, discount, promoCode and item discount in all order projections.
All payment forms/checks/callbacks use persisted discounted total + delivery. Reject composition edits for
promo orders with the shared 409 message. No changes to callback signatures or ownership rules.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestPromoOrderSnapshot: stored original prices, line allocations, eligibility, code and totals survive campaign edits.
- TestPromoEligibilitySnapshot: overlapping targets qualify once; ineligible lines stay excluded; eligible lines
  with zero allocated discount remain eligible for RMS; historical/no-code orders have false eligibility.
- TestPromoOrderDuplicate: same code retries same order; different/no code does not reuse it; expired existing retry works.
- TestPromoOrderConcurrency: concurrent edits produce one consistent campaign version, never mixed targets/rate.
- TestPromoExpectedTotal: missing value 400; changed price/delivery/discount 409 and no writes; matching value succeeds.
- TestPromoPayment: exact discounted amount accepted; original/wrong amount rejected; repeat callback stays idempotent.
- TestPromoOrderLegacy: old/no-code orders, ownership and blocked-product retry safety remain unchanged.
- TestPromoOrderMutation: add/remove items on promo order → 409 without changes.
Use real database, clock.Fixed and existing payment test helpers; never send provider requests.

## API documentation
Update POST /orders and every order response schema; document monetary semantics, 409 recovery and retry behavior.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
