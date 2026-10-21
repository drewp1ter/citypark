package deliveryprice_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type env struct {
	db     *sqlx.DB
	router http.Handler
	admin  string
	user   string
}

func newEnv(t *testing.T) env {
	t.Helper()
	db := testutil.NewDB(t)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	deliveryprice.NewHandler(db, tokens).Routes(r)

	adminID := testutil.InsertUser(t, db, "admin", "+79000000001", auth.RoleAdmin)
	userID := testutil.InsertUser(t, db, "user", "+79000000002", auth.RoleUser)
	admin, err := tokens.Issue(adminID, auth.RoleAdmin)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}
	user, err := tokens.Issue(userID, auth.RoleUser)
	if err != nil {
		t.Fatalf("issue user token: %v", err)
	}
	return env{db: db, router: r, admin: admin, user: user}
}

func (e env) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.DoJSON(t, e.router, method, path, token, body)
}

func (e env) count(t *testing.T) int {
	t.Helper()
	return testutil.Count(t, e.db, `SELECT count(*) FROM delivery_prices`)
}

type errorBody struct {
	StatusCode int               `json:"statusCode"`
	Message    string            `json:"message"`
	Errors     map[string]string `json:"errors"`
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d, body %s", rec.Code, want, rec.Body.String())
	}
}

func TestList(t *testing.T) {
	e := newEnv(t)
	testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)
	testutil.InsertDeliveryPrice(t, e.db, "Адлер", 20000)

	rec := e.do(t, http.MethodGet, "/delivery-prices", "", nil)

	wantStatus(t, rec, http.StatusOK)
	var list []map[string]any
	testutil.Decode(t, rec, &list)
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	first := list[0]
	if len(first) != 6 {
		t.Errorf("keys = %v, want exactly id, name, value, price, createdAt, updatedAt", first)
	}
	for _, key := range []string{"id", "name", "value", "price", "createdAt", "updatedAt"} {
		if _, ok := first[key]; !ok {
			t.Errorf("missing key %q in %v", key, first)
		}
	}
	if first["name"] != "Сочи" || first["price"] != float64(150) {
		t.Errorf("first = %v, want name Сочи and price 150", first)
	}
	if list[1]["name"] != "Адлер" {
		t.Errorf("second name = %v, want Адлер (ordered by id)", list[1]["name"])
	}
}

func TestListEmpty(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, http.MethodGet, "/delivery-prices", "", nil)

	wantStatus(t, rec, http.StatusOK)
	if got := rec.Body.String(); got != "[]" && got != "[]\n" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestCreate(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"deliveryPrices": []map[string]any{
		{"name": "Сочи", "value": "Россия, Сочи", "price": 150},
		{"name": "Адлер", "price": 99.5},
		{"name": "  Сириус  ", "value": "  Федеральная территория  ", "price": 300},
	}}

	rec := e.do(t, http.MethodPost, "/delivery-prices", e.admin, body)

	wantStatus(t, rec, http.StatusCreated)
	var created []map[string]any
	testutil.Decode(t, rec, &created)
	if len(created) != 3 {
		t.Fatalf("created = %d objects, want 3", len(created))
	}
	if created[1]["price"] != 99.5 {
		t.Errorf("price = %v, want 99.5", created[1]["price"])
	}
	if e.count(t) != 3 {
		t.Errorf("rows = %d, want 3", e.count(t))
	}
	var kopecks int64
	if err := e.db.Get(&kopecks, `SELECT price_kopecks FROM delivery_prices WHERE name = 'Адлер'`); err != nil {
		t.Fatalf("select Адлер: %v", err)
	}
	if kopecks != 9950 {
		t.Errorf("price_kopecks = %d, want 9950", kopecks)
	}
	var name, value string
	if err := e.db.QueryRow(`SELECT name, value FROM delivery_prices WHERE id = $1`, int64(created[2]["id"].(float64))).
		Scan(&name, &value); err != nil {
		t.Fatalf("select Сириус: %v", err)
	}
	if name != "Сириус" || value != "Федеральная территория" {
		t.Errorf("name = %q, value = %q, want trimmed values", name, value)
	}
}

