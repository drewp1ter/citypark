# Task 28: Order read endpoints

## Goal
`GET /orders/my`, `GET /orders/{id}` and `GET /orders`.

## Depends on
11, 25. Task 27 must be done too (it creates `handler.go`).

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/store.go`, `internal/order/handler.go`, `internal/auth/middleware.go`.

## Files to change
- `internal/order/store.go`: add `List` and `ListByCustomer`.
- `internal/order/handler.go`: add three routes.

## Files to create
- `internal/order/read_handler_test.go`

## Spec

### Store
```go
// List returns all orders, newest first (ORDER BY o.created_at DESC, o.id DESC), and the total count.
func (s *Store) List(ctx context.Context, limit, offset int) ([]Order, int, error)

// ListByCustomer returns one customer's orders, newest first (ORDER BY o.id DESC), and the total count.
func (s *Store) ListByCustomer(ctx context.Context, customerID int64, limit, offset int) ([]Order, int, error)
```
- Reuse the order `SELECT` from task 25 with `LIMIT`/`OFFSET`, then `loadItems` for the page.
- The count is a separate `SELECT count(*)` with the same filter and no limit.
- The slice is never nil.

### Routes
| Route | Role | Behaviour |
|---|---|---|
| `GET /orders/my?page=&pageSize=` | user | `ListByCustomer` for the signed-in user |
| `GET /orders/{id}` | user | `ByID`; allowed for the order's customer and for admins; otherwise 404 |
| `GET /orders?page=&pageSize=` | admin | `List` |

- Paging: `httpx.Pagination(r, h.opts.DefaultPageSize)`.
- List responses: `200 {"orders": [Order, ...], "count": 12}` where `count` is the total, not the page size.
- `GET /orders/{id}`: `httpx.PathID(r, "id")`; `store.ErrNotFound` → `httpx.NotFound()`. A normal user asking for someone else's order also gets `httpx.NotFound()` (do not reveal that it exists).
- Build the store with `NewStore(h.svc.db)`.

Register the routes like this so that `/orders/my` is not captured by `/orders/{id}` (chi prefers the static path) and roles are separate:
```go
r.Group(func(r chi.Router) {
    r.Use(h.tokens.RequireRole(auth.RoleUser))
    r.Get("/orders/my", httpx.Handle(h.listMine))
    r.Get("/orders/{id}", httpx.Handle(h.get))
})
r.Group(func(r chi.Router) {
    r.Use(h.tokens.RequireRole(auth.RoleAdmin))
    r.Get("/orders", httpx.Handle(h.list))
})
```

Order JSON:
```json
{
  "id": 1, "createdAt": "2024-05-24T12:00:00Z", "updatedAt": "2024-05-24T12:00:00Z",
  "status": "new", "comment": "", "paymentType": "cash", "total": 599, "deliveryPrice": 150,
  "customer": { "id": 2, "name": "Иван", "phone": "+79991234567" },
  "address": { "id": 1, "city": "Сочи", "address": "ул. Ленина, 1" },
  "items": [{ "productId": 1, "name": "Шашлык", "categoryName": "Мангал", "amount": 1, "offeredPrice": 599 }]
}
```

## Tests to write
`db := testutil.NewDB(t)`. Seed: users `user` and `other` (role user), `admin` (role admin); a category with products; two orders for `user` and one for `other`, inserted with `NewStore(db).Insert`. Tokens from `tokens.Issue`.

- `TestListMine`: 200; `count` is 2; two orders, newest first (higher id first); each has `items` with names and an `address`.
- `TestListMinePaging`: `?pageSize=1&page=2` → one order (the older one), `count` still 2.
- `TestListMineEmpty`: a user without orders → `{"orders":[],"count":0}`.
- `TestListMineUnauthorized`: no token → 401 `{"statusCode":401,"message":"Требуется авторизация"}`.
- `TestGetOwn`: 200 and the body has exactly the keys `id, createdAt, updatedAt, status, comment, paymentType, total, deliveryPrice, customer, address, items`.
- `TestGetOthers`: `user` asks for the order of `other` → 404.
- `TestGetAsAdmin`: admin asks for any order → 200.
- `TestGetMissing`: 404. `TestGetUnauthorized`: 401.
- `TestListAll`: admin → 200, `count` 3, three orders, each with `customer.name`.
- `TestListAllForbidden`: user token → 403 `{"statusCode":403,"message":"Недостаточно прав"}`.
- `TestListAllUnauthorized`: 401.
- `TestListBadPaging`: `?page=0` → 400.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Changing orders (task 29).
