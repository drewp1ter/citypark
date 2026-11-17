# Task 74: Hide dishes marked hidden in iiko

## Goal
Omit hidden dishes from every customer menu consumer, including the website and Mini App.

## Depends on
Backend 108. Explicit owner-requested fix independent of hotel reservation task 73.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/entities/dish/api/menu-api.ts`, `src/entities/dish/api/get-menu.ts`, `../citypark-go/docs/API.md`.

## Files to create
- `scripts/qa/dish-visibility-harness.mjs`

## Files to change
- `src/entities/dish/api/menu-api.ts`: add the boolean `isHidden` response field.
- `src/entities/dish/api/get-menu.ts`: filter hidden products before converting them to dishes.

## Spec
Fetch all product pages before filtering. Keep dishes only when `!isHidden` and their category is visible.
The shared getMenu seam covers homepage selection, delivery menu, dish pages, checkout, refreshes and Mini App.
Hidden IDs resolve as missing dishes; existing dish-page handling returns 404, and existing priceLines excludes stale cart IDs.
Restore dishes on the next fresh menu read after iiko clears the flag and the backend sync succeeds.
Manual ordering blocks, category visibility and saved homepage selections retain their existing behavior.

## Tests to write
Run an isolated TypeScript harness against the actual menu API/getMenu/mapping/selection/pricing modules.
Verify true/false/missing flags, hidden categories, multiple pages, homepage manual selections, stale carts,
ID lookups and visibility restoration. This isolated harness does not replace real-backend acceptance.

## API documentation
Consume backend 108's additive Product.isHidden field. No new website endpoint.

## Done when
`node scripts/qa/dish-visibility-harness.mjs`, `npm run check`, `npm run build` pass.
Read-only real-backend smoke checks for menu and dish pages; report if the running backend needs the new build.

## Out of scope
Admin changes, backend changes, live iiko writes/syncs, orders, payments and deployment.
