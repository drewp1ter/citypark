# Task 69: Opaque customer sessions

## Goal
Add revocable customer sessions and wire them into customer profile, address and
order access, keeping existing staff Bearer JWT authentication intact.

## Depends on
68.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/auth/middleware.go`, `internal/app/app.go`.

## Files to create
- `migrations/00022_customer_sessions.sql`
- `internal/customer/session.go`, `internal/customer/session_test.go`
- `internal/customer/handler.go`, `internal/customer/handler_test.go`

## Files to change
- `internal/order/handler.go`, `internal/address/handler.go`, `internal/user/handler.go`: inject customer/staff authentication adapters.
- `internal/user/auth_handler.go`: leave staff issuance unchanged; stop issuing customer JWTs when customer auth is enabled.
- `internal/app/app.go`, `internal/config/config.go`, config/app tests, `.env.example`: feature gate and dependencies.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
- `CUSTOMER_AUTH_ENABLED=false` by default until 71 is ready. Disabled customer
  auth routes return 503 `Вход временно недоступен`; staff routes remain usable.
- Tokens: `cs_` followed by base64url of 32 crypto/rand bytes. Persist SHA-256
  only. Absolute lifetime 30 days, never extended by reads. Store authentication
  time/method/surface; server sets surface `web` for password/Google and `mini`
  for Telegram. Neither request bodies nor browser headers may choose a surface.
- Customer calls use `Authorization: Bearer cs_<random>`. Route adapters check
  exact single header, token length/encoding, expiry, revocation, account role and
  account existence; malformed or unknown credentials return 401
  `Требуется авторизация`. Expiry is exclusive (`now >= expires_at` fails).
- When enabled, reject legacy role=user JWTs on customer profile/address/order
  routes; still accept valid staff JWTs only where admin access existed. Admin
  sessions may not be created in customer storage. Optional checkout with no
  header remains guest; any supplied invalid credential fails before writes.
- Add customer `GET /customer/me` -> 200 `{"id":1,"name":"Иван","email":"","phone":"","role":"user"}`;
  `PUT /customer/me` takes `{"name":"Иван","phone":""}` and returns the same
  shape; no email/password/role mutation. Forbid those fields with 400
  `Недопустимые поля профиля`. Supplied phone normalizes; duplicate -> 409
  `Телефон уже используется`. Preserve existing admin /user response shape.
- `POST /auth/customer/logout` requires a valid session and returns 204 after
  revocation. A subsequent use returns 401. All private responses/errors no-store.

```go
type Session struct { UserID int64; TokenHash []byte; AuthenticatedAt, ExpiresAt time.Time; Method, Surface string }
func (s *Service) CreateSession(ctx context.Context, userID int64, method, surface string) (token string, expiresAt time.Time, err error)
func (s *Service) Authenticate(ctx context.Context, token string) (Session, error)
func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error
func (s *Store) RevokeAllSessions(ctx context.Context, userID int64) error
```
Add `NewService(db *sqlx.DB, c clock.Clock) *Service`; extend constructor options
when mail/provider dependencies arrive, keeping them injected. Store remains
transaction-compatible via store.DBTX. Service takes injected store/clock. Customer identity goes into existing
auth.NewContext for order ownership; never convert staff tokens into customers.

```sql
CREATE TABLE customer_sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    authenticated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    method text NOT NULL CHECK (method IN ('password','google','telegram')),
    surface text NOT NULL CHECK (surface IN ('web','mini')),
    revoked_at timestamptz
);
CREATE INDEX customer_sessions_user_id_idx ON customer_sessions(user_id);
INSERT INTO customer_sessions (token_hash,user_id,authenticated_at,expires_at,method,surface)
VALUES ($1,$2,$3,$4,$5,$6);
SELECT user_id,token_hash,authenticated_at,expires_at,method,surface
FROM customer_sessions WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>$2;
UPDATE customer_sessions SET revoked_at=$2 WHERE token_hash=$1 AND revoked_at IS NULL;
UPDATE customer_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL;
```
Down drops the table. No raw tokens in DB, logs, traces, error objects or URLs.

## Tests to write
- `TestCustomerSessions`: random tokens, hashed storage, 30-day boundary,
  revocation/all-session revocation, unknown/malformed/expired/deleted/admin user.
- `TestCustomerAuthenticationRoutes`: required/optional credentials; repeated,
  empty or wrong scheme; legacy customer JWT denied when enabled; no fallback.
- `TestCustomerProfile`: stable response, optional phone, protected fields,
  conflict, validation, missing session, no-store on success/errors.
- `TestCustomerOrderAndAddressAccess`: own history/details/addresses, cross-account
  denial, guest separation, phone snapshots unchanged, staff access regression.
Use existing DB/HTTP helpers and clock.Fixed. Cover the disabled feature gate.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 69: Opaque customer sessions`.


## Out of scope
Public customer password login/registration (71), OAuth/Telegram issuance (74–75), cookie handling (website 35), staff auth redesign.
