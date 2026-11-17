# Task 108: Import dish visibility from iiko

## Goal
Import item-level `isHidden`, expose it for staff clients and reject hidden dishes in new ordering operations.

## Depends on
104. Explicit owner-requested catalog fix; independent of hotel tasks 106–107.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `internal/catalog/types.go`, `internal/order/service.go`.

## Files to create
- `migrations/00038_product_is_hidden.sql`
- `internal/iiko/product_hidden_test.go`
- `internal/catalog/product_hidden_test.go`
- `internal/catalog/product_hidden_migration_test.go`
- `internal/order/product_hidden_test.go`

## Files to change
- iiko item types/client tests and sync: decode and persist the flag.
- Catalog types/store and response-key tests: expose `isHidden`; hidden means unavailable.
- Order shared block validation and admin item handler: reject hidden dishes.
- API.md and OpenAPI: document additive field and ordering behavior.

## Spec
Add `IsHidden bool` with `json:"isHidden"` to MenuItem and ProductView, and `db:"is_hidden"` to Product.
Migration up: `ALTER TABLE products ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;`
Migration down: `ALTER TABLE products DROP COLUMN is_hidden;`
Sync includes `is_hidden` in INSERT and `is_hidden = EXCLUDED.is_hidden` in UPDATE. Missing flags decode as false.
Hidden priced products remain synced and listed, with stable IDs, images, manual blocks and descriptions preserved.
`isAvailableForOrder` additionally requires `!IsHidden`. This field is independent of `isOrderBlocked` and deletion.
Quotes, creation, duplicate creation and admin item additions/amount changes reject hidden dishes with HTTP 503:
`Заказ блюда «{name}» временно недоступен.` No order/payment changes on rejection.
Sale windows and manual-block behavior of admin edits remain as before. Historical orders, payment retries and category visibility are unchanged.

## Tests to write
- TestMenuItemHidden: true/false/omitted decode from a fake HTTP menu.
- TestSyncHiddenProducts: visible/hidden/restored, stable ID and staff fields, deletion/reappearance.
- TestProductHiddenMigration: existing row default, NOT NULL and rollback preserving products.
- TestHiddenProductResponses: lists, search, category reads and ID reads retain flag and fresh availability.
- TestCreateHiddenProduct: new and duplicate requests return 503 without side effects.
- TestQuoteHiddenProduct: 503 without orders/payments.
- TestAddHiddenItem: absent and existing order positions reject edits without changing historical amounts/prices.

## API documentation
Update product lists/mutation response schema, POST /orders, POST /orders/quote and POST /orders/{id}/items.
Customer clients must filter hidden dishes; staff clients retain them. Deploy before website 74/admin 37.

## Done when
CONVENTIONS checks pass; changed internal packages have at least 80% coverage. Use configured TEST_DB_DSN; never start services.

## Out of scope
Frontend implementation, category visibility changes, payment policy changes, production sync or deployment.
