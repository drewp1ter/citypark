package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

// seedPending inserts an online order that costs 790.00 (699.00 items + 91.00 delivery)
// and sets it to pending, like Create does before the customer pays.
func seedPending(t *testing.T, e *svcEnv) int64 {
	t.Helper()
	e.payments.form = PaymentForm{Amount: "790.00"}
	e.payments.valid = true

	customer := testutil.InsertUser(t, e.db, "Иван", "+79990000000", "user")
	id := insertOrder(t, e.db, InsertParams{
		CustomerID:    customer,
		PaymentType:   PaymentOnline,
		ItemsHash:     ItemsHash([]ItemInput{{ProductID: e.productA, Amount: 1}, {ProductID: e.productB, Amount: 1}}),
		DeliveryPrice: 9100,
		Items: []InsertItem{
			{ProductID: e.productA, Amount: 1, OfferedPrice: 59900},
			{ProductID: e.productB, Amount: 1, OfferedPrice: 10000},
		},
	})
	if err := NewStore(e.db).SetStatus(context.Background(), id, StatusPending); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	return id
}

func callback(id int64) PaymentCallback {
	return PaymentCallback{
		TransactionID: strconv.FormatInt(id, 10),
		OperationID:   "555",
		Amount:        "790.00",
		Signature:     "998b6268fe3a6c7f4eeb8ccec65a9566",
	}
}

func (e *svcEnv) wantPaidOnce(t *testing.T, id int64) {
	t.Helper()
	o := e.order(t, id)
	if o.Status != StatusPaid || o.PaymentID != "555" {
		t.Errorf("Status = %q, PaymentID = %q; want paid and 555", o.Status, o.PaymentID)
	}
	if len(e.notifier.paid) != 1 {
		t.Fatalf("OrderPaid calls = %d, want 1", len(e.notifier.paid))
	}
	if got := e.notifier.paid[0]; got.ID != id || got.Status != StatusPaid {
		t.Errorf("notified order = id %d status %q, want id %d status paid", got.ID, got.Status, id)
	}
}

func (e *svcEnv) wantUntouched(t *testing.T, id int64, status string) {
	t.Helper()
	if o := e.order(t, id); o.Status != status || o.PaymentID != "" {
		t.Errorf("Status = %q, PaymentID = %q; want %q and empty", o.Status, o.PaymentID, status)
	}
	if len(e.notifier.paid) != 0 {
		t.Errorf("OrderPaid called: %+v", e.notifier.paid)
	}
}

func TestPaymentConfirmed(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
		t.Fatalf("PaymentConfirmed: %v", err)
	}
	e.wantPaidOnce(t, id)
}

func TestPaymentConfirmedTwice(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	for i := range 2 {
		if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	e.wantPaidOnce(t, id)
}

func TestPaymentConfirmedFromNew(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	if err := NewStore(e.db).SetStatus(context.Background(), id, StatusNew); err != nil {
		t.Fatal(err)
	}

	if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
		t.Fatalf("PaymentConfirmed: %v", err)
	}
	e.wantPaidOnce(t, id)
}

func TestPaymentBadSignature(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	e.payments.valid = false

	err := e.svc.PaymentConfirmed(context.Background(), callback(id))
	wantHTTPError(t, err, http.StatusForbidden, "Неверная подпись")
	e.wantUntouched(t, id, StatusPending)
}

func TestPaymentBadSignatureUnknownOrder(t *testing.T) {
	e := newSvcEnv(t, friday)
	seedPending(t, e)
	e.payments.valid = false

	err := e.svc.PaymentConfirmed(context.Background(), callback(999))
	wantHTTPError(t, err, http.StatusForbidden, "Неверная подпись")
}

func TestPaymentUnknownOrder(t *testing.T) {
	e := newSvcEnv(t, friday)
	seedPending(t, e)

	err := e.svc.PaymentConfirmed(context.Background(), callback(999))
	wantHTTPError(t, err, http.StatusNotFound, "Заказ не найден")
}

