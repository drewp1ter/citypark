# Task 28: Promo order breakdown

## Goal
Show stored promo-code details in admin order views.

## Depends on
admin27; backend89

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](../../../citypark-go/docs/tasks/PROMOCODES.md)
- `src/features/orders/ui/order-details.tsx`
- `src/features/orders/api.ts`

## Files to create
- `e2e/promo-orders.spec.ts`

## Files to change
- `src/features/orders/ui/order-details.tsx`
- `src/shared/api/schema.d.ts`

## Spec
Show stored promo-code details in admin order views.
Render original food subtotal, code, discount, discounted food total, delivery and amount paid/payable from the
backend snapshot. Show line discounts in details; do not subtract discount twice from total. Preserve the
existing presentation for orders without codes. Reuse the existing formatRub helper. Keep live order totals
consistent if they use the same response; do not implement campaign editing inside order details.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- e2e/promo-orders.spec.ts: promo and legacy fixtures, capped discount, pickup and delivery, immutable display after
  campaign edit, exact payable reconciliation and both themes at mobile/desktop sizes.
Run npm run e2e -- e2e/promo-orders.spec.ts; use existing test payment mechanisms, never real charges.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run npm run api:gen, npm run check, npm run build, and the named npm run e2e spec against the
configured existing backend. Never start the backend/database. Tick only this task after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
