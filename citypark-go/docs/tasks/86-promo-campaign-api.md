# Task 86: Promo campaign admin API

## Goal
Expose campaign administration to admins only.

## Depends on
85

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/promo/store.go`
- `internal/app/app.go`

## Files to create
- `internal/promo/handler.go`
- `internal/promo/handler_test.go`

## Files to change
- `internal/app/app.go`
- `internal/apidocs/openapi.yaml`
- `docs/API.md`

## Spec
Expose campaign administration to admins only.
Implement the four campaign routes in the shared contract with NewHandler and Routes using existing auth/httpx
patterns. Wrap create/update in store.InTx. Return full campaign objects and non-nil arrays. Enforce immutable
codes, exact validation messages and 409 duplicates. Timestamps are RFC3339; disabled campaigns remain visible.
No public list, deletion, analytics or redemption route. Register routes in app wiring.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestPromoAdminAuth: absent token → 401, customer token → 403 on every route.
- TestPromoAdminCRUD: create 201, get/list/update 200; saved state matches response.
- TestPromoAdminErrors: missing id 404, duplicate code 409, malformed JSON and each invalid field 400.
- TestPromoAdminDisable: disable active code, preserve configuration and include it in lists.
Use chi, testutil.DoJSON and real test database; assert each contract error text.

## API documentation
Add all four /promo-codes routes to docs/API.md and OpenAPI, including schemas, examples and every error status. Capture examples from tests.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
