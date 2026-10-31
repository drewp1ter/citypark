# Task 45: Promo-code visibility and automatic recalculation

## Goal
Implement the global promo-code controls and checkout behavior described below, preserving existing orders.

## Depends on
Backend task 90 and website task 44.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/widgets/food-ordering/model/use-checkout-quote.ts
- `src/widgets/food-ordering/ui/promo-code-field.tsx`
- `src/widgets/food-ordering/ui/checkout-form.tsx``

## Files to create
None expected; extend existing implementation and tests.

## Files to change
The implementation files listed above, their existing tests, and the supporting files explicitly named in the spec.

## Spec
- Add promoCodesEnabled to ApiSiteSettings, SiteSettings, conversion and fallback defaults (false).
- Load settings alongside menu/towns/hours in website and Mini App checkout pages and pass the flag to the shared CheckoutForm. Keep existing settings cache (60 seconds) and page refresh behavior; backend enforcement is authoritative between refreshes.
- When disabled, hide the entire promo section, remove the surface-specific saved session code, and quote/submit an empty code. Invalidate previously discounted totals immediately. Missing settings default false.
- Replace explicit apply with a trailing 500 ms debounce on normalized promo text (existing trim/ASCII-uppercase rules). Persist the settled code. Clearing through typing or Remove recalculates immediately. Preserve enabled-code restoration after payment return.
- Remove the Apply/Recalculate button and its onApply prop. Keep Remove. Enter in the promo field prevents checkout submission; automatic calculation handles it.
- Invalidate old totals and block submission immediately during editing, debounce and requests. Remove the explicit-apply error. Ignore obsolete responses, cancel obsolete timers on changes/unmount, and prevent periodic/cart/delivery refreshes from quoting an old code during typing. Cart/delivery changes without an active typing delay still quote immediately.
- Keep loading feedback and accessible errors. Render general quote failures outside the conditional promo section so disabled promo UI cannot hide a checkout failure. Backend rejection after a switch change remains visible until settings refresh or the user removes the code.
- Retain authoritative totals, expected-total validation, ownership and payment-return behavior. Read installed Next.js documentation before changing its APIs.

## Tests to write
- Extend scripts/qa/flows/promo-checkout.mjs: both surfaces hide disabled fields, clear persisted codes and submit no code; enabled fields restore and apply automatically.
- Verify one settled request after rapid typing, no request before 500 ms, immediate clearing/removal, normalized codes, ignored out-of-order responses, and submission blocked throughout editing/requesting.
- Cover cart/town changes during debounce, invalid codes, transport failure, backend disabling between requests and existing expected-total conflict refresh.
- Update existing QA locators that click Apply/Recalculate; preserve desktop/mobile and light/dark coverage. Authenticated Mini App tests require supplied sessions; record unavailable checks honestly.

## API documentation
Consume backend task 90; do not change backend files.

## Done when
```bash
npm run check
npm run build
# Run the production server at BASE_URL, then:
node scripts/qa/flows/promo-checkout.mjs
node scripts/qa/flows/checkout-and-contacts.mjs
```
All checks required by CONVENTIONS pass. Backend changed-package statement coverage is at least 80% where applicable. Use configured services; do not start the backend/database. Live QA writes require existing session authorization; otherwise document the exact pending checks.

## Out of scope
Other projects, unrelated task completion, individual campaign redesign, retroactive repricing, real payments and iiko dispatch. On completion, tick this task and commit according to the project workflow.
