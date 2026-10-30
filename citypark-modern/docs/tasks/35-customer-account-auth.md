# Task 35: Customer sign-in and verification

## Goal
Add the server-only customer API boundary, secure session cookies and Russian
registration, sign-in and email-verification screens; preserve guest ordering.

## Depends on
34; backend71.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md`, `src/pages/checkout/api/orders-api.ts`, `docs/PLAN.md`.

## Files to create
- `src/shared/api/customer.ts`, `src/shared/api/customer-session.ts`
- `src/pages/account/index.ts`, `src/pages/account/api/actions.ts`, `src/pages/account/model/schema.ts`
- `src/pages/account/ui/sign-in-page.tsx`, `register-page.tsx`, `verify-page.tsx`, `auth-form.tsx`
- `app/(plain)/account/sign-in/page.tsx`, `register/page.tsx`, `verify/page.tsx`
- `scripts/qa/flows/customer-auth.mjs`

## Files to change
- `src/shared/api/index.ts`, `src/shared/config/env.ts`, `.env.example`: server-only public origin/config.
- `src/pages/checkout/api/orders-api.ts`, `place-order.ts`: authenticated calls and guest-cookie bridge.
- `src/widgets/site-header` public UI: link to sign-in; do not make public pages depend on private data.
- `public/sw.js`: preserve its offline-only cache; ensure account/auth/private responses are never cached.

## Spec
Read the installed Next.js16 guides for each cookies/redirect/Route Handler/
caching API used before implementation. Use FSD public APIs, Russian copy,
existing token colors and primitives, both themes, labeled inputs, keyboard focus
and >=44px targets. Root app route files only delegate to src; keep server
handlers/actions in src and export them from the root shim. No backend fetches
from browser code, service-worker caching of private responses or secret NEXT_PUBLIC
variables. These new screens have no finished reference; use existing components.
- Routes /account/sign-in, /account/register, /account/verify. Screen headings
  `Вход`, `Регистрация`, `Подтверждение email`. Fields `Имя`, `Email`, `Пароль`,
  optional `Телефон`. Buttons `Войти`, `Зарегистрироваться`, `Подтвердить email`,
  `Отправить письмо повторно`; link `Продолжить без входа` -> /menu.
- Add server-only `CUSTOMER_PUBLIC_ORIGIN` to frontend config and `.env.example`,
  matching the backend origin (HTTPS production; localhost HTTP development).
  Server-only API client imports `server-only`; fetch cache:no-store, validates
  status/JSON and redacts errors. Customer auth headers are server-derived only.
- Cookie `__Host-citypark_customer`: Secure,HttpOnly,SameSite=Lax,Path=/, no
  Domain, absolute expiry from backend, max30days. Local HTTP development uses
  distinct `citypark_customer_dev` only in development. Mutate cookies only in
  Server Actions/Route Handlers, never Server Components. Never put token in
  client props/HTML/localStorage/sessionStorage/URL or logs.
- Sign-in action calls /auth/customer/sign-in, stores token and redirects only
  to validated local return path (default /account; provisional /menu until37).
  Reject //, external origins, backslashes and encoded bypasses. Do not turn a
  present invalid session into a guest checkout; show `Войдите повторно, чтобы
  продолжить` and preserve cart/form. Explicit logout revokes then clears cookie;
  if backend unavailable clear local cookie and report remote failure safely.
- Registration/resend always show backend generic result. Password fields never
  echoed into action state; failed fields preserved except secrets. Verify GET
  renders POST confirmation, never auto-consumes token; Referrer-Policy:no-referrer
  on token pages, no third-party assets/analytics. Confirm then link to sign-in.
- Checkout forwards valid customer cookie as backend Bearer token. Guest checkout
  forwards backend citypark_guest_checkout cookie and captures its Set-Cookie
  into a separate Next HttpOnly/Secure/Lax/Path=/ cookie (same expiry), never a
  browser auth token. This bridges server-side fetches for retry isolation from67.
  Do not copy arbitrary backend cookie/header values; allowlist guest cookie.
- All account/auth routes and responses private,no-store, including errors.
  Keep static public pages/cache behavior; header may show static `Личный кабинет`
  link rather than reading cookies from every public layout. Existing checkout
  price/time/address validations and Moneta redirect behavior stay unchanged.

```ts
// server-only modules; methods return sanitized data, never raw tokens to clients
export async function customerRequest<T>(path: string, init?: RequestInit): Promise<T>
export async function getCustomerSessionToken(): Promise<string | undefined>
export async function clearCustomerSession(): Promise<void>
```
Backend contracts: register/resend/confirm/sign-in from71, logout/me from69.
Render exact Russian backend validation messages, generic fallback
`Не удалось выполнить запрос. Попробуйте позже`.

## Tests to write
- `customer-auth.mjs`: registration validation and password non-echo, generic
  resend, verification confirmation requires POST, successful sign-in/logout,
  expired-session checkout fails without becoming guest; cart and input survive.
- Inspect response headers/cookies, rendered HTML and browser storage: correct
  flags, no token leakage or private caching; safe local return paths.
- Guest checkout retries forward only allowlisted guest scope and remain separate
  from authenticated history; real order creation requires explicit owner approval.
  Gate write scenarios behind `QA_CUSTOMER_WRITES=1`; default flow verifies safe
  forms/navigation only and prints skipped owner checks. Use real test accounts
  on owner-approved backend, no backend mock. Check390/1440px and both themes.

## API documentation
Consume the documented backend routes; do not change the backend. Keep request and response types local to the server client and never expose token-bearing envelopes to Client Components.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 35: Customer sign-in and verification`.


## Out of scope
Recovery/Google (36), account/history (37), Mini App (38–39), actual backend writes without approval.
