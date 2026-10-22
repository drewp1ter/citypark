package hotel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type pricesFixture struct {
	db     *sqlx.DB
	router http.Handler
	admin  string
	user   string
}

// newPricesFixture returns a router over the migrated database. NewDB does not truncate hotel_prices and
// breakfast_items (the migration seeds them), so the state found at the start is put back after the test.
func newPricesFixture(t *testing.T) pricesFixture {
	t.Helper()
	db := testutil.NewDB(t)
	before, err := NewStore(db).GetPrices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := NewStore(db).ReplacePrices(context.Background(), before.Fees, before.Breakfast); err != nil {
			t.Errorf("restore prices: %v", err)
		}
	})

	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	NewHandler(NewStore(db), "", tokens).Routes(r)
	return pricesFixture{db: db, router: r, admin: admin, user: user}
}

func (f pricesFixture) get(t *testing.T) Prices {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/hotel/prices", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body %s", rec.Code, rec.Body)
	}
	var p Prices
	testutil.Decode(t, rec, &p)
	return p
}

func TestPricesSeed(t *testing.T) {
	f := newPricesFixture(t)

	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/hotel/prices", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var raw map[string]any
	testutil.Decode(t, rec, &raw)
	if raw["breakfastDelivery"] != float64(100) || raw["lateCheckoutPerHour"] != float64(500) || raw["smokingFine"] != float64(2000) {
		t.Errorf("fees = %v, %v, %v, want 100, 500, 2000", raw["breakfastDelivery"], raw["lateCheckoutPerHour"], raw["smokingFine"])
	}
	if raw["updatedAt"] == "" || raw["updatedAt"] == nil {
		t.Error("updatedAt is empty")
	}
	var p Prices
	testutil.Decode(t, rec, &p)
	want := []BreakfastItem{
		{"Омлет, 130 г", "Омлет", 25000, false},
		{"Сырники", "", 30000, false},
		{"Блины с творогом", "", 20000, true},
		{"Круассан с сёмгой", "", 35000, false},
		{"Каша с фруктами", "", 8000, true},
	}
	if len(p.Breakfast) != len(want) {
		t.Fatalf("items = %d, want %d: %+v", len(p.Breakfast), len(want), p.Breakfast)
	}
	for i := range want {
		if p.Breakfast[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, p.Breakfast[i], want[i])
		}
	}
	if last := p.Breakfast[4]; !last.DesktopOnly || last.Price != money.FromRubles(80) {
		t.Errorf("last item = %+v, want desktopOnly with price 80", last)
	}
	t.Logf("GET body: %s", rec.Body)
}

func TestPutPrices(t *testing.T) {
	f := newPricesFixture(t)
	body := map[string]any{
		"breakfastDelivery": 150, "lateCheckoutPerHour": 600.5, "smokingFine": 3000,
		"breakfast": []map[string]any{
			{"name": "Яичница", "shortName": "Яйца", "price": 199.99, "desktopOnly": true},
			{"name": "Омлет", "price": 0},
		},
	}

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	t.Logf("PUT body: %s", rec.Body)
	var put Prices
	testutil.Decode(t, rec, &put)
	got := f.get(t)
	if got.BreakfastDelivery != 15000 || got.LateCheckoutPerHour != 60050 || got.SmokingFine != 300000 {
		t.Errorf("fees = %+v, want 15000, 60050, 300000 kopecks", got.Fees)
	}
	want := []BreakfastItem{{"Яичница", "Яйца", 19999, true}, {"Омлет", "", 0, false}}
	if len(got.Breakfast) != 2 || got.Breakfast[0] != want[0] || got.Breakfast[1] != want[1] {
		t.Errorf("items = %+v, want %+v", got.Breakfast, want)
	}
	if put.Fees != got.Fees || len(put.Breakfast) != 2 || put.Breakfast[0] != want[0] || !put.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("PUT response = %+v, want the stored state %+v", put, got)
	}
	if n := testutil.Count(t, f.db, `SELECT count(*) FROM hotel_prices`); n != 1 {
		t.Errorf("hotel_prices rows = %d, want 1", n)
	}
	if n := testutil.Count(t, f.db, `SELECT count(*) FROM breakfast_items WHERE sort IN (1, 2)`); n != 2 {
		t.Errorf("items with sort 1 and 2 = %d, want 2", n)
	}
}

