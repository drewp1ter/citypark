# Search discovery release plan

Production: https://cityparkvip.ru. Deployment is pending. Owner registers with Yandex Webmaster later.

## Prioritized queue

- 61 (P0): canonical metadata, staging/private noindex.
- 62 (P0; depends on 61): sitemap from public static routes and existing backend room/news lists; exclude account/auth/checkout/payment/Mini App URLs, missing content and query variants. Add sitemap URL to robots. Never treat a failed backend fetch as an empty successful sitemap. Check valid XML, URL uniqueness, exclusions and backend failure behavior; check/build required.
- 63 (P1; depends on 62): audit raw server HTML for contacts, hours, menu/prices and room content; fix inaccessible public facts using current data sources. Add focused HTTP checks; check/build required.
- 64 (P1; depends on 63): Hotel/Restaurant JSON-LD using visible verified facts, stable IDs, canonical URLs and safe serialization. Omit invented ratings/availability. Validate JSON and visible-data consistency; check/build required.
- 65 (P1; depends on 63): review delivery/hotel/parking questions, list missing owner facts, publish only verified Russian answers with existing components. Check/build required for code changes.
- 66 (P2; depends on 62): write Webmaster runbook: add production URL, obtain HTML verification file, put unchanged file in public, deploy, verify and retain file; submit sitemap and inspect indexing/Alice visibility. No placeholder verification token.
- 67 (P2; depends on deployment and 64–66): public HTTPS/redirect/canonical/sitemap/noindex checks. Owner then registers and supplies verification file; track indexing errors and Alice visibility baseline. Indexing/citations are monitored outcomes, never guaranteed.

Create each detailed task using TASK_TEMPLATE.md before beginning it; execute one numbered task per session.

## Deployment acceptance

Keep SITE_INDEXING_ENABLED=false on staging. Enable true only on the public production deployment and rebuild. Check initial HTML for canonical and robots tags on public pages and noindex on account, checkout/payment returns and Mini App routes. Check room/news canonicals with actual backend identifiers. Missing content must still return 404. Until verified on the deployed site these checks remain pending.

Yandex documentation: https://yandex.ru/support/webmaster/ru/service/quick-start and https://yandex.ru/support/webmaster/ru/service/rights.html.

## Task 63 audit results (2026-11-05)

Production-build HTTP acceptance passed against the configured live backend: contact address/email/company/phone links, restaurant hours on contacts/home/menu, public restaurant/hotel main content, all 5 room listings and detail names/prices/descriptions, and all 24 linked menu category responses with 246 dish descriptions/prices. Category descriptions are checked wherever supplied. No rendering fixes were needed.

The acceptance script fetches HTML and uses an inert DOMParser, stripping scripts/styles/templates before checking main content; React payloads cannot satisfy text assertions. It does not execute website JavaScript, submit forms or modify backend data. Nonempty catalogs and successful backend responses are required. Run `node scripts/qa/public-content-http.mjs` against a production build using BASE_URL (default http://localhost:3112).

Limitations: this proves server HTML availability, not mobile visibility of every responsive duplicate or search inclusion. Menu categories are reachable through ordinary query links and render independently; they currently canonicalize to /menu, so category-specific indexing is not established by this audit. Existing fallback hours/contacts may become stale during backend failures; this audit requires a healthy backend and checks current values. Public deployment checks and Yandex indexing/visibility remain pending.
