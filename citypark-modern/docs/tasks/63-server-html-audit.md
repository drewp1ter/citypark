# Task 63: Public server HTML audit

## Goal
Verify that public business facts can be read from HTTP HTML without executing website JavaScript or interacting with controls. Fix any missing facts using existing data sources.

## Depends on
62.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/menu/ui/menu-page.tsx`, `src/widgets/food-ordering/ui/menu-catalog.tsx`, `src/pages/contacts/ui/contacts-page.tsx`.

## Files to create
- `scripts/qa/public-content-http.mjs`: read-only live-backend HTML acceptance.

## Files to change
Task progress and `docs/SEARCH_DISCOVERY.md` audit results. Change public rendering only if checks reveal a gap.

## Spec
Fetch public pages over HTTP without website JavaScript. Parse HTML with DOMParser; discard scripts, styles and templates so serialized React props cannot pass as content. Compare contacts/address/email and telephone links, restaurant hours, room names/prices/summaries/detail descriptions, and category descriptions and dish names/descriptions/prices to existing backend responses. Crawl menu category links already in HTML; no category click required. Check all room detail links and categories, not only the first. An empty catalog cannot count as coverage. Treat failed backend requests or missing expected text as failures; never mutate backend data. Record coverage and limitations without changing factual copy or introducing hidden crawler-only content.

## Tests to write
Production HTTP script: public pages return 200 and one h1/main; contact facts match settings; hours match working-hours; all rooms link to details with name, price and description; every menu category has a crawlable link and its server HTML contains the expected category text and all dishes with their descriptions and prices. Products in hidden categories are excluded according to existing getMenu behavior.

## API documentation
No API changes.

## Done when
`npm run check`, `npm run build`, and `node scripts/qa/public-content-http.mjs` pass against a production server (BASE_URL defaults to http://localhost:3112). Record actual coverage in the release plan.

## Results
Production HTTP audit passed: 5 rooms, 24 category links and 246 dishes; existing server rendering already exposes the checked facts. Scripts/styles/templates exclusion is checked explicitly. No application rendering changes were necessary. Build passes with the existing dynamic-rendering diagnostics.

## Out of scope
Structured data (64), new factual content (65), changes to canonical strategy, and Webmaster registration.
