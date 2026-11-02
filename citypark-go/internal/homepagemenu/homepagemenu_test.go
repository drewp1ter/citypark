package homepagemenu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func setup(t *testing.T) (*sqlx.DB, http.Handler, string, string) {
	t.Helper()
	db := testutil.NewDB(t)
	db.MustExec(`INSERT INTO homepage_menu(id,data) VALUES(1,'{"categories":[]}') ON CONFLICT(id) DO UPDATE SET data=EXCLUDED.data`)
	t.Cleanup(func() { db.MustExec(`UPDATE homepage_menu SET data='{"categories":[]}' WHERE id=1`) })
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	NewHandler(NewStore(db), tokens).Routes(r)
	return db, r, admin, user
}

func TestHomepageMenuPersistenceAndValidation(t *testing.T) {
	db, r, admin, user := setup(t)
	c := testutil.InsertCategory(t, db, testutil.Category{})
	other := testutil.InsertCategory(t, db, testutil.Category{})
	hidden := testutil.InsertCategory(t, db, testutil.Category{IsHidden: true})
	deleted := testutil.InsertCategory(t, db, testutil.Category{IsDeleted: true})
	p := testutil.InsertProduct(t, db, testutil.Product{CategoryID: c})
	p2 := testutil.InsertProduct(t, db, testutil.Product{CategoryID: c})
	gone := testutil.InsertProduct(t, db, testutil.Product{CategoryID: c, IsDeleted: true})
	db.MustExec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, p)
	read := func() Configuration {
		t.Helper()
		rec := testutil.DoJSON(t, r, "GET", "/homepage-menu", "", nil)
		if rec.Code != 200 {
			t.Fatal(rec.Code, rec.Body)
		}
		var out Configuration
		testutil.Decode(t, rec, &out)
		return out
	}
	if got := read(); got.Categories == nil || len(got.Categories) != 0 {
		t.Fatal(got)
	}
	good := Configuration{[]Category{{other, []int64{}}, {c, []int64{p2, p}}}}
	for _, tc := range []struct {
		token  string
		status int
	}{{"", 401}, {user, 403}, {admin, 200}} {
		rec := testutil.DoJSON(t, r, "PUT", "/homepage-menu", tc.token, good)
		if rec.Code != tc.status {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	if !reflect.DeepEqual(read(), good) {
		t.Fatal(read())
	}
	var saved []byte
	if err := db.Get(&saved, `SELECT data FROM homepage_menu`); err != nil {
		t.Fatal(err)
	}
	t.Log(string(saved))
	tests := []struct {
		in             Configuration
		field, message string
	}{
		{Configuration{}, "categories", "Укажите список категорий"},
		{Configuration{[]Category{{0, []int64{}}}}, "categories[0].categoryId", "Укажите уникальную категорию"},
		{Configuration{[]Category{{c, []int64{}}, {c, []int64{}}}}, "categories[1].categoryId", "Укажите уникальную категорию"},
		{Configuration{[]Category{{c, nil}}}, "categories[0].productIds", "Укажите список из не более трёх блюд"},
		{Configuration{[]Category{{c, []int64{p, p2, p, p2}}}}, "categories[0].productIds", "Укажите список из не более трёх блюд"},
		{Configuration{[]Category{{c, []int64{p, p}}}}, "categories[0].productIds", "Укажите уникальные блюда"},
		{Configuration{[]Category{{c, []int64{-1}}}}, "categories[0].productIds", "Укажите уникальные блюда"},
		{Configuration{[]Category{{hidden, []int64{}}}}, "categories[0].categoryId", "Категория недоступна"},
		{Configuration{[]Category{{deleted, []int64{}}}}, "categories[0].categoryId", "Категория недоступна"},
		{Configuration{[]Category{{999999, []int64{}}}}, "categories[0].categoryId", "Категория недоступна"},
		{Configuration{[]Category{{other, []int64{p}}}}, "categories[0].productIds", "Блюдо недоступно или относится к другой категории"},
		{Configuration{[]Category{{c, []int64{gone}}}}, "categories[0].productIds", "Блюдо недоступно или относится к другой категории"},
		{Configuration{[]Category{{c, []int64{999999}}}}, "categories[0].productIds", "Блюдо недоступно или относится к другой категории"},
	}
	for _, tc := range tests {
		rec := testutil.DoJSON(t, r, "PUT", "/homepage-menu", admin, tc.in)
		var e httpx.Error
		testutil.Decode(t, rec, &e)
		if rec.Code != 400 || e.Message != validationMessage || e.Fields[tc.field] != tc.message {
			t.Fatal(rec.Code, rec.Body)
		}
		if !reflect.DeepEqual(read(), good) {
			t.Fatal("invalid save changed configuration")
		}
	}
	for _, body := range []string{`{`, `null`, `{}`, `{"categories":null}`, `{"categories":"bad"}`, `{"categories":[{"categoryId":1.5,"productIds":[]}]}`} {
		req := httptest.NewRequest("PUT", "/homepage-menu", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+admin)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != 400 {
			t.Fatal(body, rec.Code, rec.Body)
		}
	}
	// Catalog changes do not erase selections; consumers resolve stale IDs against the live menu.
	db.MustExec(`UPDATE products SET category_id=$1 WHERE id=$2`, other, p)
	db.MustExec(`UPDATE categories SET is_hidden=true WHERE id=$1`, c)
	if !reflect.DeepEqual(read(), good) {
		t.Fatal("catalog edits changed selection")
	}
	rec := testutil.DoJSON(t, r, "PUT", "/homepage-menu", admin, Configuration{[]Category{}})
	if rec.Code != 200 || len(read().Categories) != 0 {
		t.Fatal(rec.Code, rec.Body)
	}
}

func TestHomepageMenuMigration(t *testing.T) {
	db, _, _, _ := setup(t)
	data, err := os.ReadFile("../../migrations/00032_homepage_menu.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(data), "-- +goose Down", 2)
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, q := range []string{`CREATE SCHEMA homepage_menu_migration_test`, `SET LOCAL search_path TO homepage_menu_migration_test`, parts[0], parts[1], parts[0]} {
		if _, err := tx.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	var out string
	if err := tx.Get(&out, `SELECT data->'categories' FROM homepage_menu`); err != nil || out != "[]" {
		t.Fatal(out, err)
	}
}

func TestHomepageMenuErrors(t *testing.T) {
	db, r, admin, _ := setup(t)
	db.MustExec(`DELETE FROM homepage_menu`)
	for _, method := range []string{"GET", "PUT"} {
		rec := testutil.DoJSON(t, r, method, "/homepage-menu", admin, Configuration{[]Category{}})
		if rec.Code != 404 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	db.MustExec(`INSERT INTO homepage_menu(id,data) VALUES(1,'{"categories":[]}')`)
	closed, err := sqlx.Open("pgx", "unused")
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	broken := NewStore(closed)
	if _, err := broken.Get(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if err := broken.Put(context.Background(), Configuration{[]Category{}}); err == nil {
		t.Fatal("expected error")
	}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	br := chi.NewRouter()
	NewHandler(broken, tokens).Routes(br)
	for _, method := range []string{"GET", "PUT"} {
		rec := testutil.DoJSON(t, br, method, "/homepage-menu", admin, Configuration{[]Category{}})
		if rec.Code != 500 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
}
