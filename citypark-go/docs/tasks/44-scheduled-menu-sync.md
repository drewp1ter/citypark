# Task 44: Scheduled menu sync

## Goal
The server syncs the iiko menu by itself every `IIKO_SYNC_INTERVAL` (and once at startup), so prices changed in iiko
reach the site without anyone pressing a button. `0` turns the timer off.

## Depends on
43.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `cmd/server/main.go`, `internal/app/app.go`,
`internal/config/config.go`.

## Background
Nothing calls `/iiko/update-products` on a schedule today. The timer uses the same `Syncer`, so the lock from task 43
keeps it from overlapping with the admin's button.

## Files to change
- `internal/iiko/sync.go`: `RunEvery` (Spec 1).
- `internal/iiko/sync_test.go`: the tests below.
- `internal/config/config.go` + its test: `Iiko.SyncInterval time.Duration` from `IIKO_SYNC_INTERVAL`, default `0`,
  parsed with `time.ParseDuration`; a negative value or a parse error → `IIKO_SYNC_INTERVAL is invalid: ...`.
- `.env.example`: `IIKO_SYNC_INTERVAL=0` in the iiko block, with the comment
  `# how often the server syncs the menu, e.g. 30m; 0 = never (production: 30m)`.
- `internal/app/app.go`: add `Syncer *iiko.Syncer` to `App` and set it.
- `cmd/server/main.go`: start the timer (Spec 2).

## Spec

### 1. `RunEvery`
```go
// RunEvery syncs once right away and then on every value from tick, until ctx is done.
// Errors are logged, never returned: a failed sync waits for the next tick.
func (s *Syncer) RunEvery(ctx context.Context, tick <-chan time.Time, log *slog.Logger)
```
- Each run calls `s.Sync(ctx)`.
- Success → `log.Info("scheduled menu sync done", "categories", n, "products", m)`.
- `ErrSyncRunning` → `log.Info("scheduled menu sync skipped: another sync is running")`.
- Other error → `log.Error("scheduled menu sync failed", "error", err)`.
- Returns when `ctx.Done()` is closed, also while waiting for a tick.

### 2. `main.go`
After `app.New` and before `ListenAndServe`:
```go
if cfg.Iiko.SyncInterval > 0 {
	if cfg.Iiko.APIKey == "" {
		log.Warn("IIKO_SYNC_INTERVAL is set but iiko is not configured; the timer is off")
	} else {
		ticker := time.NewTicker(cfg.Iiko.SyncInterval)
		defer ticker.Stop()
		go a.Syncer.RunEvery(ctx, ticker.C, log)
	}
}
```
`ctx` is the signal context, so the timer stops on shutdown.

## Tests to write
With the fake `Source` from the existing tests, a `slog` logger writing to a `bytes.Buffer`, and a
`chan time.Time` you control:
- `TestRunEverySyncsAtStartAndOnTick`: start `RunEvery` in a goroutine; wait for the first sync (fake counts `Fetch`
  calls); send one tick → second `Fetch`; cancel ctx → `RunEvery` returns (use a `done` channel with a 1 s timeout).
  The log contains `scheduled menu sync done` twice.
- `TestRunEveryLogsError`: fake `Fetch` returns an error → the log contains `scheduled menu sync failed`; the loop
  keeps running (a tick causes another `Fetch`).
- `TestRunEverySkipsWhenRunning`: hold the lock with a blocked `Sync` from another goroutine, start `RunEvery` →
  the log contains `skipped`.
- Config: `IIKO_SYNC_INTERVAL` missing → 0; `30m` → 30 minutes; `-1m` and `abc` → error.

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/iiko` and `internal/config` is at least 80%, and
with `IIKO_SYNC_INTERVAL=1m` and valid iiko settings `make run` logs `scheduled menu sync done` at startup.

## Out of scope
A cron syntax, a sync status endpoint, retries inside one run.
