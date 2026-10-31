package order

import (
	"context"
	"errors"
	"math"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/promo"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func quoteCampaign(t *testing.T, e *svcEnv, in promo.Input) {
	t.Helper()
	if err := store.InTx(context.Background(), e.db, func(tx *sqlx.Tx) error { _, err := promo.NewStore(tx).Create(context.Background(), in); return err }); err != nil {
		t.Fatal(err)
	}
}
func baseCampaign() promo.Input {
	return promo.Input{Code: "SAVE10", Percentage: 10, Enabled: true, Target: "all"}
}
func TestQuote(t *testing.T) {
	e := newSvcEnv(t, friday)
	quoteCampaign(t, e, baseCampaign())
	if _, err := e.db.Exec(`UPDATE delivery_prices SET free_from_kopecks=60000 WHERE name='Сочи'`); err != nil {
		t.Fatal(err)
	}
	in := QuoteInput{Items: []ItemInput{{e.productA, 1}, {e.productB, 1}, {e.productB, 1}}, Fulfillment: "delivery", City: "Сочи", PromoCode: " save10 "}
	q, err := e.svc.Quote(context.Background(), in)
	if err != nil || q.Subtotal != 79900 || q.Discount != 7990 || q.DeliveryPrice != 0 || len(q.Items) != 2 || q.Items[1].Amount != 2 {
		t.Fatalf("%+v %v", q, err)
	}
	in.Items = []ItemInput{{e.productA, 1}, {e.productB, 1}}
	q, err = e.svc.Quote(context.Background(), in)
	if err != nil || q.Total != 62910 || q.DeliveryPrice != 0 {
		t.Fatalf("%+v %v", q, err)
	}
	if _, err = e.db.Exec(`UPDATE delivery_prices SET free_from_kopecks=65000 WHERE name='Сочи'`); err != nil {
		t.Fatal(err)
	}
	r := testutil.DoJSON(t, newRouter(e, defaultOpts()), "POST", "/orders/quote", "", in)
	if r.Code != 200 {
		t.Fatalf("%d %s", r.Code, r.Body)
	}
	testutil.Decode(t, r, &q)
	if q.DeliveryPrice != 15000 || q.PayableTotal != 77910 {
		t.Fatal(q)
	}
	if err = os.WriteFile("/tmp/order-quote-example.json", r.Body.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if r.Header().Get("Cache-Control") == "" {
		t.Fatal("missing private cache policy")
	}
	in.PromoCode = ""
	q, err = e.svc.Quote(context.Background(), in)
	if err != nil || q.Discount != 0 || q.EligibleSubtotal != 0 || q.PromoCode != nil || q.DeliveryPrice != 0 {
		t.Fatalf("%+v %v", q, err)
	}
	in.Fulfillment = "pickup"
	in.PromoCode = "SAVE10"
	q, err = e.svc.Quote(context.Background(), in)
	if err != nil || q.DeliveryPrice != 0 {
		t.Fatalf("%+v %v", q, err)
	}
	for _, table := range []string{"orders", "users", "addresses"} {
		if n := testutil.Count(t, e.db, "SELECT count(*) FROM "+table); n != 0 {
			t.Fatalf("quote wrote %s", table)
		}
	}
	if len(e.notifier.pending) != 0 || len(e.notifier.paid) != 0 {
		t.Fatal("quote notified")
	}
}
func TestQuoteCatalogChanges(t *testing.T) {
	e := newSvcEnv(t, friday)
	c := baseCampaign()
	c.Target = "selected"
	c.CategoryIDs = []int64{e.category}
	quoteCampaign(t, e, c)
	in := QuoteInput{Items: []ItemInput{{e.productA, 1}}, Fulfillment: "pickup", PromoCode: c.Code}
	if _, err := e.db.Exec(`UPDATE products SET price_kopecks=12345 WHERE id=$1`, e.productA); err != nil {
		t.Fatal(err)
	}
	q, err := e.svc.Quote(context.Background(), in)
	if err != nil || q.Items[0].OfferedPrice != 12345 || q.Discount != 1234 {
		t.Fatalf("%+v %v", q, err)
	}
	other := testutil.InsertCategory(t, e.db, testutil.Category{})
	if _, err = e.db.Exec(`UPDATE products SET category_id=$2 WHERE id=$1`, e.productA, other); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Quote(context.Background(), in)
	quoteField(t, err, "В корзине нет блюд для этого промокода.")
	p := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, PriceKopecks: 100})
	in.Items = []ItemInput{{p, 1}}
	if q, err = e.svc.Quote(context.Background(), in); err != nil || q.Discount != 10 {
		t.Fatalf("%+v %v", q, err)
	}
	if _, err = e.db.Exec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, p); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Quote(context.Background(), in)
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 503 {
		t.Fatal(err)
	}
}
func quoteField(t *testing.T, err error, want string) {
	t.Helper()
	var he *httpx.Error
	if !errors.As(err, &he) || he.Status != 400 || he.Fields["promoCode"] != want {
		t.Fatalf("want %q got %v", want, err)
	}
}
func TestQuoteErrors(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := QuoteInput{Items: []ItemInput{{e.productA, 1}}, Fulfillment: "pickup"}
	for _, code := range []string{"?", "MISSING"} {
		in.PromoCode = code
		_, err := e.svc.Quote(context.Background(), in)
		want := "Промокод недействителен."
		if code == "?" {
			want = "Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний."
		}
		quoteField(t, err, want)
	}
	c := baseCampaign()
	c.Enabled = false
	quoteCampaign(t, e, c)
	in.PromoCode = c.Code
	_, err := e.svc.Quote(context.Background(), in)
	quoteField(t, err, "Промокод недействителен.")
	if _, err = e.db.Exec(`UPDATE promo_codes SET enabled=true, minimum_eligible_subtotal_kopecks=999999`); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Quote(context.Background(), in)
	quoteField(t, err, "Недостаточная сумма подходящих блюд для промокода.")
	in.PromoCode = ""
	in.Fulfillment = "delivery"
	in.City = "unknown"
	_, err = e.svc.Quote(context.Background(), in)
	wantHTTPError(t, err, 400, "Указано не существующее имя населенного пункта.")
	in.Fulfillment = "pickup"
	in.Items = []ItemInput{{999999, 1}}
	_, err = e.svc.Quote(context.Background(), in)
	wantHTTPError(t, err, 400, "Продукт не найден.")
	router := newRouter(e, defaultOpts())
	badReq := httptest.NewRequest("POST", "/orders/quote", strings.NewReader(`{"items":`))
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != 400 {
		t.Fatal(badRec.Body)
	}
	for _, body := range []any{QuoteInput{}, QuoteInput{Items: []ItemInput{{e.productA, 0}}, Fulfillment: "pickup"}, QuoteInput{Items: []ItemInput{{e.productA, 1}}, Fulfillment: "delivery"}} {
		r := testutil.DoJSON(t, router, "POST", "/orders/quote", "", body)
		if r.Code != 400 {
			t.Fatalf("%d %s", r.Code, r.Body)
		}
	}
	r := testutil.DoJSON(t, router, "POST", "/orders/quote", "invalid", in)
	if r.Code != 401 {
		t.Fatalf("auth %d", r.Code)
	}
	in.Items = []ItemInput{{e.productA, math.MaxInt}, {e.productA, 1}}
	_, err = e.svc.Quote(context.Background(), in)
	wantHTTPError(t, err, 400, "Некорректное количество.")
	for _, table := range []string{"orders", "users"} {
		if n := testutil.Count(t, e.db, "SELECT count(*) FROM "+table); n != 0 {
			t.Fatal("quote wrote data")
		}
	}
	e.db.Close()
	in.Items = []ItemInput{{e.productA, 1}}
	if _, err = e.svc.Quote(context.Background(), in); err == nil {
		t.Fatal("closed DB accepted")
	}
}

func TestQuoteOverflow(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := QuoteInput{Items: []ItemInput{{e.productA, 2}}, Fulfillment: "pickup"}
	if _, err := e.db.Exec(`UPDATE products SET price_kopecks=$2 WHERE id=$1`, e.productA, int64(math.MaxInt64)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Quote(context.Background(), in); err == nil {
		t.Fatal("line overflow accepted")
	}
	in.Items = []ItemInput{{e.productA, 1}, {e.productB, 1}}
	if _, err := e.svc.Quote(context.Background(), in); err == nil {
		t.Fatal("subtotal overflow accepted")
	}
	in.Items = []ItemInput{{e.productA, 1}}
	in.Fulfillment = "delivery"
	in.City = "Сочи"
	if _, err := e.svc.Quote(context.Background(), in); err == nil {
		t.Fatal("payable overflow accepted")
	}
}
