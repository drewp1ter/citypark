package order

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const (
	hoursStart = "9:00"
	hoursEnd   = "22:00"
)

var (
	nightTime = time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC)
	saturday  = time.Date(2024, 5, 25, 15, 0, 0, 0, time.UTC)
)

func newRouter(e *svcEnv, opts HandlerOptions) http.Handler {
	r := chi.NewRouter()
	NewHandler(e.svc, nil, opts).Routes(r)
	return r
}

func defaultOpts() HandlerOptions {
	return HandlerOptions{WorkingHoursStart: hoursStart, WorkingHoursEnd: hoursEnd, DefaultPageSize: 20}
}

func orderBody(e *svcEnv, paymentType string, withAddress bool) map[string]any {
	body := map[string]any{
		"phone":       "+79991234567",
		"name":        "Иван",
		"comment":     "Без лука",
		"paymentType": paymentType,
		"items":       []map[string]any{{"productId": e.productA, "amount": 2}},
	}
	if withAddress {
		body["address"] = map[string]any{"city": "Сочи", "address": "ул. Ленина, 1"}
	}
	return body
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder, status int) httpx.Error {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body.String())
	}
	var he httpx.Error
	testutil.Decode(t, rec, &he)
	return he
}

func wantKeys(t *testing.T, got map[string]any, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("body keys = %v, want exactly %v", got, want)
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("body %v has no key %q", got, k)
		}
	}
}

func wantFieldErrors(t *testing.T, he httpx.Error, keys ...string) {
	t.Helper()
	if len(he.Fields) != len(keys) {
		t.Errorf("errors = %v, want keys %v", he.Fields, keys)
	}
	for _, k := range keys {
		if he.Fields[k] == "" {
			t.Errorf("errors = %v, missing key %q", he.Fields, k)
		}
	}
}

func TestCreateOrder(t *testing.T) {
	e := newSvcEnv(t, friday)
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, true))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	testutil.Decode(t, rec, &got)
	wantKeys(t, got, "orderId", "redirectUrl")
	if got["redirectUrl"] != cashURL {
		t.Errorf("redirectUrl = %v, want %q", got["redirectUrl"], cashURL)
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 1 {
		t.Errorf("orders = %d, want 1", n)
	}
}

func TestCreateOrderPickup(t *testing.T) {
	e := newSvcEnv(t, friday)
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, false))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body.String())
	}
	if n := e.count(t, `SELECT count(*) FROM orders WHERE address_id IS NULL`); n != 1 {
		t.Errorf("pickup orders = %d, want 1", n)
	}
}

func TestCreateOrderNight(t *testing.T) {
	e := newSvcEnv(t, nightTime)
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, true))
	want := `{"statusCode":503,"message":"Извините, но мы не можем принять ваш заказ.\nЗаказы принимаются с 9:00 до 22:00"}`
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateOrderNoWorkingHours(t *testing.T) {
	e := newSvcEnv(t, nightTime)
	opts := defaultOpts()
	opts.WorkingHoursStart = ""
	rec := testutil.DoJSON(t, newRouter(e, opts), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, true))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateOrderCategoryWeekend(t *testing.T) {
	e := newSvcEnv(t, saturday)
	if _, err := e.db.Exec(`UPDATE categories SET sale_days = '1111100' WHERE id = $1`, e.category); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, false))
	he := decodeError(t, rec, http.StatusServiceUnavailable)
	if !strings.HasPrefix(he.Message, "Возможность заказа") {
		t.Errorf("message = %q, want prefix %q", he.Message, "Возможность заказа")
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateOrderProductWeekend(t *testing.T) {
	e := newSvcEnv(t, saturday)
	if _, err := e.db.Exec(`UPDATE products SET sale_days = '1111100' WHERE id = $1`, e.productA); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentCash, false))
	he := decodeError(t, rec, http.StatusServiceUnavailable)
	if !strings.HasPrefix(he.Message, "Возможность заказа") {
		t.Errorf("message = %q, want prefix %q", he.Message, "Возможность заказа")
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateOrderBadAmount(t *testing.T) {
	e := newSvcEnv(t, friday)
	body := orderBody(e, PaymentCash, false)
	body["items"] = []map[string]any{{"productId": e.productA, "amount": 0}}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", body)
	he := decodeError(t, rec, http.StatusBadRequest)
	wantFieldErrors(t, he, "items[0].amount")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateOrderBadProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	body := orderBody(e, PaymentCash, false)
	body["items"] = []map[string]any{{"productId": 999, "amount": 1}}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", body)
	want := `{"statusCode":400,"message":"Продукт не найден."}`
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestCreateOrderValidation(t *testing.T) {
	e := newSvcEnv(t, friday)
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", map[string]any{})
	he := decodeError(t, rec, http.StatusBadRequest)
	wantFieldErrors(t, he, "phone", "name", "paymentType", "items")
}

func TestCreateOrderBadPaymentType(t *testing.T) {
	e := newSvcEnv(t, friday)
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, "card", false))
	he := decodeError(t, rec, http.StatusBadRequest)
	wantFieldErrors(t, he, "paymentType")
}

func TestCreateOrderBadAddress(t *testing.T) {
	e := newSvcEnv(t, friday)
	body := orderBody(e, PaymentCash, false)
	body["address"] = map[string]any{"city": ""}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", body)
	he := decodeError(t, rec, http.StatusBadRequest)
	wantFieldErrors(t, he, "address.city", "address.address")
}

func TestCreateOrderBadJSON(t *testing.T) {
	e := newSvcEnv(t, friday)
	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	newRouter(e, defaultOpts()).ServeHTTP(rec, req)
	he := decodeError(t, rec, http.StatusBadRequest)
	if he.Message != "Некорректный JSON" {
		t.Errorf("message = %q, want %q", he.Message, "Некорректный JSON")
	}
}

func TestCreateOrderOnline(t *testing.T) {
	e := newSvcEnv(t, friday)
	e.payments.form = PaymentForm{Amount: "790.00", Signature: "sig"}
	rec := testutil.DoJSON(t, newRouter(e, defaultOpts()), http.MethodPost, "/orders", "", orderBody(e, PaymentOnline, true))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	testutil.Decode(t, rec, &got)
	wantKeys(t, got, "orderId", "amount", "mntSignature")
	if got["amount"] != "790.00" || got["mntSignature"] != "sig" {
		t.Errorf("amount/mntSignature = %v / %v, want 790.00 / sig", got["amount"], got["mntSignature"])
	}
}

func TestWorkingHoursBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		hour, min  int
		want       int
	}{
		{"before opening", hoursStart, hoursEnd, 8, 59, http.StatusServiceUnavailable},
		{"at opening", hoursStart, hoursEnd, 9, 0, http.StatusOK},
		{"before closing", hoursStart, hoursEnd, 21, 59, http.StatusOK},
		{"at closing", hoursStart, hoursEnd, 22, 0, http.StatusServiceUnavailable},
		{"unparsable start", "abc", hoursEnd, 5, 0, http.StatusOK},
		{"unparsable end", hoursStart, "abc", 5, 0, http.StatusOK},
		{"empty both", "", "", 5, 0, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2024, 5, 24, tt.hour, tt.min, 0, 0, time.UTC)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			h := WorkingHours(clock.Fixed(now), tt.start, tt.end)(next)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/orders", nil))
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
