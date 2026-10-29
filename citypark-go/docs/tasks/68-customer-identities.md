# Task 68: Customer identities and optional phones

## Goal
Represent linked identity methods independently of checkout contacts, support
accounts without phones and reserve authentication-email changes for verification.

## Depends on
67.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/user/store.go`, `internal/user/handler.go`.

## Files to create
- `migrations/00021_customer_identities.sql`
- `internal/customer/store.go`, `internal/customer/store_test.go`

## Files to change
- `internal/user/store.go`, `internal/user/handler.go` and their tests: nullable phone storage, stable JSON, protect email.
- `internal/user/auth_handler.go` and tests: optional phone for new customer registration, preserve admin branch.
- `internal/order/store.go` and tests: COALESCE nullable legacy phones on reads.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: profile/registration rules.

## Spec
- `users.id` is the sole account ID. No phone matching or new phone-based claims.
  Keep the unique phone constraint; SQL NULL means absent, not an empty phone.
  Keep `user.User.Phone string` and existing JSON: select `COALESCE(phone, '') AS phone`
  and write `NULLIF($phone, '')`. Audit every direct scan, seed and admin bootstrap.
- Keep `users.email`, hashes and `is_email_confirmed` as the email/password
  credential state. Add `IsEmailConfirmed bool` to the internal user row/select
  mapping without changing the public Response JSON. Do not infer verification
  from Google email claims. New
  password registrations require verification in 71; do not enable new auth yet.
- Identity rows are keyed by validated provider subject; email subjects are
  trimmed/lowercased addresses, Google subjects are provider `sub`, Telegram
  subjects are canonical positive decimal Telegram IDs. Never merge by email.
- Backfill email identity reservations only for non-admin users with nonempty
  email and either password hash. Preserve `is_email_confirmed` without promoting
  it. Reject normalized-email collisions transactionally with a clear migration
  error; do not select a winner. No identity for phone-only contacts or admins.
- For non-admin `PUT /user`, omitted/unchanged email preserves it; a different
  email returns 409 `Для изменения email подтвердите новый адрес`. Name/phone
  updates must not change authentication fields. Preserve staff behavior/shape.
- Registration accepts omitted phone; supplied phone still validates as before.
  Multiple absent phones work; duplicate supplied phones retain generic signup
  behavior. Schema backfill is the only legacy identity association in this task.

```go
// package customer; Store uses store.DBTX, NewStore(db store.DBTX).
type Identity struct {
    UserID int64 `db:"user_id"`
    Provider string `db:"provider"`
    Subject string `db:"subject"`
}
func (s *Store) IdentityOwner(ctx context.Context, provider, subject string) (int64, error)
func (s *Store) AddIdentity(ctx context.Context, identity Identity) error
```

Both methods use parameterized SQL; missing owner returns store.ErrNotFound;
unique conflicts are propagated for callers to map without reassignment.

```sql
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;
UPDATE users SET phone = NULL, updated_at = now() WHERE phone = '';
CREATE TABLE customer_identities (
    user_id bigint NOT NULL REFERENCES users(id),
    provider text NOT NULL CHECK (provider IN ('email','google','telegram')),
    subject text NOT NULL CHECK (subject <> ''),
    PRIMARY KEY (provider, subject), UNIQUE (user_id, provider)
);
INSERT INTO customer_identities (user_id, provider, subject)
SELECT id, 'email', lower(btrim(email)) FROM users
WHERE role = 'user' AND NULLIF(btrim(email), '') IS NOT NULL
AND (password_hash <> '' OR legacy_password_hash <> '');
SELECT user_id FROM customer_identities WHERE provider = $1 AND subject = $2;
INSERT INTO customer_identities (user_id, provider, subject) VALUES ($1, $2, $3);
```
Down drops identities, replaces NULL phones with deterministic unique placeholders
`'unassigned:' || id`, then restores NOT NULL; never invent callable phone numbers.
No identity rows or nullable-phone values may leak secrets in JSON.

## Tests to write
- `TestIdentityStore`: exact owner, missing provider/subject, duplicate subject
  across accounts and duplicate provider per account; assert rows are unchanged.
- `TestIdentityMigration`: password/legacy reservations, normalized collision
  rollback, admins/contacts excluded, confirmation flags preserved, up/down.
- `TestOptionalCustomerPhone`: multiple absent phones; valid/invalid/duplicate
  supplied phones; old admin sign-in/bootstrap and order scans still work.
- `TestProfileEmailProtection`: omitted/unchanged email, changed email 409,
  credentials unchanged after name/phone changes, unauthenticated 401.
Use testutil.NewDB, existing HTTP helpers; no live providers.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 68: Customer identities and optional phones`.


## Out of scope
Sessions (69), SMTP/tokens (70), verified signup/email change (71), password reset (72), provider authentication (73–75).
