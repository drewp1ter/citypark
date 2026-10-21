# Task 39: Hidden categories

## Goal
The sync stores the iiko `isHidden` flag of each items group in a new `categories.is_hidden` column, and `GET /categories` no longer lists hidden categories.

## Depends on
38.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `internal/catalog/store.go`, `internal/testutil/builders.go`.

Never open `internal/iiko/testdata/`.

## Files to create
- `migrations/00003_category_is_hidden.sql`

## Files to change
- `internal/iiko/types.go`: `ItemsGroup` gets `IsHidden`.
- `internal/iiko/sync.go`: `syncCategories` writes `is_hidden`.
- `internal/catalog/types.go`: `Category` gets `IsHidden`.
- `internal/catalog/store.go`: `categoryColumns` selects `is_hidden`; `ListCategories` skips hidden rows.
- `internal/testutil/builders.go`: `Category` gets `IsHidden`, and `InsertCategory` writes it.
- `internal/iiko/sync_test.go`, `internal/iiko/client_test.go`, `internal/catalog/category_handler_test.go`: the tests below.
- `docs/API.md`: notes of `GET /categories` and `POST /iiko/update-products`.

## Spec

iiko sends `isHidden` on some items groups only, for example:
```json
{ "id": "b615621a-535a-426a-b9b5-45a55b30051f", "name": "...", "description": "30 мин.", "isHidden": true, "items": [] }
```
A missing field means "not hidden".

Migration (goose, like `00002_menus.sql`):
```sql
-- +goose Up
ALTER TABLE categories ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE categories DROP COLUMN is_hidden;
```

Go:
```go
// internal/iiko/types.go
type ItemsGroup struct {
    ID       string     `json:"id"`
    Name     string     `json:"name"`
    IsHidden bool       `json:"isHidden"`
    Items    []MenuItem `json:"items"`
}

// internal/catalog/types.go: new field in Category
IsHidden bool `db:"is_hidden"`

// internal/testutil/builders.go: new field in Category
IsHidden bool
```

Sync: the category upsert in `syncCategories` becomes
```sql
INSERT INTO categories (iiko_id, menu_id, name, description, sort_order, is_deleted, sale_start, sale_end, sale_days, is_hidden)
VALUES ($1, $2, $3, '', $4, false, $5, $6, $7, $8)
ON CONFLICT (iiko_id) DO UPDATE SET
    menu_id = EXCLUDED.menu_id, name = EXCLUDED.name, description = '', sort_order = EXCLUDED.sort_order,
    is_deleted = false, sale_start = EXCLUDED.sale_start, sale_end = EXCLUDED.sale_end,
    sale_days = EXCLUDED.sale_days, is_hidden = EXCLUDED.is_hidden, updated_at = now()
RETURNING id
```
with `$8 = g.IsHidden`. Hidden groups are still synced, with their products; they are not soft-deleted. A group that is no longer hidden in iiko gets `is_hidden = false` on the next sync. `Result` counts hidden categories too.

Catalog:
- `categoryColumns` = `id, name, description, sale_start, sale_end, sale_days, is_hidden`.
- `ListCategories`: `WHERE NOT is_deleted AND NOT is_hidden ORDER BY sort_order, id`.
- `CategoryByID`, `ProductsByCategory`, product search and order creation are **unchanged**: a hidden category is still returned by `GET /categories/{id}` and its products are still listed and orderable.
- `CategoryView` (the JSON) is unchanged: there is no `isHidden` field in responses.

## Tests to write
Use `testutil.NewDB`, the existing `fakeSource` in `sync_test.go`, and `testutil.DoJSON` for handlers.

- `TestSyncHiddenCategories` (`internal/iiko/sync_test.go`): a menu with group A `IsHidden: true` and group B without it → `is_hidden` is `true` for A and `false` for B, read back from `categories`; A's products are synced. A second sync where A has `IsHidden: false` → A has `is_hidden = false`.
- `TestMenu` (`internal/iiko/client_test.go`): add `"isHidden": true` to one group of the served JSON and check that its `ItemsGroup.IsHidden` is `true` and that a group without the field has `false`.
- `TestListCategoriesHidden` (`internal/catalog/category_handler_test.go`): insert a visible and a hidden category with `testutil.InsertCategory` → `GET /categories` is 200 and lists only the visible one.
- `TestGetCategoryHidden` (same file): `GET /categories/{id}` of a hidden category → 200 with its id.
- Existing tests keep passing (`internal/store/migrate_test.go` included).

## API documentation
- `GET /categories`: the first line becomes "Returns every category that is not deleted and not hidden in iiko, ordered by sort order and then by `id`." Add to the notes: "A category hidden in iiko is not listed, but `GET /categories/{id}` and its products still work."
- `POST /iiko/update-products`: add to the notes: "The `isHidden` flag of a menu group is stored; hidden groups are synced with their products but are not listed by `GET /categories`."
- `internal/apidocs/openapi.yaml`: no change, the JSON shapes are the same.

## Done when
The commands in CONVENTIONS "Done means" pass, and coverage of `internal/iiko` and `internal/catalog` is at least 80% each.

## Out of scope
Hiding categories from `GET /categories/{id}`, products, search or orders; exposing `isHidden` in JSON; storing the group `description` from iiko; changing `docs/design/schema.sql`.
