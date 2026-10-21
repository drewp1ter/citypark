# Task 05: Solid menu toolbar and category arrows

## Goal
On `/menu`, the sticky toolbar (search + category chips) stops looking like a black strip that floats over the dishes,
and the chips row loses its native horizontal scrollbar. Mouse users scroll the chips with «‹» / «›» arrows instead.

## Depends on
None.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/menu/ui/menu-catalog.tsx`, `src/widgets/site-header/ui/site-header.tsx`,
`src/shared/ui/chip.tsx`, `src/shared/ui/icon-button.tsx`, `src/shared/ui/icons.tsx`, `src/app/styles/globals.css`,
`scripts/qa/flows/menu.mjs` (the "sticky toolbar" part), `design/reference/dark/desktop/menu.html` (lines 18–60).

## Background
- From 1280 px the menu-page header is `xl:bg-transparent`, so dishes show between the header pill and the toolbar.
  The toolbar has `bg-bg`, so it reads as a separate black (white in light theme) stripe.
- The search box and the chips are two sticky elements in a grid with `md:gap-x-2`. The dishes show through that
  8 px gap.
- The chips `<nav>` has `overflow-x-auto`, so browsers draw a scrollbar under the chips.

## Files to create
- `src/pages/menu/ui/category-tabs.tsx`: `CategoryTabs`, a client component (Spec 2). Do not export it from the
  page's `index.ts`.

## Files to change
- `src/widgets/site-header/ui/site-header.tsx`: solid header on the menu page (Spec 1).
- `src/pages/menu/ui/menu-catalog.tsx`: close the gap; render the chips through `CategoryTabs` (Spec 1, 2).
- `src/app/styles/globals.css`: `no-scrollbar` utility (Spec 2).
- `src/shared/ui/icons.tsx` (+ `index.ts`): `ChevronLeftIcon`, `ChevronRightIcon`, only if `ChevronDownIcon` can't be
  rotated with a class (Spec 2).
- `scripts/qa/flows/menu.mjs`: new checks (Spec 3).

## Spec

### 1. One solid top area
- Menu-page header: remove `xl:bg-transparent`, so the header keeps `bg-bg` at every width. The header's bottom edge
  must meet the toolbar's top edge with no gap at 390, 768, 1024, 1280 and 1440 px (the toolbar is `top-[74px]`, and
  `md:top-[84px]` from 768 px). If they don't meet at some width, change the header's bottom padding at that
  breakpoint (`pb-*` / `xl:pb-*`). Do not move the toolbar's `top` values; the cart panel and the QA expect them.
- Other pages keep the transparent header; only the `menuPage` branch changes.
- Close the gap between the search box and the chips from 768 px: the toolbar background must cover it (for example,
  `md:-ml-2 md:pl-2` on the chips' sticky wrapper). The search box and the chips must not move.
- Both themes: the top area is one block of `bg-bg` from the top of the window to the bottom of the toolbar.

### 2. `CategoryTabs` (client)
- Props: `categories`, `selectedId` (the current category id). It renders the current `<nav aria-label="Разделы меню">`
  with the same `Chip` links (`href`, `scroll={false}`, `selected`, `className="px-4"`). Keep the nav's sticky
  classes, its `bg-bg`, and its padding. `MenuCatalog` renders `<CategoryTabs … />` in place of the nav.
- Scrollbar: add `@utility no-scrollbar` to `globals.css` (`scrollbar-width: none;` and
  `&::-webkit-scrollbar { display: none; }`, with a one-line comment like the other utilities). Put it on the
  scrolling element. Touch swipe and trackpad scrolling must keep working.
- Arrows, from 768 px only (below 768 px the row is swiped):
  - «‹» at the left edge when the row can scroll left, «›» at the right edge when it can scroll right. Neither shows
    when all chips fit. Recompute on scroll and on resize (`ResizeObserver` on the scroller).
  - A click scrolls by 80 % of the visible width; `behavior: "smooth"`, but `"auto"` under
    `prefers-reduced-motion: reduce`.
  - Use `IconButton variant="surface"` (44 px). Use `ChevronDownIcon` turned with `rotate-90` / `-rotate-90` if it
    accepts `className`; otherwise add `ChevronLeftIcon` / `ChevronRightIcon` next to it.
  - Labels: `aria-label="Прокрутить разделы влево"` / `"…вправо"`. Give both `tabIndex={-1}` and `aria-hidden`:
    keyboard and screen-reader users already reach every chip with Tab, so the arrows must not add Tab stops.
  - The arrows sit over the row's ends (absolutely positioned in a `relative` wrapper), vertically centred. They
    must not change the row's height.
- Edge fade, from 768 px: when the row can scroll in a direction, the chips fade out at that edge (about 48 px). Use
  a CSS mask (`mask-image` with a transparent → black gradient, or Tailwind v4's `mask-*` utilities). A mask needs
  no colour, so no new tokens. No fade at an edge with nothing more to scroll.
- On first render, scroll the selected chip into view within the row (set the scroller's `scrollLeft`). Do not use
  `scrollIntoView`; it can scroll the page too.
- Keyboard: when Tab focuses a chip that is partly hidden, the browser scrolls it into view. Check that the focus
  ring is not cut off by the fade or covered by an arrow. If it is, add `scroll-padding-inline` / `scroll-px-*` on the
  scroller so the focused chip stops clear of the arrows.

### 3. QA in `menu.mjs`
In the "sticky toolbar" part (desktop, 1440 px, after the scroll to 700):
- `out("toolbar: header bottom / chips top (px)", …)`: `header` `getBoundingClientRect().bottom` and the nav's
  wrapper top (the sticky element). The two numbers must be equal or the header's bottom must be larger.
- `out("chips: scrollbar-width", …)`: `getComputedStyle(scroller).scrollbarWidth` → `none`.
- `out("chips: arrows at start", …)`: whether «‹» / «›» are visible → `false / true` when the chips overflow at
  1440 px; print `scrollWidth > clientWidth` with it so a short menu explains `false / false`.
- Click «›», wait 600 ms: `out("chips: after ›", …)`: `scrollLeft > 0` and «‹» visible → `true / true`.
In the mobile part (390 px):
- `out("mobile: chip arrows", …)`: both arrows hidden → `false / false`.

## Tests to write
No unit-test runner; check by hand at 390, 768, 1024, 1280 and 1440 px, light and dark:
- scroll the dish list: no dish shows between the header and the toolbar, or between the search box and the chips;
- no scrollbar under the chips in Chrome and Firefox;
- arrows and fades appear only where there is more to scroll; the arrows work with a mouse;
- Tab through the chips: each one scrolls into view with a visible focus ring;
- `?category=<a late category>`: the selected chip is visible on load.

## Done when
CONVENTIONS "Done means" passes, and with the backend running and a production build on port 3111:
`node scripts/qa/flows/menu.mjs`, `npm run qa:a11y`, `npm run qa:keyboard` and `npm run qa:responsive` print no
unexpected values.

## Out of scope
The header on other pages, the cart panel, the mobile cart bar and sheet, dish cards, the search behaviour, and any
change in `../citypark-go/`.
