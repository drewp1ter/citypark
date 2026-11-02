package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/logging"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
)

type transport struct {
	mu      sync.Mutex
	events  []*sentry.Event
	flushed bool
}

func (f *transport) Configure(sentry.ClientOptions) {}
func (f *transport) SendEvent(e *sentry.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}
func (f *transport) Flush(time.Duration) bool              { return true }
func (f *transport) FlushWithContext(context.Context) bool { f.flushed = true; return true }
func (f *transport) Close()                                {}

func TestMonitoring(t *testing.T) {
	tr := &transport{}
	client, err := New("https://public@example.test/1", "test", "citypark-go@test", 1, tr)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(logging.Recover(slog.New(slog.NewTextHandler(io.Discard, nil))), Middleware(client))
	var hubs []*sentry.Hub
	router.Get("/orders/{id}", httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		hubs = append(hubs, sentry.GetHubFromContext(r.Context()))
		return errors.New("private@example.test password=secret")
	}))
	router.Get("/expected", httpx.Handle(func(w http.ResponseWriter, r *http.Request) error {
		return httpx.BadRequest("Ожидаемая ошибка")
	}))
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) { panic("private@example.test") })
	for _, path := range []string{"/orders/123?token=secret", "/orders/456", "/expected", "/panic", "/missing"} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer secret")
		req.Header.Set("Cookie", "session=secret")
		req.Header.Set("sentry-trace", "0123456789abcdef0123456789abcdef-0123456789abcdef-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		want := 500
		if path == "/expected" {
			want = 400
		}
		if path == "/missing" {
			want = 404
		}
		if rec.Code != want {
			t.Fatalf("%s status %d want %d", path, rec.Code, want)
		}
	}
	if hubs[0] == nil || hubs[0] == hubs[1] {
		t.Fatal("requests must have independent hubs")
	}
	errorsCount, txCount := 0, 0
	for _, event := range tr.events {
		if event.Type == "transaction" {
			txCount++
		} else {
			errorsCount++
		}
		if event.Type != "transaction" && len(event.Exception) > 0 {
			if len(event.Fingerprint) != 2 || event.Fingerprint[1] != "GET /orders/{id}" {
				t.Fatalf("missing route grouping: %v", event.Fingerprint)
			}
		}
		if event.Environment != "test" || event.Release != "citypark-go@test" {
			t.Fatal("missing tags")
		}
		if event.Contexts["trace"]["trace_id"] == nil {
			t.Fatal("missing trace")
		}
		data, _ := json.Marshal(event)
		for _, secret := range []string{"private@example.test", "token=", "Bearer", "session=", "/orders/123", "/orders/456"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("leaked %s: %s", secret, data)
			}
		}
		if event.Type == "transaction" && event.Transaction == "GET /orders/{id}" && fmt.Sprint(event.Contexts["trace"]["trace_id"]) != "0123456789abcdef0123456789abcdef" {
			t.Fatal("trace did not continue")
		}
	}
	if errorsCount != 3 || txCount != 4 {
		t.Fatalf("errors=%d transactions=%d", errorsCount, txCount)
	}
	if !Flush(context.Background(), client) || !tr.flushed {
		t.Fatal("not flushed")
	}
}

func TestMonitoringDisabledAndInvalid(t *testing.T) {
	if _, err := New("bad dsn", "test", "", 1); err == nil {
		t.Fatal("invalid DSN accepted")
	}
	client, err := New("", "test", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	reached := false
	Middleware(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		if sentry.GetHubFromContext(r.Context()) != nil {
			t.Fatal("disabled hub")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if !reached {
		t.Fatal("handler not reached")
	}
}

func TestScrub(t *testing.T) {
	event := &sentry.Event{Request: &sentry.Request{URL: "https://secret"}, User: sentry.User{Email: "secret"}, Message: "secret", Breadcrumbs: []*sentry.Breadcrumb{{Message: "secret"}}, Tags: map[string]string{"secret": "secret"}, Contexts: map[string]sentry.Context{"trace": {"trace_id": "safe", "secret": "secret"}, "private": {"secret": "secret"}}, Exception: []sentry.Exception{{Value: "secret", Mechanism: &sentry.Mechanism{Description: "secret", Data: map[string]any{"secret": "secret"}}, Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{{Function: "handler", Vars: map[string]any{"secret": "secret"}}}}}}, Spans: []*sentry.Span{{Name: "secret", Description: "secret", Data: map[string]any{"secret": "secret"}, Tags: map[string]string{"secret": "secret"}}}}
	scrub(event, nil)
	data, _ := json.Marshal(event)
	if strings.Contains(string(data), "secret") {
		t.Fatalf("leaked: %s", data)
	}
	if event.Exception[0].Stacktrace.Frames[0].Function != "handler" {
		t.Fatal("stack removed")
	}
	e := scrub(&sentry.Event{Message: "private panic"}, nil)
	if e.Message != "Unexpected application error" {
		t.Fatal("missing safe panic message")
	}
}
