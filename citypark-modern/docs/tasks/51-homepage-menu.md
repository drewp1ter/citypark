# Task 51: Homepage category switching and curated dishes

## Goal
Homepage visitors switch categories in place and see dishes selected by staff.

## Depends on
website50; backend92. Admin31 supplies the editor.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/home/ui/home-page.tsx`, `src/pages/home/ui/delivery-section.tsx`, `src/entities/dish/api/menu-api.ts`.

## Files to create
- `src/entities/dish/lib/select-homepage-menu.ts`
- `src/pages/home/ui/delivery-showcase.tsx`, `src/pages/home/ui/dish-teaser.tsx`
- `scripts/qa/flows/homepage-menu.mjs`

## Files to change
- Home page, server delivery wrapper and dish public API/fetcher.

## Spec
Fetch `/homepage-menu` server-side without caching; failures use empty configuration. Resolve raw IDs against the live visible menu.
Empty configuration uses first four categories. Empty or wholly invalid manual selection uses up to three category dishes sorted by availability then photos, stable menu order for ties. Never top up valid manual choices.
Omit hidden/deleted categories, deleted/moved dishes and categories without any dishes. Hide teaser if no groups remain.
Selected blocked/out-of-hours dishes stay visible with existing ordering behavior. Preserve cart behavior and refresh menu data on focus and every 60 seconds.
Client category buttons switch in place on both viewports, expose pressed state, preserve focus and scroll position. Mobile categories scroll horizontally. First category is selected initially; fall back to first group if selection disappears.
Server wrapper resolves editable copy, working hours and delivery facts, passing only serializable props and a server-rendered full-menu card into the client boundary.

## Tests to write
Pure resolver: defaults, ranking, stable ties, exact manual order, partial/all stale choices, empty/hidden categories, blocked dishes and moved products.
Browser: category switching and keyboard, URL/scroll stability, manual order, cart persistence, both themes at 390/1440px, API failure fallback; existing menu/cart regression.

## Done when
Conventions checks/build, homepage-menu and menu regression flows pass against existing services. Restore any authorized temporary config writes.

## Out of scope
Checkout, payment and Mini App interface changes.
