package address_test

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

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type env struct {
	db     *sqlx.DB
	tokens *auth.Tokens
	router http.Handler
}

func newEnv(t *testing.T) env {
	t.Helper()
	db := testutil.NewDB(t)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	address.NewHandler(address.NewStore(db), tokens).Routes(r)
	return env{db: db, tokens: tokens, router: r}
}

func (e env) token(t *testing.T, userID int64) string {
	t.Helper()
	token, err := e.tokens.Issue(userID, auth.RoleUser)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func (e env) do(t *testing.T, method, path string, userID int64, body any) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.DoJSON(t, e.router, method, path, e.token(t, userID), body)
}

func (e env) user(t *testing.T, phone string) int64 {
	t.Helper()
	return testutil.InsertUser(t, e.db, "user", phone, auth.RoleUser)
}

func countLinks(t *testing.T, db *sqlx.DB, userID, addressID int64) int {
	t.Helper()
	return testutil.Count(t, db,
		`SELECT count(*) FROM user_addresses WHERE user_id = $1 AND address_id = $2`, userID, addressID)
}

func TestFindOrCreate(t *testing.T) {
	db := testutil.NewDB(t)
	s := address.NewStore(db)
	ctx := context.Background()

	first, err := s.FindOrCreate(ctx, "Сочи", "ул. Ленина, 1")
	if err != nil {
		t.Fatalf("first FindOrCreate: %v", err)
	}
	second, err := s.FindOrCreate(ctx, "Сочи", "ул. Ленина, 1")
	if err != nil {
		t.Fatalf("second FindOrCreate: %v", err)
	}

	if first.ID == 0 || first.ID != second.ID {
		t.Errorf("ids = %d and %d, want the same non-zero id", first.ID, second.ID)
	}
	if first.City != "Сочи" || first.Address != "ул. Ленина, 1" {
		t.Errorf("address = %+v", first)
	}
	if !second.UpdatedAt.Equal(first.UpdatedAt) {
		t.Errorf("updatedAt changed from %v to %v", first.UpdatedAt, second.UpdatedAt)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Errorf("addresses = %d, want 1", n)
	}
}

func TestLinkTwice(t *testing.T) {
	db := testutil.NewDB(t)
	s := address.NewStore(db)
	ctx := context.Background()
	userID := testutil.InsertUser(t, db, "user", "+79990000001", auth.RoleUser)
	a, err := s.FindOrCreate(ctx, "Сочи", "ул. Ленина, 1")
	if err != nil {
		t.Fatalf("FindOrCreate: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := s.LinkUser(ctx, userID, a.ID); err != nil {
			t.Fatalf("LinkUser call %d: %v", i+1, err)
		}
	}

	if n := countLinks(t, db, userID, a.ID); n != 1 {
		t.Errorf("links = %d, want 1", n)
	}
}

func TestUnlinkMissing(t *testing.T) {
	db := testutil.NewDB(t)
	s := address.NewStore(db)
	userID := testutil.InsertUser(t, db, "user", "+79990000001", auth.RoleUser)

	if err := s.UnlinkUser(context.Background(), userID, 999); err != nil {
		t.Fatalf("UnlinkUser: %v", err)
	}
}

func TestListByUserEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.InsertUser(t, db, "user", "+79990000001", auth.RoleUser)

	list, err := address.NewStore(db).ListByUser(context.Background(), userID)

	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Errorf("list = %#v, want a non-nil empty slice", list)
	}
}

func TestByID(t *testing.T) {
	db := testutil.NewDB(t)
	s := address.NewStore(db)
	ctx := context.Background()
	id := testutil.InsertAddress(t, db, 0, "Сочи", "ул. Ленина, 1")

	got, err := s.ByID(ctx, id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.ID != id || got.City != "Сочи" || got.Address != "ул. Ленина, 1" {
		t.Errorf("address = %+v", got)
	}

	if _, err := s.ByID(ctx, id+1); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing id: err = %v, want store.ErrNotFound", err)
	}
}

func TestCreate(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodPost, "/addresses", userID, map[string]any{"city": "Сочи", "address": "ул. Ленина, 1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var body map[string]any
	testutil.Decode(t, rec, &body)
	for _, key := range []string{"id", "city", "address", "createdAt", "updatedAt"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response has no %q: %v", key, body)
		}
	}
	if body["city"] != "Сочи" || body["address"] != "ул. Ленина, 1" {
		t.Errorf("response = %v", body)
	}
	id := int64(body["id"].(float64))
	if n := countLinks(t, e.db, userID, id); n != 1 {
		t.Errorf("links = %d, want 1", n)
	}
}

