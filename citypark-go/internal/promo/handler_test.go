package promo

import (
	"encoding/json"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type apiEnv struct {
	db          *sqlx.DB
	router      http.Handler
	admin, user string
}

func api(t *testing.T) apiEnv {
	db := testutil.NewDB(t)
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	a, e := tokens.Issue(1, auth.RoleAdmin)
	if e != nil {
		t.Fatal(e)
	}
	u, e := tokens.Issue(2, auth.RoleUser)
	if e != nil {
		t.Fatal(e)
	}
	r := chi.NewRouter()
	NewHandler(db, tokens).Routes(r)
	return apiEnv{db, r, a, u}
}
func (e apiEnv) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	return testutil.DoJSON(t, e.router, method, path, e.admin, body)
}
func TestPromoAdminAuth(t *testing.T) {
	e := api(t)
	for _, route := range []struct{ method, path string }{{"GET", "/promo-codes"}, {"GET", "/promo-codes/1"}, {"POST", "/promo-codes"}, {"PUT", "/promo-codes/1"}} {
		for _, token := range []string{"", e.user} {
			r := testutil.DoJSON(t, e.router, route.method, route.path, token, input())
			want := 401
			if token != "" {
				want = 403
			}
			if r.Code != want {
				t.Fatalf("%s %s: %d %s", route.method, route.path, r.Code, r.Body)
			}
		}
	}
}
func TestPromoAdminCRUD(t *testing.T) {
	e := api(t)
	r := e.do(t, "GET", "/promo-codes", nil)
	if r.Code != 200 || r.Body.String() != "[]\n" {
		t.Fatalf("empty: %d %s", r.Code, r.Body)
	}
	in := input()
	r = e.do(t, "POST", "/promo-codes", in)
	if r.Code != 201 {
		t.Fatalf("create %d %s", r.Code, r.Body)
	}
	var c Campaign
	testutil.Decode(t, r, &c)
	if c.Code != "TEST_10" || c.ProductIDs == nil || c.CategoryIDs == nil {
		t.Fatal(c)
	}
	if err := os.WriteFile("/tmp/promo-campaign-example.json", r.Body.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	path := "/promo-codes/" + strconv.FormatInt(c.ID, 10)
	r = e.do(t, "GET", path, nil)
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	var got Campaign
	testutil.Decode(t, r, &got)
	if got.Code != c.Code || got.ID != c.ID {
		t.Fatal(got)
	}
	in.Percentage = 20
	r = e.do(t, "PUT", path, in)
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	var percentage int
	if err := e.db.Get(&percentage, `SELECT percentage FROM promo_codes WHERE id=$1`, c.ID); err != nil || percentage != 20 {
		t.Fatalf("persisted %d %v", percentage, err)
	}
	r = e.do(t, "GET", "/promo-codes", nil)
	var list []Campaign
	testutil.Decode(t, r, &list)
	if r.Code != 200 || len(list) != 1 || list[0].Percentage != 20 {
		t.Fatal(r.Body)
	}
}
func TestPromoAdminErrors(t *testing.T) {
	e := api(t)
	r := e.do(t, "POST", "/promo-codes", input())
	var c Campaign
	testutil.Decode(t, r, &c)
	path := "/promo-codes/" + strconv.FormatInt(c.ID, 10)
	assert := func(r *httptest.ResponseRecorder, status int, field, message string) {
		t.Helper()
		var b struct {
			Message string
			Errors  map[string]string
		}
		testutil.Decode(t, r, &b)
		if r.Code != status {
			t.Fatalf("want %d got %d %s", status, r.Code, r.Body)
		}
		got := b.Message
		if field != "" {
			got = b.Errors[field]
		}
		if got != message {
			t.Fatalf("want %q got %s", message, r.Body)
		}
	}
	for _, method := range []string{"GET", "PUT"} {
		assert(e.do(t, method, "/promo-codes/999", input()), 404, "", "Промокод не найден.")
		if r := e.do(t, method, "/promo-codes/bad", input()); r.Code != 404 {
			t.Fatal(r.Body)
		}
	}
	assert(e.do(t, "POST", "/promo-codes", input()), 409, "", "Такой промокод уже существует.")
	in := input()
	in.Code = "changed"
	assert(e.do(t, "PUT", path, in), 409, "", "Код нельзя изменить после создания.")
	cases := []struct {
		field, message string
		value          any
	}{
		{"code", "Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.", "?"},
		{"percentage", "Укажите целый процент от 1 до 99.", 100},
		{"minimumEligibleSubtotal", "Укажите положительную сумму.", 0},
		{"maximumDiscount", "Укажите положительную сумму.", -1},
		{"target", "Выберите все блюда или укажите блюда и категории.", "selected"},
		{"endsAt", "Дата окончания должна быть позже даты начала.", "2026-01-01T00:00:00Z"},
	}
	for _, tc := range cases {
		body := map[string]any{"code": "other", "percentage": 10, "enabled": true, "target": "all"}
		body[tc.field] = tc.value
		if tc.field == "endsAt" {
			body["startsAt"] = tc.value
		}
		assert(e.do(t, "POST", "/promo-codes", body), 400, tc.field, tc.message)
	}
	for _, field := range []string{"productIds", "categoryIds"} {
		body := map[string]any{"code": "other", "percentage": 10, "target": "selected", field: []int64{999999}}
		assert(e.do(t, "POST", "/promo-codes", body), 400, field, "Выбранное блюдо или категория не найдены.")
	}
	for _, body := range []any{json.RawMessage(`{"code":`), map[string]any{"percentage": 1.5}, map[string]any{"startsAt": "bad"}} {
		if raw, ok := body.(json.RawMessage); ok {
			req := httptest.NewRequest("POST", "/promo-codes", strings.NewReader(string(raw)))
			req.Header.Set("Authorization", "Bearer "+e.admin)
			rec := httptest.NewRecorder()
			e.router.ServeHTTP(rec, req)
			assert(rec, 400, "", "Некорректный JSON")
		} else {
			assert(e.do(t, "POST", "/promo-codes", body), 400, "", "Некорректный JSON")
		}
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM promo_codes`); n != 1 {
		t.Fatal(n)
	}
}
func TestPromoAdminDisable(t *testing.T) {
	e := api(t)
	in := input()
	r := e.do(t, "POST", "/promo-codes", in)
	var c Campaign
	testutil.Decode(t, r, &c)
	in.Enabled = false
	r = e.do(t, "PUT", "/promo-codes/"+strconv.FormatInt(c.ID, 10), in)
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	r = e.do(t, "GET", "/promo-codes", nil)
	var list []Campaign
	testutil.Decode(t, r, &list)
	if len(list) != 1 || list[0].Enabled || list[0].Code != c.Code || list[0].Percentage != c.Percentage {
		t.Fatal(r.Body)
	}
}
func TestPromoAdminDatabaseErrors(t *testing.T) {
	e := api(t)
	e.db.Close()
	for _, route := range []struct{ method, path string }{{"GET", "/promo-codes"}, {"GET", "/promo-codes/1"}, {"POST", "/promo-codes"}} {
		r := e.do(t, route.method, route.path, input())
		if r.Code != 500 {
			t.Fatal(r.Body)
		}
	}
}
