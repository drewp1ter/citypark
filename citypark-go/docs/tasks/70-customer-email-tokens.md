# Task 70: SMTP and single-use email tokens

## Goal
Provide configurable email delivery and reusable transactional hashed email-token
storage without exposing verification or recovery routes yet.

## Depends on
69.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/config/config.go`, `internal/customer/session.go`.

## Files to create
- `migrations/00023_customer_email_tokens.sql`
- `internal/customer/email.go`, `internal/customer/email_test.go`
- `internal/mailer/smtp.go`, `internal/mailer/smtp_test.go`

## Files to change
- `internal/config/config.go`, config tests, `.env.example`, `internal/app/app.go`: SMTP and public origin wiring.

## Spec
- Config: `CUSTOMER_PUBLIC_ORIGIN` is an absolute HTTPS origin in production
  (no path/query/userinfo; localhost HTTP allowed only development/test).
  `SMTP_HOST`, `SMTP_PORT` (default 587), `SMTP_USERNAME`, `SMTP_PASSWORD`,
  `SMTP_FROM`, `SMTP_TLS_MODE` (`starttls` default or `tls`). All-empty SMTP means
  disabled; partial config fails validation when customer auth is enabled.
- SMTP transport uses standard library, explicit TLS with hostname verification,
  TLS >=1.2, bounded dial/I/O deadlines and context cancellation. Require TLS
  before sending credentials; no production plaintext mode. Injectable dialer
  permits a local fake SMTP server in tests. Reject CR/LF in mail headers.
- Interface `Mailer.Send(ctx context.Context, to, subject, body string) error`;
  never log body, recipient token, passwords or auth commands. Log failures with
  purpose and nonsecret operation ID; return a sanitized internal error.
- Email tokens use 32 random bytes base64url; SHA-256 only in DB, bound to user,
  purpose and normalized target email. `verify` expires after 24h; `reset` after
  1h; `change-email` after 24h. Tokens never authorize another purpose/account.
- Consumption must run inside the same transaction as the resulting account
  mutation. Single conditional UPDATE with RETURNING prevents concurrent reuse.
  Failed downstream mutation rolls back consumption. Public routes arrive later.
- Invalidate earlier outstanding tokens for the same user/purpose when replacing
  one. Failure to deliver invalidates the new token; public generic response
  handling is defined in 71/72. No sending email before transaction commit.

```go
type EmailToken struct { UserID int64; Purpose, Email string }
func (s *Service) IssueEmailToken(ctx context.Context, userID int64, purpose, email string) (raw string, err error)
func (s *Store) ConsumeEmailToken(ctx context.Context, hash []byte, purpose string, now time.Time) (EmailToken, error)
```
Service has injected random source/clock/mailer; missing, consumed, wrong-purpose
and expired tokens map to one internal invalid-token error.

```sql
CREATE TABLE customer_email_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32),
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('verify','reset','change-email')),
    email text NOT NULL, expires_at timestamptz NOT NULL, consumed_at timestamptz
);
CREATE INDEX customer_email_tokens_user_idx ON customer_email_tokens(user_id,purpose);
UPDATE customer_email_tokens SET consumed_at=$3
WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL;
INSERT INTO customer_email_tokens(token_hash,user_id,purpose,email,expires_at)
VALUES ($1,$2,$3,$4,$5);
UPDATE customer_email_tokens SET consumed_at=$3
WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>$3
RETURNING user_id,purpose,email;
```
Down drops the table. Keep raw proof out of SQL errors and logs.

## Tests to write
- `TestSMTPConfig`: absent/partial/valid config; unsafe origins, bad ports/modes,
  invalid sender, CRLF rejection; old staff Telegram config stays unchanged.
- `TestSMTPSend`: TLS/STARTTLS success, unavailable TLS, certificate failure,
  authentication refusal, recipient rejection, timeout/cancellation; assert no
  credential transmission before TLS and sanitized logs.
- `TestEmailTokens`: exact lifetimes, hashes only, different purpose/user, expiry
  boundary, resend invalidation, delivery failure, reuse, concurrent consume.
- `TestEmailTokenTransaction`: failed account update rolls consumption back;
  successful transaction consumes once. Use existing DB and an injected mailer.

## API documentation
No public endpoints or OpenAPI changes in this task.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 70: SMTP and single-use email tokens`.


## Out of scope
Verification/signup routes (71), reset routes/rate limits (72), live SMTP sends and real credentials.