func TestCreateTrims(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodPost, "/addresses", userID, map[string]any{"city": "  Сочи ", "address": "\tул. Ленина, 1\n"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got address.Address
	testutil.Decode(t, rec, &got)
	if got.City != "Сочи" || got.Address != "ул. Ленина, 1" {
		t.Errorf("address = %+v, want trimmed values", got)
	}
}

func TestCreateTwice(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")
	body := map[string]any{"city": "Сочи", "address": "ул. Ленина, 1"}

	var ids [2]int64
	for i := range ids {
		rec := e.do(t, http.MethodPost, "/addresses", userID, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("call %d: status = %d, body %s", i+1, rec.Code, rec.Body)
		}
		var got address.Address
		testutil.Decode(t, rec, &got)
		ids[i] = got.ID
	}

	if ids[0] != ids[1] {
		t.Errorf("ids = %v, want the same id", ids)
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Errorf("addresses = %d, want 1", n)
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM user_addresses`); n != 1 {
		t.Errorf("links = %d, want 1", n)
	}
}

func TestCreateSharedAddress(t *testing.T) {
	e := newEnv(t)
	a := e.user(t, "+79990000001")
	b := e.user(t, "+79990000002")
	body := map[string]any{"city": "Сочи", "address": "ул. Ленина, 1"}

	for _, id := range []int64{a, b} {
		if rec := e.do(t, http.MethodPost, "/addresses", id, body); rec.Code != http.StatusCreated {
			t.Fatalf("user %d: status = %d, body %s", id, rec.Code, rec.Body)
		}
	}

	if n := testutil.Count(t, e.db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Errorf("addresses = %d, want 1", n)
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM user_addresses`); n != 2 {
		t.Errorf("links = %d, want 2", n)
	}
}

func TestCreateValidation(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
		keys []string
	}{
		{"empty city and missing address", map[string]any{"city": ""}, []string{"city", "address"}},
		{"whitespace only", map[string]any{"city": "   ", "address": " "}, []string{"city", "address"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			userID := e.user(t, "+79990000001")

			rec := e.do(t, http.MethodPost, "/addresses", userID, tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body)
			}
			var body struct {
				Errors map[string]string `json:"errors"`
			}
			testutil.Decode(t, rec, &body)
			for _, k := range tt.keys {
				if body.Errors[k] == "" {
					t.Errorf("errors = %v, want key %q", body.Errors, k)
				}
			}
			if n := testutil.Count(t, e.db, `SELECT count(*) FROM addresses`); n != 0 {
				t.Errorf("addresses = %d, want 0", n)
			}
		})
	}
}

func TestCreateBadJSON(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodPost, "/addresses", userID, "not an object")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body)
	}
}

func TestCreateUnauthorized(t *testing.T) {
	e := newEnv(t)

	rec := testutil.DoJSON(t, e.router, http.MethodPost, "/addresses", "", map[string]any{"city": "Сочи", "address": "ул. Ленина, 1"})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestList(t *testing.T) {
	e := newEnv(t)
	a := e.user(t, "+79990000001")
	b := e.user(t, "+79990000002")
	first := testutil.InsertAddress(t, e.db, a, "Сочи", "ул. Ленина, 1")
	testutil.InsertAddress(t, e.db, b, "Сочи", "ул. Мира, 2")
	second := testutil.InsertAddress(t, e.db, a, "Адлер", "ул. Кирова, 3")

	rec := e.do(t, http.MethodGet, "/addresses", a, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got []address.Address
	testutil.Decode(t, rec, &got)
	if len(got) != 2 || got[0].ID != first || got[1].ID != second {
		t.Fatalf("addresses = %+v, want ids [%d %d]", got, first, second)
	}
	if got[0].City != "Сочи" || got[1].Address != "ул. Кирова, 3" {
		t.Errorf("addresses = %+v", got)
	}
}

func TestListEmptyIsJSONArray(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodGet, "/addresses", userID, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Errorf("body = %q, want %q", got, "[]\n")
	}
}

func TestListUnauthorized(t *testing.T) {
	e := newEnv(t)

	rec := testutil.DoJSON(t, e.router, http.MethodGet, "/addresses", "", nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestDelete(t *testing.T) {
	e := newEnv(t)
	a := e.user(t, "+79990000001")
	b := e.user(t, "+79990000002")
	id := testutil.InsertAddress(t, e.db, a, "Сочи", "ул. Ленина, 1")
	if _, err := e.db.Exec(`INSERT INTO user_addresses (user_id, address_id) VALUES ($1, $2)`, b, id); err != nil {
		t.Fatalf("link second user: %v", err)
	}

	rec := e.do(t, http.MethodDelete, "/addresses/"+strconv.FormatInt(id, 10), a, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body %s", rec.Code, rec.Body)
	}
	if n := countLinks(t, e.db, a, id); n != 0 {
		t.Errorf("link of the deleting user = %d, want 0", n)
	}
	if n := countLinks(t, e.db, b, id); n != 1 {
		t.Errorf("link of the other user = %d, want 1", n)
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM addresses WHERE id = $1`, id); n != 1 {
		t.Errorf("address rows = %d, want 1", n)
	}
}

func TestDeleteMissing(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodDelete, "/addresses/999", userID, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body %s", rec.Code, rec.Body)
	}
}

func TestDeleteBadID(t *testing.T) {
	e := newEnv(t)
	userID := e.user(t, "+79990000001")

	rec := e.do(t, http.MethodDelete, "/addresses/abc", userID, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body %s", rec.Code, rec.Body)
	}
}

func TestDeleteUnauthorized(t *testing.T) {
	e := newEnv(t)

	rec := testutil.DoJSON(t, e.router, http.MethodDelete, "/addresses/1", "", nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
