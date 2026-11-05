package address_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type bookAPI struct {
	db       *sqlx.DB
	router   http.Handler
	token    string
	sessions *customer.Service
	tokens   *auth.Tokens
}

func newBookAPI(t *testing.T, enabled bool) bookAPI {
	t.Helper()
	db, _ := bookFixture(t)
	c := clock.Fixed(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	sessions := customer.NewService(db, c)
	tokens := auth.NewTokens("test", time.Hour, c)
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: enabled}
	r := chi.NewRouter()
	address.NewBookHandler(address.NewStore(db), address.NewService(db), adapter.Required(false)).Routes(r)
	var token string
	var err error
	if enabled {
		token, _, err = sessions.CreateSession(context.Background(), 1, "password", "web")
	} else {
		token, err = tokens.Issue(1, auth.RoleUser)
	}
	if err != nil {
		t.Fatal(err)
	}
	return bookAPI{db, r, token, sessions, tokens}
}
func (f bookAPI) call(t *testing.T, method, path string, body any, want int) *httptest.ResponseRecorder {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, method, "/customer/addresses"+path, f.token, body)
	if rec.Code != want || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("%s %s: %d %s cache=%q want %d", method, path, rec.Code, rec.Body, rec.Header().Get("Cache-Control"), want)
	}
	return rec
}
func apiFields() address.Fields {
	return address.Fields{Name: "Дом", City: "Белореченск", Street: "ул. Гоголя, 61", Apartment: "12", Entrance: "2, этаж 3"}
}
func (f bookAPI) create(t *testing.T, in address.Fields) address.Entry {
	t.Helper()
	rec := f.call(t, "POST", "", in, 201)
	var out struct {
		Entry   address.Entry `json:"entry"`
		Created bool          `json:"created"`
	}
	testutil.Decode(t, rec, &out)
	if !out.Created {
		t.Fatal(rec.Body)
	}
	return out.Entry
}
func expectBookError(t *testing.T, rec *httptest.ResponseRecorder, message string, fields map[string]string) {
	t.Helper()
	var out struct {
		Status  int               `json:"statusCode"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"errors"`
	}
	testutil.Decode(t, rec, &out)
	if out.Status != rec.Code || out.Message != message || !reflect.DeepEqual(out.Fields, fields) {
		t.Fatal(rec.Body)
	}
}
func TestBookAPIList(t *testing.T) {
	f := newBookAPI(t, true)
	if rec := f.call(t, "GET", "", nil, 200); rec.Body.String() != "{\"addresses\":[]}\n" {
		t.Fatal(rec.Body)
	}
	e := f.create(t, apiFields())
	if e.AddressID != 0 {
		t.Fatal("internal ID leaked")
	}
	bookExec(t, f.db, `INSERT INTO addresses(city,address) VALUES('Город','  старая строка  '); INSERT INTO user_addresses(user_id,address_id) SELECT 1,id FROM addresses WHERE city='Город'`)
	rec := f.call(t, "GET", "", nil, 200)
	var out struct {
		Addresses []address.Entry `json:"addresses"`
	}
	testutil.Decode(t, rec, &out)
	if len(out.Addresses) != 2 || out.Addresses[0] != e || out.Addresses[1].ID <= e.ID || !out.Addresses[1].NeedsReview || out.Addresses[1].Street != "" || out.Addresses[1].Address != "  старая строка  " {
		t.Fatal(rec.Body)
	}
	var raw map[string][]map[string]any
	testutil.Decode(t, rec, &raw)
	want := []string{"id", "name", "city", "address", "street", "apartment", "entrance", "isDefault", "needsReview", "createdAt", "updatedAt"}
	for _, item := range raw["addresses"] {
		if len(item) != len(want) {
			t.Fatal(item)
		}
		for _, key := range want {
			if _, ok := item[key]; !ok {
				t.Fatal(key, item)
			}
		}
	}
	t.Logf("GET example: %s", rec.Body)
}
func TestBookAPICreate(t *testing.T) {
	f := newBookAPI(t, true)
	e := f.create(t, apiFields())
	data, _ := json.Marshal(struct {
		Entry   address.Entry `json:"entry"`
		Created bool          `json:"created"`
	}{e, true})
	t.Logf("POST new example: %s", data)
	in := apiFields()
	in.Name = "Другое"
	in.City = " БЕЛОРЕЧЕНСК "
	in.Street = "УЛ. ГОГОЛЯ, 61"
	rec := f.call(t, "POST", "", in, 200)
	var out struct {
		Entry   address.Entry `json:"entry"`
		Created bool          `json:"created"`
	}
	testutil.Decode(t, rec, &out)
	if out.Created || out.Entry != e || testutil.Count(t, f.db, `SELECT count(*) FROM user_addresses`) != 1 {
		t.Fatal(rec.Body)
	}
	t.Logf("POST duplicate example: %s", rec.Body)
}
func TestBookAPIUpdate(t *testing.T) {
	f := newBookAPI(t, true)
	e := f.create(t, apiFields())
	shared := bookEntries(t, f.db, 1)[0]
	other := createBookEntry(t, address.NewService(f.db), 2, apiFields())
	bookExec(t, f.db, `INSERT INTO orders(customer_id,address_id,items_hash,delivery_address) VALUES(1,$1,'hash',$2)`, shared.AddressID, e.Address)
	in := address.Fields{City: "Город", Street: "Дом 3"}
	rec := f.call(t, "PUT", "/"+strconv.FormatInt(e.ID, 10), in, 200)
	var out struct {
		Entry address.Entry `json:"entry"`
	}
	testutil.Decode(t, rec, &out)
	if out.Entry.ID != e.ID || out.Entry.CreatedAt != e.CreatedAt || !out.Entry.IsDefault || out.Entry.Address != "Дом 3" {
		t.Fatal(rec.Body)
	}
	t.Logf("PUT example: %s", rec.Body)
	second := f.create(t, address.Fields{City: "Город", Street: "Дом 4"})
	rec = f.call(t, "PUT", "/"+strconv.FormatInt(second.ID, 10)+"/default", nil, 200)
	testutil.Decode(t, rec, &out)
	if !out.Entry.IsDefault || out.Entry.ID != second.ID {
		t.Fatal(rec.Body)
	}
	t.Logf("DEFAULT example: %s", rec.Body)
	if rec := f.call(t, "DELETE", "/"+strconv.FormatInt(second.ID, 10), nil, 204); rec.Body.Len() != 0 {
		t.Fatal(rec.Body)
	}
	if got := bookEntries(t, f.db, 1); len(got) != 1 || got[0].IsDefault {
		t.Fatal(got)
	}
	if got := bookEntries(t, f.db, 2); len(got) != 1 || got[0] != other {
		t.Fatal(got)
	}
	if testutil.Count(t, f.db, `SELECT count(*) FROM orders WHERE address_id=$1 AND delivery_address=$2`, shared.AddressID, e.Address) != 1 {
		t.Fatal("order changed")
	}
}
func TestBookAPIValidation(t *testing.T) {
	f := newBookAPI(t, true)
	e := f.create(t, apiFields())
	invalid := []address.Fields{{}, {City: " ", Street: " "}, {Name: strings.Repeat("я", 41), City: strings.Repeat("я", 81), Street: strings.Repeat("я", 121), Apartment: strings.Repeat("я", 21), Entrance: strings.Repeat("я", 21)}, {City: "Город", Street: "я"}}
	for _, in := range invalid {
		for _, method := range []string{"POST", "PUT"} {
			path := ""
			if method == "PUT" {
				path = "/" + strconv.FormatInt(e.ID, 10)
			}
			rec := f.call(t, method, path, in, 400)
			expectBookError(t, rec, "Проверьте адрес", address.FieldErrors(address.CleanFields(in)))
		}
	}
	bodies := []string{"", "{", "null", "[]", "true", "1", `"text"`, `{} {}`, `{} trailing`, `{"city":1}`, `{"city":null}`, `{"city":[]}`, `{"city":{}}`, `{"userId":1}`, `{"id":1}`, `{"isDefault":true}`, `{"unknown":"x"}`, `{"name":null}`, `{"apartment":false}`, `{"entrance":null}`, `{"street":null}`, `{"name":"` + strings.Repeat("x", 1<<20) + `"}`}
	for _, body := range bodies {
		for _, method := range []string{"POST", "PUT"} {
			path := "/customer/addresses"
			if method == "PUT" {
				path += "/" + strconv.FormatInt(e.ID, 10)
			}
			req := httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+f.token)
			rec := httptest.NewRecorder()
			f.router.ServeHTTP(rec, req)
			if rec.Code != 400 || rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("%s %d %s", body[:min(len(body), 50)], rec.Code, rec.Body)
			}
			expectBookError(t, rec, "Некорректный JSON", nil)
		}
	}
	if got := bookEntries(t, f.db, 1); len(got) != 1 || got[0].ID != e.ID || got[0].Name != e.Name {
		t.Fatal(got)
	}
}
func TestBookAPIIDs(t *testing.T) {
	f := newBookAPI(t, true)
	f.create(t, apiFields())
	other := createBookEntry(t, address.NewService(f.db), 2, apiFields())
	before := bookEntries(t, f.db, 1)
	for _, id := range []string{"abc", "0", "-1", "9223372036854775808", "999", strconv.FormatInt(other.ID, 10)} {
		for _, route := range []struct{ method, suffix string }{{"PUT", ""}, {"PUT", "/default"}, {"DELETE", ""}} {
			rec := f.call(t, route.method, "/"+id+route.suffix, apiFields(), 404)
			expectBookError(t, rec, "Не найдено", nil)
		}
	}
	if got := bookEntries(t, f.db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
}
func TestBookAPIConflict(t *testing.T) {
	f := newBookAPI(t, true)
	f.create(t, apiFields())
	e := f.create(t, address.Fields{City: "Город", Street: "Дом 2"})
	before := bookEntries(t, f.db, 1)
	rec := f.call(t, "PUT", "/"+strconv.FormatInt(e.ID, 10), apiFields(), 409)
	expectBookError(t, rec, "Такой адрес уже сохранён", map[string]string{"street": "Такой адрес уже сохранён"})
	if got := bookEntries(t, f.db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
}
func TestBookAPIAuth(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		f := newBookAPI(t, enabled)
		web, _, err := f.sessions.CreateSession(context.Background(), 1, "password", "web")
		if err != nil {
			t.Fatal(err)
		}
		mini, _, err := f.sessions.CreateSession(context.Background(), 1, "telegram", "mini")
		if err != nil {
			t.Fatal(err)
		}
		revoked, _, _ := f.sessions.CreateSession(context.Background(), 1, "password", "web")
		expired, _, _ := f.sessions.CreateSession(context.Background(), 1, "password", "web")
		bookExec(t, f.db, `UPDATE customer_sessions SET revoked_at=now() WHERE token_hash=$1`, customerTokenHash(t, revoked))
		bookExec(t, f.db, `UPDATE customer_sessions SET expires_at='2020-01-01' WHERE token_hash=$1`, customerTokenHash(t, expired))
		user, _ := f.tokens.Issue(1, auth.RoleUser)
		staff, _ := f.tokens.Issue(1, auth.RoleAdmin)
		unknown, _ := f.tokens.Issue(1, "unknown")
		for _, tc := range []struct {
			token   string
			on, off int
		}{{web, 200, 503}, {mini, 200, 503}, {revoked, 401, 503}, {expired, 401, 503}, {user, 401, 200}, {staff, 401, 403}, {unknown, 401, 401}, {"", 401, 401}, {"bad", 401, 401}} {
			f.token = tc.token
			want := tc.off
			if enabled {
				want = tc.on
			}
			rec := f.call(t, "GET", "", nil, want)
			if want != 200 {
				msg := map[int]string{401: "Требуется авторизация", 403: "Недостаточно прав", 503: "Вход временно недоступен"}[want]
				expectBookError(t, rec, msg, nil)
			}
		}
	}
	// The group must enforce identity even with a custom access middleware.
	db, _ := bookFixture(t)
	for _, role := range []string{"", "admin", "unknown"} {
		r := chi.NewRouter()
		access := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if role != "" {
					r = r.WithContext(auth.NewContext(r.Context(), auth.Identity{UserID: 1, Role: role}))
				}
				next.ServeHTTP(w, r)
			})
		}
		address.NewBookHandler(address.NewStore(db), address.NewService(db), access).Routes(r)
		rec := testutil.DoJSON(t, r, "GET", "/customer/addresses", "", nil)
		want := 403
		if role == "" {
			want = 401
		}
		if rec.Code != want || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(rec.Code, rec.Body)
		}
	}
}
func TestBookAPIFailures(t *testing.T) {
	f := newBookAPI(t, false)
	e := f.create(t, apiFields())
	before := bookEntries(t, f.db, 1)
	bookExec(t, f.db, `INSERT INTO users(id,name,phone,merged_into_user_id) VALUES(3,'Merged','3',2)`)
	merged, _ := f.tokens.Issue(3, auth.RoleUser)
	f.token = merged
	expectBookError(t, f.call(t, "POST", "", apiFields(), 401), "Требуется авторизация", nil)
	f.token, _ = f.tokens.Issue(1, auth.RoleUser)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	bookExec(t, f.db, `CREATE FUNCTION test_book_api_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '%',NEW.address; END $$; CREATE TRIGGER test_book_api_failure BEFORE INSERT ON addresses FOR EACH ROW EXECUTE FUNCTION test_book_api_failure()`)
	t.Cleanup(func() {
		bookExec(t, f.db, `DROP TRIGGER test_book_api_failure ON addresses; DROP FUNCTION test_book_api_failure()`)
	})
	var events []*sentry.Event
	client, err := sentry.NewClient(sentry.ClientOptions{Dsn: "https://public@example.test/1", BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
		events = append(events, event)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	original := f.router
	f.router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		original.ServeHTTP(w, r.WithContext(sentry.SetHubOnContext(r.Context(), sentry.NewHub(client, sentry.NewScope()))))
	})
	in := address.Fields{Name: "PRIVATE NAME", City: "PRIVATE CITY", Street: "PRIVATE STREET"}
	for _, route := range []struct{ method, path string }{{"POST", ""}, {"PUT", "/" + strconv.FormatInt(e.ID, 10)}} {
		expectBookError(t, f.call(t, route.method, route.path, in, 500), "Внутренняя ошибка сервера", nil)
	}
	captured, _ := json.Marshal(events)
	if len(events) != 2 || bytes.Contains(captured, []byte("PRIVATE")) {
		t.Fatalf("monitoring leaked fields or missed failures: %s", captured)
	}
	if strings.Contains(logs.String(), "PRIVATE") {
		t.Fatal(logs.String())
	}
	if got := bookEntries(t, f.db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	// A closed DB also exercises read/default/delete failures without leaking SQL.
	broken, _ := sqlx.Open("pgx", "")
	_ = broken.Close()
	r := chi.NewRouter()
	access := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.NewContext(r.Context(), auth.Identity{UserID: 1, Role: auth.RoleUser})))
		})
	}
	address.NewBookHandler(address.NewStore(broken), address.NewService(broken), access).Routes(r)
	f.router = r
	for _, route := range []struct{ method, path string }{{"GET", ""}, {"PUT", "/1/default"}, {"DELETE", "/1"}} {
		expectBookError(t, f.call(t, route.method, route.path, nil, 500), "Внутренняя ошибка сервера", nil)
	}
}

