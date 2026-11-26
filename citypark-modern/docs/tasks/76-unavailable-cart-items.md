# Task 76: Make unavailable cart items removable

## Goal

Customers can identify and remove unavailable dishes from the cart or checkout and continue with the remaining dishes.

## Depends on

Task 74.

## Read first

- `docs/tasks/CONVENTIONS.md`
- `src/entities/dish/lib/price-lines.ts`
- `src/widgets/cart-summary/ui/cart-sheet.tsx`
- `src/widgets/food-ordering/ui/checkout-form.tsx`

## Files to create

- `src/features/remove-unavailable-items/index.ts` and `src/features/remove-unavailable-items/ui/unavailable-items.tsx` for the shared recovery controls.
- `scripts/qa/flows/unavailable-cart.mjs` for read-only browser acceptance.

## Files to change

- `src/entities/dish/lib/price-lines.ts` and `src/entities/dish/index.ts` classify and expose unavailable lines.
- The menu layout, site header and cart summary show stored-cart counts and support all-unavailable carts.
- The food-ordering quote and submit actions return unavailable item keys. The checkout form and quote hook display them and discard stale results.
- The dish visibility, checkout address and checkout browser checks cover the new behavior.

## Spec

- Treat items with a missing or hidden dish, missing variant or price, schedule closure, or manual block as unavailable. Exclude them from totals and prevent checkout or payment until every item is orderable.
- Show «Удалите недоступные блюда, чтобы продолжить.» and a «Удалить» button for each affected line on desktop and mobile, including the Mini App. Use the public menu name when present and «Недоступное блюдо» otherwise.
- Remove the exact dish and variant from the existing cart store. Recalculate checkout for the remaining items and retain form values. Show the empty state after removal of the last line.
- The server checks the full submitted cart. If its menu is newer, return affected keys for that cart snapshot. Refresh the menu and show removal controls immediately. Generic backend failures continue through the existing error path.

## Tests to write

- Dish visibility harness covers mixed and all-unavailable carts, restored availability, server quote rejection, and a late response after removal.
- Checkout address harness checks that an unavailable item prevents dispatch and address saving on both surfaces while retaining entered values.
- Browser flow covers 390 px and 1440 px in light and dark themes, keyboard removal, all-unavailable and mixed carts, recalculation, and reload persistence.

## Done when

- `npm run check` and `npm run build` pass.
- `node scripts/qa/dish-visibility-harness.mjs` and `node scripts/qa/checkout-address-harness.mjs` pass.
- `node --env-file=.env scripts/qa/flows/unavailable-cart.mjs` passes against the production build using the configured backend.

## Out of scope

Go API changes, cart storage migration, and live order or payment creation.
