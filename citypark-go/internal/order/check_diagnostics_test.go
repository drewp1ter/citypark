package order

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
)

func TestPaymentCheckOutcomeReasons(t *testing.T) {
	for _, tc := range []struct {
		name, status, payment, amount string
		numeric, unknown              bool
		elapsed                       time.Duration
		code, reason, retry           string
		found                         bool
	}{
		{name: "numeric id", numeric: true, code: "500", reason: CheckInvalidID},
		{name: "unknown uuid", unknown: true, code: "500", reason: CheckOrderNotFound},
		{name: "canceled", status: StatusCanceled, amount: "790.00", code: "500", reason: CheckOrderCanceled, found: true},
		{name: "historical unpaid", status: StatusConfirmed, amount: "790.00", code: "500", reason: CheckNotPayable, found: true},
		{name: "paid", status: StatusPaid, payment: "555", amount: "790.00", code: "200", found: true},
		{name: "other day", status: StatusPending, amount: "790.00", elapsed: 48 * time.Hour, code: "500", reason: CheckRetryNotAllowed, retry: "expired", found: true},
		{name: "wrong amount", status: StatusPending, amount: "1.00", code: "500", reason: CheckAmountMismatch, found: true},
		{name: "no amount", status: StatusPending, code: "100", found: true},
		{name: "same amount", status: StatusPending, amount: "790.00", code: "402", found: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			status := tc.status
			if status == "" {
				status = StatusPending
			}
			setOrderState(t, e.db, id, status, tc.payment)
			e.svc.clock = clock.Fixed(friday.Add(tc.elapsed))
			cb := e.callback(t, id)
			cb.Command, cb.Amount = "CHECK", tc.amount
			switch {
			case tc.numeric:
				cb.TransactionID = strconv.FormatInt(id, 10)
			case tc.unknown:
				cb.TransactionID = unknownUUID
			}
			got, err := e.svc.PaymentCheck(context.Background(), cb)
			if err != nil {
				t.Fatal(err)
			}
			if got.Response.ResultCode != tc.code || got.Reason != tc.reason || got.RetryReason != tc.retry {
				t.Fatalf("code %s reason %q retry %q; want %s %q %q", got.Response.ResultCode, got.Reason, got.RetryReason, tc.code, tc.reason, tc.retry)
			}
			if tc.found && (got.OrderID != id || got.OrderStatus != status || got.ExpectedAmount != "790.00") {
				t.Fatalf("order details %+v", got)
			}
			if !tc.found && (got.OrderID != 0 || got.OrderStatus != "" || got.ExpectedAmount != "") {
				t.Fatalf("unexpected order details %+v", got)
			}
		})
	}
}

// captureLog replaces the default logger for one test and returns what it wrote.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return buf
}

func checkRequest(e *svcEnv, opts HandlerOptions, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/moneta/check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	newRouter(e, opts).ServeHTTP(rec, req)
	return rec
}

func TestMonetaCheckLogsAndReports(t *testing.T) {
	const signature = "deadbeefdeadbeefdeadbeefdeadbeef"
	for _, tc := range []struct {
		name         string
		command      string
		invalidSign  bool
		unknown      bool
		status       int
		reason, code string
		level        string
	}{
		{name: "no command", command: "", status: 400, reason: CheckBadRequest, level: "WARN"},
		{name: "bad signature", command: "CHECK", invalidSign: true, status: 403, reason: CheckBadSignature, level: "WARN"},
		{name: "unknown order", command: "CHECK", unknown: true, status: 200, reason: CheckOrderNotFound, code: "500", level: "WARN"},
		{name: "payable order", command: "CHECK", status: 200, code: "100", level: "INFO"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			e.payments.valid = !tc.invalidSign
			buf := captureLog(t)
			var reasons []string
			opts := defaultOpts()
			opts.OnCheckFailure = func(_ context.Context, reason string) { reasons = append(reasons, reason) }
			txID := e.uuid(t, id)
			if tc.unknown {
				txID = unknownUUID
			}
			form := url.Values{"MNT_TRANSACTION_ID": {txID}, "MNT_SIGNATURE": {signature}}
			if tc.command != "" {
				form.Set("MNT_COMMAND", tc.command)
			}
			if rec := checkRequest(e, opts, form); rec.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.status, rec.Body)
			}
			wantCalls := 0
			if tc.reason != "" {
				wantCalls = 1
			}
			if len(reasons) != wantCalls || (wantCalls == 1 && reasons[0] != tc.reason) {
				t.Fatalf("reported %v, want reason %q", reasons, tc.reason)
			}
			line := buf.String()
			for _, want := range []string{"msg=\"moneta check\"", "level=" + tc.level, "transaction_id=" + txID,
				"http_status=" + strconv.Itoa(tc.status), "result_code=" + tc.code} {
				if tc.code == "" && strings.HasPrefix(want, "result_code") {
					want = "result_code=\"\""
				}
				if !strings.Contains(line, want) {
					t.Fatalf("log %q lacks %q", line, want)
				}
			}
			if tc.reason != "" && !strings.Contains(line, "reason="+tc.reason) {
				t.Fatalf("log %q lacks reason", line)
			}
			if strings.Contains(line, signature) {
				t.Fatalf("log leaks the signature: %q", line)
			}
		})
	}
}

func TestMonetaCheckWithoutReporter(t *testing.T) {
	e := newSvcEnv(t, friday)
	seedPending(t, e)
	buf := captureLog(t)
	rec := checkRequest(e, defaultOpts(), url.Values{"MNT_COMMAND": {"CHECK"}, "MNT_TRANSACTION_ID": {unknownUUID}, "MNT_SIGNATURE": {"x"}})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<MNT_RESULT_CODE>500</MNT_RESULT_CODE>") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(buf.String(), "reason=order_not_found") {
		t.Fatalf("log %q", buf)
	}
}

func TestMonetaCheckInternalError(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	e.svc.SetPaymentHours(fakeHours{err: context.DeadlineExceeded})
	var reasons []string
	opts := defaultOpts()
	opts.OnCheckFailure = func(_ context.Context, reason string) { reasons = append(reasons, reason) }
	form := url.Values{"MNT_COMMAND": {"CHECK"}, "MNT_TRANSACTION_ID": {e.uuid(t, id)}, "MNT_SIGNATURE": {"x"}}
	if rec := checkRequest(e, opts, form); rec.Code != 500 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if len(reasons) != 1 || reasons[0] != CheckInternalError {
		t.Fatalf("reported %v", reasons)
	}
}
