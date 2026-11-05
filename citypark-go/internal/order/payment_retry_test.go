package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/workinghours"
)

type countingPaymentHours struct {
	calls int
	err   error
}

func (h *countingPaymentHours) Get(context.Context) (workinghours.Hours, time.Time, error) {
	h.calls++
	return workinghours.Hours{Start: hoursStart, End: hoursEnd}, time.Time{}, h.err
}

func TestPaymentRetryEligibility(t *testing.T) {
	loc := time.FixedZone("Moscow", 3*3600)
	now := time.Date(2026, 11, 5, 0, 30, 0, 0, loc)
	for _, tc := range []struct {
		name, status, payment string
		created               time.Time
		reason                string
	}{
		{"new", StatusNew, "", now, ""}, {"pending", StatusPending, "", now, ""},
		{"paid", StatusPaid, "", now, "not_payable"}, {"confirmed", StatusConfirmed, "", now, "not_payable"},
		{"canceled", StatusCanceled, "", now, "not_payable"}, {"payment id", StatusPending, "123", now, "not_payable"},
		{"yesterday", StatusPending, "", now.Add(-time.Hour), "expired"},
		{"tomorrow", StatusPending, "", now.AddDate(0, 0, 1), "expired"},
		{"UTC previous day", StatusPending, "", now.UTC(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := paymentState(Order{Status: tc.status, PaymentID: tc.payment, CreatedAt: tc.created}, now)
			if got.Reason != tc.reason || got.Allowed != (tc.reason == "") {
				t.Fatalf("%+v", got)
			}
		})
	}
	for _, tc := range []struct {
		hour, minute int
		allowed      bool
	}{{10, 59, false}, {11, 0, true}, {22, 59, true}, {23, 0, false}} {
		now := time.Date(2026, 11, 5, tc.hour, tc.minute, 0, 0, loc)
		svc := &Service{clock: clock.Fixed(now)}
		svc.SetPaymentHours(fakeHours{h: workinghours.Hours{Start: "11:00", End: "23:00"}})
		got, err := svc.customerOrders(context.Background(), []Order{{Status: StatusPending, CreatedAt: now}})
		if err != nil || got[0].PaymentRetry.Allowed != tc.allowed {
			t.Fatalf("%+v: %+v %v", tc, got, err)
		}
	}
}

func prepareRetry(t *testing.T) (*readEnv, int64) {
	t.Helper()
	e := newReadEnv(t)
	id := e.userOrders[0]
	e.db.MustExec(`UPDATE orders SET created_at=$2 WHERE account_id=$1`, e.userID, friday)
	return e, id
}

func TestRetryPayment(t *testing.T) {
	e, id := prepareRetry(t)
	ctx := context.Background()
	// Saved discounts and delivery remain authoritative after catalog changes.
	e.db.MustExec(`UPDATE order_items SET discount_kopecks=100 WHERE order_id=$1`, id)
	e.db.MustExec(`UPDATE orders SET discount_kopecks=100 WHERE id=$1`, id)
	before := e.order(t, id)
	e.db.MustExec(`UPDATE products SET price_kopecks=1, is_order_blocked=true WHERE id=$1`, e.productA)
	counts := []int{e.count(t, "SELECT count(*) FROM orders"), e.count(t, "SELECT count(*) FROM order_items"), e.count(t, "SELECT count(*) FROM users")}
	for range 2 {
		got, err := e.svc.RetryPayment(ctx, id, e.userID)
		if err != nil || got.OrderID != id || got.RedirectURL != e.payments.form.URL {
			t.Fatalf("%+v %v", got, err)
		}
		req := e.payments.requests[len(e.payments.requests)-1]
		if req.OrderID != id || req.Amount != before.Total+before.DeliveryPrice {
			t.Fatalf("%+v", req)
		}
	}
	if !reflect.DeepEqual(before, e.order(t, id)) || len(e.notifier.pending)+len(e.notifier.paid) != 0 {
		t.Fatal("retry changed order or notified")
	}
	after := []int{e.count(t, "SELECT count(*) FROM orders"), e.count(t, "SELECT count(*) FROM order_items"), e.count(t, "SELECT count(*) FROM users")}
	if !reflect.DeepEqual(counts, after) {
		t.Fatal("retry created rows")
	}
}

func TestRetryPaymentHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, token, id, state, message string
		status                          int
	}{
		{"success", "owner", "", "", "", 200},
		{"missing session", "", "", "", "Требуется авторизация", 401},
		{"invalid session", "bad", "", "", "Требуется авторизация", 401},
		{"admin", "admin", "", "", "Недостаточно прав", 403},
		{"foreign", "other", "", "", "Не найдено", 404},
		{"missing order", "owner", "999999", "", "Не найдено", 404},
		{"invalid id", "owner", "bad", "", "Не найдено", 404},
		{"paid", "owner", "", "paid", "Этот заказ уже оплачен или недоступен для оплаты.", 409},
		{"expired", "owner", "", "expired", "Срок оплаты заказа истёк. Оформите новый заказ.", 409},
		{"closed", "owner", "", "closed", "Оплата доступна только в часы работы ресторана.", 503},
		{"hours failure", "owner", "", "hours", "", 500},
		{"missing hours", "owner", "", "nil", "", 500},
		{"empty URL", "owner", "", "url", "", 500},
		{"database failure", "owner", "", "db", "", 500},
		{"guest", "owner", "", "guest", "Не найдено", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, id := prepareRetry(t)
			switch tc.state {
			case "paid":
				e.db.MustExec(`UPDATE orders SET status='paid' WHERE id=$1`, id)
			case "expired":
				e.db.MustExec(`UPDATE orders SET created_at=created_at-interval '1 day' WHERE id=$1`, id)
			case "closed":
				e.svc.SetPaymentHours(fakeHours{h: workinghours.Hours{Start: "00:00", End: "01:00"}})
			case "hours":
				e.svc.SetPaymentHours(fakeHours{err: errors.New("hours unavailable")})
			case "nil":
				e.svc.SetPaymentHours(nil)
			case "url":
				e.payments.form.URL = ""
			case "db":
				e.db.Close()
			case "guest":
				e.db.MustExec(`UPDATE orders SET account_id=NULL WHERE id=$1`, id)
			}
			token := map[string]string{"owner": e.userToken, "other": e.otherToken, "admin": e.adminToken, "bad": "invalid"}[tc.token]
			raw := fmt.Sprint(id)
			if tc.id != "" {
				raw = tc.id
			}
			rec := testutil.DoJSON(t, e.router, "POST", "/orders/"+raw+"/payment-retry", token, nil)
			if rec.Code != tc.status || rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			if tc.message != "" {
				var body struct{ Message string }
				testutil.Decode(t, rec, &body)
				if body.Message != tc.message {
					t.Fatalf("message %q", body.Message)
				}
			}
			if rec.Code == 200 {
				t.Logf("retry response: %s", rec.Body)
			}
		})
	}
}

func TestCustomerOrderPaymentRetry(t *testing.T) {
	e, id := prepareRetry(t)
	hours := &countingPaymentHours{}
	e.svc.SetPaymentHours(hours)
	got := e.getList(t, "/orders/my", e.userToken)
	if hours.calls != 1 || len(got.Orders) != 2 {
		t.Fatalf("calls %d orders %d", hours.calls, len(got.Orders))
	}
	for _, o := range got.Orders {
		if o["paymentRetry"].(map[string]any)["allowed"] != true {
			t.Fatal(o)
		}
	}
	for _, token := range []string{e.userToken, e.adminToken} {
		rec := testutil.DoJSON(t, e.router, "GET", fmt.Sprintf("/orders/%d", id), token, nil)
		var body map[string]any
		testutil.Decode(t, rec, &body)
		_, exists := body["paymentRetry"]
		if exists != (token == e.userToken) {
			t.Fatal(body)
		}
	}
	for _, o := range e.getList(t, "/orders", e.adminToken).Orders {
		if _, exists := o["paymentRetry"]; exists {
			t.Fatal(o)
		}
	}
	hours.err = errors.New("hours unavailable")
	for _, path := range []string{"/orders/my", fmt.Sprintf("/orders/%d", id)} {
		rec := testutil.DoJSON(t, e.router, "GET", path, e.userToken, nil)
		if rec.Code != 500 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	}
	empty, err := e.svc.customerOrders(context.Background(), nil)
	data, _ := json.Marshal(empty)
	if err != nil || string(data) != "[]" {
		t.Fatalf("%s %v", data, err)
	}
}

func TestPaymentCheckRetryEligibility(t *testing.T) {
	for _, tc := range []struct {
		name    string
		elapsed time.Duration
		fail    bool
		paid    bool
		code    string
	}{
		{"open", 0, false, false, "402"}, {"next day", 24 * time.Hour, false, false, "500"},
		{"closed", 10 * time.Hour, false, false, "500"}, {"paid after expiry", 24 * time.Hour, true, true, "200"},
		{"hours unavailable", 0, true, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			e.svc.clock = clock.Fixed(friday.Add(tc.elapsed))
			if tc.fail {
				e.svc.SetPaymentHours(fakeHours{err: errors.New("hours unavailable")})
			}
			if tc.paid {
				setOrderState(t, e.db, id, StatusPaid, "555")
			}
			cb := callback(id)
			cb.Command = "CHECK"
			got, err := e.svc.PaymentCheck(context.Background(), cb)
			if tc.code == "" {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || got.ResultCode != tc.code {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestPaymentRetryConfirmationRace(t *testing.T) {
	e, id := prepareRetry(t)
	e.payments.valid = true
	e.payments.form.Amount = "749.00"
	if _, err := e.svc.RetryPayment(context.Background(), id, e.userID); err != nil {
		t.Fatal(err)
	}
	cb := callback(id)
	cb.Amount = "749.00"
	for range 2 {
		if err := e.svc.PaymentConfirmed(context.Background(), cb); err != nil {
			t.Fatal(err)
		}
	}
	e.wantPaidOnce(t, id)
	_, err := e.svc.RetryPayment(context.Background(), id, e.userID)
	wantHTTPError(t, err, http.StatusConflict, "Этот заказ уже оплачен или недоступен для оплаты.")
}

func TestDelayedPaymentConfirmation(t *testing.T) {
	for _, state := range []string{StatusPending, StatusCanceled} {
		t.Run(state, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			id := seedPending(t, e)
			setOrderState(t, e.db, id, state, "")
			e.svc.clock = clock.Fixed(friday.Add(36 * time.Hour))
			e.svc.SetPaymentHours(nil)
			if err := e.svc.PaymentConfirmed(context.Background(), callback(id)); err != nil {
				t.Fatal(err)
			}
			if state == StatusPending {
				e.wantPaidOnce(t, id)
			} else {
				e.wantUntouched(t, id, state)
			}
		})
	}
}
