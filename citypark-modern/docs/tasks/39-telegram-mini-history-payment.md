# Task 39: Mini App pairing, history and payment return

## Goal
Complete Mini App account pairing/history and preserve the ordering context
through payment success, failure, cancellation and return navigation.

## Depends on
38.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md`, `src/pages/account/ui/pairing-panel.tsx`, `src/pages/checkout/ui/checkout-done-page.tsx`.

## Files to create
- `src/pages/mini/ui/mini-account-page.tsx`, `mini-history-page.tsx`, `mini-order-page.tsx`, `mini-payment-return-page.tsx`
- `app/(mini)/mini/account/page.tsx`, `orders/page.tsx`, `orders/[id]/page.tsx`, `checkout/done/page.tsx`
- `src/features/account-pairing/index.ts`, `api/actions.ts`, `ui/pairing-status.tsx`
- `src/entities/order/index.ts`, `api/orders.ts`, `model/types.ts`
- `scripts/qa/flows/telegram-pairing-payment.mjs`

## Files to change
- `src/pages/account` and `src/pages/order-history`: consume shared pairing feature/order entity instead of importing mini pages.
- `src/pages/mini/index.ts`, shared ordering widget and checkout actions: server-bound surface and payment return context.
- `src/pages/checkout/ui/checkout-done-page.tsx`, `api/orders-api.ts`: distinguish regular/mini return safely; preserve website.
- `src/shared/api/customer-session.ts`: pending payment context cookie.

## Spec
Read the installed Next.js16 guides for each cookies/redirect/Route Handler/
caching API used before implementation. Use FSD public APIs, Russian copy,
existing token colors and primitives, both themes, labeled inputs, keyboard focus
and >=44px targets. Root app route files only delegate to src; keep server
handlers/actions in src and export them from the root shim. No backend fetches
from browser code, service-worker caching of private responses or secret NEXT_PUBLIC
variables. These new screens have no finished reference; use existing components.
- /mini/account shows Telegram identity/account name, `Связать с сайтом`,
  `История заказов`; no Google OAuth button. Show `Google можно подключить в
  личном кабинете на сайте` with regular website link, never launch embedded OAuth.
- Create pairing via explicit action, show short-lived code and `Скопировать код`,
  expiry and instructions to sign into website and paste it there. Creation is
  not confirmation. Once redeemed, show peer name and explicit `Подтвердить связь`
  plus `Отменить`; explain account/history transfer. Reuse shared status feature,
 3s visibility-aware polling and recent-auth errors from37; no code in URLs/logs.
- After merge, source session401 is expected; show `Связь подтверждена. Откройте
  приложение заново` so fresh launch signs into destination. Do not reuse consumed
  initData automatically. Future launch shows combined owned history; guests
  remain excluded. Confirmations never accept IDs from client as authority.
- /mini/orders and /mini/orders/{id} use mini server session and shared order
  entity, same pagination/authorization/error semantics as37; BackButton goes to
  known mini parent. Website and mini pages must not import each other's slices.
- Before redirecting to Moneta, save server-issued pending payment context with
  order ID and trusted surface in Secure,HttpOnly,Lax cookie with <=24h lifetime;
  never trust a form field as surface or a returned order ID as ownership.
  Retain existing backend payment URL/signature generation. Moneta callback is
  the sole payment authority; return URL/query parameters never mark paid.
- Existing Moneta success/failure URLs may remain regular /checkout/done and
  /checkout?payment=failed; a narrow server adapter uses validated stored context
  and matching returned MNT_TRANSACTION_ID to route to /mini/checkout/done or
  /mini/checkout?payment=failed. Do not change global URLs per order or introduce
  unsigned redirect parameters. Missing context shows neutral return UI, never
  another customer's order. Copy Mini App state into URL only if nonsecret and
  validated; no auth or pairing proofs in payment URLs.
- On return, load authorized order if mini session exists. Pending is
  `Платёж ещё обрабатывается`, failed/canceled `Оплата не завершена`, paid
  `Заказ оплачен`. Provide `К заказу` and `В меню`; never infer status from success
  redirect. Preserve cart on failure/cancel/unknown; clear once only after verified
  paid status. Poll pending status briefly with bounded retry, no duplicate order
  or payment on return/reload. Missing session offers relaunch; it must not become
  guest checkout or expose details. Website guest flow remains supported with
  neutral confirmation, not unauthorized order-history access.
- Document external-payment handoff behavior and safe reopening through configured
  bot Mini App link for clients losing embedded context. Manual device check40
  must verify cookies/navigation survive actual payment provider behavior.

## Tests to write
- `telegram-pairing-payment.mjs`: both confirmations and stale/conflict/replay/
  cancel states; source reauthentication; shared owned history vs guest separation.
- Payment return success/pending/failure/cancel/reload, forged ID or surface,
  missing cookie/session, no cart loss on unknown status, no duplicate submission,
  no status mutation from browser return parameters; website regression flows.
- Automated read-only/UI assertions run by default. Real pairing/order/payment
  writes are owner-approved QA_CUSTOMER_WRITES=1; document inability to automate
  cross-app navigation rather than substituting fake backend success. Verify
  both themes/safe areas and app back-button destinations.

## API documentation
Consume the documented backend routes; do not change the backend. Keep request and response types local to the server client and never expose token-bearing envelopes to Client Components.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 39: Mini App pairing, history and payment return`.


## Out of scope
Changes to Moneta staff callback/signing, guest history/claims, real payments without approval, public launch (40).