func TestPutPricesEmptyBreakfast(t *testing.T) {
	f := newPricesFixture(t)
	body := map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3, "breakfast": []any{}}

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"breakfast":[]`) {
		t.Errorf("PUT body = %s, want breakfast []", rec.Body)
	}
	get := testutil.DoJSON(t, f.router, http.MethodGet, "/hotel/prices", "", nil)
	if !strings.Contains(get.Body.String(), `"breakfast":[]`) {
		t.Errorf("GET body = %s, want breakfast []", get.Body)
	}
}

func TestPutPricesOmittedBreakfast(t *testing.T) {
	f := newPricesFixture(t)
	body := map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3}

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if got := f.get(t); len(got.Breakfast) != 0 {
		t.Errorf("items = %+v, want none", got.Breakfast)
	}
}

func TestPutPricesInvalid(t *testing.T) {
	validFees := func() map[string]any {
		return map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3}
	}
	with := func(mut func(m map[string]any)) map[string]any {
		m := validFees()
		mut(m)
		return m
	}
	manyItems := make([]map[string]any, 31)
	for i := range manyItems {
		manyItems[i] = map[string]any{"name": "x", "price": 1}
	}
	tests := []struct {
		name  string
		body  map[string]any
		field string
		msg   string
	}{
		{"negative fee", with(func(m map[string]any) { m["smokingFine"] = -1 }), "smokingFine", "значение должно быть не меньше 0"},
		{"missing fee", map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2}, "smokingFine", "поле не должно быть пустым"},
		{"fee too big", with(func(m map[string]any) { m["breakfastDelivery"] = 100001 }), "breakfastDelivery", "значение должно быть не больше 100000"},
		{"fee three decimals", with(func(m map[string]any) { m["lateCheckoutPerHour"] = 1.005 }), "lateCheckoutPerHour", "не более 2 знаков после запятой"},
		{"empty name", with(func(m map[string]any) { m["breakfast"] = []map[string]any{{"name": "", "price": 1}} }), "breakfast[0].name", "поле не должно быть пустым"},
		{"blank name", with(func(m map[string]any) { m["breakfast"] = []map[string]any{{"name": "  ", "price": 1}} }), "breakfast[0].name", "поле не должно быть пустым"},
		{"long name", with(func(m map[string]any) {
			m["breakfast"] = []map[string]any{{"name": strings.Repeat("я", 81), "price": 1}}
		}), "breakfast[0].name", "значение должно быть не больше 80"},
		{"long short name", with(func(m map[string]any) {
			m["breakfast"] = []map[string]any{{"name": "a", "shortName": strings.Repeat("я", 41), "price": 1}}
		}), "breakfast[0].shortName", "значение должно быть не больше 40"},
		{"missing price", with(func(m map[string]any) { m["breakfast"] = []map[string]any{{"name": "a"}} }), "breakfast[0].price", "поле не должно быть пустым"},
		{"price three decimals", with(func(m map[string]any) {
			m["breakfast"] = []map[string]any{{"name": "a", "price": 1}, {"name": "b", "price": 10.555}}
		}), "breakfast[1].price", "не более 2 знаков после запятой"},
		{"31 items", with(func(m map[string]any) { m["breakfast"] = manyItems }), "breakfast", "значение должно быть не больше 30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPricesFixture(t)
			before := f.get(t)

			rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
			}
			var e errorBody
			testutil.Decode(t, rec, &e)
			if e.Message != "Ошибка валидации, пожалуйста проверьте введенные вами данные" || e.Errors[tt.field] != tt.msg {
				t.Errorf("body = %+v, want errors[%q] = %q", e, tt.field, tt.msg)
			}
			after := f.get(t)
			if after.Fees != before.Fees || len(after.Breakfast) != len(before.Breakfast) || !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Errorf("state changed: before %+v, after %+v", before, after)
			}
		})
	}
}

func TestPutPricesInvalidJSON(t *testing.T) {
	f := newPricesFixture(t)

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, "not an object")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400: %s", rec.Code, rec.Body)
	}
}

func TestPutPricesTrimsNames(t *testing.T) {
	f := newPricesFixture(t)
	body := map[string]any{
		"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3,
		"breakfast": []map[string]any{{"name": "  Блин  ", "shortName": " Б ", "price": 5}},
	}

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	got := f.get(t)
	if len(got.Breakfast) != 1 || got.Breakfast[0].Name != "Блин" || got.Breakfast[0].ShortName != "Б" {
		t.Errorf("items = %+v, want name %q and shortName %q", got.Breakfast, "Блин", "Б")
	}
}

func TestPutPricesAuth(t *testing.T) {
	body := map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3}
	tests := []struct {
		name string
		user func(f pricesFixture) string
		want int
	}{
		{"no token", func(pricesFixture) string { return "" }, http.StatusUnauthorized},
		{"user role", func(f pricesFixture) string { return f.user }, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPricesFixture(t)
			before := f.get(t)

			rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", tt.user(f), body)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
			if after := f.get(t); after.Fees != before.Fees {
				t.Errorf("fees changed: %+v -> %+v", before.Fees, after.Fees)
			}
		})
	}
}

func TestGetPricesMissingRow(t *testing.T) {
	f := newPricesFixture(t)
	if _, err := f.db.Exec(`DELETE FROM hotel_prices`); err != nil {
		t.Fatal(err)
	}

	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/hotel/prices", "", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", rec.Code, rec.Body)
	}
	var e errorBody
	testutil.Decode(t, rec, &e)
	if e.Message != "Не найдено" {
		t.Errorf("message = %q, want %q", e.Message, "Не найдено")
	}

	// A PUT creates the row again.
	body := map[string]any{"breakfastDelivery": 1, "lateCheckoutPerHour": 2, "smokingFine": 3}
	if rec := testutil.DoJSON(t, f.router, http.MethodPut, "/hotel/prices", f.admin, body); rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if got := f.get(t); got.SmokingFine != 300 {
		t.Errorf("smokingFine = %d, want 300", got.SmokingFine)
	}
}

func TestReplacePricesInTx(t *testing.T) {
	f := newPricesFixture(t)
	ctx := context.Background()
	before := f.get(t)

	tx, err := f.db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	in, err := NewStore(tx).ReplacePrices(ctx, Fees{1, 2, 3}, []BreakfastItem{{Name: "A", Price: 4}})
	if err != nil {
		t.Fatal(err)
	}
	if in.Fees != (Fees{1, 2, 3}) || len(in.Breakfast) != 1 {
		t.Errorf("inside the transaction: %+v", in)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	if after := f.get(t); after.Fees != before.Fees || len(after.Breakfast) != len(before.Breakfast) {
		t.Errorf("rollback left %+v, want %+v", after, before)
	}
}

type onlyDBTX struct{ store.DBTX }

func TestReplacePricesUnsupportedDB(t *testing.T) {
	db := testutil.NewDB(t)

	_, err := NewStore(onlyDBTX{db}).ReplacePrices(context.Background(), Fees{}, nil)

	if err == nil || !strings.Contains(err.Error(), "needs *sqlx.DB or *sqlx.Tx") {
		t.Errorf("err = %v, want the unsupported-type error", err)
	}
}

func TestReplacePricesRollsBackOnError(t *testing.T) {
	f := newPricesFixture(t)
	before := f.get(t)

	// A negative price violates the CHECK on the second step, after the fees were already updated.
	_, err := NewStore(f.db).ReplacePrices(context.Background(), Fees{1, 2, 3}, []BreakfastItem{{Name: "A", Price: -1}})

	if !store.IsCheckViolation(err) {
		t.Fatalf("err = %v, want a check violation", err)
	}
	if after := f.get(t); after.Fees != before.Fees || len(after.Breakfast) != len(before.Breakfast) {
		t.Errorf("state changed: before %+v, after %+v", before, after)
	}
}

func TestHasKopecksPrecision(t *testing.T) {
	tests := []struct {
		in   float64
		want bool
	}{
		{0, true}, {10, true}, {10.5, true}, {199.99, true}, {0.1 + 0.2, true},
		{10.555, false}, {1.005, false}, {0.001, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.in), func(t *testing.T) {
			if got := hasKopecksPrecision(tt.in); got != tt.want {
				t.Errorf("hasKopecksPrecision(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPricesJSONShape(t *testing.T) {
	p := Prices{Fees: Fees{10000, 50050, 200000}, Breakfast: []BreakfastItem{{"A", "", 25000, true}}}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"breakfastDelivery":100,"lateCheckoutPerHour":500.5,"smokingFine":2000,` +
		`"breakfast":[{"name":"A","shortName":"","price":250,"desktopOnly":true}],"updatedAt":"0001-01-01T00:00:00Z"}`
	if string(b) != want {
		t.Errorf("json = %s, want %s", b, want)
	}
}
