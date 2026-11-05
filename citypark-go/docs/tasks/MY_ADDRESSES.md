# My addresses — implementation plan

Status: scope confirmed on 2026-11-05; executable task files created in both projects. No implementation or deployment has started.

## Confirmed outcome

Customers can manage delivery addresses in «Мои адреса» and reuse them at checkout.
Guests store addresses in browser `localStorage`; signed-in customers store them on the backend.
Both support names such as «Дом» and «Работа», editing, deleting and a default address.
Checkout prefills the default, permits another saved address or manual entry, and offers an initially
unchecked «Сохранить адрес» checkbox. After login, offer to import guest addresses; import only with
consent and skip duplicates. Do not automatically save every delivery or import addresses silently.

## Findings from the current code

- `internal/address/store.go`: `addresses` rows are shared by city/address text. `user_addresses` links
  customers to these rows; it has no name, stable entry ID, structured fields or default flag.
- `internal/order/service.go` and `internal/order/store.go`: orders reference shared addresses and
  preserve delivery snapshots. Creating an order deliberately does not populate the address book.
- `internal/customer/merge.go`: account pairing merges `user_addresses`; new metadata must survive it.
- Existing `/addresses` endpoints use shared address IDs. Do not silently reinterpret those IDs.
- Website `src/widgets/food-ordering/ui/checkout-form.tsx` is also used by the Telegram Mini App.
  Town/street are controlled; apartment and entrance currently use uncontrolled inputs.
- `use-checkout-location.ts` starts geolocation for an empty checkout. Saved-address loading must
  finish before deciding whether geolocation should start.
- `src/shared/customer/customer.ts` accepts object responses, not top-level arrays. A new list API
  should return an envelope rather than weakening validation of every customer request.
- Website and Mini App sessions are separate. Server calls must explicitly select the correct surface.

## Implementation decisions

These engineering defaults are specified in the task files within the confirmed scope; they are not additional user requirements. The task files below define the executable contracts and acceptance checks.

### Data and compatibility

Keep immutable shared `addresses` rows for order compatibility. Extend the per-user `user_addresses`
association with a stable generated entry ID, name, structured delivery fields, default flag and timestamps.
Keep its existing `(user_id, address_id)` uniqueness and foreign keys. An edit points only the caller's
entry to a found/created shared address; it never rewrites the shared row or an order snapshot.

The address-book model is:

| Field | Rule |
| --- | --- |
| `id` | Stable saved-entry ID; guest UUID or backend entry ID; never interchangeable with shared address ID |
| `name` | Optional, trimmed, maximum 40 characters; display address text when empty |
| `city` | Required, trimmed, maximum 80 characters |
| `street` | Street and house, 3–120 characters for new/edited entries |
| `apartment` | Optional, maximum 20 characters |
| `entrance` | Optional, maximum 20 characters; retains existing combined entrance/floor field |
| `isDefault` | Boolean; at most one per owner |
| `createdAt`, `updatedAt` | Server timestamps for account entries |

Use the existing order address formatting for the shared address text. Keep apartment and entrance
separate in saved entries so selecting an address restores the form exactly. Do not add phone numbers,
order comments or coordinates to the address book.

Migrate existing associations without losing text. Preserve legacy unstructured text verbatim and mark
it as requiring review when it cannot safely satisfy the structured form. Never guess apartment/floor
by splitting punctuation, truncate old values, or hide an address because it cannot currently be delivered to.
Existing `/addresses` routes retain their response shape and shared-ID semantics; their writes must
participate in the same default/ownership rules. Legacy rows created later need the same review fallback.

Use conservative duplicate detection: Unicode normalization, trim/collapse whitespace and case folding
of city and the consistently formatted complete delivery address. Do not expand abbreviations, omit
apartment/entrance, or use fuzzy geocoding. The backend is authoritative for account duplicates.
Existing duplicates discovered during migration remain visible; new creates/imports reuse a matching entry.
An edit colliding with another saved entry returns a field-level conflict instead of deleting either entry.

Serialize address-book writes per user in transactions, including legacy writes and account merge;
enforce at most one default with a partial unique index. First address in an empty book becomes default.
Changing default is atomic. Deleting the default leaves no default until the customer chooses another.
Import preserves an existing account default; in an empty book, prefer the guest default, otherwise the
first imported entry. Duplicate imports never overwrite account names or fields.

### API boundary

Add a structured customer address-book API while retaining the legacy API:

