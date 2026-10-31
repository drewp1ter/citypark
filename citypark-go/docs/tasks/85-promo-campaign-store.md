# Task 85: Promo campaign persistence

## Goal
Add campaign persistence with transactional targeting replacement.

## Depends on
84

## Read first
- `docs/tasks/CONVENTIONS.md`
- [Shared promo-code contract](PROMOCODES.md)
- `internal/catalog/store.go`
- `internal/store/store.go`

## Files to create
- `migrations/00029_promo_campaigns.sql`
- `internal/promo/types.go`
- `internal/promo/store.go`
- `internal/promo/store_test.go`

## Files to change
None.

## Spec
Add campaign persistence with transactional targeting replacement.
Create promo_codes (id, code unique, percentage, enabled, target, minimum_eligible_subtotal_kopecks,
maximum_discount_kopecks, starts_at, ends_at, created_at, updated_at), promo_code_products and
promo_code_categories (composite promo/id primary keys; FKs to campaigns and catalog). Use bigint money,
CHECK percentage BETWEEN 1 AND 99, positive optional money, target IN ('all','selected'), and valid date range.
Use the next free migration number if 00029 has since been allocated. The down migration drops these tables.
Export Campaign, Input (fields from the shared JSON contract), Store and NewStore(db store.DBTX).
Methods: List(ctx context.Context) ([]Campaign,error), ByID(ctx context.Context,id int64) (Campaign,error), ByCode(ctx context.Context,code string)
(Campaign,error), Create(ctx context.Context,in Input) (Campaign,error), Update(ctx context.Context,id int64,in Input) (Campaign,error).
Create/Update callers supply a transaction; lock campaign row FOR UPDATE before replacing targets. Reads used
for order pricing acquire FOR SHARE on the campaign row before loading selections. Parameterize SQL, return
store.ErrNotFound for absent rows, preserve code on update and ensure rollback on invalid targets.
Normalize and validate at the package boundary; store sorted unique non-nil target arrays. Menu sync must not
reset campaign targeting. FK deletion may remove a target link, never broaden selected targeting to all.

The shared contract fixes business rules, wire shapes, money arithmetic, messages and defaults.
Do not broaden scope or change those rules. Read installed library docs before using new library APIs.

## Tests to write
- TestCampaignRoundTrip: create/read/update/list with nullable limits/dates and mixed targets → exact persisted values.
- TestCampaignValidation: invalid percentage, money, date, target shape/code → rejected without writes.
- TestCampaignUniqueCode: normalized duplicates → unique violation; code edit → rejected.
- TestCampaignTargets: missing IDs → error; duplicate IDs → one link; failed replacement → rollback.
- TestCampaignLocking: target replacement cannot interleave with a locked pricing read.
Use testutil.NewDB, serial tests and database row assertions.

## API documentation
No endpoint changes in this task; consume the published backend contract.

## Done when
Run all CONVENTIONS.md checks: gofmt -l . (empty), go build ./..., go vet ./...,
go test -p 1 ./..., and go test -p 1 -cover for every changed internal package (at least 80%).
Load TEST_DB_DSN from .env.development; do not start PostgreSQL. Update API.md/OpenAPI where applicable.
Tick only this implemented task and use the required task commit after all checks pass.

## Out of scope
Other numbered promo tasks, refunds, loyalty, redemption limits, analytics, stacking, and unrelated fixes.
Do not mark dependencies complete or alter their existing blocker notes.
