# Search discovery release plan

Production: https://cityparkvip.ru. Deployment is pending. Owner registers with Yandex Webmaster later.

Owner instructions: [Yandex Webmaster runbook](YANDEX_WEBMASTER.md).

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

## Task 64 structured data (2026-11-05)

The contacts page renders one native JSON-LD block with Schema.org Hotel and Restaurant nodes. Stable identifiers are https://cityparkvip.ru/hotel#business and https://cityparkvip.ru/restaurant#business; each points to its canonical page. Both use the shared brand name, postal address, email and respective telephone number. Restaurant openingHours follows the existing shared daily-hours model already displayed on the contacts page. No ratings, offers, live availability, geographic coordinates or unverified business claims are emitted.

The serializer receives the same fetched settings/hours objects as the visible contacts, including existing fallback values. It escapes less-than characters to prevent backend text from closing the script element. This preserves the existing fallback behavior and its freshness limitations; it does not establish a new source of business facts.

Validation passed: isolated normal/changed/empty data and malicious-text round-trip checks; typecheck, lint and FSD checks; production build; production HTTP agreement with live backend values, visible contact text/links and the logo accessible name; the task 63 raw-HTML audit (5 rooms, 24 categories, 246 dishes). Existing dynamic-rendering diagnostics remain during build. No live backend writes. External validator submission, deployed-site verification and Yandex inclusion remain pending.

Commands: `node scripts/qa/business-json-ld-harness.mjs`, `node scripts/qa/business-json-ld-http.mjs`, `node scripts/qa/public-content-http.mjs`. HTTP checks use BASE_URL (default http://localhost:3112) and the configured live backend.

References: [Schema.org Hotel](https://schema.org/Hotel), [Restaurant](https://schema.org/Restaurant); implementation follows the installed Next.js JSON-LD guide at node_modules/next/dist/docs/01-app/02-guides/json-ld.md.

## Task 65 customer questions (2026-11-05)

Added five always-visible answers at /contacts#questions: address, guest parking, restaurant/delivery hours, hotel details/prices/rules and the delivery menu. Mutable address/phone/hour values come from the existing contact-page fetches. Parking repeats the exact reference copy. Existing hotel policies and variable delivery charges are linked rather than copied into new fixed promises.

Source review and unresolved owner questions are in [CUSTOMER_QUESTIONS.md](CUSTOMER_QUESTIONS.md). Further answers about delivery cutoff/total time, cancellation/refunds, booking conditions, parking security/capacity, accessibility and allergen handling require verified owner facts and remain unpublished. No backend writes or API changes.

Validation: production build, check, raw-HTML answer/value/link checks, 390/1440px browser checks in both themes without website JavaScript, keyboard focus, no horizontal overflow and single-line phone links. Screenshots reviewed in /tmp/citypark-task65. Existing business JSON-LD HTTP acceptance passes. Existing dynamic-rendering diagnostics remain during build; production deployment/indexing acceptance remains pending. Run `node scripts/qa/customer-questions.mjs` against a production build (BASE_URL defaults to http://localhost:3112).

## Task 66 owner runbook (2026-11-05)

Prepared [YANDEX_WEBMASTER.md](YANDEX_WEBMASTER.md) using the current official Yandex instructions and repository configuration. It covers production readiness, the real HTML verification file and its byte-for-byte response check, standalone public-file deployment, sitemap submission, indexing troubleshooting and Alice visibility evidence. All operational steps remain pending; no account, domain, deployment, verification artifact or Webmaster submission was changed. Task 67 is the deployment/owner handoff gate, not completed by writing these instructions.

Task 66 validation: `npm run check`, `npm run build`, local-link/configuration review and `git diff --check` passed. Existing dynamic-rendering diagnostics remain during build. All runbook operational statuses remain Pending.

## Task 67 production preflight — blocked (2026-11-05, 22:12 UTC)

Read-only HTTPS requests confirm that production is reachable, but the intended frontend is not deployed. All four responses below contain the same prior-application HTML shell:

| URL | HTTP status | Content-Type | Body length |
|---|---|---|---|
| https://cityparkvip.ru/ | 200 | text/html | 1,788 bytes |
| https://cityparkvip.ru/contacts | 200 | text/html | 1,788 bytes |
| https://cityparkvip.ru/robots.txt | 200 | text/html | 1,788 bytes |
| https://cityparkvip.ru/sitemap.xml | 200 | text/html | 1,788 bytes |

Each response body has SHA-256 `a95a277da0639cbabf2db17e2db8acc1db2e4eedc75a9e4cea9e061c14a56123`. The shell contains the old title, a client application root and an existing Yandex verification meta tag. The responses lack the intended Next.js assets, contacts FAQ and business JSON-LD. Additional /hotel, /rooms and /menu requests also returned the prior shell. The robots and sitemap URLs therefore fail their required formats despite returning 200.

The existing verification tag does not establish which account owns the site or whether verification remains valid. Before replacing the old frontend, the owner should check existing Webmaster access and retain the active verification method, or establish the genuine replacement method described in the runbook. No verification value was copied into the new app.

Task 67 is skipped for now at the owner’s request (2026-11-05); acceptance has not passed. Resume when the owner returns to it after the intended frontend is deployed with correct production configuration and the matching backend is available for read-only comparison. Redirect, canonical/private-noindex, missing-content 404, production content/JSON-LD, Webmaster processing and Alice visibility acceptance remain unrun; an earlier local test pass cannot substitute for them. All runbook operational statuses remain Pending. No deployment, DNS/settings changes, Webmaster submissions or backend writes were performed. No application code was changed; this is blocker evidence, not a completed release check.