func TestCreateUnauthorized(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"deliveryPrices": []map[string]any{{"name": "Сочи", "price": 150}}}

	rec := e.do(t, http.MethodPost, "/delivery-prices", "", body)

	wantStatus(t, rec, http.StatusUnauthorized)
	if e.count(t) != 0 {
		t.Errorf("rows = %d, want 0", e.count(t))
	}
}

func TestCreateForbidden(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"deliveryPrices": []map[string]any{{"name": "Сочи", "price": 150}}}

	rec := e.do(t, http.MethodPost, "/delivery-prices", e.user, body)

	wantStatus(t, rec, http.StatusForbidden)
	if e.count(t) != 0 {
		t.Errorf("rows = %d, want 0", e.count(t))
	}
}

func TestCreateValidation(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		name     string
		body     any
		wantKeys []string
	}{
		{
			name:     "empty name and zero price",
			body:     map[string]any{"deliveryPrices": []map[string]any{{"name": "", "price": 0}}},
			wantKeys: []string{"deliveryPrices[0].name", "deliveryPrices[0].price"},
		},
		{
			name:     "whitespace-only name",
			body:     map[string]any{"deliveryPrices": []map[string]any{{"name": "   ", "price": 10}}},
			wantKeys: []string{"deliveryPrices[0].name"},
		},
		{
			name:     "empty array",
			body:     map[string]any{"deliveryPrices": []map[string]any{}},
			wantKeys: []string{"deliveryPrices"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := e.do(t, http.MethodPost, "/delivery-prices", e.admin, tt.body)

			wantStatus(t, rec, http.StatusBadRequest)
			var got errorBody
			testutil.Decode(t, rec, &got)
			if len(got.Errors) != len(tt.wantKeys) {
				t.Errorf("errors = %v, want keys %v", got.Errors, tt.wantKeys)
			}
			for _, key := range tt.wantKeys {
				if _, ok := got.Errors[key]; !ok {
					t.Errorf("errors = %v, missing key %q", got.Errors, key)
				}
			}
		})
	}
}

func TestCreateMalformedJSON(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, http.MethodPost, "/delivery-prices", e.admin, "not an object")

	wantStatus(t, rec, http.StatusBadRequest)
	var got errorBody
	testutil.Decode(t, rec, &got)
	if got.Message != "Некорректный JSON" {
		t.Errorf("message = %q, want Некорректный JSON", got.Message)
	}
}

func TestCreateDuplicateRollsBack(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"deliveryPrices": []map[string]any{
		{"name": "A", "price": 100},
		{"name": "B", "price": 200},
		{"name": "A", "price": 300},
	}}

	rec := e.do(t, http.MethodPost, "/delivery-prices", e.admin, body)

	wantStatus(t, rec, http.StatusConflict)
	var got errorBody
	testutil.Decode(t, rec, &got)
	if got.Message != "name должно быть уникальным" {
		t.Errorf("message = %q, want %q", got.Message, "name должно быть уникальным")
	}
	if e.count(t) != 0 {
		t.Errorf("rows = %d, want 0 after rollback", e.count(t))
	}
}

func TestUpdate(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)
	body := map[string]any{"name": "updated name", "value": "updated value", "price": 99}

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), e.admin, body)

	wantStatus(t, rec, http.StatusOK)
	var got map[string]any
	testutil.Decode(t, rec, &got)
	if got["name"] != "updated name" || got["value"] != "updated value" || got["price"] != float64(99) {
		t.Errorf("response = %v", got)
	}
	var name, value string
	var kopecks int64
	err := e.db.QueryRow(`SELECT name, value, price_kopecks FROM delivery_prices WHERE id = $1`, id).
		Scan(&name, &value, &kopecks)
	if err != nil {
		t.Fatalf("select row: %v", err)
	}
	if name != "updated name" || value != "updated value" || kopecks != 9900 {
		t.Errorf("row = %q, %q, %d, want updated name, updated value, 9900", name, value, kopecks)
	}
}

func TestUpdateSetsUpdatedAt(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)
	// Age the row so that now() is observably later than updated_at.
	if _, err := e.db.Exec(`UPDATE delivery_prices SET updated_at = now() - interval '1 hour' WHERE id = $1`, id); err != nil {
		t.Fatalf("age row: %v", err)
	}

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), e.admin,
		map[string]any{"name": "Сочи", "price": 150})

	wantStatus(t, rec, http.StatusOK)
	n := testutil.Count(t, e.db,
		`SELECT count(*) FROM delivery_prices WHERE id = $1 AND updated_at > now() - interval '1 minute'`, id)
	if n != 1 {
		t.Errorf("updated_at was not refreshed")
	}
}

