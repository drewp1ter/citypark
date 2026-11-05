# Task 62: Public sitemap

## Goal
Publish a complete sitemap of public pages and advertise it through robots.txt.

## Depends on
61.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/SEARCH_DISCOVERY.md`, `src/entities/article/api/get-news.ts`, `src/entities/room/api/get-rooms.ts`.

## Files to create
- `app/sitemap.ts`: metadata route and dynamic configuration.
- `src/app/search/sitemap.ts`: sitemap generator.
- `src/shared/config/search.ts`: shared production origin.
- `scripts/qa/sitemap-harness.mjs`: isolated logic checks.
- `scripts/qa/sitemap-http.mjs`: read-only production HTTP checks.

## Files to change
Shared config exports, root metadata base, robots sitemap reference and task progress.

## Spec
Use https://cityparkvip.ru for all canonical and sitemap URLs. Include /, /restaurant, /hotel, /rooms, /menu, /news, /contacts, /documents, /offer and /privacy. Read rooms and all paginated news through existing entity getters; include only returned identifiers as encoded path segments. Deduplicate URLs. Exclude all account/auth/checkout/payment/Mini App/API routes and query variants by using this explicit public allowlist. Do not invent last-modified dates. Generate on request while retaining existing entity fetch caches. Propagate list fetch errors rather than returning a successful partial sitemap. No backend writes or changes.

## Tests to write
Harness: empty lists retain static pages; duplicate room IDs yield one URL; unusual slugs are encoded; no private routes or query strings; backend errors reject. Production HTTP: XML response has sitemap namespace, matches backend room/news identifiers and robots advertises production URL.

## API documentation
No backend API changes. New public website route: GET /sitemap.xml.

## Done when
`node scripts/qa/sitemap-harness.mjs`, `npm run check`, `npm run build`, and `node scripts/qa/sitemap-http.mjs` against a production server on BASE_URL (default http://localhost:3112) pass.

## Out of scope
HTML content audit (63), structured data (64), Webmaster registration (owner later).
