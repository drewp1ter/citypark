# Task 55b: Address API adapters and reusable fields

## Goal
Provide an explicit-surface server adapter and reusable address controls so management and checkout share one contract.

## Depends on
Website 55 and backend 97. Do not modify backend files.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `../citypark-go/docs/API.md` — customer addresses/auth errors.
- `src/shared/customer/customer.ts`
- `src/widgets/food-ordering/ui/address-field.tsx`

## Files to create
- `src/entities/address/api/server.ts` — server-only reads/writes shared by feature actions and order action.
- `src/entities/address/ui/address-field.tsx` — moved suggestion control.
- `src/entities/address/ui/address-fields.tsx` — controlled city/street/apartment/entrance inputs.
- `scripts/qa/address-api-harness.mjs`

## Files to change
- `src/entities/address/model/types.ts`, `src/entities/address/model/schema.ts` — result/context types, response validation.
- `src/entities/address/index.ts` — export browser-safe controls only.
- `src/widgets/food-ordering/ui/checkout-form.tsx` — import the moved AddressField through the entity API.
- `src/widgets/food-ordering/ui/address-field.tsx` — remove after all callers migrate.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Read installed Next.js docs for server-only boundaries before implementation. Keep customerRequest's
object-only response validation; new endpoints have envelopes. The browser must never receive a backend
URL or bearer token. Export from the separate `@/entities/address/api/server` entry point:
```ts
type AddressContext =
 | { kind: 'guest'; surface: 'web' }
 | { kind: 'account'; surface: AddressSurface; accountId: number }
 | { kind: 'auth-required'; surface: AddressSurface }
type AddressBookLoad = { context: AddressContext; entries: SavedAddress[]; error?: string }
async function loadAddressBook(surface: AddressSurface): Promise<AddressBookLoad>
async function createSavedAddress(surface: AddressSurface, expectedAccountId: number, fields: AddressFields): Promise<{ entry: SavedAddress; created: boolean }>
async function updateSavedAddress(surface: AddressSurface, expectedAccountId: number, id: number, fields: AddressFields): Promise<SavedAddress>
async function setSavedDefault(surface: AddressSurface, expectedAccountId: number, id: number): Promise<SavedAddress>
async function deleteSavedAddress(surface: AddressSurface, expectedAccountId: number, id: number): Promise<void>
```
Use only getCustomerSessionToken for web, getMiniSessionToken for mini; no fallback. No token means guest
only for web, auth-required for mini. With a token, fetch `/customer/me` using that exact token and validate
the profile; then fetch the address envelope. A 401 is auth-required, not guest. Non-auth failures show
`Не удалось загрузить адреса. Попробуйте ещё раз.` and must not silently reinterpret the visitor as guest.
When profile lookup fails before ownership is known, use auth-required context with the load error;
account writes stay unavailable until revalidation. A list failure after valid profile keeps account context.

Mutations re-read the same surface session/profile and compare expectedAccountId to the authenticated
profile before writing. This client-supplied value is a stale-context guard, never backend ownership authority.
Mismatch/missing/expired session → `Войдите повторно, чтобы сохранить адрес`; no write/fallback. IDs are
positive safe integers. Validate fields and all response shapes, including created flag, Entry IDs/types,
timestamps/review flags and array envelope. Forward no unknown request properties.

Known backend validation/conflict errors retain their documented messages. Malformed/network/500 failures
use `Не удалось сохранить адрес. Попробуйте ещё раз.` (delete/default use the same generic failure).
Do not catch redirect exceptions here; these helpers do not redirect. No raw addresses in console/Sentry.

Move existing AddressField behavior intact: DaData same-origin request, debounce/cancellation, manual
fallback, accessibility and token styling. AddressFields accepts controlled AddressFields values,
onChange, delivery towns, suggestionsEnabled, field errors and onManualEdit. It renders only city/street/
apartment/entrance; name belongs to management/checkout save UI. Avoid importing entities/dish from
entities/address: accept a structural readonly array of `{ name: string }` town options.
Keep labels `Населённый пункт`, `Улица и дом`, `Квартира / офис`, `Подъезд, этаж`.
Keep missing old town visible for editing; consumers decide delivery availability. Selecting/programmatically
filling values must not be misclassified as a manual edit that discards the selected address.

## Tests to write
- `addressSurfaceIsolation`: web/mini credentials never cross; no token, expired token, changed account fail safely.
- `addressAPIContract`: all verbs/paths/body envelopes/created flag and every response-validation failure.
- `addressAPIErrors`: field errors, conflict, 401/404/503/network/500; no writes on failed ownership guard.
- `addressAPIPrivacy`: no URL/token/address text in client output or captured logs.
- `addressFieldRegression`: suggestion/manual keyboard behavior and geolocation flow remain unchanged after extraction.

Harness tests may stub transport only for isolated failure coverage, never as a substitute for live integration.

## API documentation
No backend changes; consume docs/API.md exactly. Keep `SavedAddress` synchronized with its Entry schema.

## Done when
Run `node scripts/qa/address-api-harness.mjs`, `npm run check`, `npm run build` and the existing
`node scripts/qa/flows/checkout-geolocation.mjs` against the production build. Read the QA script setup;
do not start the backend or send live mutations. Tick 55b and commit `task 55b: share saved address API and fields`.

## Out of scope
Management screens, checkout saving, post-login prompt, new UI dependencies and deployment.