func customerTokenHash(t *testing.T, token string) []byte {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, "cs_"))
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	return hash[:]
}

type failingBookBody struct{}

func (failingBookBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failingBookBody) Close() error             { return nil }
func TestBookAPIReadError(t *testing.T) {
	f := newBookAPI(t, false)
	req := httptest.NewRequest("POST", "/customer/addresses", nil)
	req.Body = failingBookBody{}
	req.Header.Set("Authorization", "Bearer "+f.token)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatal(rec.Code, rec.Body)
	}
	expectBookError(t, rec, "Некорректный JSON", nil)
}

func bookFixture(t *testing.T) (*sqlx.DB, *address.Service) {
	t.Helper()
	db := testutil.NewDB(t)
	if _, err := db.Exec(`INSERT INTO users(id,name,phone) VALUES(1,'One','1'),(2,'Two','2')`); err != nil {
		t.Fatal(err)
	}
	return db, address.NewService(db)
}
func createBookEntry(t *testing.T, s *address.Service, user int64, in address.Fields) address.Entry {
	t.Helper()
	e, created, err := s.CreateEntry(context.Background(), user, in)
	if err != nil || !created {
		t.Fatalf("create: %+v %v %v", e, created, err)
	}
	return e
}
func bookExec(t *testing.T, db *sqlx.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
func bookEntries(t *testing.T, db *sqlx.DB, user int64) []address.Entry {
	t.Helper()
	e, err := address.NewStore(db).ListEntries(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