| Method/path | Behavior |
| --- | --- |
| `GET /customer/addresses` | Current customer's entries as `{ "addresses": [...] }` |
| `POST /customer/addresses` | Create or return an existing duplicate; no metadata overwrite on duplicate |
| `PUT /customer/addresses/{id}` | Edit the caller's saved entry, preserving its stable ID |
| `PUT /customer/addresses/{id}/default` | Atomically make this entry the default |
| `DELETE /customer/addresses/{id}` | Delete only the caller's entry; never shared/order data |

All routes use the existing customer access middleware, private/no-store responses and normal API errors.
Derive ownership from the session, never a submitted user ID. Foreign and missing IDs are indistinguishable.
Allow explicit web or Mini App customer sessions according to the existing feature flag policy.
Define exact status codes, Russian validation messages and response examples in the API task before coding;
update `docs/API.md` and `internal/apidocs/openapi.yaml` together with implementation.

Guest import can call the idempotent create operation sequentially; a bulk endpoint is unnecessary for v1.
The create response must identify whether an entry was created or already existed so import can report counts.

### Guest storage and session transitions

Use a versioned `citypark.addresses.v1` storage envelope with validated entries and stable guest UUIDs.
Do not read browser storage during SSR. Follow the cart's hydration/external-store pattern and handle
cross-tab changes, corrupt JSON, unknown versions, blocked storage and quota failures explicitly.
Storage failures must not display a false persisted-success message; keep entered values available.
Do not overwrite an unknown future schema with an empty book.

Account addresses stay in request/client memory, never guest `localStorage`. Logout or account changes
discard the previous account's in-memory data; late responses cannot populate a different account's view.
An expired session is an authentication error, not permission to write account data into guest storage.

Offer «Добавить сохранённые адреса в аккаунт?» after successful authentication, including Google login
and returning directly to checkout. Use a shared authenticated client bridge rather than wiring only the
password form. Scope the offer to the current surface/account and guest-book revision. «Не сейчас»
suppresses the same offer for that session/revision; management provides a manual import action.

On acceptance, skip duplicates and preserve account metadata/defaults. Remove only guest entries whose
import or duplicate match was acknowledged, and only if their local revision still matches the submitted
snapshot. Failed or concurrently edited entries remain available for retry. A partial failure reports
counts and preserves retryability. Replaying a request must not create duplicates. Rejection retains
guest entries. Do not mix website and Mini App credentials or assume browser storage crosses devices.

### Interface and checkout behavior

- Add `/account/addresses`, accessible to guests without redirecting them to login. Link it from the
  account screen, website navigation and checkout. Guest copy: «Адреса сохраняются только в этом браузере».
- Because the Mini App shares checkout, provide the same account-backed management at `/mini/addresses`
  with its own session. Guests remain a website flow.
- Reuse existing tokens, controls, Russian labels and address suggestions. Include add/edit forms,
  delete confirmation, default selection, empty/loading/error states and keyboard/focus handling.
- Preserve unavailable-town addresses in management. At checkout mark them unavailable and require
  another address or manual correction; always validate against current delivery towns and pricing.
- Checkout priority: restored submission values/manual edits > explicit saved selection > default >
  geolocation > current town fallback. A late fetch, storage event or geolocation result must never
  replace customer edits. Apply a default once after address-book readiness, not after every refresh.
- Selecting an entry updates city, street, apartment and entrance together. Changing city invalidates
  the old selection. Editing selected fields makes a checkout draft; it never silently changes the book.
- «Другой адрес» clears selection and permits manual entry. Pickup hides address saving and performs
  no address writes. New checkout saves may include an optional name; default selection stays in management.
- Save timing: when checkout is submitted with «Сохранить адрес», validate and save the address
  before creating the order. Saving is independent of payment and may succeed even if order creation later
  fails. State this behavior in the UI. On save failure, retain the draft and allow retry or unchecking the
  box to continue; do not create an order until that choice is resolved. Unchecking never deletes an
  already acknowledged saved entry. Duplicate submissions reuse the entry.
- Keep existing order/payment handoff and server pricing. Address IDs never substitute for delivery
  validation, and selecting an address never silently overwrites the customer's chosen delivery method.
- Exclude address text/names from logs, analytics and Sentry payloads/breadcrumbs; preserve current privacy filters.

## Delivery sequence

The task files below contain exact paths, named tests, UI/API errors and completion commands. The larger
storage and frontend-foundation slices are split into 96/96b/96c and 55/55b to keep work bounded.
Do one task and one commit per implementation session. Do not tick progress for planning alone.

