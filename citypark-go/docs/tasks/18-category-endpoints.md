# Task 18: Category endpoints

## Goal
`GET /categories` and `GET /categories/{id}`.

## Depends on
17.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/catalog/types.go`, `internal/catalog/store.go`.

## Files to create
- `internal/catalog/handler.go`
- `internal/catalog/category_handler_test.go`

## Spec

```go
package catalog

type Handler struct { /* store *Store, clock clock.Clock, defaultPageSize int */ }

// defaultPageSize is used by the product search route added in task 19.
func NewHandler(s *Store, c clock.Clock, defaultPageSize int) *Handler

// Routes registers the public catalog routes.
func (h *Handler) Routes(r chi.Router)
```

| Route | Behaviour |
|---|---|
| `GET /categories` | `ListCategories`; 200 with `[]CategoryView` built by `View(h.clock.Now())` |
| `GET /categories/{id}` | `httpx.PathID(r, "id")`; `CategoryByID`; `store.ErrNotFound` → `httpx.NotFound()`; 200 with `CategoryView` |

Both routes are public. An empty list is the JSON `[]`, not `null`.

Response example:
```json
{
  "id": 1,
  "name": "Мангал",
  "description": "",
  "images": [{ "id": 1, "url": "https://example.com/a.jpg", "type": "from_iiko" }],
  "isAvailableForOrder": true,
  "salesDays": "Пн, Вт, Ср, Чт, Пт",
  "startTime": "9:00",
  "endTime": "21:00"
}
```

## Tests to write
`db := testutil.NewDB(t)`; clock `clock.Fixed(time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC))` (a Friday).

- `TestListCategories`: three categories (`SortOrder` 1, 0, 2) and one deleted → 200, three items in sort order; the deleted one is absent.
- `TestListCategoriesEmpty`: body is `[]`.
- `TestListCategoriesAvailability`: one category with window 9:00–21:00 and days `1111100`, one with days `0000011` → first `isAvailableForOrder: true`, second `false`.
- `TestGetCategory`: 200; the body has exactly the keys `id, name, description, images, isAvailableForOrder, salesDays, startTime, endTime`; the image inserted with plain SQL is present.
- `TestGetCategoryMissing`: `/categories/999` → 404 with body `{"statusCode":404,"message":"Не найдено"}`.
- `TestGetCategoryDeleted`: 404.
- `TestGetCategoryBadID`: `/categories/abc` → 404.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Product routes (task 19). Creating or editing categories through the API (the menu comes from iiko).
