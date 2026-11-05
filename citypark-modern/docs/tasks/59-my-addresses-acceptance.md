# Task 59: Saved-address integration acceptance and release checklist

## Goal
Verify the complete guest/account/checkout/import feature against a production build and document release prerequisites.

## Depends on
Website 55–58 including 55b; backend 96–98 including 96b/96c. All must be checked first.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `scripts/qa/flows/my-addresses.mjs`
- `scripts/qa/flows/checkout-geolocation.mjs`
- `../citypark-go/docs/tasks/MY_ADDRESSES.md`

## Files to create
- `docs/MY_ADDRESSES_RELEASE.md` — reproducible setup, acceptance results and deployment/rollback prerequisites.

## Files to change
- `scripts/qa/flows/my-addresses.mjs` — complete scenario matrix and explicit assertions/cleanup instructions.
- `scripts/qa/address-store-harness.mjs`, `scripts/qa/address-api-harness.mjs`,
  `scripts/qa/address-management-harness.mjs`, `scripts/qa/checkout-address-harness.mjs`,
  `scripts/qa/address-import-harness.mjs` — fill only identified coverage gaps.
- `docs/tasks/PROGRESS.md` — completion with evidence and remaining owner-run checks, if any.

## Spec
Do not implement missing product behavior silently in a QA-only task. If acceptance finds a defect, record
the failing case and affected earlier task in PROGRESS.md and stop per conventions; do not weaken a test.
Do not change backend files. Never mark unavailable checks as passed.

Use BASE_URL (default http://localhost:3111) against `npm run build` plus the existing production server
pattern. Read-only backend comes from configured MENU_API_URL; do not start a replacement backend/DB.
Use isolated non-production accounts whose sessions the owner supplies securely. Do not commit credentials
or print tokens/addresses. Under AGENTS.md live setup/mutation requires the owner to run or explicitly
authorize the concrete commands. Script header must list setup/cleanup, environment variable names and
which stages write address data. No implicit credential guessing, real payments or order notifications.

Required matrix (named assertions print expected and actual outcomes without private values):
- `guestCRUD`: empty/create/rename/edit/conflict/delete/default, reload, cross-tab refresh, quota/corrupt/
  unknown-version recovery and no silent data reset.
- `accountCRUD`: web and mini ownership isolation; two independent authenticated contexts see persisted
  account changes; account data never written to guest localStorage.
- `sharedLocationSafety`: two users at the same location can independently edit/default/delete; existing
  order snapshots stay unchanged (verify through backend tests/read-only fixtures, not new live orders).
- `legacyReview`: existing raw address remains visible and requires explicit structured correction.
- `checkoutPrefill`: all four fields, defaults once, manual/error values override late location/storage;
  removed delivery town and needsReview entry cannot silently become a valid selection.
- `checkoutSave`: unchecked and pickup have no writes; checked saves before order creation, duplicate
  retry safe; failures preserve fields and allow opt-out. Verify order-call sequencing in the harness,
  and stop browser checkout before any real order/payment side effect.
- `loginImport`: password and Google return paths, prompt once per session/revision, decline/manual retry,
  duplicate skip/counts, default preservation, partial failure, lost response and concurrent guest edit.
- `sessionChanges`: expiry/logout/account switch/web-mini separation; no stale response applied or guest fallback.
- `privacy`: no saved-address fields/names/coordinates in console, monitoring payloads or analytics.
- `responsiveA11y`: management/selector/import in light/dark at 390/1440 px, labels, focus, announcements,
  keyboard completion, 44 px targets and no horizontal overflow.

Keep isolated failure harnesses clearly separate from actual backend integration. Record native Telegram
webview and Google credential requirements in release docs with owner checklist; unavailable requirements
block completion unless the owner explicitly accepts handoff (record that acceptance verbatim/date).

Release order: backend migration/store/API/legacy compatibility/merge changes as one compatible unit;
then website. Never deploy 96 without 98. Pre-release backup/rollback ownership belongs to the operator.
Once structured metadata exists, migration Down intentionally refuses loss; roll back website behavior
without dropping saved-address columns. Document legacy endpoint compatibility and current feature-flag
auth matrix. This task creates a checklist; it does not deploy or request deployment approval.

## Tests to write
The named matrix above is required. Expand existing harnesses only for gaps revealed by integration.
Retain existing menu/cart, checkout-geolocation, customer-auth, Telegram and payment-retry assertions.
Use deterministic time/network fixtures only in isolated tests, never to disguise a failed real dependency.

## API documentation
Verify deployed test backend matches docs/API.md/OpenAPI, including list envelope, created flag and distinct
saved-entry/shared-address IDs. Report mismatches as blockers; do not alter backend documentation here.

## Done when
- All five address harness commands pass.
- `npm run check` and `npm run build` pass with no errors/warnings covered by check.
- Production `node scripts/qa/flows/my-addresses.mjs` passes all available authorized integration scenarios.
- Run `node scripts/qa/flows/menu.mjs`, `node scripts/qa/flows/checkout-geolocation.mjs`,
  `node scripts/qa/payment-retry-harness.mjs`, `node scripts/qa/flows/payment-retry.mjs` and applicable
  customer-auth/Telegram regression flows using their documented setup and side-effect boundaries.
- Release document states exact results, backend task completion, and any explicit owner-accepted manual handoff.
- Tick 59 and commit `task 59: verify saved address flows` only when required acceptance is satisfied.

## Out of scope
Deployment, production address/order mutations, payment callbacks, data cleanup and unrelated refactors.
