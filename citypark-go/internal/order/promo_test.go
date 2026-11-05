package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func promoInput(t *testing.T, e *svcEnv) CreateInput {
	t.Helper()
	in := e.input(true)
	in.Items = []ItemInput{{e.productA, 1}, {e.productB, 1}}
	in.PromoCode = " save10 "
	q, err := e.svc.Quote(context.Background(), QuoteInput{Items: in.Items, Fulfillment: "delivery", City: in.Address.City, PromoCode: in.PromoCode})
	if err != nil {
		t.Fatal(err)
	}
	in.ExpectedPayableTotal = &q.PayableTotal
	return in
}
func TestPromoOrderSnapshot(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, res.OrderID)
	if o.Subtotal != 69900 || o.Discount != 6990 || o.Total != 62910 || o.DeliveryPrice != 15000 || o.PromoCode == nil || *o.PromoCode != "SAVE10" || *o.PromoPercentage != 10 || o.Items[0].OfferedPrice != 59900 || o.Items[0].Discount != 5990 {
		t.Fatalf("snapshot %+v", o)
	}
	if len(e.payments.requests) != 1 || e.payments.requests[0].Amount != 77910 {
		t.Fatal(e.payments.requests)
	}
	data, _ := json.Marshal(o)
	if err = os.WriteFile("/tmp/promo-order-example.json", data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.db.Exec(`UPDATE promo_codes SET percentage=90, enabled=false, updated_at=now()`); err != nil {
		t.Fatal(err)
	}
	if _, err = e.db.Exec(`UPDATE products SET price_kopecks=99999`); err != nil {
		t.Fatal(err)
	}
	got := e.order(t, o.ID)
	if !reflect.DeepEqual(o, got) {
		t.Fatalf("snapshot changed: %+v", got)
	}
	list, _, err := NewStore(e.db).List(context.Background(), ListFilter{}, 10, 0)
	if err != nil || len(list) != 1 || list[0].Discount != o.Discount {
		t.Fatalf("list %v %v", list, err)
	}
}
func TestPromoEligibilitySnapshot(t *testing.T) {
	e := newSvcEnv(t, friday)
	other := testutil.InsertCategory(t, e.db, testutil.Category{})
	tiny := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, PriceKopecks: 1})
	excluded := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: other, PriceKopecks: 100})
	c := baseCampaign()
	c.Target = "selected"
	c.ProductIDs = []int64{e.productA}
	c.CategoryIDs = []int64{e.category}
	cap := money.Kopecks(1)
	c.MaximumDiscount = &cap
	quoteCampaign(t, e, c)
	in := e.input(false)
	in.Items = []ItemInput{{e.productA, 1}, {tiny, 1}, {excluded, 1}}
	in.PromoCode = c.Code
	expected := money.Kopecks(60000)
	in.ExpectedPayableTotal = &expected
	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, res.OrderID)
	for _, it := range o.Items {
		if it.ProductID == excluded {
			if it.PromoEligible || it.Discount != 0 {
				t.Fatal(it)
			}
		} else if !it.PromoEligible {
			t.Fatal(it)
		}
		if it.ProductID == tiny && it.Discount != 0 {
			t.Fatal(it)
		}
	}
	if o.Discount != 1 {
		t.Fatal(o)
	}
	if _, err = e.db.Exec(`DELETE FROM promo_code_products; DELETE FROM promo_code_categories`); err != nil {
		t.Fatal(err)
	}
	got := e.order(t, o.ID)
	if !reflect.DeepEqual(got, o) {
		t.Fatal("eligibility changed")
	}
}
func TestPromoOrderDuplicate(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	first, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.db.Exec(`UPDATE promo_codes SET enabled=false, ends_at='2000-01-01', updated_at=now()`); err != nil {
		t.Fatal(err)
	}
	if _, err = e.db.Exec(`UPDATE delivery_prices SET price_kopecks=99999`); err != nil {
		t.Fatal(err)
	}
	again, err := e.svc.Create(context.Background(), in)
	if err != nil || again.OrderID != first.OrderID {
		t.Fatalf("retry %v %v", again, err)
	}
	wrong := money.Kopecks(1)
	in.ExpectedPayableTotal = &wrong
	_, err = e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, 409, "Сумма заказа изменилась. Проверьте итог и повторите оформление.")
	in.PromoCode = ""
	in.ExpectedPayableTotal = nil
	plain, err := e.svc.Create(context.Background(), in)
	if err != nil || plain.OrderID == first.OrderID {
		t.Fatalf("plain %v %v", plain, err)
	}
	c := baseCampaign()
	c.Code = "OTHER"
	quoteCampaign(t, e, c)
	in.PromoCode = c.Code
	in.ExpectedPayableTotal = moneyPtr(162909)
	other, err := e.svc.Create(context.Background(), in)
	if err != nil || other.OrderID == first.OrderID || other.OrderID == plain.OrderID {
		t.Fatalf("other %v %v", other, err)
	}
}
func moneyPtr(v money.Kopecks) *money.Kopecks { return &v }
func TestPromoExpectedTotal(t *testing.T) {
	for _, kind := range []string{"missing", "negative", "price", "delivery", "discount", "valid"} {
		t.Run(kind, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			quoteCampaign(t, e, baseCampaign())
			in := promoInput(t, e)
			switch kind {
			case "missing":
				in.ExpectedPayableTotal = nil
			case "negative":
				in.ExpectedPayableTotal = moneyPtr(-1)
			case "price":
				e.db.MustExec(`UPDATE products SET price_kopecks=price_kopecks+100`)
			case "delivery":
				e.db.MustExec(`UPDATE delivery_prices SET price_kopecks=price_kopecks+100`)
			case "discount":
				e.db.MustExec(`UPDATE promo_codes SET percentage=20`)
			}
			_, err := e.svc.Create(context.Background(), in)
			if kind == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if kind == "missing" || kind == "negative" {
				var he *httpx.Error
				if !errors.As(err, &he) || he.Status != 400 || he.Fields["expectedPayableTotal"] != "Подтвердите итоговую сумму заказа." {
					t.Fatal(err)
				}
			} else {
				wantHTTPError(t, err, 409, "Сумма заказа изменилась. Проверьте итог и повторите оформление.")
			}
			for _, table := range []string{"orders", "users", "addresses"} {
				if testutil.Count(t, e.db, "SELECT count(*) FROM "+table) != 0 {
					t.Fatal("failed order wrote " + table)
				}
			}
		})
	}
}
func TestPromoPayment(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	e.db.MustExec(`UPDATE site_settings SET data=jsonb_set(data::jsonb, '{promoCodesEnabled}', 'false') WHERE id=1`)
	if retry, err := e.svc.Create(context.Background(), in); err != nil || retry.OrderID != res.OrderID {
		t.Fatalf("retry %v %v", retry, err)
	}
	e.payments.valid = true
	e.payments.form.Amount = "779.10"
	cb := e.callback(t, res.OrderID)
	cb.Amount = "849.00"
	wantHTTPError(t, e.svc.PaymentConfirmed(context.Background(), cb), 400, "Неверная сумма")
	cb.Amount = "779.10"
	if err = e.svc.PaymentConfirmed(context.Background(), cb); err != nil {
		t.Fatal(err)
	}
	if err = e.svc.PaymentConfirmed(context.Background(), cb); err != nil {
		t.Fatal(err)
	}
	e.wantPaidOnce(t, res.OrderID)
	check, err := e.svc.PaymentCheck(context.Background(), cb)
	if err != nil || check.Response.Amount != "779.10" || check.Response.ResultCode != "200" {
		t.Fatalf("check %v %v", check, err)
	}
}
func TestPromoOrderLegacy(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(false))
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, res.OrderID)
	if o.PromoCode != nil || o.Discount != 0 || o.Subtotal != o.Total || o.Items[0].PromoEligible {
		t.Fatal(o)
	}
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	res, err = e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	e.db.MustExec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, e.productA)
	_, err = e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, 503, "Заказ блюда «Шашлык» временно недоступен.")
}
func TestPromoOrderMutation(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	res, err := e.svc.Create(context.Background(), promoInput(t, e))
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, res.OrderID)
	tokens := auth.NewTokens("secret", time.Hour, e.svc.clock)
	token, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	router := newRouter(e, defaultOpts())
	for _, route := range []struct {
		method, path string
		body         any
	}{{"POST", fmt.Sprintf("/orders/%d/items", o.ID), ItemInput{ProductID: e.productB, Amount: 2}}, {"DELETE", fmt.Sprintf("/orders/%d/items/%d", o.ID, e.productA), nil}} {
		r := testutil.DoJSON(t, router, route.method, route.path, token, route.body)
		if r.Code != 409 {
			t.Fatalf("mutation %d %s", r.Code, r.Body)
		}
	}
	s := NewStore(e.db)
	for _, err := range []error{s.UpsertItem(context.Background(), o.ID, InsertItem{ProductID: e.productB, Amount: 5, OfferedPrice: 1}), s.RemoveItem(context.Background(), o.ID, e.productA)} {
		wantHTTPError(t, err, 409, "Нельзя изменять состав заказа с промокодом.")
	}
	if got := e.order(t, o.ID); !reflect.DeepEqual(got, o) {
		t.Fatal("mutated snapshot")
	}
}
func TestPromoOrderConcurrency(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	tx, err := e.db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var id int64
	if err = tx.Get(&id, `SELECT id FROM promo_codes FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE promo_codes SET percentage=20, updated_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, e := e.svc.Create(context.Background(), in); done <- e }()
	// The writer holds its row lock while replacing selections; creation must wait.
	time.Sleep(20 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("creation bypassed campaign lock: %v", err)
	default:
	}
	if _, err = tx.Exec(`UPDATE promo_codes SET target='selected', updated_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO promo_code_products(promo_code_id,product_id) VALUES($1,$2)`, id, e.productB); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		wantHTTPError(t, err, 409, "Сумма заказа изменилась. Проверьте итог и повторите оформление.")
	case <-time.After(5 * time.Second):
		t.Fatal("creation did not finish")
	}
	if testutil.Count(t, e.db, `SELECT count(*) FROM orders`) != 0 {
		t.Fatal("mixed campaign snapshot accepted")
	}
	in.ExpectedPayableTotal = moneyPtr(82900)
	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, res.OrderID)
	if o.Discount != 2000 || o.Items[0].PromoEligible || !o.Items[1].PromoEligible {
		t.Fatal(fmt.Sprint(o))
	}
}

func TestPromoExpectedHTTP(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	router := newRouter(e, defaultOpts())
	for _, value := range []any{nil, -1, "bad"} {
		body := orderBody(e, false)
		body["promoCode"] = "SAVE10"
		body["expectedPayableTotal"] = value
		r := testutil.DoJSON(t, router, "POST", "/orders", "", body)
		var he httpx.Error
		testutil.Decode(t, r, &he)
		if r.Code != 400 || he.Fields["expectedPayableTotal"] != "Подтвердите итоговую сумму заказа." {
			t.Fatalf("%d %s", r.Code, r.Body)
		}
	}
	if testutil.Count(t, e.db, `SELECT count(*) FROM orders`) != 0 {
		t.Fatal("invalid confirmation wrote order")
	}
}

func TestPromoRollbackProtection(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	res, err := e.svc.Create(context.Background(), promoInput(t, e))
	if err != nil {
		t.Fatal(err)
	}
	e.db.MustExec(`UPDATE orders SET status='paid',payment_id='555' WHERE id=$1`, res.OrderID)
	migration, err := os.ReadFile("../../migrations/00030_order_promos.sql")
	if err != nil {
		t.Fatal(err)
	}
	down := strings.SplitN(string(migration), "-- +goose Down", 2)[1]
	tx, err := e.db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(down); err == nil || !strings.Contains(err.Error(), "cannot discard order promo snapshots") {
		t.Fatalf("rollback protection: %v", err)
	}
}

func TestGlobalPromoSwitch(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	in := promoInput(t, e)
	e.db.MustExec(`UPDATE site_settings SET data=jsonb_set(data::jsonb, '{promoCodesEnabled}', 'false') WHERE id=1`)
	quote := QuoteInput{Items: in.Items, Fulfillment: "pickup", PromoCode: in.PromoCode}
	router := newRouter(e, defaultOpts())
	rec := testutil.DoJSON(t, router, "POST", "/orders/quote", "", quote)
	var response struct {
		Errors map[string]string `json:"errors"`
	}
	testutil.Decode(t, rec, &response)
	if rec.Code != 400 || response.Errors["promoCode"] != "Промокоды временно отключены." {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	body := orderBody(e, true)
	body["promoCode"] = "SAVE10"
	body["expectedPayableTotal"] = 779.10
	rec = testutil.DoJSON(t, router, "POST", "/orders", "", body)
	testutil.Decode(t, rec, &response)
	if rec.Code != 400 || response.Errors["promoCode"] != "Промокоды временно отключены." {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	e.svc.ConfigurePromo("")
	_, err := e.svc.Create(context.Background(), in)
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 400 || he.Fields["promoCode"] != "Промокоды временно отключены." {
		t.Fatal(err)
	}
	for _, table := range []string{"orders", "users", "addresses"} {
		if testutil.Count(t, e.db, "SELECT count(*) FROM "+table) != 0 {
			t.Fatal(table)
		}
	}
	quote.PromoCode = ""
	if _, err := e.svc.Quote(context.Background(), quote); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Create(context.Background(), e.input(false)); err != nil {
		t.Fatal(err)
	}
	e.db.MustExec(`DELETE FROM site_settings WHERE id=1`)
	quote.PromoCode = "SAVE10"
	if _, err := e.svc.Quote(context.Background(), quote); err == nil {
		t.Fatal("missing settings permitted quote")
	}
	if _, err := e.svc.Create(context.Background(), in); err == nil {
		t.Fatal("missing settings permitted order")
	}
	e.db.MustExec(`INSERT INTO site_settings (id, data) VALUES (1, '[]')`)
	if _, err := e.svc.Quote(context.Background(), quote); err == nil {
		t.Fatal("unreadable settings permitted quote")
	}
	if _, err := e.svc.Create(context.Background(), in); err == nil {
		t.Fatal("unreadable settings permitted order")
	}
}
