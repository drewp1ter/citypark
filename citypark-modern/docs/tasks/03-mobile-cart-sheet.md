# Task 03: Cart sheet on the mobile menu

## Goal
Below 1024 px, a tap on the cart bar on `/menu` opens a bottom sheet with the cart: lines with steppers, delivery,
total and «Оформить заказ». Today the bar goes straight to `/checkout`, so a phone user never sees the cart.

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/widgets/cart-summary/ui/cart-bar.tsx`, `src/widgets/cart-summary/ui/cart-panel.tsx`,
`src/widgets/cart-summary/ui/use-priced-cart.ts`, `src/widgets/site-header/ui/mobile-menu.tsx` (the native
`<dialog>` pattern), `src/features/add-to-cart/ui/add-to-cart-button.tsx`, `src/app/styles/globals.css`
(`body:has(dialog[open])`), `scripts/qa/flows/menu.mjs` (the "mobile:" checks).

## Files to create
- `src/widgets/cart-summary/ui/cart-sheet.tsx`: `CartSheet` (Spec 2).

## Files to change
- `src/widgets/cart-summary/ui/cart-bar.tsx`: the bar opens the sheet (Spec 1).
- `design/tokens.css`, `src/app/styles/tokens.css`, the `@theme` block: the `--scrim` token (Spec 2).
- `scripts/qa/flows/menu.mjs`: update the mobile checks (Spec 3).

## Spec

### 1. `CartBar`
- Keep the look, the `nav aria-label="Корзина"` and the `count === 0 → null` rule.
- The `AppLink` becomes a `<button type="button">` with the same classes plus `w-full cursor-pointer`,
  `aria-haspopup="dialog"`, `aria-expanded={open}`. A click calls `showModal()` on the sheet's dialog.
- The right pill text changes from `{price} ↗` to `{price}` + `ChevronDownIcon` from `@/shared/ui` with `rotate-180`
  (points up), `aria-hidden`.
- Render `<CartSheet>` next to the button, passing `dishes`, a dialog ref and `onClose` that sets `open=false`.
- If the cart becomes empty while the sheet is open, close the dialog (the bar unmounts at 0).

### 2. `CartSheet` (client, not exported from the slice's `index.ts`)
- A native modal `<dialog aria-labelledby>` like `MobileMenu`: Esc closes, focus is trapped, the page does not scroll.
  A click on the backdrop (the dialog element itself, `event.target === dialog`) closes it. Add `lg:hidden` so a
  resize to desktop never shows it.
- Placement: bottom sheet, `fixed inset-x-0 bottom-0 top-auto m-0 w-full max-w-none max-h-[85dvh] overflow-y-auto
  rounded-t-xl bg-inverse-bg p-5 pb-6 text-inverse-text backdrop:bg-scrim`. There is no backdrop colour yet: add a
  `--scrim` token (semi-transparent dark, for both themes) to `design/tokens.css` and `src/app/styles/tokens.css`, and
  expose it in `@theme` as `--color-scrim`, like the other tokens. No hex colours in components.
- Respect `prefers-reduced-motion` (only add a slide-up transition under
  `motion-safe:`; no animation is fine too).
- Content, same tokens and copy as `CartPanel`:
  - Header row: `h2` «Корзина» (id for `aria-labelledby`) + `{count} блюд(о/а)` + close `IconButton`
    `aria-label="Закрыть корзину"` with `CloseIcon`.
  - Lines from `usePricedCart(dishes)`: photo 48 px, name (+ ` · {variant.label}` when the dish has sizes),
    line total, and the stepper `AddToCartButton dish={item.dish} variantId={item.variant.id}` (layout `"icon"`),
    which changes the quantity and removes the line at 0.
  - «Доставка» row from `quoteDelivery({ method: "delivery", zone: "city", subtotal })`, «Итого» row,
    `Button href="/checkout"` «Оформить заказ» (closes the sheet on click), and the phone line «или 8 918 311-97-91».
- Do not duplicate `CartPanel` by copy-paste where a small shared piece is obvious (e.g. the totals); but do not
  change `CartPanel`'s markup or the desktop look.
- Widgets may import from `features` (`@/features/add-to-cart`); check with Steiger.

### 3. QA in `menu.mjs`
- `mobile: cart bar`: read the bar button (`nav[aria-label='Корзина'] button`) instead of `a[href='/checkout']`;
  expected text unchanged except the arrow.
- New: tap the bar → `out("mobile: sheet", <text of the dialog>)` (expected: the 3 seeded lines, Итого, Оформить заказ);
  press `+` in the first line → the total in the sheet and on the bar grow; Esc → `dialog` not open, focus back on the bar.
- New: tap «Оформить заказ» in the sheet → URL is `/checkout`.
- Tablet 900 check: use the bar button locator; keep everything else.

## Tests to write
No unit-test runner; check by hand at 390 px and 900 px, light and dark: sheet opens and closes (button, Esc,
backdrop), the page behind does not scroll, a long cart scrolls inside the sheet, steppers work, removing the last
portion closes the sheet, keyboard focus is trapped and returns to the bar, touch targets ≥ 44 px.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs`, `npm run qa:a11y`, `npm run qa:keyboard` and `npm run qa:responsive` print no
unexpected values.

## Out of scope
The header cart button on other pages, the desktop `CartPanel`, `/checkout`, and any change in `../citypark-go/`.

