package order

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestParseTransactionID(t *testing.T) {
	const lower = "3f2b8c1e-9d4a-4e7b-a1c6-5d0e8f7a2b94"
	for _, tt := range []struct {
		name, in, want string
		ok             bool
	}{
		{"lowercase", lower, lower, true},
		{"uppercase", "3F2B8C1E-9D4A-4E7B-A1C6-5D0E8F7A2B94", lower, true},
		{"empty", "", "", false},
		{"old numeric id", "15", "", false},
		{"no dashes", "3f2b8c1e9d4a4e7ba1c65d0e8f7a2b94", "", false},
		{"one character too long", lower + "0", "", false},
		{"non-hex character", "3f2b8c1e-9d4a-4e7b-a1c6-5d0e8f7a2b9g", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTransactionID(tt.in)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("parseTransactionID(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCreateOrderUsesPaymentUUID(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	var stored string
	if err := e.db.Get(&stored, `SELECT payment_uuid::text FROM orders WHERE id = $1`, res.OrderID); err != nil {
		t.Fatal(err)
	}
	if len(e.payments.requests) != 1 {
		t.Fatalf("payment requests = %d, want 1", len(e.payments.requests))
	}
	got := e.payments.requests[0].TransactionID
	if got != stored {
		t.Errorf("TransactionID = %q, want the stored payment_uuid %q", got, stored)
	}
	if _, ok := parseTransactionID(got); !ok || got == strconv.FormatInt(res.OrderID, 10) {
		t.Errorf("TransactionID = %q, want a canonical UUID that is not the order number %d", got, res.OrderID)
	}
}

func TestOrderPaymentUUIDsAreUnique(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	store := NewStore(db)
	first := insertOrder(t, db, sd.params(true))
	second := insertOrder(t, db, sd.params(true))
	a, err := store.ByID(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.ByID(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if a.PaymentUUID == b.PaymentUUID {
		t.Fatalf("both orders have payment UUID %q", a.PaymentUUID)
	}
	for _, id := range []string{a.PaymentUUID, b.PaymentUUID} {
		if got, ok := parseTransactionID(id); !ok || got != id {
			t.Errorf("payment UUID %q is not canonical lowercase", id)
		}
	}
	found, err := store.ByPaymentUUID(context.Background(), b.PaymentUUID)
	if err != nil || found.ID != second {
		t.Errorf("ByPaymentUUID = order %d, %v; want %d", found.ID, err, second)
	}
}

func TestPaymentConfirmedByUUID(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	cb := e.callback(t, id)
	cb.TransactionID = strings.ToUpper(cb.TransactionID) // the letter case must not matter

	if err := e.svc.PaymentConfirmed(context.Background(), cb); err != nil {
		t.Fatalf("PaymentConfirmed: %v", err)
	}
	e.wantPaidOnce(t, id)
}

func TestPaymentConfirmedRejectsNumericID(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	cb := callbackUUID(strconv.FormatInt(id, 10))

	err := e.svc.PaymentConfirmed(context.Background(), cb)
	wantHTTPError(t, err, http.StatusBadRequest, "Некорректное уведомление")
	e.wantUntouched(t, id, StatusPending)
}

func TestPaymentConfirmedUnknownUUID(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := seedPending(t, e)

	err := e.svc.PaymentConfirmed(context.Background(), callbackUUID("9c1d7a52-6e0b-4b8f-8a3d-2f4e6c8a0b13"))
	wantHTTPError(t, err, http.StatusNotFound, "Заказ не найден")
	e.wantUntouched(t, id, StatusPending)
}

func TestPaymentCheckByUUID(t *testing.T) {
	for _, tt := range []struct {
		name, status, payment, code, amount string
		numeric, unknown                    bool
	}{
		{name: "new", status: StatusNew, code: "100", amount: "790.00"},
		{name: "paid", status: StatusPaid, payment: "555", code: "200", amount: "790.00"},
		{name: "canceled", status: StatusCanceled, code: "500", amount: "790.00"},
		{name: "numeric id", status: StatusNew, code: "500", numeric: true},
		{name: "unknown uuid", status: StatusNew, code: "500", unknown: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			setOrderState(t, e.db, id, tt.status, tt.payment)
			cb := e.callback(t, id)
			cb.Command, cb.Amount = "CHECK", ""
			switch {
			case tt.numeric:
				cb.TransactionID = strconv.FormatInt(id, 10)
			case tt.unknown:
				cb.TransactionID = unknownUUID
			}
			got, err := e.svc.PaymentCheck(context.Background(), cb)
			if err != nil || got.Response.ResultCode != tt.code || got.Response.Amount != tt.amount || got.Response.TransactionID != cb.TransactionID {
				t.Fatalf("response %+v, %v; want code %s amount %q echoing %q", got, err, tt.code, tt.amount, cb.TransactionID)
			}
		})
	}

	e := newSvcEnv(t, friday)
	id := seedPending(t, e)
	e.payments.valid = false
	_, err := e.svc.PaymentCheck(context.Background(), e.callback(t, id))
	wantHTTPError(t, err, http.StatusForbidden, "Неверная подпись")
}

// signingPayments builds real MONETA URLs so the test can read MNT_TRANSACTION_ID back.
type signingPayments struct {
	*fakePayments
	signer *moneta.Signer
}

func (p signingPayments) Form(r PaymentRequest) PaymentForm {
	p.fakePayments.requests = append(p.fakePayments.requests, r)
	amount := moneta.FormatAmount(r.Amount)
	return PaymentForm{Amount: amount, URL: p.signer.PaymentURL("https://pay.test/", r.TransactionID, amount, "", "", r.SubscriberID)}
}

func TestRetryPaymentSignsSameUUID(t *testing.T) {
	e, id := prepareRetry(t)
	e.svc.payments = signingPayments{e.payments, moneta.New("100", "secret", false)}
	want := e.order(t, id).PaymentUUID

	for i := range 2 {
		res, err := e.svc.RetryPayment(context.Background(), id, e.userID)
		if err != nil {
			t.Fatalf("retry %d: %v", i+1, err)
		}
		u, err := url.Parse(res.RedirectURL)
		if err != nil {
			t.Fatal(err)
		}
		if got := u.Query().Get("MNT_TRANSACTION_ID"); got != want {
			t.Errorf("retry %d: MNT_TRANSACTION_ID = %q, want the order's payment_uuid %q", i+1, got, want)
		}
	}
}
