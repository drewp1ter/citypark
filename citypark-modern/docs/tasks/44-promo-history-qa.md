# Task 44: Promo history and cross-surface QA

## Goal
Display saved promo breakdowns and verify the complete customer flow.

## Depends on
website43; admin28

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](../../../citypark-go/docs/tasks/PROMOCODES.md)
- `src/pages/order-history/ui/order-detail-page.tsx`
- `src/pages/mini/ui/mini-order-page.tsx`

## Files to create
- `scripts/qa/flows/promo-history.mjs`

## Files to change
- `src/entities/order/model/types.ts`
- `src/entities/order/api/orders.ts`
- `src/pages/order-history/ui/order-detail-page.tsx`
- `src/pages/mini/ui/mini-order-page.tsx`

## Spec
Display saved promo breakdowns and verify the complete customer flow.
Extend the existing shared order response adapter/types used by these pages with backend snapshot fields;
render original subtotal, discount/code, discounted food, delivery and final total consistently. Do not derive
historical discounts from current campaigns or current menu prices. Existing no-code history stays unchanged.
Verify website guest/account and mini authenticated checkout, payment return and history. Provider checks are
manual release acceptance; automated checks use existing test mechanisms. Document any device/session/provider
checks not run instead of claiming them complete. Run all prerequisite checks before campaign activation.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- promo-history.mjs: saved discounted order after campaign edit/disable, legacy/no-code order, pickup/delivery,
  account ownership isolation, website and mini rendering, line/header arithmetic and dark/light mobile/desktop.
- Re-run promo-checkout.mjs, checkout-and-contacts.mjs and telegram-pairing-payment.mjs.
- Release checklist: MONETA payable includes delivery; iiko food payment equals discounted food; notifications and
  both order histories match. Record actual results and required manual checks without creating real provider orders.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run npm run check and npm run build. Run the named QA scripts against the production build using
BASE_URL (default http://localhost:3111). Do not start the backend/database. Tick only this task after checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