| Task | Deliverable | Depends on |
| --- | --- | --- |
| [Backend 96](96-address-book-model.md) | Migration, per-user read model, normalization/validation | Backend 95 |
| [Backend 96b](96b-address-book-writes.md) | Transactional create/deduplicate/edit | Backend 96 |
| [Backend 96c](96c-address-book-defaults.md) | Default/deletion and legacy write compatibility | Backend 96b |
| [Backend 97](97-customer-address-api.md) | Structured API, ownership/validation, wiring, API/OpenAPI | Backend 96c |
| [Backend 98](98-address-book-account-merge.md) | Metadata/default-preserving account merge | Backend 97 |
| [Website 55](../../../citypark-modern/docs/tasks/55-guest-address-store.md) | Shared model/formatting and versioned guest store | Website 54; backend 97 |
| [Website 55b](../../../citypark-modern/docs/tasks/55b-address-api-and-fields.md) | Server adapter and reusable fields | Website 55; backend 97 |
| [Website 56](../../../citypark-modern/docs/tasks/56-my-addresses-management.md) | Guest/account and Mini App management | Website 55b; backend 98 |
| [Website 57](../../../citypark-modern/docs/tasks/57-checkout-saved-addresses.md) | Checkout selection/readiness and opt-in saving | Website 56; backend 98 |
| [Website 58](../../../citypark-modern/docs/tasks/58-import-guest-addresses.md) | Post-login import, retries, session isolation | Website 57; backend 98 |
| [Website 59](../../../citypark-modern/docs/tasks/59-my-addresses-acceptance.md) | Production integration QA and release checklist | All above |

Backend 96 and 98 must ship together before enabling the feature: old account merge SQL otherwise drops
metadata. If a slice exceeds the template's one-session size, split it before assigning it; keep migration
and API compatibility requirements explicit in both halves. Do not deploy an intermediate incompatible state.

Frontend structure: `entities/address` owns the shared model, formatting and storage/read adapters;
`features/manage-addresses` owns mutations/editor UI and its Server Actions; `features/import-addresses`
owns the import prompt. Pages and `widgets/food-ordering` compose these downward through public APIs.
Move reusable address suggestion primitives out of the food-ordering widget into a lower layer so the
management feature does not import a widget or another feature. Root route files remain re-exports.
Read installed Next.js docs before implementation. Browser code calls only same-origin Next endpoints/actions.

## Acceptance and verification

Backend tests must demonstrate:

- Migration retains existing address text, links and order snapshots; legacy API shapes/IDs remain valid.
- Two customers sharing a location can independently rename, edit, delete and choose defaults.
- Structured create/read/update/delete, limits, invalid IDs, 401/403 policy and foreign/missing-ID behavior.
- Duplicate create/import retries, collision edits, apartment differences and Unicode/whitespace cases.
- Concurrent default changes and deletes preserve at most one default, with transactions rolling back on failure.
- Account merge preserves unique entries and destination metadata, resolves defaults deterministically,
  keeps old orders unchanged and does not leave entries owned by the merged-away account.

Frontend tests must demonstrate:

- Guest add/edit/delete/default survives reload; malformed/blocked storage and cross-tab updates are handled.
- Account changes persist across independent authenticated browser contexts; account data never leaks into guest storage.
- Every checkout field is restored exactly; manual edits win over late defaults/geolocation; pickup does not save.
- Address save success/failure/retry and opt-out preserve checkout values and existing payment behavior.
- Import acceptance/rejection, duplicate skip, lost-response retry, partial failure, concurrent local edits,
  password/Google login, expired sessions and switching accounts are covered.
- Removed delivery towns, offline API failures and legacy unstructured entries have usable recovery paths.
- Both themes at 390/1440 px, keyboard/focus, accessible labels/errors and 44 px touch targets pass.
- Existing menu/cart, checkout pricing, autofill and payment-retry flows retain their behavior.

Backend commands: load the existing `.env.development` without printing secrets; run `gofmt -l .`,
`go build ./...`, `go vet ./...`, `go test -p 1 ./...` and coverage for every changed internal package
(at least 80%). Never start Docker/databases; request the existing database if unavailable.

Website commands: `npm run check`, `npm run build`, then a new
`node scripts/qa/flows/my-addresses.mjs` against a production server (`BASE_URL`, default port 3111),
plus affected existing flows. Use isolated test accounts and the test backend for address mutations.
Do not create/pay production orders or send live notifications as part of address QA. Record any
native Telegram/device checks that require owner execution instead of claiming they passed.

Release backend compatibility and merge support first, then the website. Verify ordinary guest checkout
still works if account-address loading fails. Deployment, live payments, a new design system, address
sharing between unrelated accounts and automatic geocoding-based address cleanup are outside this plan.
