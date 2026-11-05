# My addresses: acceptance results and release checklist

Status (2026-11-05): automated, guest and authorized web-session acceptance pass. Mini App, login-return, legacy-fixture
and native Telegram acceptance is **not done** and is not claimed. The owner accepted the manual handoff on 2026-11-05 ("i'll check mini manually, let's commit", recorded in
`docs/tasks/PROGRESS.md`). The checks below stay pending with the owner.

## Prerequisites

- Website tasks 55, 55b, 56, 57, 58 and backend tasks 96, 96b, 96c, 97, 98 are all ticked.
- Production build served on `:3111`: `npm run build && npm start -- -p 3111`.
- Read-only backend at `MENU_API_URL` already running. This task starts no backend or database.

## Results (this run)

| Check | Result |
|---|---|
| `address-store`, `address-api`, `address-management`, `checkout-address`, `address-import` harnesses | pass, exit 0 |
| `payment-retry-harness.mjs` | pass, exit 0 |
| `npm run check` (typecheck, eslint, Steiger) | pass, no problems |
| `npm run build` | pass |
| `flows/my-addresses.mjs` (guest, no sessions) | pass: 390/1440 px, light/dark, privacy, 44 px targets; account stages BLOCKED |
| `flows/my-addresses.mjs` authorized (`QA_ADDRESS_WRITES=1 QA_IMPORT_WRITES=1`, two disposable web accounts on the dev backend, owner-approved 2026-11-05) | pass: web owned persistence/edit/default with no guest storage; real-session import prompt/dismiss/manual/logout/navigation and sequential import with counts, cleanup, duplicates and default preservation at 390/1440 px, both themes. Afterwards both accounts had 0 addresses and both sessions were logged out |
| `flows/menu.mjs` | pass |
| `flows/checkout-geolocation.mjs` | pass; Mini App geolocation PENDING |
| `flows/payment-retry.mjs` | pass; owned-order and retry POST PENDING |
| Backend `go test -p 1 ./internal/address/ ./internal/app/ -run 'Address|Book|Merge|Shared'` | pass (test DB on :5433) |

Not run: `customer-auth`, `telegram-mini` and other flows needing owner sessions or provider access.

## Matrix status

`node --env-file=.env scripts/qa/flows/my-addresses.mjs` prints the named matrix.

| Name | Status | Remaining |
|---|---|---|
| guestCRUD | pass | none |
| accountCRUD | partial | web pass with a real session; second account was signed in but the flow has no cross-account assertions; Mini App needs `QA_MINI_SESSION` |
| sharedLocationSafety | partial | backend tests pass (`TestBookOwnerIsolation`, `TestDeleteEntry`, `TestBookConcurrentCreates`); live two-account run |
| legacyReview | blocked | seed legacy/unavailable addresses on a disposable account |
| checkoutPrefill | pass | none |
| checkoutSave | pass | harness sequencing only; no real order or payment |
| loginImport | partial | web import with a real session passes; real password and Google return paths and native relaunch remain |
| sessionChanges | partial | logout/navigation with a real web session pass; expiry and account switch remain |
| privacy | pass | none |
| responsiveA11y | pass | native Telegram keyboard check |

## Owner checklist

Stages that write address data: only with `QA_ADDRESS_WRITES=1` (temporary addresses, removed in `finally`) and
`QA_IMPORT_WRITES=1` (temporary imported rows). Use disposable non-production accounts. Never commit sessions.

1. Mini App: run the authorized flow with `QA_MINI_SESSION` from Telegram (web stages already pass). Optionally add cross-account assertions with the second web session.
2. Verify password and Google login return to `/menu`, `/checkout` and an arbitrary page, and that import is offered once.
3. In native Telegram: relaunch, import prompt, keyboard completion, geolocation permission.
4. Compare the deployed test backend with `docs/API.md`/OpenAPI: list envelope, `created` flag, distinct saved-entry and
   shared-address IDs. Report mismatches as blockers.

## Release order and rollback

- Deploy backend 96, 96b, 96c, 97 and 98 as one compatible unit (migration, store, API, legacy compatibility,
  account merge). Never deploy 96 without 98. Then deploy the website.
- The operator owns the pre-release backup and rollback plan.
- Once structured metadata exists, the migration `Down` intentionally refuses to lose data. Roll back website
  behavior only; do not drop saved-address columns.
- Legacy address endpoints stay compatible (`TestLegacyBookCreate`, `TestLegacyBookDelete`, `TestLegacyBookPermissions`).
- This document is a checklist. It does not deploy or ask for deployment approval.
