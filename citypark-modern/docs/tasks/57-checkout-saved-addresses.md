# Task 57: Saved addresses and opt-in saving at checkout

## Goal
Use guest/account addresses in both checkout surfaces, with deterministic prefill and an explicit save choice.

## Depends on
Website 56; backend 98. Do not modify backend files.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `src/widgets/food-ordering/ui/checkout-form.tsx`
- `src/widgets/food-ordering/api/place-order.ts`
- `src/widgets/food-ordering/model/use-checkout-location.ts`

## Files to create
- `src/widgets/food-ordering/model/use-checkout-address.ts`
- `src/widgets/food-ordering/ui/saved-address-selector.tsx`
- `scripts/qa/checkout-address-harness.mjs`

## Files to change
- `src/pages/checkout/ui/checkout-page.tsx`, `src/pages/mini/ui/mini-checkout-page.tsx` — load current-surface book/context.
- `src/widgets/food-ordering/ui/checkout-form.tsx` — controlled address fields, selection/save UI and guest persistence.
- `src/widgets/food-ordering/model/use-checkout-location.ts` — readiness/manual-selection precedence.
- `src/widgets/food-ordering/model/schema.ts` — explicit save intent/name and restored-state fields.
- `src/widgets/food-ordering/api/place-order.ts` — account save before order creation; isolate save errors.
- `scripts/qa/flows/my-addresses.mjs`, `scripts/qa/flows/checkout-geolocation.mjs` — new cases/regressions.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Read installed Next.js docs for Server Actions/redirects. Extend CheckoutForm props with server-derived
AddressBookLoad. Guest mode reads the versioned guest store after hydration; account mode uses only its
loaded entries. Mini never falls back to website/guest credentials. An address-list fetch failure displays
retry feedback but still permits manual checkout with saving unchecked; retain account context for any
future save attempt. Do not make the whole checkout fail just because optional address loading failed.

All city/street/apartment/entrance inputs are controlled and updated atomically from a selection. Selection
IDs are tagged by source (`guest`/`account`) and never submitted as authoritative delivery information.
No default/geolocation until guest hydration/account loading has settled. Priority:
1. Submitted-error values and subsequent manual edits.
2. Explicit saved selection.
3. Valid default, applied once when ready.
4. Geolocation, then the existing town fallback.
Defaults needing review or outside current delivery towns are not applied. A late lookup/geolocation/storage
event cannot overwrite an edit/selection. Cancel in-flight geolocation updates on selection/manual changes.
Programmatic selection must not trigger a manual-edit callback that clears itself. Restored values include
all four address fields plus name/save intent. Changes after an error are not overwritten by stale results.

Selector labels: `Сохранённые адреса`, `Другой адрес`, `Мои адреса`. Unavailable entries show
`Доставка недоступна`; legacy entries show `Уточните адрес в разделе «Мои адреса»` and cannot auto-fill.
«Другой адрес» clears selection and all four address fields, keeping delivery method/schedule/contact/cart
unchanged. Editing any selected field clears selection but retains other draft fields; the book is not edited.
Changing city clears street/apartment/entrance to avoid carrying a different unit to a new town.

Delivery mode includes an initially unchecked `Сохранить адрес` and optional `Название адреса` when checked.
Copy: `Адрес будет сохранён при оформлении заказа, независимо от результата оплаты.` Selecting an existing
entry does not check the box; checked duplicates are idempotent. Pickup hides the controls and ignores
save intent, with no address writes. Default selection remains in management, not a new checkout toggle.

Save timing is before creating the order, after validating the address fields (not necessarily after all
other order validation). Therefore a saved address may remain if another checkout field/order/payment later
fails; this is explicit opt-in, not an automatically saved delivery. Guest flow validates AddressFields and
persists locally before dispatching the existing form action. On storage failure retain draft and stop that
submission; allow retry or unchecking. Never trust a client claim of account ownership.

Account flow revalidates fields and expectedAccountId server-side against that surface's current session,
then calls createSavedAddress before createOrder. A session that was present but expired must not become
a guest save. Guest-vs-account intent is checked against current session on submit; if identity changed,
return the reauthentication/context error and never redirect a guest save into the account book. A guest
local save already acknowledged before a concurrent login remains local; it can be imported later with
consent. Do not claim browser storage and server session changes form an atomic transaction. Preserve existing order auth.

On account save failure, return a separate address-save error and restored values; do not create an order.
Copy: `Не удалось сохранить адрес. Повторите попытку или снимите отметку «Сохранить адрес».` Validation/auth
errors may use the more specific contract message. Successful save is not rolled back for a later order
failure. Retry reuses an equivalent entry without overwriting names/defaults. Unchecking does not delete
an address saved by an earlier attempt. Do not log fields in catch paths.

Keep server-side town/availability/pricing validation, signed payment context, redirects outside catch
blocks, cart behavior and order ownership unchanged. Different field content recomputes delivery quote
through the existing city/method flow; address IDs never bypass eligibility. Link management to the right
surface. An entry removed elsewhere should cease to be selected but must not silently erase manual text.

## Tests to write
- `checkoutAddressPriority`: hydration/default/manual/error restoration/late location races; all four fields exact.
- `checkoutAddressSelection`: change selection/manual city/other address; unit values never leak between entries.
- `checkoutUnavailableAddresses`: removed town, legacy review and API failure still allow safe manual entry.
- `checkoutGuestSave`: unchecked/pickup no write; checked valid save, duplicate retry, invalid/storage failure stops action.
- `checkoutAccountSave`: correct surface/owner, changed/expired session, failure before createOrder; duplicate and successful
  save followed by order failure; no unintended second order or mutation of an existing entry.
- `checkoutPaymentRegression`: existing pricing, payment return/retry and cart behavior unchanged.

## API documentation
No backend changes. Explain save-before-order behavior in the QA header and visible copy above.

## Done when
`node scripts/qa/checkout-address-harness.mjs`, `npm run check`, `npm run build`, the expanded
`node scripts/qa/flows/my-addresses.mjs`, `node scripts/qa/flows/checkout-geolocation.mjs` and affected
payment-retry harness/flow pass against a production build. Test order-dispatch ordering in isolated harnesses;
do not create/pay live orders. Live address writes require the owner per AGENTS.md. Tick 57 and commit
`task 57: reuse saved addresses at checkout` after required checks pass.

## Out of scope
Automatic saving, address-ID-only order payloads, order/payment redesign, login import, deployment.
