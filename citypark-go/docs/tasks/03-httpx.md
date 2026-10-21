# Task 03: httpx — JSON responses and errors

## Goal
Shared HTTP helpers that every handler uses: JSON output, one error type, body decoding, path ids, pagination.

## Depends on
01.

## Read first
`docs/tasks/CONVENTIONS.md`.

## Files to create
- `internal/httpx/httpx.go`
- `internal/httpx/httpx_test.go`

## Files to change
- `internal/health/handler.go`: use `httpx.Handle` and `httpx.JSON` instead of hand-written JSON.

## Spec

```go
package httpx

// Error is an expected failure that is shown to the API user.
type Error struct {
    Status  int               `json:"statusCode"`
    Message string            `json:"message"`
    Fields  map[string]string `json:"errors,omitempty"`
}

func (e *Error) Error() string // returns Message

func NewError(status int, message string) *Error
func BadRequest(message string) *Error // 400
func Unauthorized() *Error             // 401 "Требуется авторизация"
func Forbidden() *Error                // 403 "Недостаточно прав"
func NotFound() *Error                 // 404 "Не найдено"

// JSON writes v with the given status and Content-Type application/json; charset=utf-8.
func JSON(w http.ResponseWriter, status int, v any)

// NoContent writes status 204 with no body.
func NoContent(w http.ResponseWriter)

// WriteError writes err as JSON. If err is (or wraps) *Error, its status and body are used.
// Any other error is logged with slog.Default().Error and answered as
// 500 {"statusCode":500,"message":"Внутренняя ошибка сервера"}.
func WriteError(w http.ResponseWriter, r *http.Request, err error)

// HandlerFunc is the handler signature used in this project.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts a HandlerFunc to http.HandlerFunc and sends returned errors to WriteError.
func Handle(h HandlerFunc) http.HandlerFunc

// DecodeJSON reads the request body into dst.
// Empty body, malformed JSON, or a JSON value of the wrong type returns BadRequest("Некорректный JSON").
// The body is limited to 1 MiB with http.MaxBytesReader.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error

// PathID reads chi.URLParam(r, name) as a positive int64. On failure it returns NotFound().
func PathID(r *http.Request, name string) (int64, error)

// Pagination reads the query parameters page (default 1) and pageSize (default defaultSize).
// Both must be integers >= 1 and pageSize must be <= 500, otherwise
// BadRequest("Некорректные параметры пагинации") is returned.
// limit = pageSize, offset = (page - 1) * pageSize.
func Pagination(r *http.Request, defaultSize int) (limit, offset int, err error)
```

Details:
- Use `errors.As` in `WriteError` so wrapped errors work.
- `JSON` must write `null`-safe output for slices: callers pass non-nil slices, do not special-case here.
- If encoding fails in `JSON`, log it with `slog.Default().Error`; do not panic.

## Tests to write
- `TestJSON`: status, content type and body.
- `TestWriteErrorKnown`: a `*Error` with `Fields` gives the exact JSON `{"statusCode":400,"message":"m","errors":{"a":"b"}}`.
- `TestWriteErrorWrapped`: `fmt.Errorf("x: %w", NotFound())` gives 404.
- `TestWriteErrorUnknown`: `errors.New("boom")` gives 500 and the Russian message, and the body does not contain `boom`.
- `TestDecodeJSON`: table test: valid object, empty body, `{bad`, `[]` into a struct.
- `TestPathID`: table test through a chi router with route `/x/{id}`: `5` ok, `0`, `-1`, `abc` give 404.
- `TestPagination`: table test: no params → `(defaultSize, 0)`; `page=3&pageSize=10` → `(10, 20)`; `page=0`, `pageSize=abc`, `pageSize=501` → 400.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Logging middleware (task 04), validation (task 05), authentication.
