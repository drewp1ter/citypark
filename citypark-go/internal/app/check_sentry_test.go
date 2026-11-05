package app_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/drewp1ter/citypark-go/internal/monitoring"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type eventTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (f *eventTransport) Configure(sentry.ClientOptions) {}
func (f *eventTransport) SendEvent(e *sentry.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}
func (f *eventTransport) Flush(time.Duration) bool              { return true }
func (f *eventTransport) FlushWithContext(context.Context) bool { return true }
func (f *eventTransport) Close()                                {}

// A rejected MONETA CHECK reaches Sentry through the real router with only the reason tag.
func TestMonetaCheckFailureReachesSentry(t *testing.T) {
	db := testutil.NewDB(t)
	tr := &eventTransport{}
	client, err := monitoring.New("https://public@example.test/1", "test", "citypark-go@test", 0, tr)
	if err != nil {
		t.Fatal(err)
	}
	a := newApp(t, db, newFakeTelegram(t), map[string]string{"MONETA_ID": "100", "MONETA_SECRET": "secret"}, monitoring.Middleware(client))

	const txID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	sum := md5.Sum([]byte("CHECK100" + txID + "RUB0secret"))
	for _, signature := range []string{"00000000000000000000000000000000", hex.EncodeToString(sum[:])} {
		values := url.Values{"MNT_COMMAND": {"CHECK"}, "MNT_ID": {"100"}, "MNT_TRANSACTION_ID": {txID},
			"MNT_CURRENCY_CODE": {"RUB"}, "MNT_TEST_MODE": {"0"}, "MNT_SIGNATURE": {signature}}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/moneta/check", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		a.Handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	reasons := map[string]bool{}
	for _, event := range tr.events {
		if event.Type != "transaction" {
			reasons[event.Tags["moneta.check_reason"]] = true
			if strings.Contains(event.Message, txID) {
				t.Fatalf("event leaks the transaction id: %q", event.Message)
			}
		}
	}
	if len(reasons) != 2 || !reasons["bad_signature"] || !reasons["order_not_found"] {
		t.Fatalf("reasons %v", reasons)
	}
}
