# Task 75: Telegram Mini App authentication

## Goal
Authenticate validated raw Telegram launches with the dedicated customer bot
and issue customer sessions without requiring email or phone.

## Depends on
69, 71 (shared rate-limit helper).

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/config/config.go`, `internal/customer/session.go`.

## Files to create
- `migrations/00025_customer_telegram_proofs.sql`
- `internal/customer/telegram.go`, `internal/customer/telegram_test.go`

## Files to change
- `internal/config/config.go`, config tests, `.env.example`, `internal/app/app.go`: separate customer bot token and Telegram auth handler.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
Read Telegram's current raw initData validation specification linked in
CUSTOMER_ACCOUNTS before coding. Use the bot-token HMAC validation flow; do not
confuse it with third-party Ed25519 validation rules.
- `CUSTOMER_TELEGRAM_BOT_TOKEN` is independent of existing TELEGRAM_TOKEN/chat IDs.
  Absent disables only Telegram auth: 503 `Вход через Telegram недоступен`.
  Never use staff bot configuration as fallback. No Bot API calls are needed.

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /auth/customer/telegram | public | validate launch and create mini session | 200 |

Body `{"initData":"query_id=...&user=...&auth_date=...&hash=..."}` max16KiB;
success uses the exact session envelope from 71 with method telegram, surface
mini. Input must be raw encoded initData, never initDataUnsafe or a client ID.
- Parse strictly, reject duplicate keys, malformed escapes, missing proof/date/
  user, malformed JSON, nonpositive/out-of-range integer Telegram IDs. Preserve
  original decoded field values for sorted data-check string; compare 32-byte
  HMAC using constant-time comparison after strict hex decoding. Match official
  hash-field exclusion rules, including signature handling, at implementation.
- Inject clock; accept auth_date no older than 5min, future skew max30s. No stale
  sessions can be made fresh by retrying. Deduplicate the canonical validated
  proof, not the raw query string (reordering cannot bypass replay protection).
- After validation, insert proof hash and create/find identity plus session in
  one transaction. Same proof reused/concurrent use has exactly one winner;
  replay returns 401. If response is lost, customer must relaunch for new proof.
  Delete expired proof records only after their whole acceptance window ends.
- Look up only canonical Telegram ID. New subject gets a new user with no email/
  phone/password; max80 trimmed display name, fallback `Пользователь`. Existing
  Telegram identities sign into their owner; matching names/emails/phones never
  attach to another user. Concurrent new launches with same ID share one owner.
- Invalid/expired/replayed proof -> 401 `Не удалось подтвердить вход через Telegram`;
  bad JSON/oversize uses existing httpx validation/size handling. Rate limit
  30/IP/15min -> 429, no-store always. Log only sanitized failure reason.

```go
func ValidateTelegramInitData(raw, botToken string, now time.Time) (TelegramLaunch, error)
type TelegramLaunch struct { ID int64; Name string; AuthDate time.Time; ProofHash []byte }
```

```sql
CREATE TABLE customer_telegram_proofs (
    proof_hash bytea PRIMARY KEY CHECK(octet_length(proof_hash)=32),
    expires_at timestamptz NOT NULL
);
INSERT INTO customer_telegram_proofs(proof_hash,expires_at) VALUES($1,$2);
DELETE FROM customer_telegram_proofs WHERE expires_at<=$1;
```
Down drops table. Store no raw initData. Identity creation uses 68 transactions.

## Tests to write
- `TestTelegramInitData`: official-algorithm fixture, Unicode/encoded values,
  tamper, wrong bot, malformed hex/query/JSON, duplicate fields, missing fields,
  unsafe client-only IDs, exact5min/future30s boundaries, invalid/large ID.
- `TestTelegramSessions`: new and existing subject; no phone/email; distinct
  subjects never merge, staff tokens/bots unaffected, no-store/disabled config.
- `TestTelegramReplay`: reordered raw query, duplicate proof and concurrent
  attempts -> one session; rollback on failed identity/session creation.
- `TestTelegramIdentityRace`: different valid launches for same subject -> one
  owner, no orphan user. Use clock.Fixed, pure fixtures and configured DB.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 75: Telegram Mini App authentication`.


## Out of scope
BotFather configuration/deployment, pairing (77), Mini App UI (website38), real Telegram launches.
