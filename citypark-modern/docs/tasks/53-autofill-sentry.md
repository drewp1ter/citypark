# Task 53: Address autofill Sentry diagnostics

## Goal
Report all unsuccessful browser and reverse-geocoding autofill attempts with useful diagnostics and no customer location data.

## Depends on
52

## Read first
- AGENTS.md and docs/tasks/CONVENTIONS.md
- docs/SENTRY.md
- src/widgets/food-ordering/model/use-checkout-location.ts
- src/app/api/address-geolocate-handler.ts

## Files to create
- Shared monitoring autofill reporter and field allowlist; typed provider errors.
- Browser and controlled-provider QA scripts.

## Files to change
- Browser location hook, checkout surface option, server geolocation handler and provider parser.
- Shared Sentry filter, existing monitoring/geolocation QA and documentation.

## Spec
Capture browser permission denial, unavailable position, timeout, unsupported API, request failures and invalid responses. Capture server validation/configuration failures, provider network/timeout/HTTP/JSON/schema failures, empty results, unsupported region, absent/ambiguous town matches, delivery-list/matching failures and partial addresses. Preserve the API response contract and manual entry behavior. The server owns recognized route failures; the browser owns transport/location failures. Ignore late callbacks after manual takeover or unmount and avoid duplicate capture after effect replay.

Group by fixed reason and stage. Allow only fixed enums, bounded numeric timing/status/count values and secure-context flags through beforeSend. Never include coordinates, address/locality text, customer information, credentials, raw provider responses or arbitrary error messages. Keep source frames and trace identifiers.

## Tests to write
Extend privacy and handler tests, exercise the real provider parser against controlled responses, and run browser failure/lifecycle scenarios with intercepted Sentry envelopes. Confirm browser/server ingestion and original TypeScript source frames through Sentry MCP.

## API documentation
Document diagnostic behavior in docs/PLAN.md and docs/SENTRY.md. No API contract changes.

## Done when
- npm run check
- node scripts/qa/sentry.mjs
- node scripts/qa/autofill-provider.mjs
- npm run build
- node --env-file=.env scripts/qa/flows/checkout-geolocation.mjs
- node --env-file=.env scripts/qa/autofill-browser.mjs
- Confirm browser/server events in Sentry and record direct issue links.

## Out of scope
Fixing the underlying provider/location failure, changing address matching, live orders/payments and production deployment. Native Telegram/device permission behavior remains a manual device check; the shared hook is verified on the website.
