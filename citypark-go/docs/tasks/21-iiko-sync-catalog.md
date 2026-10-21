# Task 21: iiko sync — categories and products

## Goal
Save the iiko menu into `categories` and `products`: insert new rows, update existing ones, and soft-delete what is no longer in iiko.

## Depends on
08, 09, 20.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/types.go`, `internal/salewindow/salewindow.go`, `internal/store/store.go`.

## Files to create
- `internal/iiko/sync.go`
- `internal/iiko/sync_test.go`

Do not open `internal/iiko/testdata/*.json`.

## Spec

```go
package iiko

// Source provides the menu. *Client implements it.
type Source interface {
    Fetch(ctx context.Context) (*Nomenclature, error)
}

type Result struct {
    TotalCategories int `json:"totalCategories"`
    TotalProducts   int `json:"totalProducts"`
}

type Syncer struct { /* db *sqlx.DB, src Source, marker string */ }

// marker is the prefix that marks an iiko group as a site category, e.g. "#".
func NewSyncer(db *sqlx.DB, src Source, marker string) *Syncer

// Sync downloads the menu and applies it in one transaction.
func (s *Syncer) Sync(ctx context.Context) (Result, error)
```

`Sync`:
1. `src.Fetch(ctx)` — before the transaction starts.
2. `store.InTx`: call `syncCategories`, then `syncProducts`. (Task 22 adds images and tags here.)
3. Return `Result{TotalCategories: len(categoryIDs), TotalProducts: len(productIDs)}`.

### `syncCategories(ctx, tx, groups) (map[string]int64, error)`
Returns a map from iiko group id to `categories.id`.

1. Keep only groups whose `Name` starts with `marker`. Other groups are ignored completely.
2. For each kept group run:
```sql
INSERT INTO categories (iiko_id, name, description, sort_order, is_deleted, sale_start, sale_end, sale_days)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (iiko_id) DO UPDATE SET
    name = EXCLUDED.name, description = EXCLUDED.description, sort_order = EXCLUDED.sort_order,
    is_deleted = EXCLUDED.is_deleted, sale_start = EXCLUDED.sale_start, sale_end = EXCLUDED.sale_end,
    sale_days = EXCLUDED.sale_days, updated_at = now()
RETURNING id
```
   - `name`: the group name with the marker prefix removed, then `strings.TrimSpace`.
   - `description`: `""` when nil.
   - `sale_*`: `w := salewindow.Parse(additionalInfo)` (`""` when nil) → `w.Start`, `w.End`, `w.Days`.
3. Soft-delete every iiko category that was not in this menu. `ids` is the list of kept iiko ids and must be a non-nil slice:
```sql
UPDATE categories SET is_deleted = true, updated_at = now()
WHERE iiko_id IS NOT NULL AND NOT is_deleted AND NOT (iiko_id = ANY($1))
```

### `syncProducts(ctx, tx, products, categoryIDs) (map[string]int64, error)`
Returns a map from iiko product id to `products.id`.

1. Keep only products whose `ParentGroup` is not nil and is a key of `categoryIDs`.
2. Skip a product with an empty `SizePrices` (it has no price).
3. For each kept product upsert on `iiko_id` with the same pattern as above. Column values:

| Column | Value |
|---|---|
| `category_id` | `categoryIDs[*ParentGroup]` |
| `name`, `measure_unit`, `weight`, `sort_order` (`Order`), `is_deleted` | from the product |
| `description` | `""` when nil |
| `fats`, `proteins`, `carbohydrates`, `calories` | `FatAmount`, `ProteinsAmount`, `CarbohydratesAmount`, `EnergyAmount` |
| `price_kopecks` | `money.FromRubles(SizePrices[0].Price.CurrentPrice)` |
| `sale_start`, `sale_end`, `sale_days` | from `salewindow.Parse(additionalInfo)` |

   `flags` is not written (it is not an iiko field; it keeps its value on update).
4. Soft-delete every iiko product that was not kept (same SQL as for categories, on `products`).

A row without `iiko_id` (created by hand) is never touched by the sync.

## Tests to write
`db := testutil.NewDB(t)`. Use a fake source:
```go
type fakeSource struct{ n *Nomenclature; err error }
func (f *fakeSource) Fetch(context.Context) (*Nomenclature, error) { return f.n, f.err }
```
Build small `Nomenclature` values in the test (2–3 groups, a few products).

- `TestSyncCreates`: groups `#Мангал`, `#Пицца`, `Служебная`; products in each → 2 categories (names `Мангал`, `Пицца` without the marker), products of the unmarked group are not saved; `Result` has the right totals; a product price `320` is stored as `32000`; nil `description` is stored as `''`.
- `TestSyncUpdates`: sync twice, the second time with a changed name and price → the same row ids, new values, no extra rows.
- `TestSyncSoftDeletesMissing`: second menu lacks one category and its products → those rows have `is_deleted = true`; the others stay `false`.
- `TestSyncRestores`: a third sync that contains the category again → `is_deleted = false`.
- `TestSyncIsDeletedFlag`: a product with `IsDeleted: true` is stored with `is_deleted = true`.
- `TestSyncSaleWindow`: group `additionalInfo` `["9:00", "21:00", 1111100]` → `sale_start = 540`, `sale_end = 1260`, `sale_days = '1111100'`; nil → `NULL, NULL, ''`.
- `TestSyncKeepsManualRows`: a category and product inserted with `testutil` (no `iiko_id`) are unchanged after a sync.
- `TestSyncSkipsProductWithoutPrice`: empty `SizePrices` → not saved, no error.
- `TestSyncFetchError`: the source returns an error → `Sync` returns it and the database is unchanged.
- `TestSyncCustomMarker`: marker `"menu:"` → group `menu: Супы` becomes category `Супы`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Images, tags and the HTTP endpoint (task 22).
