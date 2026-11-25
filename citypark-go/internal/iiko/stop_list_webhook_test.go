package iiko

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

const stopListEvent = `{"eventType":"StopListUpdate","organizationId":"ORG","eventInfo":{"terminalGroupsStopListsUpdates":[{"id":"DELIVERY","isFull":false}]}}`

func TestStopListWebhookReceive(t *testing.T) {
	for _, tc := range []struct {
		name, token, auth, terminal, body string
		status                            int
		queued                            bool
	}{
		{"disabled", "", "", "delivery", "[]", 404, false},
		{"unauthorized", "secret", "wrong", "delivery", "[]", 401, false},
		{"invalid batch", "secret", "secret", "delivery", stopListEvent, 400, false},
		{"missing groups", "secret", "secret", "delivery", `[{"eventType":"StopListUpdate","organizationId":"org"}]`, 400, false},
		{"other organization", "secret", "secret", "delivery", "[" + strings.Replace(stopListEvent, "ORG", "other", 1) + "]", 200, false},
		{"other terminal", "secret", "secret", "other", "[" + stopListEvent + "]", 200, false},
		{"sole terminal fallback", "secret", "secret", "", "[" + stopListEvent + "]", 200, true},
		{"mixed events", "secret", "secret", "delivery", `[{"eventType":"DeliveryOrderUpdate","organizationId":"org","eventInfo":{}},` + stopListEvent + "]", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewStopListWebhook(tc.token, "org", tc.terminal, func(context.Context) error {
				t.Error("refresh called inline")
				return nil
			})
			handler := httpx.Handle(h.Receive)
			for range 2 {
				req := httptest.NewRequest(http.MethodPost, "/webhooks/iiko", strings.NewReader(tc.body))
				req.Header.Set("Authorization", tc.auth)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != tc.status {
					t.Fatalf("status = %d, want %d", rec.Code, tc.status)
				}
			}
			if got := len(h.pending) == 1; got != tc.queued {
				t.Errorf("refresh queued = %t, want %t", got, tc.queued)
			}
		})
	}
}

func TestStopListWebhookWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	called := make(chan struct{}, 1)
	h := NewStopListWebhook("secret", "org", "delivery", func(context.Context) error {
		called <- struct{}{}
		return nil
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	h.enqueue()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("refresh was not called")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

func TestStopListWebhookWorkerRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var calls atomic.Int32
		h := NewStopListWebhook("secret", "org", "delivery", func(context.Context) error {
			if calls.Add(1) == 1 {
				return errors.New("temporarily unavailable")
			}
			return nil
		})
		go h.Run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
		h.enqueue()
		synctest.Wait()
		time.Sleep(29 * time.Second)
		synctest.Wait()
		if got := calls.Load(); got != 1 {
			t.Fatalf("refresh called %d times before retry delay", got)
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if got := calls.Load(); got != 2 {
			t.Fatalf("refresh called %d times after retry delay", got)
		}
	})
}
