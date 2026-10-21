# Task 40: Product descriptions from iiko

## Goal
The sync stores the iiko menu item `description` (trimmed) in `products.description`, so `GET /products` and
`GET /products/{id}` return it. Category descriptions stay `''`.

## Depends on
39.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/types.go`, `internal/iiko/sync.go`, `internal/iiko/sync_test.go`.

Never open `internal/iiko/testdata/`.

## Files to create
None.

## Files to change
- `internal/iiko/types.go`: `MenuItem` gets `Description`.
- `internal/iiko/sync.go`: `syncProducts` writes the trimmed description instead of `""`.
- `internal/iiko/client_test.go`: `menuJSON` and `TestMenu` (see Tests).
- `internal/iiko/sync_test.go`: the tests below.
- `docs/API.md`: notes of `POST /iiko/update-products`.
- `internal/apidocs/openapi.yaml`: only if it says that the product description is always empty.

## Spec

Today task 38 stores `''` as the product description, and `MenuItem` does not decode the field. iiko menu/v3 sends it
on each item, sometimes as `null`:
```json
{ "productId": "p-2", "name": "Хинкали", "description": "Говядина, свинина, лук", "menuItemId": "m-2", "sizePrices": [ … ] }
```

Go:
```go
// internal/iiko/types.go
type MenuItem struct {
    ProductID   string          `json:"productId"`
    Name        string          `json:"name"`
    Description string          `json:"description"`
    SizePrices  []ItemSizePrice `json:"sizePrices"`
}
```
A JSON `null` leaves a Go string `""`, so no pointer is needed.

Sync: in `syncProducts`, the 4th argument of the product upsert (`$4`, `description`) becomes
`strings.TrimSpace(item.Description)`. The SQL stays the same: the upsert already has
`description = EXCLUDED.description`, so a re-sync replaces an old description, and an empty iiko description clears it.

Unchanged: category descriptions are still `''`; the `search` column still indexes only `name`; no migration.

## Tests to write
Use `testutil.NewDB`, `runSync`, `getProduct` and `item()` from `sync_test.go`.

- `TestMenu` (`client_test.go`): add `"description":"  Говядина, лук  "` to item `p-2` and `"description":null` to item
  `p-1` in `menuJSON` → `ItemsGroups[1].Items[0].Description == "  Говядина, лук  "` (the client does not trim),
  `ItemsGroups[0].Items[0].Description == ""`.
- `TestSyncProductDescription` (table test, one sync per row; set `Description` on the `p-shashlik` item of `baseMenu()`):
  - `"Свекла, чеснок"` → stored `"Свекла, чеснок"`;
  - `"  Свекла  \n"` → `"Свекла"`;
  - `""` → `""`.
- `TestSyncProductDescriptionUpdated`: sync with `"Старое"`, then sync again with `"Новое"` → `"Новое"`; a third sync
  with `""` → `""`.
- `TestSyncProducts`: unchanged (the base menu has no descriptions, so `""` is still right).
- `TestSyncClearsOldData`: the product with `'Вкусно'` now gets the item's description (`""` in `baseMenu()`), so the
  check stays, but change its message to say the description is replaced from iiko. The category description is
  still cleared.

## API documentation
`POST /iiko/update-products`, Notes: products now take their description from the iiko menu item (trimmed; empty when
iiko has none). Remove "descriptions" from the list of things the update no longer takes, and say that category
descriptions are still reset. In the `GET /products` example, a non-empty `description` is fine but not required.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/iiko` is at least 80%, and `docs/API.md` is
updated. With real iiko credentials: after `POST /iiko/update-products`, `curl localhost:3000/api/v1/products?pageSize=500`
returns products with a non-empty `description` (report the count; if it is 0, write that in `PROGRESS.md`).

## Out of scope
Category descriptions, adding the description to the `search` tsvector, schema changes, and the website
(`../citypark-modern` task 02 shows the text).
