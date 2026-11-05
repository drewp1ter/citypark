package order

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/moneta"
)

func TestPaymentCheckStates(t *testing.T) {
	for _, tc := range []struct{ name, status, payment, amount, code string }{
		{"pending", StatusPending, "", "790.00", "402"},
		{"new", StatusNew, "", "790.00", "402"},
		{"missing amount", StatusPending, "", "", "100"},
		{"wrong amount", StatusPending, "", "1.00", "500"},
		{"paid", StatusPaid, "555", "790.00", "200"},
		{"confirmed", StatusConfirmed, "555", "790.00", "200"},
		{"canceled", StatusCanceled, "", "790.00", "500"},
		{"canceled paid", StatusCanceled, "555", "790.00", "500"},
		{"historical unpaid", StatusConfirmed, "", "790.00", "500"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			setOrderState(t, e.db, id, tc.status, tc.payment)
			cb := e.callback(t, id)
			cb.Command = "CHECK"
			cb.Amount = tc.amount
			got, err := e.svc.PaymentCheck(context.Background(), cb)
			if err != nil || got.ResultCode != tc.code || got.Amount != "790.00" {
				t.Fatalf("response %+v, %v", got, err)
			}
			o := e.order(t, id)
			if o.Status != tc.status || o.PaymentID != tc.payment || len(e.notifier.paid) != 0 {
				t.Fatal("check changed order")
			}
		})
	}
}

func TestPaymentCheckErrors(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	for _, id := range []string{strconv.FormatInt(id, 10), unknownUUID, "abc", "0", "-1"} {
		cb := PaymentCallback{Command: "CHECK", TransactionID: id}
		got, err := e.svc.PaymentCheck(context.Background(), cb)
		if err != nil || got.ResultCode != "500" {
			t.Fatalf("%s: %+v %v", id, got, err)
		}
	}
	e.payments.valid = false
	_, err := e.svc.PaymentCheck(context.Background(), PaymentCallback{})
	wantHTTPError(t, err, http.StatusForbidden, "Неверная подпись")
}

func TestCheckHTTP(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			form := url.Values{"MNT_COMMAND": {"CHECK"}, "MNT_TRANSACTION_ID": {e.uuid(t, id)}, "MNT_AMOUNT": {"790.00"}}
			path := "/webhooks/moneta/check"
			body := ""
			if method == http.MethodGet {
				path += "?" + form.Encode()
			} else {
				body = form.Encode()
			}
			req := httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			newRouter(e, defaultOpts()).ServeHTTP(rec, req)
			var got moneta.CheckResponse
			if rec.Code != 200 || xml.Unmarshal(rec.Body.Bytes(), &got) != nil || got.ResultCode != "402" {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			sum := md5.Sum([]byte("402100" + e.uuid(t, id) + "secret"))
			if got.Signature != hex.EncodeToString(sum[:]) {
				t.Fatal("bad response signature")
			}
			e.wantUntouched(t, id, StatusPending)
		})
	}
}

func TestMonetaMalformedForms(t *testing.T) {
	for _, path := range []string{"/webhooks/moneta", "/webhooks/moneta/check"} {
		for _, body := range []string{"MNT_COMMAND=%zz", "MNT_COMMAND=CHECK&MNT_COMMAND=CHECK", "MNT_COMMAND=PAY"} {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			newRouter(e, defaultOpts()).ServeHTTP(rec, req)
			if rec.Code != 400 {
				t.Fatalf("%s %s: %d", path, body, rec.Code)
			}
			e.wantUntouched(t, id, StatusPending)
		}
	}
}

func TestPayGET(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	form := url.Values{
		"MNT_TRANSACTION_ID": {e.uuid(t, id)},
		"MNT_OPERATION_ID":   {"555"},
		"MNT_AMOUNT":         {"790.00"},
	}
	req := httptest.NewRequest(http.MethodGet, "/webhooks/moneta?"+form.Encode(), nil)
	rec := httptest.NewRecorder()
	newRouter(e, defaultOpts()).ServeHTTP(rec, req)
	wantPlain(t, rec, 200, "SUCCESS")
	e.wantPaidOnce(t, id)
}

func TestCheckHTTPBadSignature(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	e.payments.valid = false
	req := httptest.NewRequest(http.MethodGet, "/webhooks/moneta/check?MNT_COMMAND=CHECK&MNT_TRANSACTION_ID="+e.uuid(t, id), nil)
	rec := httptest.NewRecorder()
	newRouter(e, defaultOpts()).ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	e.wantUntouched(t, id, StatusPending)
}

func TestPayGETInvalidAmount(t *testing.T) {
	for _, tc := range []struct{ name, amount string }{
		{"omitted", ""},
		{"mismatch", "1.00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			form := url.Values{
				"MNT_TRANSACTION_ID": {e.uuid(t, id)},
				"MNT_OPERATION_ID":   {"555"},
			}
			if tc.amount != "" {
				form.Set("MNT_AMOUNT", tc.amount)
			}
			req := httptest.NewRequest(http.MethodGet, "/webhooks/moneta?"+form.Encode(), nil)
			rec := httptest.NewRecorder()
			newRouter(e, defaultOpts()).ServeHTTP(rec, req)
			wantPlain(t, rec, http.StatusBadRequest, "FAIL")
			e.wantUntouched(t, id, StatusPending)
		})
	}
}
