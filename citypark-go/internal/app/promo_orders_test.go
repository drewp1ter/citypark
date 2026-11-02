package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/app"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/config"
	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/telegram"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const promoRMS = "01234567-89ab-4def-8123-456789abcdef"

func promoOrder() order.Order {
	code := "AUTUMN50"
	return order.Order{ID: 7, PromoCode: &code, Subtotal: 55001, Discount: 101, Total: 54900, DeliveryPrice: 10000, Customer: order.Customer{Name: "Тест", Phone: "+79990000000"}, Items: []order.Item{
		{ProductID: 1, IikoID: "p1", Amount: 3, OfferedPrice: 10000, Discount: 61, PromoEligible: true},
		{ProductID: 2, IikoID: "p2", Amount: 2, OfferedPrice: 10000, Discount: 40, PromoEligible: true},
		{ProductID: 3, IikoID: "p3", Amount: 1, OfferedPrice: 5000},
		{ProductID: 4, IikoID: "p4", Amount: 1, OfferedPrice: 1, PromoEligible: true},
	}}
}
func TestPromoIikoPayload(t *testing.T) {
	for _, courier := range []bool{false, true} {
		o := promoOrder()
		if courier {
			o.Address = &order.Address{City: "Белореченск", Address: "Тест, 1"}
		} else {
			o.DeliveryPrice = 0
		}
		s := iikoSettings()
		s.PromoDiscountTypeID = promoRMS
		d, err := app.DeliveryOrderForTest(o, s, "guid")
		if err != nil {
			t.Fatal(err)
		}
		if d.Payments[0].Sum != 549 || !d.Payments[0].IsProcessedExternally || len(d.Items) != 4 || d.DiscountsInfo == nil || len(d.DiscountsInfo.Discounts) != 1 {
			t.Fatal(d)
		}
		discount := d.DiscountsInfo.Discounts[0]
		if discount.Type != "RMS" || discount.DiscountTypeID != promoRMS || discount.Sum != 1.01 {
			t.Fatal(discount)
		}
		seen := map[string]bool{}
		want := []string{}
		for i, it := range d.Items {
			if !uuidV4.MatchString(it.PositionID) || seen[it.PositionID] {
				t.Fatal(it)
			}
			seen[it.PositionID] = true
			if it.Price != o.Items[i].OfferedPrice.Rubles() || it.Amount != float64(o.Items[i].Amount) {
				t.Fatal(it)
			}
			if o.Items[i].PromoEligible {
				want = append(want, it.PositionID)
			}
		}
		if !reflect.DeepEqual(want, discount.SelectivePositions) {
			t.Fatal(discount)
		}
		if courier && (d.DeliveryPoint == nil || !strings.Contains(d.Comment, "Доставка: 100 ₽ (оплачена)")) {
			t.Fatal(d)
		}
		o.Discount = 0
		o.Total = o.Subtotal
		d, err = app.DeliveryOrderForTest(o, s, "guid")
		if err != nil || d.DiscountsInfo != nil {
			t.Fatalf("zero: %v %v", d, err)
		}
		b, _ := json.Marshal(d)
		if strings.Contains(string(b), "discountsInfo") {
			t.Fatal(string(b))
		}
	}
}
func TestPromoIikoRetry(t *testing.T) {
	db := testutil.NewDB(t)
	saved := iikoFixture(t, db, "p1")
	o := promoOrder()
	o.ID = saved.ID
	var payloads []iiko.DeliveryOrder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/access_token" {
			io.WriteString(w, `{"token":"token"}`)
			return
		}
		if r.URL.Path != "/1/deliveries/create" {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Order iiko.DeliveryOrder `json:"order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		payloads = append(payloads, body.Order)
		if len(payloads) == 1 {
			db.MustExec(`UPDATE products SET price_kopecks=1`)
			w.WriteHeader(500)
			return
		}
		io.WriteString(w, `{"orderInfo":{"id":"iiko-order-1","creationStatus":"Success"}}`)
	}))
	defer server.Close()
	c := iiko.NewClient(server.URL, iiko.Credentials{APIKey: "key"}, "org", "", server.Client())
	s := iikoSettings()
	s.PromoDiscountTypeID = promoRMS
	n := app.NewIikoOrders(c, db, &fakeSender{}, alertChat, s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.OrderPaid(context.Background(), o)
	n.Wait()
	if len(payloads) != 2 || !reflect.DeepEqual(payloads[0], payloads[1]) {
		t.Fatal(payloads)
	}
	if savedIikoID(t, db, o.ID) != "iiko-order-1" {
		t.Fatal("result not saved")
	}
}
func TestPromoStaffMessage(t *testing.T) {
	var texts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		texts = append(texts, body.Text)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	client := telegram.NewClient(server.URL, "test", server.Client())
	n := app.NewTelegramNotifier(client, "chat", slog.New(slog.NewTextHandler(io.Discard, nil)))
	o := promoOrder()
	o.Address = &order.Address{City: "Белореченск", Address: "Тест, 1"}
	n.OrderPaid(context.Background(), o)
	n.Wait()
	if len(texts) != 2 {
		t.Fatal(texts)
	}
	for _, want := range []string{"AUTUMN50", "Скидка по промокоду: 1.01 ₽", "Итого к оплате: 649 ₽"} {
		if !strings.Contains(texts[0], want) {
			t.Fatalf("missing %q: %s", want, texts[0])
		}
	}
}
func TestPromoConfiguration(t *testing.T) {
	db := testutil.NewDB(t)
	// Exercise the missing-provider configuration guard with the independent global switch enabled.
	var originalSettings []byte
	if err := db.Get(&originalSettings, `SELECT data FROM site_settings WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`UPDATE site_settings SET data=jsonb_set(data, '{promoCodesEnabled}', 'true') WHERE id=1`)
	t.Cleanup(func() { db.MustExec(`UPDATE site_settings SET data=$1 WHERE id=1`, originalSettings) })
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.FixedZone("MSK", 10800))
	c, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(c, db, slog.New(slog.NewTextHandler(io.Discard, nil)), clock.Fixed(now))
	defer a.Hub.Close()
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	p := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, PriceKopecks: 10000})
	db.MustExec(`INSERT INTO promo_codes(code,percentage,enabled,target) VALUES('SAVE10',10,true,'all')`)
	body := map[string]any{"phone": "+79990000000", "name": "Тест", "items": []map[string]any{{"productId": p, "amount": 1}}, "promoCode": "SAVE10", "expectedPayableTotal": 90}
	r := testutil.DoJSON(t, a.Handler, "POST", "/api/v1/orders", "", body)
	if r.Code != 503 || !strings.Contains(r.Body.String(), "Заказы с промокодом временно недоступны.") {
		t.Fatalf("%d %s", r.Code, r.Body)
	}
	for _, table := range []string{"orders", "users"} {
		if testutil.Count(t, db, "SELECT count(*) FROM "+table) != 0 {
			t.Fatal("guard wrote " + table)
		}
	}
	// No-code stays usable while promo configuration is absent.
	delete(body, "promoCode")
	delete(body, "expectedPayableTotal")
	r = testutil.DoJSON(t, a.Handler, "POST", "/api/v1/orders", "", body)
	if r.Code != 201 {
		t.Fatalf("plain %d %s", r.Code, r.Body)
	}
	// Authenticate a fixture account and create a pending promo snapshot with its own scope.
	uid := testutil.InsertUser(t, db, "Аккаунт", "+79990000001", auth.RoleUser)
	tokens := auth.NewTokens(c.JWT.Secret, c.JWT.ExpiresIn, clock.Fixed(now))
	token, err := tokens.Issue(uid, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	code := "SAVE10"
	pct := 10
	var campaignID int64
	db.Get(&campaignID, `SELECT id FROM promo_codes WHERE code='SAVE10'`)
	name, phone := "Тест", "+79990000000"
	id, err := order.NewStore(db).Insert(context.Background(), order.InsertParams{CustomerID: uid, AccountID: &uid, ContactName: &name, ContactPhone: &phone, ItemsHash: order.ItemsHash([]order.ItemInput{{ProductID: p, Amount: 1}}), PromoCode: &code, PromoCodeID: &campaignID, PromoPercentage: &pct, Discount: 1000, Items: []order.InsertItem{{ProductID: p, Amount: 1, OfferedPrice: 10000, Discount: 1000, PromoEligible: true}}})
	if err != nil {
		t.Fatal(err)
	}
	body["promoCode"] = "SAVE10"
	body["expectedPayableTotal"] = 90
	db.MustExec(`UPDATE promo_codes SET enabled=false`)
	r = testutil.DoJSON(t, a.Handler, "POST", "/api/v1/orders", token, body)
	if r.Code != 201 {
		t.Fatalf("retry %d %s", r.Code, r.Body)
	}
	var res order.CreateResult
	testutil.Decode(t, r, &res)
	if res.OrderID != id {
		t.Fatal(res)
	}
	a.Notifier.Wait()
	a.IikoOrders.Wait()
}
func TestPromoIntegrationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	saved := iikoFixture(t, db, "p1")
	o := promoOrder()
	o.ID = saved.ID
	s := iikoSettings()
	creator, sender := runIikoOrders(t, db, s, o)
	if len(creator.calls()) != 0 || len(sender.calls()) != 1 {
		t.Fatal("missing RMS silently dispatched")
	}
	s.PromoDiscountTypeID = promoRMS
	creator, sender = runIikoOrders(t, db, s, o, errors.New("unavailable"))
	if len(creator.calls()) != 3 || len(sender.calls()) != 1 {
		t.Fatal("retry/alert behavior")
	}
	for _, d := range creator.calls() {
		if d.DiscountsInfo == nil || d.DiscountsInfo.Discounts[0].Sum != 1.01 || d.Payments[0].Sum != 549 {
			t.Fatal("undiscounted fallback")
		}
	}
	for i := range o.Items {
		o.Items[i].PromoEligible = false
	}
	creator, sender = runIikoOrders(t, db, s, o)
	if len(creator.calls()) != 0 || len(sender.calls()) != 1 {
		t.Fatal("missing eligibility accepted")
	}
	s.TerminalGroupID = ""
	creator, sender = runIikoOrders(t, db, s, o)
	if len(creator.calls()) != 0 || len(sender.calls()) != 1 {
		t.Fatal("missing transport config not alerted")
	}
}
