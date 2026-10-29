# Task 72: Customer password recovery

## Goal
Add private, generic password-recovery requests and atomic single-use resets
that revoke every existing customer session.

## Depends on
71.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/customer/email_handler.go`, `internal/customer/limits.go`.

## Files to create
- `internal/customer/reset.go`, `internal/customer/reset_test.go`

## Files to change
- `internal/customer/handler.go`, `internal/app/app.go`: recovery routes.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /auth/customer/password/forgot | public | generic recovery request | 202 |
| POST | /auth/customer/password/reset | public | consume proof and replace password | 204 |

- Forgot takes `{"email":"ivan@example.com"}`, returns
  `{"message":"Если аккаунт найден, письмо для восстановления отправлено"}`
  for found/unknown/admin/unverified/provider-only accounts, send throttling and
  SMTP failure. Send only for an already enabled verified email/password method.
  No implicit new password method and no account lookup by phone.
- Link `/account/reset?token=...` uses CUSTOMER_PUBLIC_ORIGIN; one-hour expiry.
  GET cannot consume a proof. Reset takes
  `{"token":"<proof>","password":"new-example-password"}`; password uses
  registration length rules. Invalid/expired/wrong-purpose/used proof -> 400
  `Ссылка недействительна или устарела`. Invalid fields use existing validation.
- Lock account before proof consumption; recheck current email and verified
  credential. Bcrypt hash, clear legacy hash, consume token, invalidate every
  outstanding reset token and revoke all sessions in a single DB transaction.
  Never return a new session automatically. Reset reuse and concurrent reset
  have one winner. Failure rolls back both credential and token changes.
- Reuse limits from 71: sends 3/email/hour,10/IP/hour (shared email-send budget),
  reset 20/IP/15min -> 429 with Retry-After; lookup/send always generic 202.
  All responses/errors no-store. No secrets or email bodies in logs.

```go
func (s *Service) ResetPassword(ctx context.Context, rawToken, password string) error
```

```sql
SELECT id,email,is_email_confirmed FROM users WHERE id=$1 FOR UPDATE;
UPDATE users SET password_hash=$2,legacy_password_hash='',updated_at=now() WHERE id=$1;
UPDATE customer_email_tokens SET consumed_at=$2
WHERE user_id=$1 AND purpose='reset' AND consumed_at IS NULL;
UPDATE customer_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL;
```
Use hashed token consumption SQL from 70 in the same transaction, with consistent
account-before-token lock order shared with verification/email change.

## Tests to write
- `TestForgotPasswordPrivacy`: known/unknown/unverified/admin/Google-only/Telegram-only
  responses identical; limits and SMTP failures preserve generic result.
- `TestResetPassword`: valid proof changes bcrypt, clears legacy hash, revokes
  all sessions; old password denied, new password works through sign-in.
- `TestResetProofFailures`: missing/invalid body, forged/expired/wrong-purpose,
  changed email, replay and simultaneous use; exactly one committed reset.
- `TestResetRollback`: DB failure leaves password/session/token state unchanged.
Use existing configured DB, fake SMTP, injected clock and HTTP helpers.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 72: Customer password recovery`.


## Out of scope
Provider linking, guest-order claiming, staff password resets, live SMTP tests.
