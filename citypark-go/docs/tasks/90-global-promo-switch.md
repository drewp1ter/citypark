# Task 90: Global promo-code switch

## Goal
Implement the global promo-code controls and checkout behavior described below, preserving existing orders.

## Depends on
Backend tasks 51 and 89.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/settings/store.go
- `internal/order/quote.go`
- `internal/order/service.go``

## Files to create
None expected; extend existing implementation and tests.

## Files to change
The implementation files listed above, their existing tests, and the supporting files explicitly named in the spec.

## Spec
- Add `PromoCodesEnabled bool` with JSON name `promoCodesEnabled` to Settings. Existing JSON without the field and omitted PUT values decode as false. Persist it in the existing JSON settings object; no migration is needed.
- GET /settings (public, 200) and PUT /settings (admin, 200) include the boolean; keep existing authorization and validation. Example added property: `"promoCodesEnabled": false`.
- For a nonempty trimmed promo code, read settings inside the pricing transaction before campaign lookup in quotes and new-order creation. Disabled returns HTTP 400 with the existing validation envelope and `errors.promoCode` equal to `Промокоды временно отключены.` Missing or unreadable settings must never permit a promo; propagate normal store/server errors.
- No-code quotes and orders remain available. Preserve campaign configuration, saved pricing snapshots, duplicate-order retries, payment callbacks and dispatch for existing orders. The switch must not retroactively remove discounts. Keep the existing RMS configuration guard, but check the global switch first for new promo orders.
- Update docs/API.md and internal/apidocs/openapi.yaml for settings, quotes and order creation; document defaults and the new error.

## Tests to write
- Settings round trip for true/false; existing JSON and omitted PUT property default false; existing 401/403/400 settings behavior remains.
- Enabled quote/order succeeds with a valid campaign and configuration; disabled quote/order returns the exact promoCode error and creates no business records.
- No-code checkout succeeds while disabled; saved-order retries and payment processing preserve their snapshots after disabling.
- Settings read failures fail promo pricing; disabled state takes precedence over missing RMS configuration.
- Update existing promo fixtures to enable the global flag explicitly; use testutil.NewDB and testutil.DoJSON, never start a database.

## API documentation
Update the endpoint sections and OpenAPI listed in the spec.

## Done when
```bash
gofmt -l .
go build ./...
go vet ./...
# Load TEST_DB_DSN from .env.development first; never start a database.
go test -p 1 ./...
go test -p 1 -cover ./internal/settings/... ./internal/order/... ./internal/apidocs/...
```
All checks required by CONVENTIONS pass. Backend changed-package statement coverage is at least 80% where applicable. Use configured services; do not start the backend/database. Live QA writes require existing session authorization; otherwise document the exact pending checks.

## Out of scope
Other projects, unrelated task completion, individual campaign redesign, retroactive repricing, real payments and iiko dispatch. On completion, tick this task and commit according to the project workflow.
