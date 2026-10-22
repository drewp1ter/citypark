# Task 53: Hotel reports from 1C

## Goal
Two admin endpoints return the hotel reports from the 1C database `CityParkHotel`: revenue by room for a period, and
revenue per day for a chart. The server holds the 1C credentials; the admin panel never talks to 1C.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/telegram/client.go` (HTTP client pattern with a base URL and a fake server
in tests), `internal/config/config.go`, `internal/order/handler.go` (admin group).

## Background
The old admin (`citypark_ai`) called the 1C HTTP services from the browser with a Basic auth header compiled into the
JS bundle. ADR 0001 moves that call behind `citypark-go`. Only the hotel still uses 1C; restaurant reports are out of
scope. What the old admin sent and read (the only knowledge we have of the 1C services):

| Report | Request | Rows it read |
|---|---|---|
| Сдача номеров | `GET {base}/hs/reports/v1/room_sales?StartPeriod=YYYYMMDDHHmmss&EndPeriod=YYYYMMDDHHmmss` | `[{"Organization": "...", "Room": "...", "Count": 3, "Sum": 12000.5}]` |
| Выручка по дням | `GET {base}/hs/reports/v1/month_sales?StartPeriod=...&EndPeriod=...` | `[{"Date": "21.10.2026", "Sum": 8000}]` (several rows per date possible) |

`base` was `/onec/CityParkHotel/ru_RU` behind a proxy; here it is the full URL in `ONEC_HOTEL_URL`, for example
`http://1c.local/CityParkHotel/ru_RU`. Auth is HTTP Basic. Ignore unknown fields in the rows.

## Files to create
- `internal/onec/client.go`, `internal/onec/handler.go`, `internal/onec/onec_test.go`

## Files to change
- `internal/config/config.go` + test: `OneC struct{ HotelURL, User, Password string }` from `ONEC_HOTEL_URL`,
  `ONEC_USER`, `ONEC_PASSWORD` (defaults `""`).
- `.env.example`: the three variables, comment `# 1C HTTP services of the hotel database; empty URL disables /reports`.
- `internal/app/app.go`: `onec.NewHandler(onec.NewClient(cfg.OneC.HotelURL, cfg.OneC.User, cfg.OneC.Password, nil), tokens, c).Routes(r)`.
- `docs/API.md` (new area `### Reports`), `internal/apidocs/openapi.yaml`.

## Spec

### Client
```go
type RoomSale struct {
	Organization string  `json:"Organization"`
	Room         string  `json:"Room"`
	Count        float64 `json:"Count"`
	Sum          float64 `json:"Sum"`
}
type DaySale struct {
	Date string  `json:"Date"` // "21.10.2026"
	Sum  float64 `json:"Sum"`
}

// NewClient: httpClient nil → &http.Client{Timeout: 30 * time.Second}. base "" → every call returns ErrDisabled.
func NewClient(base, user, password string, httpClient *http.Client) *Client
func (c *Client) RoomSales(ctx context.Context, from, to time.Time) ([]RoomSale, error)
func (c *Client) DaySales(ctx context.Context, from, to time.Time) ([]DaySale, error)

var ErrDisabled = errors.New("1c is not configured")
```
`StartPeriod` = `from` formatted `20060102150405`; `EndPeriod` = `to` (the handler passes the end of the last day).
A non-200 status, invalid JSON or a transport error → an error that includes the status/cause (never the password).
`null` body → empty slice.

### Routes
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| GET | `/reports/hotel/sales` | admin | revenue by organization and room | 200 |
| GET | `/reports/hotel/daily` | admin | revenue per day, every day of the period | 200 |

Parameters (both): `from`, `to`: required `YYYY-MM-DD`, local dates in the clock's location; `to` ≥ `from`; at most
366 days. Errors: 400 `Некорректные параметры отчёта` with fields (`дата в формате ГГГГ-ММ-ДД`,
`не раньше даты from`, `период не больше 366 дней`). The client gets `from 00:00:00` and `to 23:59:59`.

`/reports/hotel/sales` response (money as `money.Kopecks` JSON, rubles; convert with `math.Round(sum * 100)`):
```json
{"rows": [{"organization": "ООО «Олимп»", "room": "VIP-номер", "count": 3, "sum": 24000}],
 "total": {"count": 3, "sum": 24000}}
```
Rows sorted by organization, then room. `count` is a JSON number (keep `float64`).

`/reports/hotel/daily` response: one entry per calendar day from `from` to `to`, sums of all 1C rows with that date,
`0` for days without rows; 1C dates outside the period are ignored; an unparsable date → 502 like other bad data.
```json
{"days": [{"date": "2026-10-20", "sum": 8000}, {"date": "2026-10-21", "sum": 0}], "total": 8000}
```

Errors from the client: `ErrDisabled` → 404 `Не найдено`; any other → 502 `Сервис отчётов недоступен`, logged with
`slog.Error` and the cause.

## Tests to write
Client, with `httptest.NewServer`:
- `TestRoomSalesRequest`: the fake checks path `/hs/reports/v1/room_sales`, `StartPeriod=20261001000000`,
  `EndPeriod=20261031235959` and the Basic auth header; returns two rows → decoded.
- `TestDaySalesNull`: body `null` → empty slice, no error.
- `TestClientErrors`: 500 → error; `{` → error; the error text does not contain the password.
- `TestClientDisabled`: base `""` → `ErrDisabled`, no request.

Handler (admin token; fake 1C server):
- `TestHotelSales`: rows in random order → sorted, total summed; `12000.5` → `12000.5` in JSON.
- `TestHotelDaily`: period 2026-10-20…2026-10-22, 1C rows `20.10.2026` 5000 and 3000, `22.10.2026` 100,
  `19.10.2026` 1 → three days `8000, 0, 100`, total 8100.
- `TestReportsInvalid`: table → 400 with the field: no `from`; `to` before `from`; 367 days; `from=1.10.2026`.
- `TestReportsUpstreamDown`: fake returns 500 → 502 `Сервис отчётов недоступен`.
- `TestReportsDisabled`: client with base `""` → 404.
- `TestReportsAuth`: no token → 401; role `user` → 403.

## API documentation
`docs/API.md`, `### Reports`: both routes with examples; notes "the data comes from 1C (hotel database); 502 means 1C
did not answer within 30 s or returned an error", "dates are local (`APP_TIMEZONE`), both inclusive".

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/onec` and `internal/config` is at least 80%, docs
updated. Then ask the owner to run, with the real `ONEC_*` values on a server that can reach 1C:
```
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" 'localhost:3000/api/v1/reports/hotel/sales?from=2026-10-01&to=2026-10-21'
```
and paste the output into `PROGRESS.md` under the task line. If the real field names differ from the table above,
note that there and do not tick the task.

## Out of scope
Restaurant reports (iiko), caching, other 1C services, CSV export.
