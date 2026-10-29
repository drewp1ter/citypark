# Task 71: Verified email registration and sign-in

## Goal
Enable customer email/password authentication only after email ownership is
verified, including resend and explicit verified email changes.

## Depends on
70.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/user/auth_handler.go`, `internal/customer/email.go`.

## Files to create
- `internal/customer/email_handler.go`, `internal/customer/email_handler_test.go`
- `internal/customer/limits.go`, `internal/customer/limits_test.go`

## Files to change
- `internal/customer/store.go`, `internal/customer/handler.go`, `internal/app/app.go`: registration/verification/session wiring.
- `internal/user/auth_handler.go`, tests: admin compatibility and customer JWT cutoff.
- `docs/API.md`, `internal/apidocs/openapi.yaml`, `.env.example`.

## Spec
Paths below are relative to /api/v1. All auth responses, including errors,
send `Cache-Control: private, no-store`. Body fields use existing validators;
password 8–64 characters, name required max80, email required normalized, phone
optional/normalized. Never use phone to find or attach credentials to an account.

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /auth/customer/register | public | create pending password account and send proof | 202 |
| POST | /auth/customer/verification/resend | public | send only for pending account | 202 |
| POST | /auth/customer/verification/confirm | public | atomically verify email token | 200 |
| POST | /auth/customer/sign-in | public | verify enabled password method and issue session | 200 |
| POST | /customer/email/change | recent web session | stage new auth email without changing current email | 202 |
| POST | /customer/email/confirm | same recent web account | consume change-email token and switch credential | 200 |

- Register: `{"name":"Иван","email":"ivan@example.com","password":"example-password"}`;
  resend: `{"email":"ivan@example.com"}`. Both return
  `{"message":"Если адрес доступен, письмо с подтверждением отправлено"}`,
  including existing email/contact collisions and SMTP failure. Never overwrite
  hashes of an existing account. Log sanitized delivery failures for retry.
- Sign-in: `{"email":"ivan@example.com","password":"example-password"}` ->
  `{"user":{"id":1,"name":"Иван","email":"ivan@example.com","phone":"","role":"user"},"token":"cs_<random>","expiresAt":"2026-11-28T12:00:00Z"}`.
  Use validated email identity owner, bcrypt (or existing legacy hash upgrade)
  and is_email_confirmed. Unknown/wrong/unverified/admin all return 401
  `неверный логин или пароль.` with comparable password-check work. No session
  on verification; the customer signs in afterwards. Google email never verifies
  or enables this password credential implicitly.
- Confirm body `{"token":"<proof>"}` -> `{"verified":true}`;
  invalid/expired/consumed -> 400 `Ссылка недействительна или устарела`.
  Confirmation locks the user, checks current normalized email against token
  email, then consumes the token and sets is_email_confirmed=true atomically.
- Change body `{"email":"new@example.com"}` -> same generic 202. Require
  authenticated_at within 5 minutes, web surface, current non-admin user;
  otherwise 401 or 403 `Подтвердите вход повторно`. Store token target only until
  confirmation; current login still works. Confirmation moves the email identity
  reservation and users.email, invalidates other email proofs and revokes all
  other sessions. Conflicting reserved email -> 409 `Способ входа уже используется`;
  roll back token consumption. Do not convert Google-only accounts to password
  accounts without a separately verified password registration flow (out of scope).
- Rate-limit all routes now using constructor-owned fixed-window counters and
  injected clock: sign-in 10 per normalized-email hash/15min and 30 per direct
  RemoteAddr IP/15min; email sends 3 per email hash/hour and 10 per IP/hour;
  confirmations 20 per IP/15min. Bounded state with expired-entry cleanup. Shared
  helper reused in 72/73/75/77; deployment assumes a single backend instance.
  Ignore forwarded-IP headers unless configured trusted proxy handling exists.
  Public lookup/send endpoints keep generic 202 even when limited. Sign-in or
  confirmation throttling -> 429 `Слишком много попыток. Попробуйте позже`,
  Retry-After seconds. Never log raw email/token/password or expose existence.
- Email links target `/account/verify?token=...` and `/account/email?token=...`
  on configured origin; GET merely renders a confirmation form. Actual mutation
  is POST so mail scanners do not consume proofs. Recovery links in 72 follow this.

Core SQL inside the transaction (token consume from 70):
```sql
SELECT id,email,is_email_confirmed FROM users WHERE id=$1 FOR UPDATE;
UPDATE users SET is_email_confirmed=true,updated_at=now() WHERE id=$1 AND lower(btrim(email))=$2;
UPDATE customer_identities SET subject=$2 WHERE user_id=$1 AND provider='email';
UPDATE users SET email=$2,is_email_confirmed=true,updated_at=now() WHERE id=$1;
UPDATE customer_sessions SET revoked_at=$3 WHERE user_id=$1 AND token_hash<>$2 AND revoked_at IS NULL;
```
Use Store methods with `ctx` first and `store.DBTX`; retain
`Service.CreateSession` and `Mailer` contracts from 69/70. Bad input uses existing
validation shape/messages; absent/invalid session -> 401 `Требуется авторизация`.

## Tests to write
- `TestVerifiedRegistration`: required email/optional phone; normalized duplicate
  and phone-contact collisions generic; no existing credential modification;
  pending sign-in denied, verified sign-in succeeds, admin login unchanged.
- `TestVerification`: valid proof, 24h boundary, wrong purpose/account/email,
  replay/concurrent consumption, resend invalidation, SMTP failure generic.
- `TestVerifiedEmailChange`: recent-session checks; current email remains until
  confirmation, reserved conflict rollback, no automatic provider/password linking,
  successful switch revokes other sessions, old email cannot sign in.
- `TestCustomerAuthLimits`: all window boundaries, IP/key isolation, forwarded-IP
  spoofing, bounded cleanup, generic lookup body and Retry-After where specified.
Use testutil.NewDB, clock.Fixed, fake mailer and existing HTTP helpers; verify
stored rows and no-store responses, plus invalid bodies and credential errors.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 71: Verified email registration and sign-in`.


## Out of scope
Password recovery (72), Google/Telegram, real email delivery, frontend screens; multi-instance rate-limit storage.
