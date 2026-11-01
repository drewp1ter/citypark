package deliverysettings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type fixture struct {
	db          *sqlx.DB
	router      http.Handler
	admin, user string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	reset := func() {
		t.Helper()
		_, err := db.Exec(`INSERT INTO delivery_settings (id,cooking,outskirts_label,outskirts_min_kopecks,outskirts_max_kopecks) VALUES(1,'~30 мин','СОТ и ДНТ',20000,35000) ON CONFLICT(id) DO UPDATE SET cooking=EXCLUDED.cooking,outskirts_label=EXCLUDED.outskirts_label,outskirts_min_kopecks=EXCLUDED.outskirts_min_kopecks,outskirts_max_kopecks=EXCLUDED.outskirts_max_kopecks`)
		if err != nil {
			t.Fatal(err)
		}
	}
	reset()
	t.Cleanup(reset)
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)))
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
	return fixture{db, r, admin, user}
}
func (f fixture) get(t *testing.T) response {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, "GET", "/delivery-settings", "", nil)
	if rec.Code != 200 {
		t.Fatalf("GET %d %s", rec.Code, rec.Body)
	}
	var out response
	testutil.Decode(t, rec, &out)
	return out
}
func payload() map[string]any {
	return map[string]any{"cooking": "~30 мин", "outskirtsLabel": "СОТ и ДНТ", "outskirtsMin": 200, "outskirtsMax": 350}
}
func TestDeliverySettingsDefaults(t *testing.T) {
	f := newFixture(t)
	s, at, err := NewStore(f.db).Get(context.Background())
	if err != nil || s != (Settings{"~30 мин", "СОТ и ДНТ", 20000, 35000}) || at.IsZero() {
		t.Fatalf("%+v %v %v", s, at, err)
	}
	migration, err := os.ReadFile("../../migrations/00031_delivery_settings.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	tx, err := f.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// Transaction-local schema isolates the migration from the shared tables.
	for _, q := range []string{`CREATE SCHEMA delivery_settings_migration_test`, `SET LOCAL search_path TO delivery_settings_migration_test`, parts[0], parts[1], parts[0]} {
		if _, err := tx.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	isolated, _, err := NewStore(tx).Get(context.Background())
	if err != nil || isolated != s {
		t.Fatalf("migration: %+v %v", isolated, err)
	}
	for _, q := range []string{`UPDATE delivery_settings SET cooking=' '`, `UPDATE delivery_settings SET cooking=repeat('я',101)`, `UPDATE delivery_settings SET outskirts_label=''`, `UPDATE delivery_settings SET outskirts_label=repeat('я',61)`, `UPDATE delivery_settings SET outskirts_min_kopecks=-100`, `UPDATE delivery_settings SET outskirts_min_kopecks=1`, `UPDATE delivery_settings SET outskirts_max_kopecks=10000001`, `UPDATE delivery_settings SET outskirts_max_kopecks=100`, `INSERT INTO delivery_settings(id,cooking,outskirts_label,outskirts_min_kopecks,outskirts_max_kopecks) VALUES(2,'x','x',0,0)`} {
		tx.MustExec("SAVEPOINT constraint_check")
		if _, err := tx.Exec(q); !store.IsCheckViolation(err) {
			t.Errorf("constraint %s: %v", q, err)
		}
		tx.MustExec("ROLLBACK TO SAVEPOINT constraint_check")
	}
}
func TestDeliverySettingsPublicGet(t *testing.T) {
	f := newFixture(t)
	before := f.get(t)
	after := f.get(t)
	if before != after || before.Cooking != "~30 мин" || before.UpdatedAt.IsZero() {
		t.Fatalf("%+v %+v", before, after)
	}
	rec := testutil.DoJSON(t, f.router, "GET", "/delivery-settings", "", nil)
	var fields map[string]any
	testutil.Decode(t, rec, &fields)
	if len(fields) != 5 {
		t.Fatal(fields)
	}
	t.Log(rec.Body.String())
}
func TestDeliverySettingsPutAuth(t *testing.T) {
	f := newFixture(t)
	before := f.get(t)
	for _, tc := range []struct {
		token  string
		status int
	}{{"", 401}, {f.user, 403}} {
		rec := testutil.DoJSON(t, f.router, "PUT", "/delivery-settings", tc.token, payload())
		if rec.Code != tc.status {
			t.Fatal(rec.Code, rec.Body)
		}
		if f.get(t) != before {
			t.Fatal("unauthorized write")
		}
	}
	rec := testutil.DoJSON(t, f.router, "PUT", "/delivery-settings", f.admin, payload())
	if rec.Code != 200 {
		t.Fatal(rec.Code, rec.Body)
	}
}
func TestDeliverySettingsPut(t *testing.T) {
	f := newFixture(t)
	for _, bounds := range [][2]int{{200, 350}, {0, 0}, {200, 200}, {0, 100000}, {100000, 100000}} {
		before := f.get(t)
		in := payload()
		in["cooking"] = " \tГотовим 30 минут\n"
		in["outskirtsLabel"] = " Район "
		in["outskirtsMin"] = bounds[0]
		in["outskirtsMax"] = bounds[1]
		rec := testutil.DoJSON(t, f.router, "PUT", "/delivery-settings", f.admin, in)
		if rec.Code != 200 {
			t.Fatal(rec.Code, rec.Body)
		}
		var out response
		testutil.Decode(t, rec, &out)
		if out.Cooking != "Готовим 30 минут" || out.OutskirtsLabel != "Район" || int64(out.OutskirtsMin) != int64(bounds[0])*100 || int64(out.OutskirtsMax) != int64(bounds[1])*100 || !out.UpdatedAt.After(before.UpdatedAt) || out != f.get(t) {
			t.Fatalf("%+v", out)
		}
	}
}
func TestDeliverySettingsStore(t *testing.T) {
	f := newFixture(t)
	s := Settings{"Время", "Район", 0, 10000000}
	at, err := NewStore(f.db).Put(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	got, saved, err := NewStore(f.db).Get(context.Background())
	if err != nil || got != s || !saved.Equal(at) {
		t.Fatalf("%+v %v %v", got, saved, err)
	}
	var amount int64
	if err := f.db.Get(&amount, `SELECT outskirts_max_kopecks FROM delivery_settings`); err != nil || amount != 10000000 {
		t.Fatal(amount, err)
	}
}
func TestDeliverySettingsValidation(t *testing.T) {
	f := newFixture(t)
	before := f.get(t)
	cases := []struct {
		field   string
		value   any
		message string
	}{
		{"cooking", "", "поле не должно быть пустым"}, {"cooking", " \t", "поле не должно быть пустым"}, {"cooking", strings.Repeat("я", 101), "значение должно быть не больше 100"},
		{"outskirtsLabel", "", "поле не должно быть пустым"}, {"outskirtsLabel", " \n", "поле не должно быть пустым"}, {"outskirtsLabel", strings.Repeat("я", 61), "значение должно быть не больше 60"},
		{"outskirtsMin", nil, "поле не должно быть пустым"}, {"outskirtsMax", nil, "поле не должно быть пустым"},
		{"outskirtsMin", -1, "значение должно быть не меньше 0"}, {"outskirtsMax", -1, "значение должно быть не меньше 0"},
		{"outskirtsMin", 100001, "значение должно быть не больше 100000"}, {"outskirtsMax", 100001, "значение должно быть не больше 100000"},
		{"outskirtsMax", 199, maxBelowMin},
	}
	for _, field := range []string{"outskirtsMin", "outskirtsMax"} {
		for _, v := range []any{1.5, "200", true} {
			cases = append(cases, struct {
				field   string
				value   any
				message string
			}{field, v, ""})
		}
	}
	for _, tc := range cases {
		in := payload()
		in[tc.field] = tc.value
		rec := testutil.DoJSON(t, f.router, "PUT", "/delivery-settings", f.admin, in)
		if rec.Code != 400 {
			t.Fatalf("%v: %d %s", in, rec.Code, rec.Body)
		}
		var e httpx.Error
		testutil.Decode(t, rec, &e)
		if tc.message != "" && (e.Fields[tc.field] != tc.message || e.Message != validationMessage) {
			t.Fatalf("%+v", e)
		}
		if tc.message == "" && e.Message != "Некорректный JSON" {
			t.Fatal(e)
		}
		if f.get(t) != before {
			t.Fatal("invalid request wrote settings")
		}
		t.Log(rec.Body.String())
	}
	for _, field := range []string{"cooking", "outskirtsLabel", "outskirtsMin", "outskirtsMax"} {
		in := payload()
		delete(in, field)
		rec := testutil.DoJSON(t, f.router, "PUT", "/delivery-settings", f.admin, in)
		var e httpx.Error
		testutil.Decode(t, rec, &e)
		if rec.Code != 400 || e.Fields[field] != "поле не должно быть пустым" {
			t.Fatal(rec.Code, e)
		}
	}
	req := httptest.NewRequest("PUT", "/delivery-settings", strings.NewReader(`{"cooking":`))
	req.Header.Set("Authorization", "Bearer "+f.admin)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	if rec.Code != 400 || f.get(t) != before {
		t.Fatal(rec.Code, rec.Body)
	}
}
func TestDeliverySettingsErrors(t *testing.T) {
	f := newFixture(t)
	f.db.MustExec(`DELETE FROM delivery_settings`)
	s := NewStore(f.db)
	if _, _, err := s.Get(context.Background()); !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := s.Put(context.Background(), Settings{}); !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "PUT"} {
		rec := testutil.DoJSON(t, f.router, method, "/delivery-settings", f.admin, payload())
		if rec.Code != 404 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	closed, err := sqlx.Open("pgx", "unused")
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	broken := NewStore(closed)
	if _, _, err := broken.Get(context.Background()); err == nil || errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := broken.Put(context.Background(), Settings{}); err == nil || errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	NewHandler(broken, tokens).Routes(r)
	for _, method := range []string{"GET", "PUT"} {
		rec := testutil.DoJSON(t, r, method, "/delivery-settings", f.admin, payload())
		var e httpx.Error
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		if rec.Code != 500 || e.Message != "Внутренняя ошибка сервера" {
			t.Fatal(rec.Code, rec.Body)
		}
	}
}
