# Task 31: Stop sending paymentType

## Goal
`POST /orders` from the checkout no longer sends `paymentType`. The backend dropped the field (orders are paid only
online) and ignores it, so this is a cleanup with no visible change.

## Depends on
26, and citypark-go task 65.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `src/pages/checkout/api/orders-api.ts`, `src/pages/checkout/api/place-order.ts`,
`../citypark-go/docs/API.md` (section `POST /orders`).

## Files to change
- `src/pages/checkout/api/orders-api.ts`: remove `paymentType: 'online'` from `CreateOrderBody`.
- `src/pages/checkout/api/place-order.ts`: remove `paymentType: 'online',` from the body.

## Spec
Nothing else changes: the response is still `201 {orderId, redirectUrl}` with the Moneta page.

## Tests to write
None: the change only removes a constant field. `npm run qa:flows` must still pass the checkout flow.

## Done when
CONVENTIONS "Done means" passes, `grep -rn paymentType src` finds nothing, and the checkout QA flow passes against a
production build with the backend running.

## Out of scope
Texts that mention cash in the hotel rules and the public offer (`offer.data.ts`, `blocks.data.ts`): the owner edits
those separately.