func TestPaymentBadTransactionID(t *testing.T) {
	for _, txID := range []string{"", "abc", "0", "-1"} {
		t.Run("id="+txID, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			cb := callback(id)
			cb.TransactionID = txID

			err := e.svc.PaymentConfirmed(context.Background(), cb)
			wantHTTPError(t, err, http.StatusBadRequest, "Некорректное уведомление")
			e.wantUntouched(t, id, StatusPending)
		})
	}
}

func TestPaymentWrongAmount(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	cb := callback(id)
	cb.Amount = "1.00"

	err := e.svc.PaymentConfirmed(context.Background(), cb)
	wantHTTPError(t, err, http.StatusBadRequest, "Неверная сумма")
	e.wantUntouched(t, id, StatusPending)
}

func TestPaymentAfterConfirmed(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	setOrderState(t, e.db, id, StatusConfirmed, "")

	if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
		t.Fatalf("PaymentConfirmed: %v", err)
	}
	e.wantUntouched(t, id, StatusConfirmed)
}

func TestPaymentAfterCanceled(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	setOrderState(t, e.db, id, StatusCanceled, "")

	if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
		t.Fatalf("PaymentConfirmed: %v", err)
	}
	e.wantUntouched(t, id, StatusCanceled)
}

// postCallback sends a Moneta form. A zero-value field is left out of the form.
func postCallback(h http.Handler, id int64, fields map[string]string) *httptest.ResponseRecorder {
	form := url.Values{
		"MNT_ID":             {"100"},
		"MNT_TRANSACTION_ID": {strconv.FormatInt(id, 10)},
		"MNT_OPERATION_ID":   {"555"},
		"MNT_AMOUNT":         {"790.00"},
		"MNT_CURRENCY_CODE":  {"RUB"},
		"MNT_TEST_MODE":      {"0"},
		"MNT_SIGNATURE":      {"998b6268fe3a6c7f4eeb8ccec65a9566"},
	}
	for k, v := range fields {
		form.Set(k, v)
	}
	return postForm(h, form.Encode())
}

func postForm(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/moneta", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func wantPlain(t *testing.T, rec *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if rec.Code != status || rec.Body.String() != body {
		t.Errorf("response = %d %q, want %d %q", rec.Code, rec.Body.String(), status, body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", ct)
	}
}

func TestCallbackSuccess(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	rec := postCallback(newRouter(e, defaultOpts()), id, nil)
	wantPlain(t, rec, http.StatusOK, "SUCCESS")
	e.wantPaidOnce(t, id)
}

func TestCallbackBadSignature(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	e.payments.valid = false

	rec := postCallback(newRouter(e, defaultOpts()), id, nil)
	wantPlain(t, rec, http.StatusForbidden, "FAIL")
	e.wantUntouched(t, id, StatusPending)
}

func TestCallbackUnknownOrder(t *testing.T) {
	e := newSvcEnv(t, friday)
	seedPending(t, e)

	rec := postCallback(newRouter(e, defaultOpts()), 999, nil)
	wantPlain(t, rec, http.StatusNotFound, "FAIL")
}

func TestCallbackEmptyBody(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	rec := postForm(newRouter(e, defaultOpts()), "")
	wantPlain(t, rec, http.StatusBadRequest, "FAIL")
	e.wantUntouched(t, id, StatusPending)
}

func TestCallbackWrongAmount(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	rec := postCallback(newRouter(e, defaultOpts()), id, map[string]string{"MNT_AMOUNT": "1.00"})
	wantPlain(t, rec, http.StatusBadRequest, "FAIL")
	e.wantUntouched(t, id, StatusPending)
}

func TestCallbackRepeated(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	h := newRouter(e, defaultOpts())

	for i := range 2 {
		wantPlain(t, postCallback(h, id, nil), http.StatusOK, "SUCCESS")
		if t.Failed() {
			t.Fatalf("call %d failed", i+1)
		}
	}
	e.wantPaidOnce(t, id)
}

func TestCallbackIgnoresWorkingHours(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 3, 0, 0, 0, time.UTC))
	id := seedPending(t, e)

	rec := postCallback(newRouter(e, defaultOpts()), id, nil)
	wantPlain(t, rec, http.StatusOK, "SUCCESS")
	e.wantPaidOnce(t, id)
}
