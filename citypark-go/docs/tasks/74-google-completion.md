# Task 74: Google completion and explicit linking

## Goal
Complete a browser-bound authorization attempt, validate the Google ID token
and sign in by subject or link only to an explicitly authenticated account.

## Depends on
73.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/google/authorization.go`, `internal/customer/store.go`.

## Files to create
- `internal/google/completion.go`, `internal/google/completion_test.go`
- `internal/google/id_token.go`, `internal/google/id_token_test.go`

## Files to change
- `internal/google/authorization.go`, `internal/app/app.go`: completion handler and dependencies.
- `docs/API.md`, `internal/apidocs/openapi.yaml`.

## Spec
Implement Google's documented OIDC code flow using injected HTTP endpoints,
bounded HTTP client timeouts and the existing JWT library; no implicit flow.
No new dependency is authorized by this task.

| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| POST | /auth/customer/google/complete | browser flow proof; bound web session if link | consume attempt and exchange/validate code | 200 |

Request `{"state":"...","flowToken":"...","code":"..."}` or cancellation
`{"state":"...","flowToken":"...","error":"access_denied"}`. Exactly one
of code/error. Return session envelope from 71 for login; `{"linked":true}` for
linking with current session retained. Error/cancellation still consumes attempt.
- Atomically claim an unexpired attempt by state and flow-token hashes before
  exchange. Concurrent/repeated callback cannot exchange twice. On network or
  provider failure require a fresh start, never reuse consumed state/code.
  Link additionally revalidates exact bound recent session and account at start
  and at commit. Never accept target ID from callback body or ID-token email.
- Exchange code server-side with configured client secret, exact callback and
  stored PKCE verifier. Verify RS256 signature using Google's discovered JWKS
  URI, issuer (`accounts.google.com` or `https://accounts.google.com`), audience,
  `azp` where multiple audiences exist, exp/iat with max30s tolerance, nonce hash,
  nonempty subject. Reject algorithm confusion, missing key/claims and unsigned
  tokens. Require email_verified=true when using returned email as provider
  metadata; never adopt it as an existing password credential or merge key.
- Cache discovery/JWKS by provider cache headers with a bounded expiry; refresh
  once for unknown kid, then reject; no arbitrary jku/x5u HTTP requests. Tests use
  local signed tokens/servers, never Google production.
- Existing Google subject signs into its owner. Unknown subject creates a new
  user with NULL phone, NULL authentication email and no password, then reserves
  the subject atomically. Concurrent first login resolves to the one identity
  owner, never creates duplicate accounts. Profile name may seed display name
  max80, fallback `Пользователь`; provider email is optional metadata, not login.
- Explicit link attaches subject only to bound account, same subject/owner is
  idempotent; subject on another account or different Google method already on
  destination -> 409 `Способ входа уже используется`. No account merges here.
- Invalid flow/code/token -> 400 `Не удалось подтвердить вход через Google`;
  cancellation -> 400 `Вход через Google отменён`; provider unavailable -> 503
  `Вход через Google недоступен`; missing/stale link session uses 73 errors.
  Rate limit completion 20/IP/15min; all responses/errors no-store. Log only
  failure category and nonsecret request ID, never proofs or full provider body.

```go
func (s *Service) Complete(ctx context.Context, state, flowToken, code, providerError string, session *customer.Session) (CompletionResult, error)
// CompletionResult holds either the customer session envelope or Linked=true.
```

```sql
UPDATE customer_oauth_attempts SET consumed_at=$3
WHERE state_hash=$1 AND flow_hash=$2 AND consumed_at IS NULL AND expires_at>$3
RETURNING nonce_hash,pkce_verifier,intent,user_id,session_hash;
UPDATE customer_oauth_attempts SET pkce_verifier='' WHERE state_hash=$1;
```
Use identity reservation SQL from 68 in account/identity/session transactions;
unique identity conflicts must not leave orphan users or partial links.

## Tests to write
- `TestGoogleCompletion`: successful login, new account, verified-email metadata
  matching another account never merges, explicit link success/idempotence/conflict.
- `TestGoogleIDTokenValidation`: tampered signature, alg none/HS256, wrong/missing
  issuer/audience/azp/nonce/subject, expired/future token, unknown kid, key rotation,
  missing/unverified email does not become an authentication email.
- `TestGoogleFlowProofs`: wrong browser flow/state, expired/replayed/concurrent
  callback, cancellation, stale/revoked/different link session, missing fields.
- `TestGoogleProviderFailure`: exchange/JWKS timeout, malformed payload, rejection,
  no second exchange on retry, no secret logging, rate limits/private responses.
Use httptest provider and generated RSA keys; configured existing DB for writes.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 74: Google completion and explicit linking`.


## Out of scope
Telegram linking/merges (76–78), Google OAuth in Telegram browsers, live Google smoke tests and frontend cookie handling.
