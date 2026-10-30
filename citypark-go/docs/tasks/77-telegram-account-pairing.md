# Task 77: Two-sided Telegram account pairing

## Goal
Pair a Mini App account to an authenticated website account using an expiring
code and explicit recent confirmation on both sides before merging.

## Depends on
76.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/customer/merge.go`, `internal/customer/session.go`.

## Files to create
- `migrations/00027_customer_pairings.sql`
- `internal/customer/pairing.go`, `internal/customer/pairing_test.go`

## Files to change
- `internal/customer/handler.go`, `internal/app/app.go`: pairing routes.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /customer/pairings | recent mini session | create single-use code | 201 |
| POST | /customer/pairings/redeem | recent web session | bind code to website account | 200 |
| GET | /customer/pairings/{id} | participating recent session | inspect own pending pairing | 200 |
| POST | /customer/pairings/{id}/confirm | participating recent session | confirm this side; merge on second confirmation | 200 |
| DELETE | /customer/pairings/{id} | participating session | cancel pending pairing | 204 |

- Recent means authenticated_at within 5min, actual server-stored surface mini/
  web; no client surface override. Source mini must have method telegram.
  Requesting code is not confirmation, redeeming is not confirmation. Require
  fresh authentication at creation, redeem and each confirm; stale ->403
  `Подтвердите вход повторно`, absent/invalid ->401 `Требуется авторизация`.
- Code: 16 random bytes base64url, returned only at creation with 5-minute
  absolute expiry. Store SHA-256, never plaintext. Pairing ID independent random
  16 bytes base64url; ID alone grants no access. Creating replaces any previous
  pending source pairing. Creation response
  `{"id":"<id>","code":"<code>","expiresAt":"2026-10-29T12:05:00Z"}`.
- Redeem `{"code":"<code>"}` atomically binds an unused/unexpired code to the
  website destination and exact source/destination session hashes; 200 status
  `{"id":"<id>","status":"pending","sourceConfirmed":false,"destinationConfirmed":false,"otherName":"Иван","expiresAt":"2026-10-29T12:05:00Z"}`.
  GET/confirm use same shape with status pending/completed/canceled, current flags;
  otherName is only peer display name, never email/phone. No raw code after create.
- One successful redemption; reuse ->400 `Код недействителен или устарел`.
  Same source/destination account ->409 `Аккаунты уже связаны`.
  Unrelated GET/confirm/delete returns404 `Не найдено`. Expired proof or pairing
  ->400 `Код недействителен или устарел`. Surface mismatch ->403
  `Подтвердите связь в нужном интерфейсе`.
- Confirmation has empty JSON `{}`. Lock user rows in ascending ID order, then
  pairing, revalidate both bound sessions are active/recent and both accounts
  unchanged. Record current side once (idempotent retries); the second side
  merges through task76 helper and marks completed in the SAME transaction.
  No merge before both flags. Conflict uses76 errors with no partial transfer.
- Completed destination may GET status; source sessions are revoked, so source
  UI must reauthenticate with a new Telegram launch to see shared history.
  Confirmation retry by valid destination returns completed, never remerges.
  Cancel consumes pending pairing; completed/canceled cancellation is idempotent.
- Limit create 5/user/hour, redeem10/IP/15min and5/user/15min, confirm20/user/15min;
  429 uses71 message/Retry-After. No-store on all responses/errors. No codes in
  URLs/logs/traces; place redeem code only in POST body.

```go
func (s *Service) ConfirmPairing(ctx context.Context, pairingID string, session Session) (PairingStatus, error)
```

```sql
CREATE TABLE customer_pairings (
    id text PRIMARY KEY, code_hash bytea NOT NULL UNIQUE,
    source_id bigint NOT NULL REFERENCES users(id), destination_id bigint REFERENCES users(id),
    source_session_hash bytea NOT NULL, destination_session_hash bytea,
    expires_at timestamptz NOT NULL, redeemed_at timestamptz,
    source_confirmed_at timestamptz, destination_confirmed_at timestamptz,
    completed_at timestamptz, canceled_at timestamptz
);
INSERT INTO customer_pairings(id,code_hash,source_id,source_session_hash,expires_at)
VALUES($1,$2,$3,$4,$5);
UPDATE customer_pairings SET destination_id=$2,destination_session_hash=$3,redeemed_at=$4
WHERE code_hash=$1 AND redeemed_at IS NULL AND expires_at>$4 AND canceled_at IS NULL
RETURNING id;
SELECT * FROM customer_pairings WHERE id=$1 FOR UPDATE;
UPDATE customer_pairings SET source_confirmed_at=$2 WHERE id=$1 AND source_confirmed_at IS NULL;
UPDATE customer_pairings SET destination_confirmed_at=$2 WHERE id=$1 AND destination_confirmed_at IS NULL;
UPDATE customer_pairings SET completed_at=$2 WHERE id=$1;
UPDATE customer_pairings SET canceled_at=$2 WHERE source_id=$1 AND completed_at IS NULL AND canceled_at IS NULL;
```
Down drops pairings. Bind all confirmation SQL to authenticated participant IDs
and session hashes in transaction, never request-supplied account IDs.

## Tests to write
- `TestPairingLifecycle`: mini creation/web redeem/GET/two confirmations in both
  orders; no merge after one confirmation; completed destination history shared.
- `TestPairingProofs`: random codes/hashes only,5min expiry, replay, wrong surface,
  stale/revoked/different session, unrelated account404, same account409.
- `TestPairingConcurrency`: simultaneous redeem has one winner; simultaneous
  confirmations merge once; replacement/cancel vs confirm cannot resurrect code.
- `TestPairingMergeConflict`: both accounts with incompatible methods and admin
  rejection leave orders/identities/sessions untouched; no deadlocks.
- `TestPairingLimitsAndPrivacy`: exact budgets, no-store, no code leakage;
  source must relaunch after merge. Use configured DB, clock and HTTP helpers.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 77: Two-sided Telegram account pairing`.


## Out of scope
Website/Mini App UI (37,39), Google account merging, guest-order claims.
