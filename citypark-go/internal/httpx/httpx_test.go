package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]int{"a": 1})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("content type = %q", got)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"a":1}` {
		t.Errorf("body = %q", got)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name    string
		err     *Error
		status  int
		message string
	}{
		{"bad request", BadRequest("m"), 400, "m"},
		{"unauthorized", Unauthorized(), 401, "Требуется авторизация"},
		{"forbidden", Forbidden(), 403, "Недостаточно прав"},
		{"not found", NotFound(), 404, "Не найдено"},
		{"custom", NewError(409, "conflict"), 409, "conflict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status || tt.err.Message != tt.message {
				t.Errorf("got %d %q", tt.err.Status, tt.err.Message)
			}
			if tt.err.Error() != tt.message {
				t.Errorf("Error() = %q", tt.err.Error())
			}
		})
	}
}

func TestWriteErrorKnown(t *testing.T) {
	rec := httptest.NewRecorder()
	e := BadRequest("m")
	e.Fields = map[string]string{"a": "b"}
	WriteError(rec, httptest.NewRequest(http.MethodGet, "/", nil), e)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	want := `{"statusCode":400,"message":"m","errors":{"a":"b"}}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestWriteErrorWrapped(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequest(http.MethodGet, "/", nil), fmt.Errorf("x: %w", NotFound()))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestWriteErrorUnknown(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("boom"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if want := `{"statusCode":500,"message":"Внутренняя ошибка сервера"}`; body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
	if strings.Contains(body, "boom") {
		t.Errorf("body leaks internal error: %s", body)
	}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name   string
		h      HandlerFunc
		status int
	}{
		{"ok", func(w http.ResponseWriter, _ *http.Request) error {
			JSON(w, http.StatusOK, map[string]bool{"ok": true})
			return nil
		}, 200},
		{"error", func(http.ResponseWriter, *http.Request) error { return Forbidden() }, 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handle(tt.h)(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d", rec.Code, tt.status)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    string
	}{
		{"valid object", `{"name":"x"}`, false, "x"},
		{"empty body", ``, true, ""},
		{"malformed", `{bad`, true, ""},
		{"wrong type", `[]`, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var dst payload
			err := DecodeJSON(httptest.NewRecorder(), r, &dst)
			if !tt.wantErr {
				if err != nil || dst.Name != tt.want {
					t.Fatalf("err = %v, name = %q", err, dst.Name)
				}
				return
			}
			var e *Error
			if !errors.As(err, &e) || e.Status != 400 || e.Message != "Некорректный JSON" {
				t.Fatalf("err = %v, want 400 Некорректный JSON", err)
			}
		})
	}
}

func TestPathID(t *testing.T) {
	tests := []struct {
		raw     string
		want    int64
		wantErr bool
	}{
		{"5", 5, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			var got int64
			var gotErr error
			router := chi.NewRouter()
			router.Get("/x/{id}", func(_ http.ResponseWriter, r *http.Request) {
				got, gotErr = PathID(r, "id")
			})
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x/"+tt.raw, nil))

			if tt.wantErr {
				var e *Error
				if !errors.As(gotErr, &e) || e.Status != 404 {
					t.Fatalf("err = %v, want 404", gotErr)
				}
				return
			}
			if gotErr != nil || got != tt.want {
				t.Errorf("got %d, %v; want %d", got, gotErr, tt.want)
			}
		})
	}
}

func TestPagination(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{"defaults", "", 20, 0, false},
		{"page and size", "page=3&pageSize=10", 10, 20, false},
		{"max size", "pageSize=500", 500, 0, false},
		{"page zero", "page=0", 0, 0, true},
		{"size not a number", "pageSize=abc", 0, 0, true},
		{"page not a number", "page=x", 0, 0, true},
		{"size too large", "pageSize=501", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)
			limit, offset, err := Pagination(r, 20)
			if tt.wantErr {
				var e *Error
				if !errors.As(err, &e) || e.Status != 400 || e.Message != "Некорректные параметры пагинации" {
					t.Fatalf("err = %v, want 400", err)
				}
				return
			}
			if err != nil || limit != tt.wantLimit || offset != tt.wantOffset {
				t.Errorf("got (%d, %d, %v), want (%d, %d)", limit, offset, err, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}
