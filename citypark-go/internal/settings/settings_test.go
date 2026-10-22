package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type fixture struct {
	db     *sqlx.DB
	router http.Handler
	admin  string
	user   string
}

// newFixture returns a router over the migrated database. NewDB does not truncate site_settings (the migration
// seeds it), so the row found at the start is put back after the test.
func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	var data []byte
	if err := db.Get(&data, `SELECT data FROM site_settings WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := db.Exec(`INSERT INTO site_settings (id, data) VALUES (1, $1)
			ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data`, data)
		if err != nil {
			t.Errorf("restore site_settings: %v", err)
		}
	})

	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(now))
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
	return fixture{db: db, router: r, admin: admin, user: user}
}

// get reads GET /settings as a generic map and as the typed struct.
func (f fixture) get(t *testing.T) (map[string]any, Settings) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/settings", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body %s", rec.Code, rec.Body)
	}
	var m map[string]any
	testutil.Decode(t, rec, &m)
	var st Settings
	testutil.Decode(t, rec, &st)
	return m, st
}

func (f fixture) put(t *testing.T, token string, body any) (int, map[string]any) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/settings", token, body)
	var out map[string]any
	testutil.Decode(t, rec, &out)
	return rec.Code, out
}

// seeded returns the settings as a mutable generic map, the way a client edits what it got from GET.
func (f fixture) seeded(t *testing.T) map[string]any {
	t.Helper()
	m, _ := f.get(t)
	delete(m, "updatedAt")
	return m
}

// setPath sets a value inside nested maps/slices: setPath(m, "phones", "hotel", "number") or ("social", 0, "url").
func setPath(t *testing.T, m any, value any, path ...any) {
	t.Helper()
	cur := m
	for i, p := range path {
		last := i == len(path)-1
		switch k := p.(type) {
		case string:
			obj := cur.(map[string]any)
			if last {
				obj[k] = value
				return
			}
			cur = obj[k]
		case int:
			arr := cur.([]any)
			if last {
				arr[k] = value
				return
			}
			cur = arr[k]
		}
	}
}

func TestSeed(t *testing.T) {
	f := newFixture(t)
	m, st := f.get(t)

	r := st.Phones.Restaurant
	if r.Display != "8 918 311-97-91" || r.Number != "+79183119791" || r.AltNumber != "+78615533003" {
		t.Errorf("restaurant phone = %+v", r)
	}
	if st.Phones.Free.Number != "88001002441" {
		t.Errorf("free number = %q", st.Phones.Free.Number)
	}
	if len(st.Social) != 2 {
		t.Errorf("social links = %d, want 2", len(st.Social))
	}
	if st.Address.Postcode != "352630" {
		t.Errorf("postcode = %q", st.Address.Postcode)
	}
	if _, ok := m["updatedAt"].(string); !ok {
		t.Errorf("updatedAt missing in %v", m)
	}
}

func TestPutSettings(t *testing.T) {
	f := newFixture(t)
	body := f.seeded(t)
	setPath(t, body, "new@citypark.ru", "email")
	body["social"] = body["social"].([]any)[:1]
	body["updatedAt"] = "2000-01-01T00:00:00Z" // ignored

	code, out := f.put(t, f.admin, body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %v", code, out)
	}
	if out["email"] != "new@citypark.ru" || out["updatedAt"] == "2000-01-01T00:00:00Z" {
		t.Errorf("PUT response = %v", out)
	}

	m, st := f.get(t)
	if st.Email != "new@citypark.ru" || len(st.Social) != 1 || st.Social[0].Label != "VK" {
		t.Errorf("after PUT: email %q, social %+v", st.Email, st.Social)
	}
	if m["updatedAt"] != out["updatedAt"] {
		t.Errorf("updatedAt GET %v != PUT %v", m["updatedAt"], out["updatedAt"])
	}
}

func TestPutSettingsEmptySocial(t *testing.T) {
	f := newFixture(t)
	body := f.seeded(t)
	body["social"] = nil

	code, out := f.put(t, f.admin, body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %v", code, out)
	}
	if links, ok := out["social"].([]any); !ok || len(links) != 0 {
		t.Errorf("response social = %#v, want []", out["social"])
	}
	m, _ := f.get(t)
	if links, ok := m["social"].([]any); !ok || len(links) != 0 {
		t.Errorf("GET social = %#v, want []", m["social"])
	}
}

func TestPutSettingsTrims(t *testing.T) {
	f := newFixture(t)
	body := f.seeded(t)
	setPath(t, body, "  a@b.ru ", "email")
	setPath(t, body, "  ВКонтакте\t", "social", 0, "name")
	setPath(t, body, " 8 918 311-97-91 ", "phones", "restaurant", "display")

	code, out := f.put(t, f.admin, body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %v", code, out)
	}
	var stored []byte
	if err := f.db.Get(&stored, `SELECT data FROM site_settings`); err != nil {
		t.Fatal(err)
	}
	var st Settings
	if err := json.Unmarshal(stored, &st); err != nil {
		t.Fatal(err)
	}
	if st.Email != "a@b.ru" || st.Social[0].Name != "ВКонтакте" || st.Phones.Restaurant.Display != "8 918 311-97-91" {
		t.Errorf("stored not trimmed: %q %q %q", st.Email, st.Social[0].Name, st.Phones.Restaurant.Display)
	}
}

func TestPutSettingsInvalid(t *testing.T) {
	tests := []struct {
		name  string
		field string
		edit  func(t *testing.T, m map[string]any)
		want  string
	}{
		{"email", "email", func(t *testing.T, m map[string]any) { setPath(t, m, "abc", "email") },
			"email должен быть электронной почтой"},
		{"postcode", "address.postcode", func(t *testing.T, m map[string]any) { setPath(t, m, "3526", "address", "postcode") },
			"некорректное значение"},
		{"number", "phones.hotel.number", func(t *testing.T, m map[string]any) { setPath(t, m, "8 918", "phones", "hotel", "number") },
			"номер: только цифры и «+»"},
		{"alt number", "phones.hotel.altNumber", func(t *testing.T, m map[string]any) { setPath(t, m, "12ab", "phones", "hotel", "altNumber") },
			"номер: только цифры и «+»"},
		{"alt display without number", "phones.hotel.altNumber", func(t *testing.T, m map[string]any) {
			setPath(t, m, "", "phones", "hotel", "altNumber")
		}, "заполните оба поля"},
		{"alt number without display", "phones.hotel.altDisplay", func(t *testing.T, m map[string]any) {
			setPath(t, m, "", "phones", "hotel", "altDisplay")
		}, "заполните оба поля"},
		{"http url", "social[0].url", func(t *testing.T, m map[string]any) { setPath(t, m, "http://vk.com", "social", 0, "url") },
			"ссылка должна начинаться с https://"},
		{"inn", "company.inn", func(t *testing.T, m map[string]any) { setPath(t, m, "abc", "company", "inn") },
			"некорректное значение"},
		{"empty label", "phones.free.label", func(t *testing.T, m map[string]any) { setPath(t, m, "  ", "phones", "free", "label") },
			"поле не должно быть пустым"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			body := f.seeded(t)
			tc.edit(t, body)
			code, out := f.put(t, f.admin, body)
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body %v", code, out)
			}
			errs, _ := out["errors"].(map[string]any)
			if errs[tc.field] != tc.want {
				t.Errorf("errors[%q] = %v, want %q (body %v)", tc.field, errs[tc.field], tc.want, out)
			}
			_, st := f.get(t)
			if st.Email != "belcitypark@mail.ru" {
				t.Errorf("invalid PUT changed the stored settings")
			}
		})
	}
}

func TestPutSettingsReportsTagAndExtraErrorsTogether(t *testing.T) {
	f := newFixture(t)
	body := f.seeded(t)
	setPath(t, body, "abc", "email")
	setPath(t, body, "http://vk.com", "social", 0, "url")

	code, out := f.put(t, f.admin, body)
	errs, _ := out["errors"].(map[string]any)
	if code != http.StatusBadRequest || errs["email"] == nil || errs["social[0].url"] != "ссылка должна начинаться с https://" {
		t.Errorf("status %d, body %v", code, out)
	}
}

func TestPutSettingsBadJSON(t *testing.T) {
	f := newFixture(t)
	code, out := f.put(t, f.admin, "not an object")
	if code != http.StatusBadRequest || out["message"] != "Некорректный JSON" {
		t.Errorf("status %d, body %v", code, out)
	}
}

func TestPutSettingsAuth(t *testing.T) {
	f := newFixture(t)
	body := f.seeded(t)
	if code, _ := f.put(t, "", body); code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", code)
	}
	if code, _ := f.put(t, f.user, body); code != http.StatusForbidden {
		t.Errorf("user role: status = %d, want 403", code)
	}
}

func TestMissingRow(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.Exec(`DELETE FROM site_settings`); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/settings", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET status = %d, want 404", rec.Code)
	}
	code, _ := f.put(t, f.admin, f.seededFromStruct(t))
	if code != http.StatusNotFound {
		t.Errorf("PUT status = %d, want 404", code)
	}
	if _, _, err := NewStore(f.db).Get(context.Background()); err != store.ErrNotFound {
		t.Errorf("Store.Get error = %v, want ErrNotFound", err)
	}
}

// seededFromStruct builds a valid body without reading the database.
func (f fixture) seededFromStruct(t *testing.T) Settings {
	t.Helper()
	var s Settings
	p := Phone{Label: "a", ShortLabel: "a", Display: "1", Number: "+79183119791"}
	s.Phones.Restaurant, s.Phones.Hotel, s.Phones.Free, s.Phones.Landline = p, p, p, p
	s.Address.Postcode, s.Address.City, s.Address.Street = "352630", "Белореченск", "ул. Гоголя, 61"
	s.Email = "a@b.ru"
	s.Company.Name, s.Company.INN, s.Company.OGRN = "ООО", "2309117400", "1092309002378"
	return s
}

func TestStoreGetBadData(t *testing.T) {
	f := newFixture(t)
	if _, err := f.db.Exec(`UPDATE site_settings SET data = '[]'`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewStore(f.db).Get(context.Background()); err == nil {
		t.Error("Get on malformed data returned no error")
	}
}
