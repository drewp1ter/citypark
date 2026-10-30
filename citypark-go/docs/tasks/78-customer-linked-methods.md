# Task 78: Linked methods and linking integration

## Goal
Expose safe linked-method information and verify that explicit Google and
Telegram linking compose correctly with customer accounts and owned orders.

## Depends on
74, 77.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/tasks/CUSTOMER_ACCOUNTS.md`, `internal/customer/handler.go`, `internal/customer/pairing.go`.

## Files to create
- `internal/customer/methods.go`, `internal/customer/methods_test.go`
- `internal/app/customer_linking_test.go`

## Files to change
- `internal/customer/handler.go`, `internal/app/app.go`: listing route and integration wiring.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: unified customer auth/linking documentation.
- `../citypark-admin/src/shared/api/schema.d.ts` only if the existing type-generation command changes the shared generated contract; read admin rules first.

## Spec
| Method | Path | Role | Behavior | Success |
|---|---|---|---|---|
| GET | /customer/methods | customer session | list current linked login methods | 200 |

- Stable shape `{"methods":[{"provider":"email","verified":true},{"provider":"telegram","verified":true}]}`;
  order email,google,telegram, omit unlinked providers, [] for no methods. Email
  verified flag comes from account confirmation; provider identities are verified
  by validated proofs when attached. No subjects, hashes, email addresses, IDs or
  token data. This endpoint alone must never link/unlink any method.
- No session/invalid session ->401 `Требуется авторизация`; staff JWT ->403
  `Недостаточно прав`; private,no-store on success/errors.
- Google explicit linking remains start/complete with intent=link (73–74).
  Telegram explicit linking remains two-sided pairings (77). Do not add bypass
  endpoints, generic account-ID merge requests or phone/email lookup linking.
- Recheck conflict mapping and account locks across Google callbacks, pairing,
  password reset and email changes. No automatic account merging on provider
  email. Method list after a merge reflects destination immediately.

```go
type Method struct { Provider string `json:"provider"`; Verified bool `json:"verified"` }
func (s *Store) Methods(ctx context.Context, userID int64) ([]Method, error)
```

```sql
SELECT i.provider,CASE WHEN i.provider='email' THEN u.is_email_confirmed ELSE true END AS verified
FROM customer_identities i JOIN users u ON u.id=i.user_id WHERE i.user_id=$1
ORDER BY CASE i.provider WHEN 'email' THEN 1 WHEN 'google' THEN 2 ELSE 3 END;
```
No new migration. All integration fixtures use fake provider/mailer implementations
and the real configured test database; keep existing admin schema fields stable.

## Tests to write
- `TestMethods`: none/each/all providers, deterministic order, unverified email,
  no secret fields, unauthenticated/admin rejection, no-store.
- `TestCustomerLinkingIntegration`: Google subject sign-in vs explicit link,
  Telegram-only account, source/destination each with orders, pairing confirms
  both sides, transferred addresses/owned history, source revocation, guest orders
  remain unclaimed; cross-account detail stays404 before and after linking.
- `TestConcurrentLinkingIntegration`: racing Google link/Telegram pairing/reset,
  incompatible credentials, replayed proof; one consistent committed result.
- `TestStaffAuthRegression`: same admin Bearer login/profile/order response
  shapes; notification/callback/iiko existing regression suites pass.

## API documentation
`docs/API.md` and `internal/apidocs/openapi.yaml`: document all routes above, security, no-store, request/response examples copied from tests and every error. Preserve admin JSON; regenerate admin types only if shared shapes change.

## Done when
Load `.env.development` and run `gofmt -l .` (no output), `go build ./...`,
`go vet ./...`, `go test -p 1 ./...`, and `go test -p 1 -cover` for every
changed internal package (each at least 80%). Use the existing database;
never start Docker or replace the database. Update API/OpenAPI for changed
endpoints, tick this task only after checks pass, and commit as
`task 78: Linked methods and linking integration`.


## Out of scope
Unlinking methods, additional providers, frontend (35–40), live provider/SMTP/payment requests.
