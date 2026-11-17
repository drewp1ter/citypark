package order

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateHiddenProduct(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "duplicate"}[duplicate], func(t *testing.T) {
			e := newSvcEnv(t, friday)
			if duplicate {
				if _, err := e.svc.Create(context.Background(), e.input(false)); err != nil {
					t.Fatal(err)
				}
			}
			e.db.MustExec(`UPDATE products SET is_hidden=true WHERE id=$1`, e.productA)
			orders, payments := e.count(t, "SELECT count(*) FROM orders"), len(e.payments.requests)
			_, err := e.svc.Create(context.Background(), e.input(false))
			wantHTTPError(t, err, http.StatusServiceUnavailable, "Заказ блюда «Шашлык» временно недоступен.")
			if e.count(t, "SELECT count(*) FROM orders") != orders || len(e.payments.requests) != payments {
				t.Fatal("hidden dish changed orders or payments")
			}
		})
	}
}

func TestQuoteHiddenProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	e.db.MustExec(`UPDATE products SET is_hidden=true WHERE id=$1`, e.productA)
	in := QuoteInput{Items: []ItemInput{{e.productA, 1}}, Fulfillment: "pickup"}
	_, err := e.svc.Quote(context.Background(), in)
	wantHTTPError(t, err, http.StatusServiceUnavailable, "Заказ блюда «Шашлык» временно недоступен.")
	if e.count(t, "SELECT count(*) FROM orders") != 0 || len(e.payments.requests) != 0 {
		t.Fatal("quote created orders or payments")
	}
}

func TestAddHiddenItem(t *testing.T) {
	e := newAdminEnv(t)
	for _, tt := range []struct {
		id   int64
		name string
	}{{e.productA, "Шашлык"}, {e.productB, "Лаваш"}} {
		e.db.MustExec(`UPDATE products SET is_hidden=true WHERE id=$1`, tt.id)
		code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": tt.id, "amount": 3})
		if code != http.StatusServiceUnavailable || body["message"] != "Заказ блюда «"+tt.name+"» временно недоступен." {
			t.Fatalf("hidden item: %d %v", code, body)
		}
	}
	o := e.order(t, e.orderID)
	if len(o.Items) != 1 || o.Items[0].Amount != 1 || o.Items[0].OfferedPrice != 59900 {
		t.Fatalf("hidden item changed historical order: %+v", o.Items)
	}
}
