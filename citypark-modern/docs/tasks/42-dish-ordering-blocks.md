# Task 42: Visible blocked dishes and checkout protection

## Goal
Keep manually blocked dishes visible without extra text or purchase controls, and prevent ordering blocked cart items.

## Depends on
41; citypark-go task 81 must be implemented and deployed for live checks.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/dish/lib/price-lines.ts`, `src/widgets/food-ordering/api/place-order.ts`, `src/widgets/cart-summary/ui/use-priced-cart.ts`.

## Files to create
- `scripts/qa/flows/dish-ordering-blocks.mjs`: targeted regression flow.

## Files to change
- `src/entities/dish/api/menu-api.ts`, `src/entities/dish/api/get-menu.ts`, `src/entities/dish/api/to-dish.ts`, `src/entities/dish/model/types.ts`, `src/entities/dish/index.ts`: carry manual-block state and support fresh availability reads.
- `src/entities/dish/lib/price-lines.ts`: retain blocked cart lines and expose checkout-blocked state instead of silently dropping them.
- `src/features/add-to-cart/ui/add-to-cart-button.tsx`, `src/widgets/cart-summary/ui/cart-line.tsx`: hide add/increment controls for blocked dishes while retaining removal.
- `src/widgets/cart-summary/ui/use-priced-cart.ts`, `src/widgets/food-ordering/ui/checkout-form.tsx`, `src/widgets/food-ordering/ui/menu-catalog.tsx`: availability refresh and cart/checkout behavior.
- `src/widgets/food-ordering/api/place-order.ts`: fresh validation and authoritative backend error handling.

## Spec
Read installed Next.js docs before editing Next.js behavior. Map `isOrderBlocked` explicitly; do not conflate manual blocks with schedule closures. Blocked dishes retain normal content, images and prices; add no unavailability text or badges. Hide add/increment controls entirely, including on existing cart rows. Keep removal available and preserve cart entries in localStorage until removed by the customer. Keep displayed quantities/totals consistent with visible retained lines.

A cart containing a blocked dish cannot submit an order. On an attempted checkout, use the existing form-error mechanism: `Удалите недоступные блюда из корзины, чтобы оформить заказ.` Do not add permanent availability labels to dish/cart displays. Never silently submit a reduced cart.

Refresh availability on cart/checkout entry, window focus and every 60 seconds while visible; clean up timers/listeners. Fresh requests must bypass the existing 60-second menu cache when needed. Preserve cart and form input on refresh errors, and fail submission if fresh validation cannot complete. Revalidate on the server immediately before POST `/orders`; surface backend rejection if availability changes afterward. Apply shared behavior to the public website and Mini App. Unblocking restores controls only if existing schedule/price rules also allow ordering.

## Tests to write
The targeted flow covers visible blocked dishes with no extra text or add/increment controls, persisted blocked cart entries, removal, attempted checkout error without order creation, no silent omission, unblock recovery, focus/timed refresh and failed refresh preserving input. Check mobile/desktop and both themes, plus shared Mini App ordering behavior. Follow existing QA gating: live backend writes require owner authorization; document setup/cleanup and do not mutate the live backend without it.

## Done when
`npm run check`, `npm run build`, the targeted production-build flow, and existing menu/checkout flows pass against configured existing services. Run relevant responsive/keyboard checks. Do not start backend/database services. Update PROGRESS and commit this task separately.

## Out of scope
Backend/admin implementation, new availability badges, automatic removal of cart items, automatic block expiration, and cancellation of existing orders/payments.