func TestUpdateValidation(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), e.admin,
		map[string]any{"name": "", "price": 0})

	wantStatus(t, rec, http.StatusBadRequest)
	var got errorBody
	testutil.Decode(t, rec, &got)
	for _, key := range []string{"name", "price"} {
		if _, ok := got.Errors[key]; !ok {
			t.Errorf("errors = %v, missing key %q", got.Errors, key)
		}
	}
}

func TestUpdateDuplicateName(t *testing.T) {
	e := newEnv(t)
	testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)
	id := testutil.InsertDeliveryPrice(t, e.db, "Адлер", 20000)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), e.admin,
		map[string]any{"name": "Сочи", "price": 100})

	wantStatus(t, rec, http.StatusConflict)
	var got errorBody
	testutil.Decode(t, rec, &got)
	if got.Message != "name должно быть уникальным" {
		t.Errorf("message = %q, want %q", got.Message, "name должно быть уникальным")
	}
}

func TestUpdateMissing(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/999", e.admin, map[string]any{"name": "x", "price": 1})

	wantStatus(t, rec, http.StatusNotFound)
}

func TestUpdateBadID(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/abc", e.admin, map[string]any{"name": "x", "price": 1})

	wantStatus(t, rec, http.StatusNotFound)
}

func TestUpdateUnauthorized(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), "",
		map[string]any{"name": "x", "price": 1})

	wantStatus(t, rec, http.StatusUnauthorized)
}

func TestUpdateForbidden(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)

	rec := e.do(t, http.MethodPatch, "/delivery-prices/"+strconv.FormatInt(id, 10), e.user,
		map[string]any{"name": "x", "price": 1})

	wantStatus(t, rec, http.StatusForbidden)
}

func TestDelete(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)
	path := "/delivery-prices/" + strconv.FormatInt(id, 10)

	rec := e.do(t, http.MethodDelete, path, e.admin, nil)

	wantStatus(t, rec, http.StatusNoContent)
	if e.count(t) != 0 {
		t.Errorf("rows = %d, want 0", e.count(t))
	}
	again := e.do(t, http.MethodDelete, path, e.admin, nil)
	wantStatus(t, again, http.StatusNoContent)
}

func TestDeleteBadID(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, http.MethodDelete, "/delivery-prices/0", e.admin, nil)

	wantStatus(t, rec, http.StatusNotFound)
}

func TestDeleteUnauthorized(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)

	rec := e.do(t, http.MethodDelete, "/delivery-prices/"+strconv.FormatInt(id, 10), "", nil)

	wantStatus(t, rec, http.StatusUnauthorized)
	if e.count(t) != 1 {
		t.Errorf("rows = %d, want 1", e.count(t))
	}
}

func TestDeleteForbidden(t *testing.T) {
	e := newEnv(t)
	id := testutil.InsertDeliveryPrice(t, e.db, "Сочи", 15000)

	rec := e.do(t, http.MethodDelete, "/delivery-prices/"+strconv.FormatInt(id, 10), e.user, nil)

	wantStatus(t, rec, http.StatusForbidden)
	if e.count(t) != 1 {
		t.Errorf("rows = %d, want 1", e.count(t))
	}
}

func TestByName(t *testing.T) {
	db := testutil.NewDB(t)
	s := deliveryprice.NewStore(db)
	ctx := context.Background()
	id := testutil.InsertDeliveryPrice(t, db, "Сочи", 15000)

	got, err := s.ByName(ctx, "Сочи")
	if err != nil {
		t.Fatalf("ByName: %v", err)
	}
	if got.ID != id || got.Name != "Сочи" || got.Price.String() != "150" {
		t.Errorf("got %+v, want id %d, name Сочи, price 150", got, id)
	}

	for _, name := range []string{"Москва", "сочи", "Сочи "} {
		_, err := s.ByName(ctx, name)
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("ByName(%q) error = %v, want store.ErrNotFound", name, err)
		}
	}
}
