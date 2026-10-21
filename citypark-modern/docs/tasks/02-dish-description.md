# Task 02: Dish description on the menu card

## Goal
The dish card on `/menu` shows `dish.composition` (the product description from the backend) at every screen size,
clamped to 2 lines, also for dishes with a size switch. Today it is hidden below 768 px.

## Depends on
01. Real texts need backend task 40 (`../citypark-go`) and a menu sync; without them every description is empty and
the card looks as today.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/menu/ui/dish-card.tsx`, `scripts/qa/flows/menu.mjs`.

## Files to create
None.

## Files to change
- `src/pages/menu/ui/dish-card.tsx`: the description (Spec 1).
- `scripts/qa/flows/menu.mjs`: one new check (Spec 2).

## Spec

### 1. `DishCard`
- When `dish.composition` is set, render it as its own `<p>` right under the `h3`, in both branches (with and without
  `SizeSwitch`). Classes: `mt-0.5 line-clamp-2 break-words text-[13px] text-text-muted md:mt-0 md:text-sm`
  (same size and colour as today's text). `line-clamp-2` is built into Tailwind v4.
- Remove the `hidden md:inline` span and the ` · ` separator. The weight stays in its own `<p>` in the no-sizes branch,
  with today's classes.
- The element right above the price keeps `flex-1`, so prices line up across the cards of one row. With an empty
  weight, the weight `<p>` renders no text but stays as the spacer.
- Tokens only; no new colours, no new props.

Result (no sizes):
```
<h3>Салат свекольный</h3>
<p class="… line-clamp-2 …">Свекла, чеснок, грецкий орех</p>   ← only when composition is set
<p class="… flex-1 …">220 г</p>
```

### 2. QA check in `menu.mjs`
After the existing desktop checks, and again at 390 px (`M`):
- Fetch `${process.env.MENU_API_URL ?? "http://localhost:3000/api/v1"}/products?pageSize=500` and take the first product
  with `isAvailableForOrder` and a non-empty `description.trim()`.
- If there is none, print `out("description on card", "skipped: no product has a description")` and go on.
- Otherwise open its category (`/menu?category=<categoryId>`), find the `article` whose `h3` text is the product name,
  and print `out("description on card <size>", …)` with `<text of "h3 + p"> / lines <= 2: <true|false>`, where the line
  check is `clientHeight <= 2 * parseFloat(getComputedStyle(el).lineHeight) + 1`. Expected: the start of the description
  and `true` at both sizes.
- Keep every other check and the output format.

## Tests to write
The project has no unit-test runner; do not add one. Check by hand with the backend running and synced:
- `/menu` at 390 px and 1440 px, light and dark: descriptions are visible, at most 2 lines with `…`, prices of one row
  are on one line, a card without a description looks as today.
- A dish with a long word (or a temporary long `composition` in dev tools) does not overflow the mobile card.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs` and `npm run qa:responsive` print no unexpected values.

## Out of scope
A "more" button or a modal with the full text, showing the description anywhere else (home page, cart, checkout),
and any change in `../citypark-go/`.
