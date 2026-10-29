# Task 73: Google authorization start

## Goal
Create browser-bound, expiring Google authorization attempts with state, nonce
and PKCE; do not enable customer sign-in until completion is implemented.

## Depends on
69, 71 (shared rate-limit helper).

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/config/config.go`, `internal/customer/session.go`.

## Files to create
- `migrations/00024_customer_oauth_attempts.sql`
- `internal/google/authorization.go`, `internal/google/authorization_test.go`

## Files to change
- `internal/config/config.go`, config tests, `.env.example`, `internal/app/app.go`: Google options and disabled-by-default route wiring.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
Read Google's server-flow documentation linked in CUSTOMER_ACCOUNTS before
implementation. No provider request or real authorization is needed in tests.
Config: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_CALLBACK_URL`;
all absent disables Google (503 `Вход через Google недоступен`), partial enabled
config fails startup. Callback is exact HTTPS `/account/google/callback` on the
configured public origin, with localhost development exception. Never accept
arbitrary callback/origin/authorization endpoint values from a request.

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /auth/customer/google/start | public login/recent web session link | prepare attempt | 200 |

Request `{"intent":"login"}` or `{"intent":"link"}`; response
`{"authorizationUrl":"https://accounts.google.com/o/oauth2/v2/auth?...","flowToken":"<random>","expiresAt":"2026-10-29T12:10:00Z"}`.
- Generate independent random 32-byte state, nonce and browser flow token plus
  valid PKCE verifier (43–128 unreserved chars); method S256. Persist state,
  nonce and flow-token hashes, verifier, intent, optional requesting user/session
  hash and ten-minute absolute expiry. PKCE verifier is a short-lived backend
  secret; never return or log it. Raw state/nonce are in the authorization URL;
  raw flowToken goes only to Next server for an HttpOnly cookie (website 36).
- Request only `openid email profile`, response_type=code, configured redirect,
  state, nonce and S256 challenge. Login must not trust/accept target account ID.
  Link requires a recent (<=5min) web session; bind both user and exact session
  hash so a different browser/session cannot finish it. Mini sessions cannot
  start linking: 403 `Откройте сайт для подключения Google`.
- Invalid intent -> 400 `Некорректный способ входа`; missing session 401,
  stale -> 403 `Подтвердите вход повторно`. Start limit 20/IP/15min -> 429.
  All responses no-store. Do not add a frontend Google button until 74 is ready.

```go
// Constructor injects store, clock, client ID, callback URL and provider endpoints.
type StartResult struct { AuthorizationURL, FlowToken string; ExpiresAt time.Time }
func (s *Service) Start(ctx context.Context, intent string, session *customer.Session) (StartResult, error)
```

```sql
CREATE TABLE customer_oauth_attempts (
    state_hash bytea PRIMARY KEY, nonce_hash bytea NOT NULL,
    flow_hash bytea NOT NULL, pkce_verifier text NOT NULL,
    intent text NOT NULL CHECK(intent IN ('login','link')),
    user_id bigint REFERENCES users(id), session_hash bytea,
    expires_at timestamptz NOT NULL, consumed_at timestamptz,
    CHECK ((intent='login' AND user_id IS NULL AND session_hash IS NULL)
        OR (intent='link' AND user_id IS NOT NULL AND session_hash IS NOT NULL))
);
INSERT INTO customer_oauth_attempts(state_hash,nonce_hash,flow_hash,pkce_verifier,intent,user_id,session_hash,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8);
```
Down drops attempts. Delete expired attempt secrets during bounded cleanup.
Completion consumes once and clears verifier in 74.

## Tests to write
- `TestGoogleStart`: exact identity scopes/callback, unique state/nonce/flow,
  correct S256 challenge and ten-minute expiry, DB contains hashes not raw proofs.
- `TestGoogleStartLink`: recent web binding; unauthenticated/stale/mini denied;
  arbitrary destination IDs/callbacks cannot influence the stored account.
- `TestGoogleConfig`: absent/partial/valid credentials, unsafe/cross-origin callback.
- `TestGoogleStartLimits`: bad intent, rate limit, no-store success/errors;
  no staff authentication/config changes. Use DB helpers and clock.Fixed.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 73: Google authorization start`.


## Out of scope
Authorization-code exchange and sign-in (74), Google button/cookie handling (website 36), real Google credentials.
