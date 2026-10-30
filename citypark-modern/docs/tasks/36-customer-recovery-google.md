# Task 36: Recovery and Google sign-in

## Goal
Add password recovery, email-change confirmation and browser-bound Google
sign-in/link callbacks through the regular website.

## Depends on
35; backend72,74.

## Read first
`docs/tasks/CONVENTIONS.md`, `../citypark-go/docs/API.md`, `src/shared/api/customer-session.ts`, `src/pages/account/api/actions.ts`.

## Files to create
- `src/pages/account/ui/recovery-page.tsx`, `reset-page.tsx`, `email-page.tsx`
- `src/pages/account/api/google.ts`, `src/pages/account/ui/google-button.tsx`
- `app/(plain)/account/recovery/page.tsx`, `reset/page.tsx`, `email/page.tsx`
- `app/account/google/callback/route.ts`
- `scripts/qa/flows/customer-recovery-google.mjs`

## Files to change
- `src/pages/account/index.ts`, `api/actions.ts`, `model/schema.ts`, `ui/sign-in-page.tsx`: forms and links.
- `src/shared/api/customer.ts`, `customer-session.ts`: OAuth attempt cookies and backend calls.

## Spec
Read the installed Next.js16 guides for each cookies/redirect/Route Handler/
caching API used before implementation. Use FSD public APIs, Russian copy,
existing token colors and primitives, both themes, labeled inputs, keyboard focus
and >=44px targets. Root app route files only delegate to src; keep server
handlers/actions in src and export them from the root shim. No backend fetches
from browser code, service-worker caching of private responses or secret NEXT_PUBLIC
variables. These new screens have no finished reference; use existing components.
- Routes /account/recovery (`Восстановление пароля`), /account/reset
  (`Новый пароль`), /account/email (`Подтверждение нового email`). Recovery uses
  generic backend202; reset success `Пароль изменён. Войдите с новым паролем`.
  Clear local session after reset; other sessions revoked by backend. GET with
  token only renders a POST form. Referrer-Policy:no-referrer; no analytics.
- `Забыли пароль?` link on sign-in. Google button `Войти через Google` starts a
  Server Action that calls /auth/customer/google/start with intent login. Account
  page37 can use same action with intent link. State, nonce, PKCE stay backend-owned.
- Store returned flowToken only in secure HttpOnly same-site ten-minute cookie
  `__Host-citypark_google_flow`, paired to returned state in server-managed
  cookie data. Allow a bounded list (max3) of pending attempts for multiple tabs,
  keyed by a hash of state; starting a fourth invalidates oldest locally.
  Do not overwrite a different tab's proof silently; do not expose flow token in
  authorization URL. Callback route matches state to proof, rejects duplicates,
  passes state/code/error plus proof to backend, then deletes only that attempt.
- Callback success stores session for login, or retains session for linking and
  redirects to /account?linked=google. Failure/cancellation redirects to
  /account/sign-in with an allowlisted error code; show `Вход через Google отменён`
  or `Не удалось подтвердить вход через Google`. Never include code/state/provider
  body/raw token in error URLs/logs. Missing/expired browser proof fails closed.
- Callback consumes only its pending proof; return destination is stored server-
  side or fixed, never taken from an untrusted callback URL. Customer headers for
  intent link come only from current cookie. All callback responses no-store.
- Avoid OAuth inside Telegram embedded browser: only regular website pages expose
  the button; mini surfaces in38 show a normal website-account instruction instead.
- Email change confirmation calls backend with current authenticated web session;
  stale -> `Войдите повторно для подтверждения email`; it never silently changes
  email on GET. Other field/error copy follows backend contract from71/72/74.

```ts
// Server Actions, imported via slice api, not through a client barrel.
export async function startGoogleSignIn(): Promise<void>
export async function startGoogleLink(): Promise<void>
// Root Route Handler delegates to the account slice's server-only callback.
export async function completeGoogleCallback(request: Request): Promise<Response>
```

## Tests to write
- `customer-recovery-google.mjs`: generic recovery, invalid/expired/reused reset,
  password not echoed, reset session clearing; GET proofs never mutate.
- Google start URL/scopes, proof cookie flags, callback without cookie/state,
  cancellation, concurrent tabs, failed exchange and link-session expiry.
- Check no proof in HTML/browser storage/logs and no external return redirects;
  confirmation screens keyboard accessible in both themes.
Use production build and real configured backend; write scenarios gated by
QA_CUSTOMER_WRITES=1 and owner approval. Document live Google login as a manual
owner check when credentials or browser consent are unavailable.

## API documentation
Consume the documented backend routes; do not change the backend. Keep request and response types local to the server client and never expose token-bearing envelopes to Client Components.

## Done when
Run `npm run check` without errors/warnings and `npm run build`. Run the
named Playwright flow against a production build on :3111, with BASE_URL
and MENU_API_URL configured. Read output and fix discrepancies. Never
start the backend or mock an unavailable backend; ask the owner to start it.
Real backend writes/orders/payments require the project's approval process.
Tick only after checks pass; commit as `task 36: Recovery and Google sign-in`.


## Out of scope
Google OAuth within Mini App, account/history screen (37), automatic merges or new providers.
