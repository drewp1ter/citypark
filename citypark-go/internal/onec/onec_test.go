package onec

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

var testNow = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func TestRoomSalesRequest(t *testing.T) {
	var gotPath, gotStart, gotEnd, gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotStart = r.URL.Query().Get("StartPeriod")
		gotEnd = r.URL.Query().Get("EndPeriod")
		gotUser, gotPass, _ = r.BasicAuth()
		w.Write([]byte(`[{"Organization":"ООО «Олимп»","Room":"VIP","Count":3,"Sum":12000.5,"Extra":1},
			{"Organization":"ИП Иванов","Room":"Стандарт","Count":1,"Sum":2000}]`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "reports", "secret", nil)
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)
	got, err := c.RoomSales(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/hs/reports/v1/room_sales" || gotStart != "20261001000000" || gotEnd != "20261031235959" {
		t.Errorf("request = %s %s %s", gotPath, gotStart, gotEnd)
	}
	if gotUser != "reports" || gotPass != "secret" {
		t.Errorf("basic auth = %q %q", gotUser, gotPass)
	}
	want := []RoomSale{
		{"ООО «Олимп»", "VIP", 3, 12000.5},
		{"ИП Иванов", "Стандарт", 1, 2000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v, want %+v", got, want)
	}
}

func TestDaySalesRequest(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`[{"Date":"21.10.2026","Sum":8000}]`))
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL, "u", "p", nil).DaySales(context.Background(), testNow, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/hs/reports/v1/month_sales" {
		t.Errorf("path = %s", gotPath)
	}
	if want := []DaySale{{"21.10.2026", 8000}}; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v, want %+v", got, want)
	}
}

func TestDaySalesNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`null`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "u", "p", nil)
	days, err := c.DaySales(context.Background(), testNow, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if days == nil || len(days) != 0 {
		t.Errorf("DaySales = %#v, want empty non-nil slice", days)
	}
	rooms, err := c.RoomSales(context.Background(), testNow, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if rooms == nil || len(rooms) != 0 {
		t.Errorf("RoomSales = %#v, want empty non-nil slice", rooms)
	}
}

func TestClientErrors(t *testing.T) {
	const password = "very-secret-password"
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantSub string
	}{
		{"status 500", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }, "status 500"},
		{"invalid json", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{`)) }, "decode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			_, err := NewClient(srv.URL, "u", password, nil).RoomSales(context.Background(), testNow, testNow)
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("error %q does not contain %q", err, tt.wantSub)
			}
			if strings.Contains(err.Error(), password) {
				t.Errorf("error %q contains the password", err)
			}
		})
	}

	t.Run("transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		base := srv.URL
		srv.Close()
		_, err := NewClient(base, "u", password, nil).DaySales(context.Background(), testNow, testNow)
		if err == nil {
			t.Fatal("want an error")
		}
		if strings.Contains(err.Error(), password) || strings.Contains(err.Error(), base) {
			t.Errorf("error %q leaks the password or the address", err)
		}
	})

	t.Run("bad base url", func(t *testing.T) {
		_, err := NewClient("http://bad host", "u", password, nil).DaySales(context.Background(), testNow, testNow)
		if err == nil || !strings.Contains(err.Error(), "build") {
			t.Errorf("error = %v, want a build error", err)
		}
	})
}

func TestClientDisabled(t *testing.T) {
	c := NewClient("", "u", "p", &http.Client{Transport: failTransport{t}})
	if _, err := c.RoomSales(context.Background(), testNow, testNow); !errors.Is(err, ErrDisabled) {
		t.Errorf("RoomSales error = %v, want ErrDisabled", err)
	}
	if _, err := c.DaySales(context.Background(), testNow, testNow); !errors.Is(err, ErrDisabled) {
		t.Errorf("DaySales error = %v, want ErrDisabled", err)
	}
}

type failTransport struct{ t *testing.T }

func (f failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.t.Error("a disabled client must not send a request")
	return nil, errors.New("unexpected request")
}

type fixture struct {
	router http.Handler
	admin  string
	user   string
	reqs   *[]string
}

// newFixture serves the handler over a fake 1C server that answers every report with body and status.
// base "" builds a disabled client instead.
func newFixture(t *testing.T, status int, body string, disabled bool) fixture {
	t.Helper()
	var reqs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, r.URL.Path+"?"+r.URL.RawQuery)
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	base := srv.URL
	if disabled {
		base = ""
	}
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(testNow))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	NewHandler(NewClient(base, "u", "p", nil), tokens, clock.Fixed(testNow)).Routes(r)
	return fixture{router: r, admin: admin, user: user, reqs: &reqs}
}

func (f fixture) get(t *testing.T, token, path string) (int, map[string]any) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, path, token, nil)
	var out map[string]any
	testutil.Decode(t, rec, &out)
	return rec.Code, out
}

func TestHotelSales(t *testing.T) {
	f := newFixture(t, 200, `[
		{"Organization":"ООО «Олимп»","Room":"VIP-номер","Count":3,"Sum":12000.5},
		{"Organization":"ИП Иванов","Room":"Стандарт","Count":2,"Sum":4000},
		{"Organization":"ООО «Олимп»","Room":"Люкс","Count":1,"Sum":9000}]`, false)

	code, out := f.get(t, f.admin, "/reports/hotel/sales?from=2026-10-01&to=2026-10-21")
	if code != 200 {
		t.Fatalf("status = %d, body %v", code, out)
	}
	want := map[string]any{
		"rows": []any{
			map[string]any{"organization": "ИП Иванов", "room": "Стандарт", "count": 2.0, "sum": 4000.0},
			map[string]any{"organization": "ООО «Олимп»", "room": "VIP-номер", "count": 3.0, "sum": 12000.5},
			map[string]any{"organization": "ООО «Олимп»", "room": "Люкс", "count": 1.0, "sum": 9000.0},
		},
		"total": map[string]any{"count": 6.0, "sum": 25000.5},
	}
	// Equal organizations are ordered by room: «VIP-номер» < «Люкс» in code point order (Latin before Cyrillic).
	if !reflect.DeepEqual(out, want) {
		t.Errorf("body = %v\nwant %v", out, want)
	}
	if got := (*f.reqs)[0]; !strings.Contains(got, "StartPeriod=20261001000000") ||
		!strings.Contains(got, "EndPeriod=20261021235959") {
		t.Errorf("1C request = %s", got)
	}
}

func TestHotelSalesEmpty(t *testing.T) {
	f := newFixture(t, 200, `null`, false)
	code, out := f.get(t, f.admin, "/reports/hotel/sales?from=2026-10-01&to=2026-10-01")
	if code != 200 {
		t.Fatalf("status = %d", code)
	}
	if rows, ok := out["rows"].([]any); !ok || len(rows) != 0 {
		t.Errorf("rows = %#v, want []", out["rows"])
	}
	if total := out["total"].(map[string]any); total["count"] != 0.0 || total["sum"] != 0.0 {
		t.Errorf("total = %v, want zeros", total)
	}
}

func TestHotelDaily(t *testing.T) {
	f := newFixture(t, 200, `[
		{"Date":"20.10.2026","Sum":5000},{"Date":"20.10.2026","Sum":3000},
		{"Date":"22.10.2026","Sum":100},{"Date":"19.10.2026","Sum":1}]`, false)

	code, out := f.get(t, f.admin, "/reports/hotel/daily?from=2026-10-20&to=2026-10-22")
	if code != 200 {
		t.Fatalf("status = %d, body %v", code, out)
	}
	want := map[string]any{
		"days": []any{
			map[string]any{"date": "2026-10-20", "sum": 8000.0},
			map[string]any{"date": "2026-10-21", "sum": 0.0},
			map[string]any{"date": "2026-10-22", "sum": 100.0},
		},
		"total": 8100.0,
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("body = %v\nwant %v", out, want)
	}
	if got := (*f.reqs)[0]; !strings.Contains(got, "/month_sales") || !strings.Contains(got, "EndPeriod=20261022235959") {
		t.Errorf("1C request = %s", got)
	}
}

func TestHotelDailyInvalidDate(t *testing.T) {
	f := newFixture(t, 200, `[{"Date":"2026-10-20","Sum":5}]`, false)
	code, out := f.get(t, f.admin, "/reports/hotel/daily?from=2026-10-20&to=2026-10-22")
	if code != 502 || out["message"] != "Сервис отчётов недоступен" {
		t.Errorf("got %d %v, want 502 Сервис отчётов недоступен", code, out)
	}
}

func TestReportsInvalid(t *testing.T) {
	const (
		badDate = "дата в формате ГГГГ-ММ-ДД"
	)
	tests := []struct {
		name, query, field, msg string
	}{
		{"no from", "to=2026-10-21", "from", badDate},
		{"no to", "from=2026-10-21", "to", badDate},
		{"to before from", "from=2026-10-21&to=2026-10-20", "to", "не раньше даты from"},
		{"367 days", "from=2026-01-01&to=2027-01-02", "to", "период не больше 366 дней"},
		{"wrong format", "from=1.10.2026&to=2026-10-21", "from", badDate},
	}
	for _, path := range []string{"/reports/hotel/sales", "/reports/hotel/daily"} {
		for _, tt := range tests {
			t.Run(path+" "+tt.name, func(t *testing.T) {
				f := newFixture(t, 200, `[]`, false)
				code, out := f.get(t, f.admin, path+"?"+tt.query)
				if code != 400 || out["message"] != "Некорректные параметры отчёта" {
					t.Fatalf("got %d %v", code, out)
				}
				if fields, _ := out["errors"].(map[string]any); fields[tt.field] != tt.msg {
					t.Errorf("errors = %v, want %s: %s", out["errors"], tt.field, tt.msg)
				}
				if len(*f.reqs) != 0 {
					t.Error("1C must not be called for invalid parameters")
				}
			})
		}
	}
}

func TestReportsPeriodLimit(t *testing.T) {
	f := newFixture(t, 200, `[]`, false)
	code, out := f.get(t, f.admin, "/reports/hotel/daily?from=2026-01-01&to=2027-01-01")
	if code != 200 {
		t.Fatalf("366 days: status = %d, body %v", code, out)
	}
	if days := out["days"].([]any); len(days) != 366 {
		t.Errorf("days = %d, want 366", len(days))
	}
}

func TestReportsUpstreamDown(t *testing.T) {
	for _, path := range []string{"/reports/hotel/sales", "/reports/hotel/daily"} {
		f := newFixture(t, 500, `oops`, false)
		code, out := f.get(t, f.admin, path+"?from=2026-10-01&to=2026-10-21")
		if code != 502 || out["message"] != "Сервис отчётов недоступен" {
			t.Errorf("%s: got %d %v", path, code, out)
		}
	}
}

func TestReportsDisabled(t *testing.T) {
	for _, path := range []string{"/reports/hotel/sales", "/reports/hotel/daily"} {
		f := newFixture(t, 200, `[]`, true)
		code, out := f.get(t, f.admin, path+"?from=2026-10-01&to=2026-10-21")
		if code != 404 || out["message"] != "Не найдено" {
			t.Errorf("%s: got %d %v", path, code, out)
		}
	}
}

func TestReportsAuth(t *testing.T) {
	f := newFixture(t, 200, `[]`, false)
	for _, path := range []string{"/reports/hotel/sales", "/reports/hotel/daily"} {
		q := path + "?from=2026-10-01&to=2026-10-21"
		if code, _ := f.get(t, "", q); code != 401 {
			t.Errorf("%s without token: status = %d, want 401", path, code)
		}
		if code, _ := f.get(t, f.user, q); code != 403 {
			t.Errorf("%s as user: status = %d, want 403", path, code)
		}
	}
	if len(*f.reqs) != 0 {
		t.Error("1C must not be called without admin rights")
	}
}
