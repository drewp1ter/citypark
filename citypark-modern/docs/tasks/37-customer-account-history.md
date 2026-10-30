# Task 37: Account, linked methods and order history

## Goal
Provide private account/profile/linking controls and paginated account-owned
order history and details on the regular website.

## Depends on
36; backend78.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md`, `src/pages/account/index.ts`, `src/shared/api/customer.ts`.

## Files to create
- `src/pages/account/ui/account-page.tsx`, `methods-panel.tsx`, `pairing-panel.tsx`
- `src/pages/order-history/index.ts`, `api/orders.ts`, `ui/order-history-page.tsx`, `ui/order-detail-page.tsx`
- `app/(plain)/account/page.tsx`, `orders/page.tsx`, `orders/[id]/page.tsx`
- `scripts/qa/flows/customer-history.mjs`

## Files to change
- `src/pages/account/index.ts`, `api/actions.ts`, `model/schema.ts`: profile, verified email change, pairing actions.
- `src/shared/api/customer.ts`: method list and pairing requests.
- `src/widgets/site-header` public navigation: stable account link.

## Spec
Read the installed Next.js16 guides for each cookies/redirect/Route Handler/
caching API used before implementation. Use FSD public APIs, Russian copy,
existing token colors and primitives, both themes, labeled inputs, keyboard focus
and >=44px targets. Root app route files only delegate to src; keep server
handlers/actions in src and export them from the root shim. No backend fetches
from browser code, service-worker caching of private responses or secret NEXT_PUBLIC
variables. These new screens have no finished reference; use existing components.
- /account heading `Личный кабинет`; show display name, optional phone, linked
  `Email и пароль`, `Google`, `Telegram` from GET /customer/me,/customer/methods.
  No email assumption for Google/Telegram-only accounts; no phone requirement
  for profile. Email uses the explicit verified-change flow from71/36, never
  a writable auth-email field sent to PUT /customer/me.
- Controls `Сохранить`, `Выйти`, `Подключить Google`, `Подключить Telegram`,
  `История заказов`. Hide unsupported password/email creation for provider-only
  accounts; do not imply an unimplemented method can be added. Already-linked
  methods show state, no unlink action. Google uses explicit intent link.
- Website Telegram flow: code input `Код из Telegram`, `Продолжить` redeems code,
  then show peer display name and explicit `Подтвердить связь` / `Отменить`.
  Clearly explain that orders and login methods will be combined and Telegram
  source sessions will end. Redeem does not confirm. Never put code in URL,
  analytics or persistent browser storage. Show each side's confirmation state.
- Poll own pending pairing status every3s while visible, stop on completion/
  cancellation/expiry/unmount; timer uses server expiresAt. Concurrent polls must
  not mutate; errors stop polling and show retry/re-auth controls. Confirm is
  explicit POST Server Action. Success refreshes account/method/history views.
- /account/orders uses GET /orders/my?page=N&pageSize=10, matching actual API
  pagination contract; validate positive page and use backend count. Preserve
  existing API field names. Empty `У вас пока нет заказов`; unavailable
  `Не удалось загрузить заказы. Попробуйте позже`. Show order ID,date,status,
  total+delivery, link to /account/orders/{id}. Never query by phone.
- /account/orders/{id} loads authorized backend detail, shows snapshot contact,
  delivery/pickup,time,items,fee,total,status. Foreign/missing -> same404;
  missing/expired session -> sign-in with safe local return path. Unowned guest
  orders cannot be claimed or shown here, even with matching phone.
- All account/history/pairing pages and errors no-store; no account data in
  shared public caches or localStorage. Existing admin response types untouched.
  Reuse date/money formatting; avoid client account-ID ownership checks.

## Tests to write
- `customer-history.mjs`: two distinct test accounts with same contact phone,
  own paginated/empty history, own detail, foreign404, guest exclusion, snapshots
  survive profile edits, expired session and API errors preserve privacy.
- Explicit linking controls, no automatic merge on page load; redeem vs confirm,
  pending/expired/canceled/conflicting/complete states; polling stops on unmount.
- Keyboard/forms,390/1440px,light/dark. Real pairing, profile writes and orders are
  owner-approved QA_CUSTOMER_WRITES=1 scenarios; default flow must label skips.
  Retain no-secret/no-cache checks from35.

## API documentation
Consume the documented backend routes; do not change the backend. Keep request and response types local to the server client and never expose token-bearing envelopes to Client Components.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 37: Account, linked methods and order history`.


## Out of scope
Mini App (38–39), guest claiming, unlinking, hotel accounts, unsupported password-method enrollment.
