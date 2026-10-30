# Telegram Mini App and customer accounts

This is the release backlog, starting at backend 66 and website 35. Follow one
task and commit per session. Task 66 is executable now; turn each subsequent
backlog entry into a task file following TASK_TEMPLATE before starting it.
Do not enable customer authentication until the ownership and credential safety
tasks are deployed. Existing staff Bearer JWT authentication stays unchanged.

## Backend sequence

| Task | Scope | Depends on |
|---|---|---|
| 66 | Add nullable order account ownership and backfill contact snapshots | 65 |
| 67 | Write/read snapshots; use account ownership for history/details; reject invalid optional authentication; isolate duplicate lookup by account or server-issued guest checkout ID; protect contact/credential data | 66 |
| 68 | Multiple provider identities per internal user; nullable account phones; verified email credential state; prevent profile updates from modifying authentication email | 67 |
| 69 | Hashed opaque customer sessions, creation/authentication/logout/revocation, 30-day absolute lifetime; preserve admin JWT routes | 68 |
| 70 | Configurable SMTP transport, hashed single-use email tokens, delivery failure handling and redacted logs | 69 |
| 71 | Require email for password registration; verify before enabling password sign-in; 24-hour verification tokens, resend, verified email changes | 70 |
| 72 | Password recovery/reset, one-hour tokens, generic lookup responses, rate limits, revoke sessions after reset | 71 |
| 73 | Google OIDC start with state, nonce, PKCE; configurable client credentials and callback; identity scopes only | 69 |
| 74 | Google callback cancellation/errors, ID-token validation and subject-based sign-in; explicit linking, never merge by matching email | 73 |
| 75 | Dedicated customer bot configuration and Telegram raw initData validation, five-minute validity, bounded clock tolerance; customer sessions without email/phone | 69 |
| 76 | Transactional account merge: lock accounts, transfer identities, owned orders and saved addresses into website account; reject admins/conflicting credentials, revoke source sessions, exclude guests | 68, 69, 75 |
| 77 | Random hashed single-use five-minute pairing code from authenticated Mini App; recent authentication and confirmation in both interfaces; atomic replay protection | 76 |
| 78 | Linked-method listing and explicit website Google/Telegram linking APIs; concurrent pairing/conflict integration tests | 74, 77 |

Keep endpoint changes accompanied by `docs/API.md` and
`internal/apidocs/openapi.yaml`. Preserve admin response shapes; regenerate admin
types if the shared contract changes. Keep private responses uncached. Complete
task 67 before exposing the new customer-session authentication in 69.

Task 76 merges are one-way data migrations: identities, owned orders and saved
addresses move to the website account in one transaction. Guest orders and
checkout contact snapshots stay as they were. The source user remains as a
historical contact with revoked sessions and proofs; rolling back the schema
marker cannot restore the previous ownership.

## Website sequence

| Task | Scope | Depends on |
|---|---|---|
| 35 | Server-only customer API client and secure HttpOnly same-site session cookies; Russian sign-in/registration/verification screens, guest checkout preserved | backend 71 |
| 36 | Recovery/reset, resend, Google sign-in/callback/cancellation, private uncached pages | 35; backend 72, 74 |
| 37 | Account page, linked methods, explicit linking and pairing confirmation; paginated history/details scoped to account | 36; backend 78 |
| 38 | `/mini` compact layout reusing menu/cart/checkout; raw initData to server, launch loading/errors, back button, viewport and safe-area handling | 37; backend 75 |
| 39 | Mini App account pairing and history; payment success/failure/cancellation and return-to-order navigation; Google linking stays on regular website | 38 |
| 40 | Production-build Playwright flows and release smoke checks; document BotFather setup, HTTPS origins, staged deployment and mobile manual matrix | 39 |

Use existing Russian copy conventions, tokens, components and FSD. Backend calls
stay on the Next.js server. Customer session tokens never enter browser storage.
Do not start Google OAuth inside Telegram's embedded browser.

## Release invariants and verification

- An authenticated owner is independent of phone contact. Contact numbers are
  required only at checkout and never establish ownership. No guest-order claims.
- Do not automatically merge on email, name or phone; Telegram alone is sufficient.
  Hotel features, other OAuth providers and staff authentication changes are out
  of scope.
- Pairing requires recent authentication and explicit confirmation on both sides;
  the destination is the website account, including when both accounts have orders.
- Test invalid, expired and replayed proofs; Google cancellation; SMTP failures;
  reset reuse; session expiry/revocation; conflicting/concurrent linking; shared
  history after linking; guest separation; cross-account denial; immutable checkout
  contacts; duplicate isolation. Preserve admin access, notifications, callbacks
  and iiko delivery payloads.
- Run `go test -p 1` with the existing configured database, required coverage and
  all task checks. Run website checks, production build and relevant Playwright
  flows when frontend work begins.
- Configure SMTP, Google credentials/callback URL, separate customer bot token
  and public HTTPS origin through server-side environment variables. Staff bots
  keep their own configuration. Logs must not contain tokens or pairing codes.
- Manually verify HTTPS Telegram launches on Android, iOS and Desktop, including
  payment success/failure/cancellation and return navigation. Follow the existing
  approval process before real backend writes, orders and payments.
- Deploy additive backend migrations before frontend features. Configure the
  customer bot Mini App entry through BotFather; enable entry points only after
  credentials are configured and smoke tests pass.

## Provider specifications to read during the relevant tasks

- Google server flow: https://developers.google.com/identity/openid-connect/openid-connect
- Telegram validation: https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
- Embedded-browser restriction: https://developers.google.com/identity/protocols/oauth2/web-server#disallowed-useragent
