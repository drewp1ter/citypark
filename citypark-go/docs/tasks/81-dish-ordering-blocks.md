# Task 81: Persistent dish ordering blocks

## Goal
Let admins block a whole dish from new ordering without hiding it. Preserve the block until manually removed, including across iiko syncs.

## Depends on
80. Provides the contract for citypark-admin task 26 and citypark-modern task 42.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/catalog/category_hours.go`, `internal/catalog/types.go`, `internal/order/service.go`.

## Files to create
- `migrations/00028_product_ordering_blocks.sql`: reversible boolean column migration.
- `internal/catalog/product_block.go`, `internal/catalog/product_block_test.go`: handler and tests.

## Files to change
- `internal/catalog/types.go`, `internal/catalog/store.go`, `internal/catalog/category_hours.go`: model, queries, setter and admin route registration.
- `internal/order/service.go`, `internal/order/service_test.go`: enforce blocks on creation and duplicate retries.
- `internal/iiko/sync_test.go`: verify sync preserves blocks; change `internal/iiko/sync.go` only if needed for preservation.
- `internal/apidocs/openapi.yaml`, `docs/API.md`: contract and examples.

## Spec
Add `products.is_order_blocked boolean NOT NULL DEFAULT false`; down migration drops the column. Product queries must load it, including order validation queries. Expose `isOrderBlocked` in product views. `isAvailableForOrder` is false when blocked, otherwise retain existing product/category schedule logic. Do not filter blocked dishes from public lists.

Add `SetProductOrderBlocked(ctx context.Context, id int64, blocked bool) error` to the catalog store, using:
```sql
UPDATE products SET is_order_blocked=$2, updated_at=now()
WHERE id=$1 AND NOT is_deleted RETURNING id
```
Map no row to the existing not-found convention. Sync must not overwrite the local flag.

| Method | Path (under /api/v1) | Role | Success |
| --- | --- | --- | --- |
| PUT | /products/{id}/ordering-block | admin | 200, updated public product view |

Request: `{ "isOrderBlocked": true }`. Require an explicit boolean; false unblocks, repeated values succeed. Use existing JSON/path validation, 401/403 authentication conventions and 404 for missing/deleted products. Missing/null flag returns 400 with field `isOrderBlocked`: `Укажите, заблокирован ли заказ блюда`; use the existing validation-envelope message.

Check blocks before persisting a new order and before returning a payment link for a duplicate retry. Reject with 503 and `Заказ блюда «<name>» временно недоступен.` Do not silently remove items. Keep existing schedule checks. Do not cancel existing orders or change payment callback behavior.

## Tests to write
Use `testutil.NewDB`, `testutil.DoJSON` and existing fixed-clock helpers; database tests run serially.
- `TestProductOrderingBlock`: default false, block/unblock, idempotence and persisted readback; public list retains the product and view combines blocks with schedules.
- `TestProductOrderingBlockErrors`: no token 401, wrong role 403, malformed/missing/null/nonboolean body 400, invalid ID per existing convention, missing/deleted product 404.
- `TestSyncPreservesProductOrderingBlock`: repeated sync keeps a block; new products default unblocked.
- `TestCreateBlockedProduct`: blocked item rejects the whole order with 503 and no order inserted; unblocked item follows schedule rules.
- `TestDuplicateBlockedProduct`: block after initial order creation, retry rejects before returning payment details; existing order remains unchanged.

## API documentation
Document PUT `/products/{id}/ordering-block`, the new field on product endpoints, effective availability, sync persistence and blocked-item errors on POST `/orders` including duplicate retries.

## Done when
Conventions build, vet, formatting and serial tests pass using `TEST_DB_DSN` from `.env.development`; changed internal packages have at least 80% coverage. Do not start services. Update PROGRESS and commit this task separately.

## Out of scope
Admin and public UI, automatic expiration, category blocks, cancelling existing orders or payments.
