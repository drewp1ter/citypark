# Task 43: Menu sync for admins, one sync at a time

## Goal
An admin can start the iiko menu sync with a JWT (`POST /iiko/sync`), and two syncs never run at the same time: a sync
that starts while another one runs fails at once with 409.

## Depends on
38.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/handler.go`, `internal/iiko/sync.go` (`Syncer`, `Sync`),
`internal/deliveryprice/handler.go` (admin route group pattern).

## Background
The admin panel (`citypark-admin`) gets an «Обновить меню из iiko» button. The existing `POST /iiko/update-products`
is protected by `IIKO_UPDATE_TOKEN`, a server secret a browser must not hold, so the button needs a JWT route.
Task 44 adds a timer that calls `Sync` too, so the lock lives in `Syncer`, not in a handler.

## Files to change
- `internal/iiko/sync.go`: the lock (Spec 1).
- `internal/iiko/handler.go`: the new route and the 409 (Spec 2).
- `internal/iiko/handler_test.go`, `internal/iiko/sync_test.go`: the tests below.
- `internal/app/app.go`: pass `tokens` to `iiko.NewHandler`.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: the new endpoint; 409 on the old one.

## Spec

### 1. Lock in `Syncer`
```go
// ErrSyncRunning is returned by Sync while another sync is in progress.
var ErrSyncRunning = errors.New("iiko sync is already running")

type Syncer struct {
	db  *sqlx.DB
	src Source
	mu  sync.Mutex // held for the whole Sync
}
```
`Sync` starts with `if !s.mu.TryLock() { return Result{}, ErrSyncRunning }` and `defer s.mu.Unlock()`. Nothing else in
`Sync` changes. `NewSyncer` keeps its signature (the zero `sync.Mutex` is ready to use).

### 2. Handler
```go
// NewHandler creates a Handler. token is IIKO_UPDATE_TOKEN ("" disables POST /iiko/update-products).
func NewHandler(s *Syncer, token string, tokens *auth.Tokens) *Handler
```
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| POST | `/iiko/update-products` | `X-Update-Token` | unchanged, plus 409 | 200 |
| POST | `/iiko/sync` | admin | runs `Sync`, returns `Result` | 200 |

Both routes map `errors.Is(err, ErrSyncRunning)` to
`httpx.NewError(http.StatusConflict, "Обновление меню уже выполняется")`. Other errors stay 500 as now. `POST
/iiko/sync` has no body. Log one `slog` info line `menu sync done` with the two totals and `by` = the admin's user id
(`auth.FromContext`).

Response (both routes): `{"totalCategories": 12, "totalProducts": 184}`.

## Tests to write
Use the fake `Source` the existing sync tests use. For the lock, write a fake whose `Fetch` blocks on a channel:
- `TestSyncRunningReturnsErr`: start `Sync` in a goroutine with the blocking fake, wait until `Fetch` was entered
  (a second channel), call `Sync` again → `ErrSyncRunning`; release the first → it returns no error.
- `TestSyncUnlocksAfterError`: fake `Fetch` returns an error → `Sync` errors; a second `Sync` with a working fake →
  no error (the lock was released).
- `TestAdminSync`: admin token → 200 and the totals; the categories exist in the database afterwards.
- `TestAdminSyncNoToken` → 401 `Требуется авторизация`; `TestAdminSyncUserRole` (role `user`) → 403
  `Недостаточно прав`.
- `TestAdminSyncConflict`: while a blocked sync runs, `POST /iiko/sync` → 409 `Обновление меню уже выполняется`, and
  `POST /iiko/update-products` with the right token → 409 too.
- Existing `update-products` tests keep passing with the new constructor (`tokens` can be any `*auth.Tokens`).

## API documentation
`docs/API.md`, `### Integrations`: new section `POST /iiko/sync` (role admin; notes: "for the admin panel's sync
button; takes a few seconds; 409 while another sync runs, also one started by the timer or by
`/iiko/update-products`"). Add the 409 row to `POST /iiko/update-products`. In its notes, fix the outdated sentence "The update no longer takes
images, …": the sync **does** save each product's iiko button image (`syncImages`, `type` `from_iiko`) and deletes
iiko images that disappeared; check `internal/iiko/sync_extra.go` and describe what it really does.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/iiko` is at least 80%, `docs/API.md` and
`openapi.yaml` cover the new endpoint.

## Out of scope
The timer (task 44), changes to what the sync writes, queueing a second sync instead of rejecting it.
