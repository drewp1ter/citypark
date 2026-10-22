# Task 42: Order search filters

## Goal
`GET /orders` (admin) takes optional filters: free text `q`, a date range `from`/`to`, `status` and `paymentType`.
`count` is the number of orders that match. The new admin panel (`citypark-admin`) uses it for the order history.

## Depends on
29.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/order/store.go` (`List`, `selectOrders`), `internal/order/handler.go` (`list`),
`internal/order/types.go`.

## Background
Decided by the owner (`../docs/adr/0001-admin-panel.md`): the admin's order history is read only and the search runs on
the server. Dates are calendar days in the restaurant's time zone (`APP_TIMEZONE`, the location of the handler's
clock), because staff think in local days.

## Files to change
- `internal/order/store.go`: `ListFilter`, `List` takes it (Spec 1).
- `internal/order/handler.go`: parse the query parameters (Spec 2).
- `internal/order/*_test.go`: the tests below (add a new file `search_test.go` if that is clearer).
- `docs/API.md`, `internal/apidocs/openapi.yaml`: the parameters of `GET /orders`.

## Spec

### 1. Store
```go
// ListFilter narrows List. Zero values mean "no condition".
type ListFilter struct {
	Query       string     // trimmed; matched against id, customer phone and customer name
	From        *time.Time // inclusive: o.created_at >= From
	Before      *time.Time // exclusive: o.created_at < Before
	Status      string     // one of the Status* constants
	PaymentType string     // "cash" or "online"
}

// List returns the orders that match f, newest first, and how many match in total.
func (s *Store) List(ctx context.Context, f ListFilter, limit, offset int) ([]Order, int, error)
```
Build one WHERE clause used by both the count and the select. Each condition takes a placeholder; empty filter → no
WHERE. Write the count as `SELECT count(*) FROM orders o JOIN users u ON u.id = o.customer_id <where>`.

`Query` (non-empty after `strings.TrimSpace`):
- digits only (`^\d+$`, at most 18 digits) → `(o.id = $n OR u.phone LIKE $m)` with `$m = '%' + digits + '%'`
  (phones are stored as `+7999…`, so a part of a phone matches);
- otherwise → `(u.name ILIKE $m OR u.phone LIKE $m)` with `$m = '%' + escaped + '%'`, where `escaped` has `\`, `%`
  and `_` escaped with a backslash.

Other conditions: `o.created_at >= $n`, `o.created_at < $n`, `o.status = $n`, `o.payment_type = $n`. Join them with
`AND`. Keep `ORDER BY o.created_at DESC, o.id DESC`.

### 2. Handler
Query parameters of `GET /orders`, besides `page` and `pageSize`:

| Field | Rules | Error (400, field in `errors`) |
|---|---|---|
| `q` | at most 100 characters after trimming | `не более 100 символов` |
| `from` | `YYYY-MM-DD` | `дата в формате ГГГГ-ММ-ДД` |
| `to` | `YYYY-MM-DD`, not before `from` | the same / `не раньше даты from` |
| `status` | `new`, `pending`, `paid`, `canceled`, `confirmed` | `недопустимое значение` |
| `paymentType` | `cash`, `online` | `недопустимое значение` |

The 400 body is `{"statusCode": 400, "message": "Некорректные параметры поиска", "errors": {"<field>": "..."}}`, with
every invalid field: `e := httpx.BadRequest("Некорректные параметры поиска"); e.Fields = fields` (the same way
`validate.Struct` fills `httpx.Error.Fields`).

Dates: parse with `time.ParseInLocation("2006-01-02", v, loc)` where `loc = h.svc.clock.Now().Location()`.
`From = from 00:00`, `Before = to + 1 day 00:00` (so `to` is inclusive). `from` alone or `to` alone are allowed.

An empty parameter (`?status=`) is the same as a missing one.

## Tests to write
Store (`testutil.NewDB`; insert users, addresses and orders directly with SQL or the existing test builders; set
`created_at` explicitly):
- `TestListFilterNone`: 3 orders → all 3, newest first, count 3.
- `TestListFilterQueryID`: `Query: "<id of order 2>"` → order 2 (plus any order whose phone contains those digits;
  choose phones that do not).
- `TestListFilterQueryPhone`: `Query: "1234"` → the orders of the customer `+79991234567`.
- `TestListFilterQueryName`: `Query: "иван"` → orders of «Иван Петров», case-insensitive.
- `TestListFilterQueryEscape`: `Query: "%"` → nothing (no user name contains `%`).
- `TestListFilterDates`: orders on 2026-10-20 23:30, 2026-10-21 10:00, 2026-10-22 00:00 (Moscow) with
  `From = 2026-10-21 00:00 MSK`, `Before = 2026-10-22 00:00 MSK` → only the middle one.
- `TestListFilterStatusAndPayment`: combine `Status: "paid"`, `PaymentType: "online"` → only matching orders; count is
  the filtered count, not the total.
- `TestListFilterPaging`: 5 matching orders, limit 2 offset 2 → 2 orders, count 5.

Handler (router with `Routes`, admin token from `tokens.Issue(id, auth.RoleAdmin)`, `testutil.DoJSON`, a clock fixed
in `Europe/Moscow`):
- `TestListOrdersSearchQuery`: `?q=Иван` (URL-encoded) → 200, only Иван's orders.
- `TestListOrdersSearchDates`: `?from=2026-10-21&to=2026-10-21` → the order of that day only.
- `TestListOrdersSearchInvalid`: table → 400 `Некорректные параметры поиска` with the field: `from=21.10.2026`,
  `to=2026-13-01`, `from=2026-10-22&to=2026-10-21`, `status=done`, `paymentType=card`, `q=` + 101 letters.
- `TestListOrdersSearchEmptyParams`: `?status=&q=` → 200, all orders.
- The existing 401/403 tests for `GET /orders` keep passing.

## API documentation
`docs/API.md`, section `GET /orders`: add the five parameters to the Parameters table, the 400 example, and notes:
"`q` matches the order number, any part of the customer's phone and the customer's name (case-insensitive)";
"`from` and `to` are local dates (`APP_TIMEZONE`), both inclusive"; "`count` counts the matching orders".

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/order` is at least 80%, `docs/API.md` and
`openapi.yaml` are updated.

## Out of scope
Sorting options, search by address or item, CSV export, any change to `GET /orders/my`.
