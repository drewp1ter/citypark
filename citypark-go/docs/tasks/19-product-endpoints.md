# Task 19: Product endpoints

## Goal
`GET /categories/{id}/products` and `GET /products` (search).

## Depends on
17. Task 18 must be done too (it creates `handler.go`).

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/catalog/handler.go`, `internal/catalog/store.go`, `internal/catalog/types.go`.

## Files to change
- `internal/catalog/handler.go`: add two routes to `Routes` and two handler methods.

## Files to create
- `internal/catalog/product_handler_test.go`

## Spec

| Route | Behaviour |
|---|---|
| `GET /categories/{id}/products` | `httpx.PathID(r, "id")`; `ProductsByCategory`; 200 with `[]ProductView`. An unknown or deleted category gives `[]`. |
| `GET /products?query=&page=&pageSize=` | search, see below; 200 with `[]ProductView` |

Both routes are public. Views are built with `View(h.clock.Now())`. An empty result is `[]`.

### Search
1. `limit, offset, err := httpx.Pagination(r, h.defaultPageSize)`.
2. Words: `strings.Fields(strings.ToLower(r.URL.Query().Get("query")))`. Keep at most the first 20 words. If there are none, pass `[]string{}` (not nil).
3. `SearchProducts(ctx, words, limit, offset)`.

Notes:
- A missing or empty `query` returns all products, paged.
- In a URL, `+` and `%20` both decode to a space, so `?query=хачапури+салат` gives two words.
- A product matches when any word matches its name, its category name or a tag.

Response item example:
```json
{
  "id": 1, "categoryId": 1, "name": "Шашлык из баранины", "description": "", "measureUnit": "порц",
  "weight": 0.23, "proteins": 0, "fats": 0, "carbohydrates": 0, "calories": 0, "flags": 0, "price": 320,
  "images": [], "isAvailableForOrder": true, "salesDays": "", "startTime": "", "endTime": ""
}
```

## Tests to write
`db := testutil.NewDB(t)`; clock fixed at Friday 2024-05-24 15:00 UTC; `defaultPageSize` 100.

- `TestProductsByCategory`: category with three products (one deleted) → 200, two items ordered by `sort_order`; each has `categoryId` of the category.
- `TestProductsByCategoryUnknown`: `/categories/999/products` → 200 and `[]`.
- `TestProductsByCategoryBadID`: `/categories/abc/products` → 404.
- `TestProductKeys`: one item has exactly the 17 keys listed for `ProductView` in task 17: `id, categoryId, name, description, measureUnit, weight, proteins, fats, carbohydrates, calories, flags, price, images, isAvailableForOrder, salesDays, startTime, endTime`.
- `TestSearchByNames`: products `Хачапури`, `Шашлык`, `Салат`; `/products?query=хачапури+салат` → exactly those two.
- `TestSearchUpperCase`: `/products?query=ШАШЛЫК` finds the product.
- `TestSearchNoQuery`: `/products` → all three.
- `TestSearchPaging`: `/products?page=2&pageSize=2` → one item; `/products?pageSize=0` → 400.
- `TestSearchNoMatch`: `/products?query=zzz` → `[]`.
- `TestProductAvailability`: a product in a category with days `0000011` → `isAvailableForOrder: false` (the clock is a Friday).

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Changing `store.go` SQL. If a query from task 17 is wrong, fix it there and say so in the commit message.
