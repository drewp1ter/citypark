# Task 38: Telegram Mini App entry and ordering

## Goal
Add /mini with launch authentication, a compact Telegram layout and the
existing menu/cart/checkout flow, without duplicating ordering business logic.

## Depends on
37; backend75.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md`, `src/app/layouts/menu-layout.tsx`, `src/pages/checkout/api/place-order.ts`.

## Files to create
- `src/features/telegram-launch/index.ts`, `api/actions.ts`, `model/types.ts`, `ui/telegram-provider.tsx`
- `src/app/layouts/mini-layout.tsx`
- `app/(mini)/mini/layout.tsx`, `page.tsx`, `checkout/page.tsx`
- `src/pages/mini/index.ts`, `ui/mini-menu-page.tsx`, `ui/mini-checkout-page.tsx`
- `scripts/qa/flows/telegram-mini.mjs`

## Files to change
- `src/pages/menu` and `src/pages/checkout`: extract reusable ordering UI into `src/widgets/food-ordering` with index.ts and UI modules; keep single-page forms local until shared.
- `src/app/layouts/index.ts`, `src/shared/api/customer-session.ts`: layout exports and isolated mini session cookie.
- `src/app/styles/globals.css`: viewport/safe-area variables from existing tokens only.

## Spec
Read the installed Next.js16 guides for each cookies/redirect/Route Handler/
caching API used before implementation. Use FSD public APIs, Russian copy,
existing token colors and primitives, both themes, labeled inputs, keyboard focus
and >=44px targets. Root app route files only delegate to src; keep server
handlers/actions in src and export them from the root shim. No backend fetches
from browser code, service-worker caching of private responses or secret NEXT_PUBLIC
variables. These new screens have no finished reference; use existing components.
- Root /mini loads Telegram WebApp SDK using the installed Next guidance. Browser
  initialization lives in a typed client feature; only its raw initData is passed
  to a Server Action calling /auth/customer/telegram. Never use initDataUnsafe as
  identity/authorization. No service-worker private caching or secrets in SDK.
- Use a separate `__Host-citypark_mini` secure,HttpOnly,Lax,Path=/ session cookie
  (development-only alternate name) so Mini App launch cannot overwrite website
  account or invalidate pairing destination. Server adapters choose cookie by
  trusted route/action, never browser-supplied token/account ID. Backend-issued
  mini surface remains required. Website auth headers/cookies must not be sent.
- Authentication completes before private Mini App content/actions; backend
  session expiry is authoritative. A valid existing mini session may be reused
  on reload; do not re-submit a consumed launch proof after network uncertainty.
  If no valid session, validate fresh raw initData once; failed/replayed/expired
  launch asks the customer to close/reopen Mini App, never creates a guest session.
- Loading `Входим через Telegram…`; unavailable/wrong environment
  `Откройте приложение через Telegram`; invalid/expired launch
  `Не удалось войти. Закройте и снова откройте приложение`.
  Do not log launch data. Fail closed when SDK/initData unavailable.
- Frame covers /mini and /mini/checkout: compact ordering navigation, no hotel
  links/full desktop footer, cart still shared ordering model. Session never
  stored in cart. Extract shared UI into widgets only when both pages use it;
  pages cannot import other page slices. Preserve website visual behavior.
- Call WebApp.ready() after essential UI readiness; expand when available;
  react to viewportChanged/safeAreaChanged/contentSafeAreaChanged with feature
  detection. Compute CSS custom heights/insets with fallbacks; keep controls above
  Telegram UI and software keyboard. Remove listeners on unmount.
- Telegram BackButton visible on nested routes; use known internal route parent
  (checkout->mini), hide at mini root; synchronize with navigation. No browser
  history.back() that can leave app unexpectedly. Respect reduced motion/focus.
- Checkout uses mini cookie, requires contact phone independently of Telegram
  identity, existing prices/hours/address validations/Moneta signing. Keep Google
  OAuth out of mini. Payment return/mini history/pairing are completed in39;
  do not enable public bot entry until40 checks are documented.

## Tests to write
- `telegram-mini.mjs`: SDK-free launch fallback, simulated SDK UI events only
  (not a fake backend), loading/expired/invalid/replayed launch, viewport/safe area,
  back-button transitions/listener cleanup,390px and desktop layout.
- With owner-approved valid Telegram launch and configured backend, authenticate
  once, reload with existing mini cookie, separate website/mini sessions, stale
  session cannot checkout as guest, no raw proof in props/logs/storage.
- Run existing menu/cart/checkout flows after UI extraction, production build.
  Real authentication/order writes require QA_CUSTOMER_WRITES=1 and approval;
  device/payment checks belong to40. No fake backend or initDataUnsafe fixtures
  may be accepted as successful authentication.

## API documentation
Consume the documented backend routes; do not change the backend. Keep request and response types local to the server client and never expose token-bearing envelopes to Client Components.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 38: Telegram Mini App entry and ordering`.


## Out of scope
Mini pairing/history/payment return (39), BotFather publication, Google OAuth in Telegram, a second checkout implementation.
