# Task 17: Catalog store

## Goal
Read access to categories, products and their images, plus the JSON view types. No HTTP in this task.

## Depends on
08, 09.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/salewindow/salewindow.go`, `internal/testutil/builders.go`.

## Files to create
- `internal/catalog/types.go`
- `internal/catalog/store.go`
- `internal/catalog/store_test.go`

## Tables (columns used here)
- `categories(id, sort_order, name, description, is_deleted, sale_start, sale_end, sale_days, search tsvector)`
- `products(id, category_id, sort_order, name, description, measure_unit, weight, proteins, fats, carbohydrates, calories, flags, price_kopecks, is_deleted, sale_start, sale_end, sale_days, search tsvector)`
- `category_images(id, category_id, url, type)`, `product_images(id, product_id, url, type)`
- `tags(id, tag)`, `product_tags(product_id, tag_id)`

`sale_start` / `sale_end` are nullable integers (minutes from midnight). `sale_days` is `''` or 7 chars of `0`/`1`.

## Spec

### `types.go`
```go
package catalog

type Image struct {
    ID       int64  `db:"id" json:"id"`
    URL      string `db:"url" json:"url"`
    Type     string `db:"type" json:"type"`
    ParentID int64  `db:"parent_id" json:"-"`
}

type Category struct {
    ID          int64  `db:"id"`
    Name        string `db:"name"`
    Description string `db:"description"`
    SaleStart   *int   `db:"sale_start"`
    SaleEnd     *int   `db:"sale_end"`
    SaleDays    string `db:"sale_days"`
    Images      []Image `db:"-"`
}

type Product struct {
    ID            int64         `db:"id"`
    CategoryID    int64         `db:"category_id"`
    Name          string        `db:"name"`
    Description   string        `db:"description"`
    MeasureUnit   string        `db:"measure_unit"`
    Weight        float64       `db:"weight"`
    Proteins      float64       `db:"proteins"`
    Fats          float64       `db:"fats"`
    Carbohydrates float64       `db:"carbohydrates"`
    Calories      float64       `db:"calories"`
    Flags         int           `db:"flags"`
    Price         money.Kopecks `db:"price_kopecks"`
    SaleStart     *int          `db:"sale_start"`
    SaleEnd       *int          `db:"sale_end"`
    SaleDays      string        `db:"sale_days"`
    CategoryName      string `db:"category_name"`
    CategorySaleStart *int   `db:"category_sale_start"`
    CategorySaleEnd   *int   `db:"category_sale_end"`
    CategorySaleDays  string `db:"category_sale_days"`
    Images        []Image `db:"-"`
}

func (c Category) Window() salewindow.Window
func (p Product) Window() salewindow.Window
func (p Product) CategoryWindow() salewindow.Window

// View builds the JSON shape. now must already be in the restaurant's time zone.
func (c Category) View(now time.Time) CategoryView
func (p Product) View(now time.Time) ProductView
```
`CategoryView` JSON fields: `id, name, description, images, isAvailableForOrder, salesDays, startTime, endTime`.
`ProductView` JSON fields: `id, categoryId, name, description, measureUnit, weight, proteins, fats, carbohydrates, calories, flags, price, images, isAvailableForOrder, salesDays, startTime, endTime`.

- `salesDays`, `startTime`, `endTime` come from the item's own window: `DaysString()`, `StartString()`, `EndString()`.
- Category `isAvailableForOrder` = its window `OpenOnDay(now) && OpenAtTime(now)`.
- Product `isAvailableForOrder` = product window open **and** category window open.
- `images` is never nil in a view (use `[]Image{}`).

### `store.go`
```go
type Store struct{ db store.DBTX }
func NewStore(db store.DBTX) *Store

// Not deleted, ordered by sort_order, id. Images loaded.
func (s *Store) ListCategories(ctx context.Context) ([]Category, error)

// store.ErrNotFound when missing or deleted. Images loaded.
func (s *Store) CategoryByID(ctx context.Context, id int64) (Category, error)

// Products of a category; product and category not deleted; ordered by sort_order, id. Images loaded.
func (s *Store) ProductsByCategory(ctx context.Context, categoryID int64) ([]Product, error)

