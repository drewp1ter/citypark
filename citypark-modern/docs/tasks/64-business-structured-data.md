# Task 64: Hotel and restaurant structured data

## Goal
Describe the two businesses in server-rendered JSON-LD using facts already visible on the contacts page.

## Depends on
63.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/pages/contacts/ui/contacts-page.tsx`, `src/shared/config/site-settings.ts`, `src/shared/config/search.ts`.

## Files to create
- `src/pages/contacts/model/business-json-ld.ts`: typed serializer.
- `scripts/qa/business-json-ld-harness.mjs`: isolated value/security checks.
- `scripts/qa/business-json-ld-http.mjs`: production HTML acceptance.

## Files to change
ContactsPage, task progress and search discovery release notes.

## Spec
Export `businessJsonLd(settings: SiteSettings, hours: WorkingHours): string`. Render its result once in a native application/ld+json script on /contacts, using the exact settings/hours objects already fetched for visible content. Emit Schema.org @context and a two-node @graph, typed Hotel and Restaurant. IDs are https://cityparkvip.ru/hotel#business and /restaurant#business; URLs are the corresponding canonical pages. Names use site.name. Both nodes include PostalAddress street/locality/postcode, their respective tel link number without the tel: prefix, and the shared email. Restaurant openingHours uses the existing daily start/end as Mo-Su HH:MM-HH:MM. Omit hotel hours, ratings, availability, offers and unverified claims. Preserve existing page copy and fallback behavior. Escape every less-than character as a JSON Unicode escape so backend text cannot close the script tag. No new dependencies or API changes.

## Tests to write
- Normal values: parseable JSON, correct types/IDs/URLs, distinct phones and exact address/email/hours.
- Changed settings and overnight hours: output reflects passed values without hardcoded contact data.
- Empty strings: valid JSON preserving the passed fallback/empty values; no invented defaults.
- Malicious text including closing-script markup, quotes and Unicode: exact round trip, no literal less-than characters, one script in the HTML parser and no injected element.
- Production HTTP: exactly one JSON-LD block on /contacts, fields match the live backend and visible text/links (brand name matches the visible logo accessible label) after removing scripts; existing contacts audit still passes.

## API documentation
No HTTP API changes.

## Done when
`node scripts/qa/business-json-ld-harness.mjs`, `npm run check`, `npm run build`, `node scripts/qa/business-json-ld-http.mjs` and `node scripts/qa/public-content-http.mjs` pass. HTTP scripts use BASE_URL (default http://localhost:3112) and the existing backend, read-only.

## Out of scope
New factual copy (65), Webmaster setup (66), external validator submission and public deployment acceptance (67).

## Results
All Done when commands passed. Production HTML contains exactly one graph and matches live backend contact values and visible content. The existing public-content audit passes for 5 rooms, 24 categories and 246 dishes. Existing dynamic-rendering build diagnostics remain; public deployment checks are pending.
