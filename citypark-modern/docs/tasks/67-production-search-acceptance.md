# Task 67: Production search acceptance

**Status: Skipped for now at the owner’s request (2026-11-05).** Acceptance has not passed. Resume when the owner returns to this task after production deployment; retain the evidence below.

## Goal
Verify the deployed search preparation and record the owner's Webmaster/indexing handoff, distinguishing technical checks from asynchronous search outcomes.

## Depends on
64–66 and deployment of the intended frontend to https://cityparkvip.ru. Webmaster operations need the owner's account and genuine verification artifact or existing valid verification method.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/YANDEX_WEBMASTER.md`, `docs/SEARCH_DISCOVERY.md`, `src/shared/config/search.ts`.

## Files to create
None required; add focused acceptance automation only if existing HTTP checks do not cover a necessary production condition.

## Files to change
Record dated evidence in `docs/SEARCH_DISCOVERY.md`; update runbook operational statuses only with evidence; leave task progress unchecked until required acceptance is satisfied or the owner explicitly accepts a documented manual handoff.

## Spec
Read-only checks against the intended public deployment: HTTPS and HTTP/www redirect behavior, correct production canonical URLs, indexable public pages, noindex on account/checkout/payment/Mini App routes, valid robots and sitemap responses, and structured data/content matching the backend serving that same deployment. Reuse the four HTTP scripts documented in the runbook; do not compare production with an unrelated local backend. Record deployment identity/date and sample room/news URLs from current public data. Confirm a missing content route returns 404. Record the owner's verification/sitemap processing/indexing evidence and Alice visibility baseline or explicit no-data status. Search inclusion or citations are not guaranteed pass conditions. Do not register, change DNS, replace production, submit to Webmaster or create an invented verification artifact automatically.

## Tests to write
Reuse `sitemap-http.mjs`, `public-content-http.mjs`, `business-json-ld-http.mjs` and `customer-questions.mjs` with production BASE_URL and its matching backend. Record public headers/redirects/canonicals/private noindex and missing-content status through read-only requests. A successful HTTP status with a fallback HTML shell is not proof of sitemap/robots correctness.

## API documentation
No API changes.

## Done when
Production HTTP acceptance and the remaining public metadata/redirect/404 checks pass; owner operational evidence or explicitly accepted manual handoff is recorded. If implementation changes are required, `npm run check` and `npm run build` also pass. Stop and record a blocker if production is still serving a different application.

## Out of scope
Unrequested deployment or live infrastructure changes, backend mutations, automated owner-account operations and promises of search inclusion.

## Preflight blocker recorded before skipping (2026-11-05, 22:12 UTC)
Production serves the prior application's identical 1,788-byte HTML shell at /, /contacts, /robots.txt and /sitemap.xml. Responses are HTTP 200 text/html, title «РГК „Сити Парк“», without the new Next.js assets, FAQ or business JSON-LD. An existing Yandex verification meta tag is present; its account/ownership status is unknown. See the dated task 67 evidence in SEARCH_DISCOVERY.md. Deployment and verification-method continuity must be addressed before resuming acceptance.
