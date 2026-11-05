# Task 55: Saved-address model and guest storage

## Goal
Provide a validated, versioned guest address book that survives reloads and can be shared by management, checkout and later import.

## Depends on
Website 54; backend 97's contract must be complete before implementation. Do not modify backend files.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `../citypark-go/docs/API.md` — structured customer addresses.
- `src/entities/cart/model/cart-store.ts`
- `src/widgets/food-ordering/model/schema.ts`

## Files to create
- `src/entities/address/index.ts` — browser-safe public API.
- `src/entities/address/model/types.ts`
- `src/entities/address/model/schema.ts`
- `src/entities/address/model/format.ts`
- `src/entities/address/model/guest-store.ts`
- `src/entities/address/model/use-guest-addresses.ts`
- `scripts/qa/address-store-harness.mjs`

## Files to change
- `src/widgets/food-ordering/api/place-order.ts` — consume the entity's formatAddress for the existing order payload; preserve identical output and ordering behavior.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Export these types/functions through the entity barrel; never export server-only code there:
```ts
type AddressFields = { name: string; city: string; street: string; apartment: string; entrance: string }
type SavedAddress = AddressFields & {
 id: number; address: string; isDefault: boolean; needsReview: boolean; createdAt: string; updatedAt: string
}
type GuestAddress = AddressFields & { id: string; revision: string; isDefault: boolean }
type GuestBook = { version: 1; revision: string; entries: GuestAddress[] }
type AddressSurface = 'web' | 'mini'
type GuestStatus = 'loading' | 'ready' | 'corrupt' | 'unsupported' | 'unavailable'
type GuestResult = { ok: true; entry?: GuestAddress; created?: boolean } | { ok: false; message: string }
function formatAddress(fields: AddressFields): string
function addressDuplicateKey(city: string, fullAddress: string): string
function useGuestAddresses(): { status: GuestStatus; book: GuestBook }
// Async so later serialization/import can use them without changing callers.
function saveGuestAddress(fields: AddressFields, id?: string): Promise<GuestResult>
function deleteGuestAddress(id: string): Promise<GuestResult>
function setGuestDefault(id: string): Promise<GuestResult>
function removeImportedGuests(snapshot: readonly { id: string; revision: string }[]): Promise<GuestResult>
```
Export `addressFieldsSchema` with backend-aligned messages/limits: name 0–40, city 1–80, street 3–120,
apartment/entrance 0–20 Unicode code points, all boundary-trimmed. Use the exact Russian field messages
in the backend contract. Format matches checkout: street, optional `кв. X`, optional `подъезд Y`, comma-separated.
Duplicate key is JSON of `[normalizedCity, normalizedFullAddress]`: NFC, collapse Unicode White_Space,
trim and lowercase. Preserve display spelling. Unit/entrance/punctuation differences remain meaningful.
Use matching backend fixtures, including Cyrillic case, NBSP and composed/decomposed й.
Replace the existing inline address join in place-order.ts with formatAddress (pass an empty optional name).
This gives the new entity an actual production consumer immediately, without enabling saving or changing
checkout behavior. Do not suppress Steiger rules for an otherwise unused slice.

Persist only GuestBook at `citypark.addresses.v1`. New entry/book revisions and IDs use crypto.randomUUID;
an entry's revision changes when its fields/default change. Do not store account addresses, sessions,
phones, order comments or coordinates. Validate the whole envelope, uniqueness of IDs and at most one
default. Reads must be stable useSyncExternalStore snapshots with a deterministic empty/loading server
snapshot; no SSR localStorage access or default-prefill before hydration readiness.

Create duplicate returns existing entry/created=false without replacing name/default/revision. An edit
colliding with another entry returns `Такой адрес уже сохранён`. First entry in an empty book becomes
default; adding to a nonempty book with no default does not choose one. Delete default leaves none.
Missing edit/default ID: `Адрес больше недоступен`; deleting an already missing ID succeeds.

Re-read the latest stored envelope before each mutation. On browser storage events (including clear),
refresh subscribers without dispatching a write. Use a single mutation path for normal operations and
import cleanup. Cleanup removes only acknowledged entries whose id AND revision still match; preserve
concurrent edits/new entries and do not promote a replacement default. LocalStorage is not a transactional
multi-tab database: do not claim fully conflict-free simultaneous writers; verify observable stale revisions
and sequential cross-tab edits. Do not silently overwrite a newer schema.

Malformed current-version data: show `Не удалось прочитать сохранённые адреса`; preserve raw data until
an explicit `Сбросить сохранённые адреса` confirmation. Unsupported version: read-only with
`Обновите страницу, чтобы открыть сохранённые адреса`; no reset/overwrite by an older client.
Blocked/quota storage: `Не удалось сохранить адрес в этом браузере`; return failure, keep form input,
and never report persistence success. Read failure status is unavailable. Reset is an explicit exported
`resetGuestAddresses(): Promise<GuestResult>` operation for corrupt v1 only; cancel is a no-op.

## Tests to write
- `guestSchemaAndFormatting`: limits/Unicode trimming/duplicate-key parity; exact messages and formatted text.
- `guestCrudAndDefault`: create, duplicate, rename/edit, collision, default switch, delete, missing IDs, reload.
- `guestHydration`: server snapshot is stable/loading; first client read does not mismatch SSR.
- `guestStorageFailures`: malformed JSON, invalid shape, duplicate IDs, multiple defaults, unsupported version,
  blocked read/write/quota; no silent overwrite, false success or dropped form values; explicit reset only.
- `guestCrossTab`: other-tab update and clear refresh snapshots; mutations read latest known data.
- `guestImportCleanup`: matched acknowledgments removed; changed revisions and unacknowledged/new entries retained.
- `guestPrivacy`: only declared guest fields are serialized; no account/session data.

Use a Node harness loading TypeScript via the existing QA harness pattern; fake storage only for isolated
failure tests. Browser persistence integration is task 56/59; these tests do not claim real-backend acceptance.

## API documentation
Consume the backend contract; no endpoint changes.

## Done when
`node scripts/qa/address-store-harness.mjs`, `npm run check` and `npm run build` pass. Tick 55 and commit
`task 55: persist guest delivery addresses`. No backend writes or database startup.

## Out of scope
Account transport/editor extraction (55b), screens (56), checkout (57), login import (58), deployment.
