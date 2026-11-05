# Task 58: Import guest addresses after login

## Goal
Offer explicit, duplicate-safe import after authentication and retain unsent/changed guest data through failures.

## Depends on
Website 57 and backend 98. Do not modify backend files.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/entities/address/api/server.ts`
- `src/entities/address/model/guest-store.ts`
- `src/app/layouts/mini-layout.tsx`

## Files to create
- `src/features/import-addresses/index.ts`
- `src/features/import-addresses/api/actions.ts`
- `src/features/import-addresses/model/import-state.ts`
- `src/features/import-addresses/ui/import-addresses.tsx`
- `src/app/addresses/address-import-bridge.tsx`
- `scripts/qa/address-import-harness.mjs`

## Files to change
- `src/app/layouts/site-layout.tsx`, `src/app/layouts/menu-layout.tsx`, `src/app/layouts/checkout-layout.tsx`,
  `src/app/layouts/mini-layout.tsx` — mount one current-surface bridge per active layout.
- `src/pages/addresses/ui/addresses-page.tsx`, `src/pages/mini/ui/mini-addresses-page.tsx` — manual import composition.
- `src/features/manage-addresses/ui/address-manager.tsx` — optional ReactNode slot only; never import another feature.
- `scripts/qa/flows/my-addresses.mjs` — auth/import/session scenarios.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Read installed Next.js session/cookie/layout/revalidation docs. The bridge resolves the current surface's
profile server-side and passes account ID/context to the client feature. Render only after authenticated
account state is known and guest storage is ready/nonempty. Include password and Google login returning
to arbitrary website pages (including menu and checkout), and mini relaunch. Do not hook only password
sign-in or expose tokens. Auth/profile failures quietly suppress the offer without breaking page content;
management exposes its usual retry error. Session mutation/logout must refresh the bridge; do not assume
cached layouts automatically re-run on every navigation. Never let web and mini bridges mount together.

Prompt: `Добавить сохранённые адреса в аккаунт?`; buttons `Добавить адреса`, `Не сейчас`.
Manual trigger in account address management: `Добавить адреса из этого браузера`.
Compose the import feature from pages/app through a slot; no imports between feature slices.
Use one client coordinator per surface to avoid duplicate automatic prompts between layout/manual instances.

Dismissal key is surface + account ID + guest-book revision in sessionStorage. Rejection keeps all guest
entries. Suppress the same revision in the same session; a new revision permits a new offer. Manual import
ignores suppression. If sessionStorage is unavailable use an in-memory dismissal marker; do not block checkout.
Logout/account switch clears private in-memory responses and cancels/ignores queued work from the old context.

Export `importAddressAction(surface, expectedAccountId, fields)` from this feature's API module; it delegates
to entity createSavedAddress with fresh session/profile verification and returns validated `{ entry, created }`
or a typed error. It accepts only one AddressFields object, not a user-selected backend URL, credentials or
bulk user ID. The coordinator handles snapshot/IDs/revisions locally; they are never ownership proof.

On acceptance snapshot the guest entries and their revisions; send requests sequentially, guest default
first, then original order. The backend automatically makes the first address default only if the account
book is empty, so an existing account default (or intentional absence in a nonempty book) is preserved.
Duplicate responses increment skipped count and never overwrite account names/defaults. A needsReview
duplicate is still acknowledged but show `Некоторые адреса нужно уточнить в разделе «Мои адреса»`.

After every acknowledged create/duplicate, remove that guest entry only if its revision still matches
using removeImportedGuests. If local cleanup fails, keep data and report failure; retrying is safe because
backend create is duplicate-safe. Lost responses are not acknowledgments. Do not clear the entire storage key.
On request/validation/auth failure stop the sequence, preserve failed/unattempted/edited entries, and offer
`Повторить` / `Не сейчас`. A response from an old account/surface must never clean up local data or update
the new account UI. A request already committed before logout cannot be undone; discard stale UI results.

Success status: `Добавлено: N. Уже сохранено: M.` (literal numeric counts, including zero).
Partial failure: `Не все адреса добавлены. Оставшиеся адреса сохранены в этом браузере.` Show completed
counts and specific auth/validation error without address text in logs. Refresh the active account list
after acknowledgments. On retry build a fresh remaining snapshot, recheck account context, and recompute
counts for that attempt. Complete import naturally leaves an empty guest book; newly edited entries survive.

## Tests to write
- `importPromptEligibility`: guest/no data/loading/errors suppress; password/Google/mini authenticated returns offer once.
- `importDismissal`: same account/revision suppressed, other account/new revision allowed, manual trigger and blocked storage.
- `importDuplicatesAndDefaults`: account duplicate metadata wins; empty book guest default first; nonempty no-default preserved.
- `importPartialFailure`: failure at first/middle/last request, response loss, local cleanup failure; only acknowledged unchanged
  entries removed, idempotent retries and exact created/skipped counts.
- `importConcurrentGuestEdit`: changed revision/new entry survives acknowledgment cleanup.
- `importSessionIsolation`: logout, expired session, account switch, web/mini mismatch and late responses do not write/clean up
  the wrong book; future requests stop immediately.
- `importAccessibility`: keyboard focus, live progress/errors, pending buttons and both themes.

## API documentation
No new backend endpoint. QA header documents session setup and consent/import scenarios.

## Done when
`node scripts/qa/address-import-harness.mjs`, `npm run check`, `npm run build` and expanded production
`node scripts/qa/flows/my-addresses.mjs` pass. Follow AGENTS.md for owner-run live mutation/auth setup;
record missing Google/native-device acceptance as a blocker, never a mocked pass. Tick 58 and commit
`task 58: import guest addresses after login`.

## Out of scope
Silent imports, merging unrelated accounts, cross-device guest sync, bulk API, deployment.
