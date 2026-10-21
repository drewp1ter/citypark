# Task 04: Quantity steppers in the desktop cart panel

## Goal
From 1024 px, each line in the cart panel on `/menu` gets the same «−  N  +» stepper as the mobile cart sheet, so the
quantity can be changed (and a line removed) without scrolling back to the dish card.

## Depends on
03.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/widgets/cart-summary/ui/cart-panel.tsx`, `src/widgets/cart-summary/ui/cart-sheet.tsx`,
`src/widgets/cart-summary/ui/use-priced-cart.ts`, `src/features/add-to-cart/ui/add-to-cart-button.tsx`,
`scripts/qa/flows/menu.mjs` (the "desktop" part).

## Files to create
- `src/widgets/cart-summary/ui/cart-line.tsx`: `CartLine` (Spec 1).

## Files to change
- `src/widgets/cart-summary/ui/cart-sheet.tsx`: render lines with `CartLine` (Spec 1).
- `src/widgets/cart-summary/ui/cart-panel.tsx`: render lines with `CartLine` (Spec 2).
- `scripts/qa/flows/menu.mjs`: desktop checks (Spec 3).

## Spec

### 1. `CartLine` (client, not exported from the slice's `index.ts`)
- Props: `item` (one element of `usePricedCart(...).items`; export its type from `use-priced-cart.ts` if needed).
- Move the `<li>` markup from `CartSheet` here unchanged: photo 48 px, name (+ ` · {variant.label}` when the dish has
  more than one size), line total under the name, and
  `<AddToCartButton dish={item.dish} variantId={item.variant.id} />` on the right.
- `CartSheet` maps `items` to `<CartLine key=… item={item} />`; its look must not change.

### 2. `CartPanel`
- Lines use `CartLine` instead of the current `× {quantity}` + price markup. The panel is 340 px wide (292 px
  inside), so the price goes under the name, as in the sheet; long names wrap (`min-w-0` on the name).
- Keep everything else: header, count, empty state, «Доставка», «Итого», «Оформить заказ», phone, sticky position,
  `hidden … lg:flex`.
- «−» on the last portion removes the line; when the cart becomes empty the existing empty text shows.
- Check both themes: the stepper is `bg-inverse-bg` on an `bg-inverse-bg` panel. If it does not stand out enough,
  that is the same as in the sheet; do not add new colours.

### 3. QA in `menu.mjs` (desktop part)
- The existing click `getByRole("button", { name: "Добавить порцию" }).last()` must keep pressing the dish card's
  stepper: scope it to `p.locator("article")`.
- `panel` output: the line text now shows the stepper quantity instead of `× N`; update the expected text.
- New: press «Добавить порцию» in the first line of `aside#cart` →
  `out("panel + -> total / header pill", …)` (both grow by the dish price).
- New: press «Убрать порцию» in the first line of `aside#cart` until the line is gone →
  `out("panel − -> lines", …)` (one line fewer; the dish card's button is the round «+» again).

## Tests to write
No unit-test runner; check by hand at 1024 px and 1440 px, light and dark: the steppers fit, names wrap, keyboard
Tab reaches «−» / «+» in the panel with a visible focus ring, touch targets ≥ 44 px, the mobile sheet looks the same
as before.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs`, `npm run qa:a11y`, `npm run qa:keyboard` and `npm run qa:responsive` print no
unexpected values.

## Out of scope
The header cart pill, `/checkout`, the mobile `CartBar`/`CartSheet` behaviour, and any change in `../citypark-go/`.
