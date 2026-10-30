# Task 76: Transactional customer account merge

## Goal
Provide an internal merge primitive that moves authenticated ownership and
identities into the website account without claiming guest orders.

## Depends on
68, 69, 75.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/customer/store.go`, `internal/address/store.go`.

## Files to create
- `migrations/00026_customer_account_merge.sql`
- `internal/customer/merge.go`, `internal/customer/merge_test.go`

## Files to change
- `internal/customer/session.go`, `internal/customer/email.go`: reject merged users and serialize issuance/token mutations with merge.
- `internal/google/completion.go`, `internal/customer/telegram.go`: account locks during identity/session creation.
- `internal/user/auth_handler.go`: merged users cannot regain customer authentication.
- `internal/customer/store.go`: shared account lock helper.

## Spec
- Internal only: callers in 77 must prove both parties confirmed. Never expose
  an endpoint that accepts arbitrary source/destination account IDs for merging.
- Destination is the website account. Require distinct, existing, non-admin,
  unmerged source/destination. Reject admins and merged accounts; no staff changes.
- Lock both users with SELECT FOR UPDATE in ascending ID order, then identities,
  tokens and sessions consistently. Other session/credential/identity mutation
  paths must take the same user lock before committing, so a concurrent login,
  reset or provider link cannot revive/reassign the source during merge.
- For provider email/google/telegram, at most one identity of each per account.
  If both accounts have different subjects for the same provider, reject. If
  both have any password/legacy-password credential, reject rather than guessing
  equivalence. Destination authentication-email reservation conflict rejects.
  Validate every conflict before moving anything.
- If destination lacks an email/password method, transfer source credential
  email/hash/confirmed state and email identity; clear authentication fields on
  source in the same transaction (release its unique email before destination
  assignment). Destination display name/phone stay unchanged; source phone/name
  remain historical contacts. Never use phones to resolve conflicts or merge.
- Transfer identities, `orders.account_id` and saved addresses. Guest orders
  (`account_id IS NULL`) and `orders.customer_id`/checkout snapshots never change.
  Duplicate saved addresses use ON CONFLICT DO NOTHING. Keep source users row
  as a checkout contact marked merged; do not delete referenced historical data.
- Revoke all source sessions, invalidate source email/OAuth proofs and mark
  merged_into_user_id. Source cannot authenticate or receive new credentials.
  Existing destination sessions stay valid. After merge, Telegram subject creates
  sessions for destination; do not redirect stale source proofs to destination.
- Conflict -> internal ErrMergeConflict mapped by caller to 409
  `Аккаунты нельзя объединить: способы входа конфликтуют`; forbidden source ->
  ErrMergeForbidden mapped to 403 `Эти аккаунты нельзя объединить`.
  No partial transfer on failure; all I/O takes ctx first.

```go
func (s *Service) MergeAccounts(ctx context.Context, sourceID, destinationID int64) error
// Add a transaction-capable internal helper so task77 can merge and consume
// pairing in the same transaction; it must not commit a nested transaction.
```

```sql
ALTER TABLE users ADD COLUMN merged_into_user_id bigint REFERENCES users(id),
    ADD CONSTRAINT users_merge_not_self CHECK(merged_into_user_id IS NULL OR merged_into_user_id<>id);
SELECT id FROM users WHERE id=ANY($1) ORDER BY id FOR UPDATE;
UPDATE orders SET account_id=$2,updated_at=now() WHERE account_id=$1;
INSERT INTO user_addresses(user_id,address_id)
SELECT $2,address_id FROM user_addresses WHERE user_id=$1 ON CONFLICT DO NOTHING;
DELETE FROM user_addresses WHERE user_id=$1;
UPDATE customer_identities SET user_id=$2 WHERE user_id=$1;
UPDATE customer_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL;
UPDATE customer_email_tokens SET consumed_at=$2 WHERE user_id=$1 AND consumed_at IS NULL;
UPDATE customer_oauth_attempts SET consumed_at=$2,pkce_verifier='' WHERE user_id=$1 AND consumed_at IS NULL;
UPDATE users SET merged_into_user_id=$2,email=NULL,password_hash='',legacy_password_hash='',
    is_email_confirmed=false,updated_at=now() WHERE id=$1;
```
Use parameterized credential-copy UPDATE after clearing source email, preserving
password hashes verbatim. Down drops marker/constraint but cannot undo data merges;
document this irreversible data effect; do not attempt to reconstruct ownership.

## Tests to write
- `TestMergeAccounts`: source/destination both have orders; all identities and
  owned orders visible to destination, snapshots/guest orders unchanged, addresses
  deduplicated, source sessions/proofs invalid, destination sessions work.
- `TestMergeCredentialTransfer`: source-only credential moves with verification;
  destination profile stays; old source cannot regain auth or reset password.
- `TestMergeConflicts`: admin/equal/missing/merged users, duplicate provider and
  two password credentials; every DB row unchanged after rejected merge.
- `TestMergeConcurrency`: opposite merges, login/reset/link racing merge; locks
  prevent double transfer, deadlock or post-merge source sessions.
Use real configured PostgreSQL, explicit goroutine barriers (no t.Parallel),
read back rows; test the caller-owned transaction helper rollback.

## API documentation
No public endpoint added. Document merge effects and one-way data migration in the release notes in `docs/tasks/CUSTOMER_ACCOUNTS.md`.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 76: Transactional customer account merge`.


## Out of scope
Pairing authorization/endpoints (77), guest-order claiming, phone merges, automatic email-based merging.
