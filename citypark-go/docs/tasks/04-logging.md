# Task 04: Logging and middleware

## Goal
Set up `log/slog` and two chi middlewares: request logging and panic recovery.

## Depends on
03.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/httpx/httpx.go`.

## Files to create
- `internal/logging/logging.go`
- `internal/logging/middleware.go`
- `internal/logging/middleware_test.go`

## Spec

```go
package logging

// New returns a logger writing to w.
//   env == "production": JSON handler
//   otherwise:           text handler
// Level is Debug when debug is true, else Info.
func New(w io.Writer, env string, debug bool) *slog.Logger

// Requests logs one line per finished request.
func Requests(log *slog.Logger) func(http.Handler) http.Handler

// Recover turns a panic in a handler into a 500 response.
func Recover(log *slog.Logger) func(http.Handler) http.Handler
```

### `Requests`
After the next handler returns, log with message `request` and these attributes:

| Key | Value |
|---|---|
| `method` | `r.Method` |
| `path` | `r.URL.Path` (never the query string or body: they can hold personal data) |
| `status` | response status code (200 when the handler never called `WriteHeader`) |
| `duration_ms` | elapsed milliseconds as an integer |
| `ip` | first value of the `X-Forwarded-For` header when present, else `r.RemoteAddr` |

Level: `Error` for status >= 500, `Warn` for 400–499, `Info` otherwise.

To capture the status, wrap the `http.ResponseWriter` in a small struct that records the code in `WriteHeader` and defaults to 200 on the first `Write`.

### `Recover`
Use `defer` + `recover()`. On panic: log at `Error` level with message `panic`, attributes `panic` (the recovered value formatted with `%v`) and `stack` (`string(debug.Stack())`), then call `httpx.WriteError(w, r, httpx.NewError(500, "Внутренняя ошибка сервера"))`.

## Tests to write
Use a `bytes.Buffer` as the log output with `New(&buf, "production", false)` so that each log line is JSON.
- `TestRequestsLogsStatus`: a handler that writes 201; the log line has `"status":201`, `"method":"POST"`, `"path":"/x"` and level `INFO`.
- `TestRequestsDefaultStatus`: a handler that only calls `w.Write`; logged status is 200.
- `TestRequestsLevels`: 404 logs `WARN`, 500 logs `ERROR`.
- `TestRequestsNoQuery`: request `/x?token=secret`; the log output does not contain `secret`.
- `TestRequestsForwardedFor`: header `X-Forwarded-For: 1.2.3.4, 5.6.7.8` logs `"ip":"1.2.3.4"`.
- `TestRecover`: a handler that panics gives status 500, body `{"statusCode":500,"message":"Внутренняя ошибка сервера"}`, and a log line with `"msg":"panic"`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Wiring the middleware into `main.go` (task 31).