// Search; see below. Images loaded.
func (s *Store) SearchProducts(ctx context.Context, words []string, limit, offset int) ([]Product, error)

// Products by id; product and category not deleted; any order. Images not loaded.
func (s *Store) ProductsByIDs(ctx context.Context, ids []int64) ([]Product, error)
```
All list methods return a non-nil slice.

Product queries share this `SELECT`:
```sql
SELECT p.id, p.category_id, p.name, p.description, p.measure_unit, p.weight, p.proteins, p.fats,
       p.carbohydrates, p.calories, p.flags, p.price_kopecks, p.sale_start, p.sale_end, p.sale_days,
       c.name AS category_name, c.sale_start AS category_sale_start,
       c.sale_end AS category_sale_end, c.sale_days AS category_sale_days
FROM products p JOIN categories c ON c.id = p.category_id
WHERE NOT p.is_deleted AND NOT c.is_deleted
```

`SearchProducts`: when `words` is empty, return all products (base query) with `ORDER BY p.sort_order, p.id LIMIT $1 OFFSET $2`. Otherwise add this condition, passing `words` as `$1` (a non-nil `[]string`):
```sql
AND (
     p.search @@ ANY (ARRAY(SELECT plainto_tsquery('russian', w) FROM unnest($1::text[]) AS w))
  OR c.search @@ ANY (ARRAY(SELECT plainto_tsquery('russian', w) FROM unnest($1::text[]) AS w))
  OR EXISTS (SELECT 1 FROM product_tags pt JOIN tags t ON t.id = pt.tag_id
             WHERE pt.product_id = p.id AND t.tag = ANY($1::text[]))
)
ORDER BY p.sort_order, p.id LIMIT $2 OFFSET $3
```
A product matches when **any** word matches its name, its category name, or one of its tags.

Images: after loading the rows, collect the ids and run one query per table, then attach by `ParentID`:
```sql
SELECT id, url, type, product_id AS parent_id FROM product_images WHERE product_id = ANY($1) ORDER BY id
```
(the same for `category_images` with `category_id`). Skip the query when there are no ids.

## Tests to write
`db := testutil.NewDB(t)`. Insert categories and products with `testutil.InsertCategory` / `InsertProduct`; insert images and tags with plain SQL.

- `TestListCategories`: three categories with `SortOrder` 2, 1, 0 plus one deleted category → the three non-deleted ones in sort order; the one with an image has it attached, the others have no images.
- `TestCategoryByID`: found; deleted → `store.ErrNotFound`; missing → `store.ErrNotFound`.
- `TestProductsByCategory`: ordered by `sort_order`; deleted product absent; `CategoryName` filled; products of a deleted category → empty slice.
- `TestSearchByName`: products `Хачапури по-аджарски`, `Шашлык из баранины`, `Салат`; words `["шашлыки"]` → only the shashlik (stemming works); words `["хачапури", "салат"]` → two products.
- `TestSearchByCategoryName`: category `Грузинская кухня`; words `["кухня"]` → all its products.
- `TestSearchByTag`: product with tag `острое` (rows in `tags` and `product_tags`); words `["острое"]` → that product.
- `TestSearchEmptyWords`: `[]string{}` → all non-deleted products; `limit 2, offset 1` pages correctly.
- `TestSearchExcludesDeleted`.
- `TestProductsByIDs`: returns the requested non-deleted products with the category window fields; an empty id list returns an empty slice.
- `TestCategoryView`: window 9:00–21:00 with days `1111100`; at Friday 2024-05-24 15:00 → `isAvailableForOrder` true, `salesDays` `Пн, Вт, Ср, Чт, Пт`, `startTime` `9:00`, `endTime` `21:00`; at Saturday 15:00 → false; at Friday 05:00 → false. Marshal to JSON and check the key names.
- `TestProductView`: product without a window in a category that is closed → `isAvailableForOrder` false and `salesDays` `""`; `price` 599 kopecks marshals as `5.99`; `images` marshals as `[]`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
HTTP handlers (tasks 18, 19). Writing to these tables (tasks 21, 22).
