# Task 38: iiko sync from menu/v3

## Goal
The menu sync reads the external menu (menu/v3) instead of the nomenclature. Categories come from `itemsGroups` and belong to the synced menu. The old nomenclature code, the category marker and the recorded fixtures are removed.

## Depends on
37.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `internal/iiko/sync_extra.go`, `internal/iiko/types.go`.

Never open `internal/iiko/testdata/*.json`. They are deleted in this task with `git rm` without being read.

## Files to change
- `internal/iiko/sync.go`: the new `Source`, `NewSyncer` and sync steps below.
- `internal/iiko/sync_extra.go`: images and tags are only cleared now.
- `internal/iiko/types.go`, `internal/iiko/client.go`: remove `Nomenclature`, `Group`, `Product`, `SizePrice`, the `Nomenclature` method and the old `Fetch`. Rename `FetchMenu` to `Fetch`.
- `internal/iiko/sync_test.go`, `internal/iiko/client_test.go`: rewrite for the new types.
- `internal/app/app.go`: `iiko.NewSyncer(db, iikoClient)`.
- `docs/API.md`: notes of `POST /iiko/update-products`.

## Files to delete
- `internal/iiko/fixtures_test.go`
- `internal/iiko/testdata/` (all files, with `git rm -r`)

## Spec

```go
// Source provides the menu. *Client implements it.
type Source interface {
    Fetch(ctx context.Context) (*Menu, error)
}

func NewSyncer(db *sqlx.DB, src Source) *Syncer
```
`Result` and the handler are unchanged.

`Sync` fetches the menu, then runs one transaction (`store.InTx`):

1. **Menu.**
```sql
INSERT INTO menus (iiko_id, name) VALUES ($1, $2)
ON CONFLICT (iiko_id) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING id
```
2. **Categories.** One per `ItemsGroups[i]`, in order. Each group becomes a category, even when none of its items is for sale. The sale window is `salewindow.Window{}`, so it is stored as `NULL, NULL, ''`:
```sql
INSERT INTO categories (iiko_id, menu_id, name, description, sort_order, is_deleted, sale_start, sale_end, sale_days)
VALUES ($1, $2, $3, '', $4, false, $5, $6, $7)
ON CONFLICT (iiko_id) DO UPDATE SET
    menu_id = EXCLUDED.menu_id, name = EXCLUDED.name, description = '', sort_order = EXCLUDED.sort_order,
    is_deleted = false, sale_start = EXCLUDED.sale_start, sale_end = EXCLUDED.sale_end,
    sale_days = EXCLUDED.sale_days, updated_at = now()
RETURNING id
```
   with `$1 = group.ID`, `$3 = strings.TrimSpace(group.Name)`, `$4 = i`.
3. **Products.** For each group and each `Items[j]`:
   - choose the size: the first one with `IsDefault`, or else the first one. Skip the item if it has no sizes or the chosen size has `Price == nil`;
   - `measure_unit`: from the `Products` entry whose `ID == item.ProductID`, or `''` if there is none (build a map once);
   - `weight = size.Weight / 1000`, because iiko sends grams and the column holds kilograms;
   - `price_kopecks = money.FromRubles(*size.Price)`, `sort_order = j`;
   - `description = ''`, `fats`, `proteins`, `carbohydrates` and `calories` are `0`, `is_deleted = false`, sale window `salewindow.Window{}`.
   Use the existing product upsert with these values. If the same `ProductID` appears twice, the later row wins.
4. **Soft delete.** `softDeleteMissing` for `categories` and `products` with the kept iiko ids, as today.
5. **Clear images.** Call `replaceImages` for `category_images` and for `product_images` with an empty set, which deletes every `type = 'from_iiko'` image. Images of another type stay.
6. **Clear tags.** `DELETE FROM product_tags`, then delete the tags that no product uses (the existing `unused` statement).

Remove the code that is no longer reached: `imageSet.add`, `splitTag`, `tagSeparator`, `tagNoise`, `uploadErrorLink`, the tag upsert loop and `deref` if unused. Keep `replaceImages` and `softDeleteMissing`.

## Tests to write
Use `testutil.NewDB` and a `fakeSource` that returns a `*Menu` built in Go. Check the rows by reading them back.

- `TestPickSize` (table test for the size-choosing helper): no sizes → skip; one default size without a price → skip; a non-default size first and a default size second → the default one; no default → the first one.
- `TestSyncCreatesMenu`: one sync → one `menus` row with `iiko_id` and `name`. A second sync with a new name → still one row, name updated.
- `TestSyncCategories`: two groups → two categories with `menu_id` = the menu's id, `sort_order` 0 and 1, `description = ''`, `sale_start IS NULL`.
- `TestSyncProducts`: a priced item → a product with `price_kopecks = 60000`, `weight = 1.0` (from `1000`), `measure_unit = 'кг'`, `sort_order` = its index, linked to its category. A priceless item → no product row. `Result` is `{TotalCategories: 2, TotalProducts: 1}`.
- `TestSyncClearsOldData`: before the sync, insert an iiko category with a description and a sale window, an iiko product with nutrition values, a `from_iiko` and a `manual` image, and a tag linked to a product → after the sync, the description, sale window and nutrition are cleared, the `from_iiko` image is gone, the `manual` image stays, and `product_tags` and `tags` are empty.
- `TestSyncSoftDeletesMissing`: sync a menu, then a menu without one group and one item → the missing category and product have `is_deleted = true`. A category without `iiko_id` is untouched.
- `TestSyncFetchError`: the source returns an error → `Sync` returns it and nothing is written.
- `TestFetch` in `client_test.go`: the renamed `FetchMenu` tests still pass under the name `Fetch`.
- `handler_test.go` passes with the new fake source.

## API documentation
`POST /iiko/update-products`: update the notes. The sync now reads the external menu `IIKO_EXTERNAL_MENU_ID`. Every menu group becomes a category. Items without a price are skipped. The sync no longer takes images, tags, descriptions, nutrition or sale windows from iiko, and it clears the ones that came from iiko before. The response shape is unchanged.

## Done when
The commands in CONVENTIONS "Done means" pass, and coverage of `internal/iiko` is at least 80%. `grep -rn "Nomenclature\|marker" internal/iiko internal/app` prints nothing. `internal/iiko/testdata` no longer exists.

## Out of scope
Exposing menus in the public API, and syncing more than one menu.
