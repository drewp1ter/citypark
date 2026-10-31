# Task 43: Promo codes in shared checkout

## Goal
Add authoritative promo pricing to the shared website and Mini App checkout.

## Depends on
website42; backend89

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](../../../citypark-go/docs/tasks/PROMOCODES.md)
- `src/widgets/food-ordering/ui/checkout-form.tsx`
- `src/widgets/food-ordering/api/place-order.ts`

## Files to create
- `src/widgets/food-ordering/api/quote-order.ts`
- `src/widgets/food-ordering/ui/promo-code-field.tsx`
- `scripts/qa/flows/promo-checkout.mjs`

## Files to change
- `src/widgets/food-ordering/api/orders-api.ts`
- `src/widgets/food-ordering/api/place-order.ts`
- `src/widgets/food-ordering/model/schema.ts`
- `src/widgets/food-ordering/ui/checkout-form.tsx`

## Spec
Add authoritative promo pricing to the shared website and Mini App checkout.
Use a Server Action for quotes and existing backend helpers; browsers never call citypark-go directly. Keep
credentials and ownership/session handling on the server. Read installed Next.js docs for affected APIs.
Implement apply/replace/remove and recalculation behavior from the shared contract. Show food subtotal,
discount, delivery and payable total and explicit delivery-fee change message. Send promoCode and last reviewed
expectedPayableTotal on submit. Use server pricing as authority; do not locally calculate campaign discounts.
Ignore stale asynchronous responses. Retain code text on invalidation; require removal or a successful fresh
quote before submission. Keep existing guest/account/Mini App payment-return and retry context intact.
Use tokens, Russian labels, accessible status/errors and 44px controls; both themes at 390 and 1440px.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- promo-checkout.mjs: apply/remove/replace, malformed/expired/minimum code, overlap and cap totals, town/cart changes,
  delivery fee appearing after discount, pickup, delayed response races, failed quotes and changed-total 409.
- Exercise website and mini routes using appropriate sessions; no real payment redirects or provider writes.
- Run existing checkout-and-contacts.mjs and telegram-mini.mjs regressions.
Run node scripts/qa/flows/promo-checkout.mjs against a production build with existing backend fixtures.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run npm run check and npm run build. Run the named QA scripts against the production build using
BASE_URL (default http://localhost:3111). Do not start the backend/database. Tick only this task after checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
