# Task 54: Payment retry from website and Telegram order history

## Goal
Show payment actions for eligible unpaid orders in website and Telegram history lists and order details. Pay the saved order through backend task 95, return to its details and preserve the customer's current cart.

## Depends on
Website task 53 and backend task 95 (`../citypark-go/docs/tasks/95-payment-retry.md`). Backend 95 must be complete before implementation; do not implement or modify backend code in this task.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `../citypark-go/docs/API.md` — customer orders and payment retry.
- `src/shared/customer/customer-session.ts` — signed payment context.
- `src/widgets/food-ordering/api/place-order.ts` — existing payment handoff.

Read the installed Next.js documentation for Server Actions, cookies, redirects and revalidation before using those APIs.

## Files to create
- `src/features/payment-retry/index.ts` — UI public API.
- `src/features/payment-retry/api/retry-payment.ts` — Server Action module.
- `src/features/payment-retry/model/types.ts` — action result and props.
- `src/features/payment-retry/ui/payment-retry.tsx` — shared interactive action.
- `scripts/qa/flows/payment-retry.mjs` — focused production-build acceptance flow.

## Files to change
- `src/entities/order/model/types.ts` — backend `paymentRetry` metadata.
- `src/pages/order-history/ui/order-history-page.tsx`, `src/pages/order-history/ui/order-detail-page.tsx` — website entry points and retry-return feedback.
- `src/pages/mini/ui/mini-history-page.tsx`, `src/pages/mini/ui/mini-order-page.tsx` — Telegram entry points and retry-return feedback.
- `src/shared/customer/customer-session.ts` — signed retry origin with backward compatibility.
- `src/pages/checkout/ui/checkout-done-page.tsx`, `src/pages/checkout/ui/checkout-page.tsx` — route website retry returns to order details before checkout/cart behavior.
- `src/pages/mini/ui/mini-payment-return-page.tsx` — route Mini App retry returns to order details before cart-clearing behavior.
- `scripts/qa/flows/telegram-pairing-payment.mjs` — retain checkout return regressions.
- `docs/tasks/PROGRESS.md` — tick only after implementation acceptance passes.

## Spec
### Entry points and eligibility
Render the shared feature on all four history/detail views, passing order ID, backend eligibility and surface (`web` or `mini`). Backend metadata is authoritative; do not recalculate dates or opening hours in the browser.

- Allowed: button `Оплатить`.
- Submitting: disabled button `Переходим к оплате…`; prevent duplicate submissions.
- Expired unpaid order: `Срок оплаты заказа истёк. Оформите новый заказ.`
- Closed-hours unpaid order: `Оплата доступна только в часы работы ресторана.`
- `not_payable`, paid, confirmed or canceled: no payment action.
- Missing/unknown metadata: no payment action, never assume eligibility.

Use existing tokens and controls, both themes, keyboard focus, at least 44 px touch targets and an accessible status/error region. Do not nest buttons inside order links. No new dependencies or cross-page FSD imports.

### Server Action
Export `retryPayment` from the feature's API module; use a typed action state with an optional Russian error message. Accept an order ID and explicit surface, validate both server-side, and obtain only that surface's session. Do not fall back between website and Mini App credentials.

Call `POST /orders/{id}/payment-retry` through the existing server-only customer request client. Never send prices, customer IDs or a return URL to the backend. The browser never calls the backend directly.

On success, validate the returned ID matches the requested ID and that the payment URL is present, set signed pending context with origin `history`, then redirect using the existing payment handoff. Keep redirects outside catch blocks. Do not clear cart contents or checkout state.

On missing session/401: website redirects to `/account/sign-in?returnTo=` with the encoded order-detail path; Mini App redirects to `/mini` for relaunch. On 404, show `Заказ недоступен.` On 409/closed-hours 503, display the documented backend message and refresh history/detail data so stale actions disappear. For network, internal or malformed-response failures, show `Не удалось перейти к оплате. Попробуйте ещё раз.` and permit retry. Distinguish the documented closed-hours response from other 503 failures.

### Payment context and returns
Extend the signed `PaymentContext` with optional `origin: 'checkout' | 'history'`; extend `setPendingPaymentContext` with a third argument defaulting to `checkout`. Verify origin values, include origin in the signed payload, and treat older valid cookies without origin as checkout. Preserve session binding, expiry and returned transaction-ID checks. Query parameters alone must never establish history origin.

Use existing configured Moneta return routes; do not alter provider URLs. A verified history-origin context redirects success returns to the matching `/account/orders/{id}?payment=returned` or `/mini/orders/{id}?payment=returned`, and failure returns to the same path with `payment=failed`. Route these before existing checkout side effects, especially cart clearing. Preserve ordinary checkout behavior.

On order details, fetch the order with the owning surface's session. Paid/confirmed state displays `Заказ оплачен`, even if the return query says failed. An unpaid `payment=returned` displays `Платёж ещё обрабатывается` and an `Обновить статус` action that refreshes server data. An unpaid `payment=failed` displays `Оплата не завершена`; eligibility controls whether another retry is offered. Canceled orders keep their status and have no retry. Query strings never mark an order paid.

An invalid, missing, expired or mismatched return context must not identify a retry order or trigger cart clearing as a new fallback. Keep existing safe missing-context behavior and login/relaunch recovery. Signed context lost in an external Telegram browser requires relaunch and navigating to history; never bypass authentication.

## Tests to write
- Both surfaces: history and detail show correct actions/messages for allowed, expired, closed, paid, confirmed, canceled and missing metadata.
- Actions: correct session only; invalid ID/surface; success; double click; missing/expired session; foreign/missing order; each denial; network/500/malformed result; redirect exceptions are not swallowed.
- Stale data: order paid, midnight crossed or restaurant closed after render; backend rejection refreshes displayed eligibility.
- Saved pricing: retry targets the existing order ID, including discounted orders; never posts a new order or sends an amount.
- Context: history/checkout origins, old cookie without origin, invalid origin, tampering, expiry, wrong transaction ID and session mismatch.
- Returns: website and Mini App success/failure/pending/late confirmation; confirmed payment wins over failure query; repeated return visits; current cart remains identical after every history retry return.
- Ordinary checkout retains its existing behavior. Missing-context and external-browser Mini App returns remain safe.
- Verify keyboard use, live errors, disabled pending state, 390/1440 px and light/dark themes.

## API documentation
No backend contract changes. Consume task 95's `paymentRetry` metadata and retry endpoint exactly. Document QA setup and any owner-run live steps in the QA script header; do not embed credentials or payment URLs in logs.

## Done when
- `npm run check` and `npm run build` pass.
- Run `node scripts/qa/flows/payment-retry.mjs` against the production build (`BASE_URL`, default `http://localhost:3111`) plus the existing Telegram payment flow's applicable regressions.
- Pure/action harness tests cover failures without changing the real backend; do not substitute a mock for required real integration checks. Authenticated integration tests use owner-provided sessions and existing test orders. Ask the owner to run any required backend writes; never create/pay real orders or invoke live webhooks without authorization.
- Record native Telegram/external-browser checks explicitly if device/session access is unavailable; do not claim they passed.
- Tick task 54 and commit `task 54: retry payments from order history` after required acceptance passes.

## Out of scope
Backend/admin changes, repricing, reordering, guest history, new checkout design, automatic retries, new provider configuration and deployment.
