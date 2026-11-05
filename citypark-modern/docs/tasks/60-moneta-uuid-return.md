# Task 60: Payment return pages use the UUID transaction id

## Goal
After backend task 99, MONETA returns a UUID in `MNT_TRANSACTION_ID` instead of the order number. The signed payment cookie and the return pages match that UUID, so the success page, the failed-payment redirect and the history redirect keep working.

## Depends on
`citypark-go` task 99 (the backend must send the UUID). The code here is written against the UUID form of `redirectUrl` and must be deployed together with task 99.

## Read first
`docs/tasks/CONVENTIONS.md`, `src/shared/customer/customer-session.ts`, `scripts/qa/payment-retry-harness.mjs`.

## Files to change
- `src/shared/customer/customer-session.ts`: the context and both functions (see Spec).
- `src/shared/customer/index.ts`: export the new helper.
- `src/widgets/food-ordering/api/place-order.ts` (line ~168) and `src/features/payment-retry/api/retry-payment.ts` (line ~25): pass the UUID taken from `redirectUrl`.
- `src/pages/checkout/ui/checkout-page.tsx` (line ~22): the Mini redirect carries the UUID.
- `scripts/qa/payment-retry-harness.mjs`, `scripts/qa/checkout-address-harness.mjs`: update to the new signatures and add the cases below.

## Spec
```ts
export type PaymentContext = { orderId: number; transactionId: string; surface: 'web' | 'mini'; issuedAt: number; origin?: 'checkout' | 'history' }
export function transactionIdFromRedirect(redirectUrl: string): string | undefined
export async function setPendingPaymentContext(orderId: number, transactionId: string, surface: 'web' | 'mini', origin?: 'checkout' | 'history'): Promise<void>
export async function getPendingPaymentContext(returnedId?: string): Promise<PaymentContext | undefined>
```
- `UUID_RE` is `/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/`. Ids are compared in lowercase.
- `transactionIdFromRedirect` parses the URL and returns the lowercase `MNT_TRANSACTION_ID` query value when it matches `UUID_RE`; otherwise `undefined` (also for an unparsable URL).
- `setPendingPaymentContext` throws `Invalid transaction id` when `transactionId` does not match `UUID_RE`; the order id, origin and session checks stay as they are. The UUID goes into the signed payload.
- `getPendingPaymentContext` returns `undefined` unless `returnedId` matches `UUID_RE` (case-insensitive) and equals `context.transactionId`. The numeric `/^[1-9]\d*$/` check and the `orderId === Number(returnedId)` check are removed. Every other check (cookie shape, 24 h age, surface, origin, HMAC) is unchanged. A cookie written before this task has no `transactionId`, so it yields `undefined`; it lives at most 24 h and no migration is needed.
- `place-order.ts` and `retry-payment.ts`: `const transactionId = transactionIdFromRedirect(redirectUrl)`. When it is `undefined`, log `console.error('payment redirect without a valid MNT_TRANSACTION_ID')`, skip `setPendingPaymentContext` and still send the customer to the payment page (the order exists; a missing cookie only loses the return-page context). Otherwise call `setPendingPaymentContext(orderId, transactionId, surface[, 'history'])`.
- `checkout-page.tsx`: the Mini redirect becomes `/mini/checkout?payment=failed&MNT_TRANSACTION_ID=${context.transactionId}`. The history redirect still uses `context.orderId` for the path.
- `checkout-done-page.tsx`: preserve `context.transactionId` in the successful Mini redirect to `/mini/checkout/done`.
- The pages that read `MNT_TRANSACTION_ID` (`checkout-done-page.tsx`, `mini-payment-return-page.tsx`, `mini-checkout-page.tsx`, the `returnedOrderId` prop) need no change: they pass the raw string on. Rename `returnedOrderId` to `returnedTransactionId` only if it stays a one-line change per file.
- Never show the UUID to the customer.

## Tests to write
Extend the existing harnesses (no new framework):
- `payment-retry-harness`: set a context with a UUID, then `getPendingPaymentContext(<same uuid>)` returns it with the right `origin` (also with the uuid upper-cased); a different UUID → `undefined`; the numeric id `'7'` → `undefined`; a tampered cookie → `undefined`; a cookie without `transactionId` → `undefined`; the other-session, bad-origin, expired and future-`issuedAt` cases keep returning `undefined`.
- `transactionIdFromRedirect`: a redirect URL with a UUID → the UUID; upper-case UUID → lowercase; no parameter, a numeric value, and `not a url` → `undefined`.
- `setPendingPaymentContext(7, 'not-a-uuid', 'web')` throws `Invalid transaction id`.
- `payment-retry-harness` page cases: a redirect URL without a valid id still redirects to the payment page and does not set the cookie (assert the fake `setPendingPaymentContext` was not called).
- `checkout-address-harness`: the fake `setPendingPaymentContext` receives `(orderId, uuid, surface)` taken from the fake `redirectUrl`.
- Update the fake backend responses in `scripts/qa/flows/payment-retry.mjs` and `scripts/qa/flows/checkout-and-contacts.mjs` if they build `MNT_TRANSACTION_ID=<number>`: use a UUID so the flows follow the real format.

## Done when
`npm run check`, the two harnesses (`node scripts/qa/payment-retry-harness.mjs`, `node scripts/qa/checkout-address-harness.mjs`), `npm run build`, and then `node scripts/qa/flows/payment-retry.mjs` and `node scripts/qa/flows/checkout-and-contacts.mjs` pass.

## Out of scope
- Any change in `../citypark-go/` (task 99 does it).
- Showing or storing the UUID anywhere except the signed payment cookie and the MONETA return query.
- Accepting numeric ids on the return pages.
