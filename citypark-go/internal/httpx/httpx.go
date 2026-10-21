// Package httpx holds the HTTP helpers shared by every handler: JSON output,
// the expected-error type, body decoding, path ids and pagination.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const (
	maxBodyBytes = 1 << 20
	maxPageSize  = 500
)

// Error is an expected failure that is shown to the API user.
type Error struct {
	Status  int               `json:"statusCode"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"errors,omitempty"`
}

// Error returns the user-facing message.
func (e *Error) Error() string { return e.Message }

// NewError builds an Error with the given status and message.
func NewError(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// BadRequest is a 400 with the given message.
func BadRequest(message string) *Error { return NewError(http.StatusBadRequest, message) }

// Unauthorized is a 401.
func Unauthorized() *Error {
	return NewError(http.StatusUnauthorized, "Требуется авторизация")
}

// Forbidden is a 403.
func Forbidden() *Error {
	return NewError(http.StatusForbidden, "Недостаточно прав")
}

// NotFound is a 404.
func NotFound() *Error {
	return NewError(http.StatusNotFound, "Не найдено")
}

// JSON writes v with the given status and Content-Type application/json; charset=utf-8.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("encode json response", "error", err)
	}
}

// NoContent writes status 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError writes err as JSON. If err is (or wraps) *Error, its status and
// body are used. Any other error is logged and answered as a generic 500.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	if errors.As(err, &e) {
		JSON(w, e.Status, e)
		return
	}
	slog.Default().Error("unhandled error", "method", r.Method, "path", r.URL.Path, "error", err)
	JSON(w, http.StatusInternalServerError, NewError(http.StatusInternalServerError, "Внутренняя ошибка сервера"))
}

// HandlerFunc is the handler signature used in this project.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts a HandlerFunc to http.HandlerFunc and sends returned errors to WriteError.
func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

// DecodeJSON reads the request body (at most 1 MiB) into dst. An empty body,
// malformed JSON or a value of the wrong type returns BadRequest.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return BadRequest("Некорректный JSON")
	}
	return nil
}

// PathID reads the URL parameter name as a positive int64. On failure it returns NotFound.
func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id < 1 {
		return 0, NotFound()
	}
	return id, nil
}

// Pagination reads the query parameters page (default 1) and pageSize
// (default defaultSize) and returns limit and offset.
func Pagination(r *http.Request, defaultSize int) (limit, offset int, err error) {
	q := r.URL.Query()
	page, err := intParam(q.Get("page"), 1)
	if err != nil {
		return 0, 0, err
	}
	size, err := intParam(q.Get("pageSize"), defaultSize)
	if err != nil {
		return 0, 0, err
	}
	if size > maxPageSize {
		return 0, 0, BadRequest("Некорректные параметры пагинации")
	}
	return size, (page - 1) * size, nil
}

// intParam parses an integer >= 1, or returns def when raw is empty.
func intParam(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, BadRequest("Некорректные параметры пагинации")
	}
	return n, nil
}
