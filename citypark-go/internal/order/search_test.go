package order

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

var moscow = time.FixedZone("MSK", 3*3600)

// searchOrder inserts a pickup cash order for the customer with a fixed created_at, status and payment type.
func searchOrder(t *testing.T, db *sqlx.DB, customer, product int64, createdAt time.Time, status, payment string) int64 {
	t.Helper()
	id := insertOrder(t, db, InsertParams{
		CustomerID:  customer,
		PaymentType: PaymentCash,
		ItemsHash:   createdAt.String(),
		Items:       []InsertItem{{ProductID: product, Amount: 1, OfferedPrice: 10000}},
	})
	_, err := db.ExecContext(context.Background(),
		`UPDATE orders SET created_at = $1, status = $2, payment_type = $3 WHERE id = $4`,
		createdAt, status, payment, id)
	if err != nil {
		t.Fatalf("update order: %v", err)
	}
	return id
}

func listIDs(t *testing.T, db *sqlx.DB, f ListFilter, limit, offset int) ([]int64, int) {
	t.Helper()
	orders, count, err := NewStore(db).List(context.Background(), f, limit, offset)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ids := []int64{}
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	return ids, count
}

type searchSeed struct {
	db      *sqlx.DB
	ivan    int64
	petr    int64
	product int64
}

func newSearchSeed(t *testing.T) searchSeed {
	t.Helper()
	db := testutil.NewDB(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Мангал"})
	return searchSeed{
		db:      db,
		ivan:    testutil.InsertUser(t, db, "Иван Петров", "+79991234567", auth.RoleUser),
		petr:    testutil.InsertUser(t, db, "Пётр Сидоров", "+79997654321", auth.RoleUser),
		product: testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык", PriceKopecks: 10000}),
	}
}

func TestListFilterNone(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	a := searchOrder(t, s.db, s.ivan, s.product, base, StatusNew, PaymentCash)
	b := searchOrder(t, s.db, s.petr, s.product, base.Add(time.Hour), StatusNew, PaymentCash)
	c := searchOrder(t, s.db, s.ivan, s.product, base.Add(2*time.Hour), StatusNew, PaymentCash)

	ids, count := listIDs(t, s.db, ListFilter{}, 10, 0)
	if want := []int64{c, b, a}; !reflect.DeepEqual(ids, want) || count != 3 {
		t.Errorf("ids = %v, count = %d; want %v and 3", ids, count, want)
	}
}

func TestListFilterQueryID(t *testing.T) {
	s := newSearchSeed(t)
	// Phones without the digits of the id, so that only the order number can match.
	a := testutil.InsertUser(t, s.db, "Анна", "+70000000000", auth.RoleUser)
	b := testutil.InsertUser(t, s.db, "Борис", "+70000000009", auth.RoleUser)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	searchOrder(t, s.db, a, s.product, base, StatusNew, PaymentCash)
	second := searchOrder(t, s.db, b, s.product, base.Add(time.Hour), StatusNew, PaymentCash)

	ids, count := listIDs(t, s.db, ListFilter{Query: " " + strconv.FormatInt(second, 10) + " "}, 10, 0)
	if want := []int64{second}; !reflect.DeepEqual(ids, want) || count != 1 {
		t.Errorf("ids = %v, count = %d; want %v and 1", ids, count, want)
	}
}

func TestListFilterQueryPhone(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	a := searchOrder(t, s.db, s.ivan, s.product, base, StatusNew, PaymentCash)
	searchOrder(t, s.db, s.petr, s.product, base.Add(time.Hour), StatusNew, PaymentCash)
	c := searchOrder(t, s.db, s.ivan, s.product, base.Add(2*time.Hour), StatusNew, PaymentCash)

	ids, count := listIDs(t, s.db, ListFilter{Query: "1234"}, 10, 0)
	if want := []int64{c, a}; !reflect.DeepEqual(ids, want) || count != 2 {
		t.Errorf("ids = %v, count = %d; want %v and 2", ids, count, want)
	}
}

func TestListFilterQueryName(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	a := searchOrder(t, s.db, s.ivan, s.product, base, StatusNew, PaymentCash)
	searchOrder(t, s.db, s.petr, s.product, base.Add(time.Hour), StatusNew, PaymentCash)

	ids, count := listIDs(t, s.db, ListFilter{Query: "иван"}, 10, 0)
	if want := []int64{a}; !reflect.DeepEqual(ids, want) || count != 1 {
		t.Errorf("ids = %v, count = %d; want %v and 1", ids, count, want)
	}
}

func TestListFilterQueryEscape(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	searchOrder(t, s.db, s.ivan, s.product, base, StatusNew, PaymentCash)

	for _, q := range []string{"%", "_", `\`, "И_ан"} {
		ids, count := listIDs(t, s.db, ListFilter{Query: q}, 10, 0)
		if len(ids) != 0 || count != 0 {
			t.Errorf("Query %q: ids = %v, count = %d; want nothing", q, ids, count)
		}
	}
}

func TestListFilterQueryLongDigits(t *testing.T) {
	s := newSearchSeed(t)
	searchOrder(t, s.db, s.ivan, s.product, time.Date(2026, 10, 20, 10, 0, 0, 0, moscow), StatusNew, PaymentCash)

	// 19 digits do not fit the id branch; they are matched against the phone and the name only.
	ids, count := listIDs(t, s.db, ListFilter{Query: "9999999999999999999"}, 10, 0)
	if len(ids) != 0 || count != 0 {
		t.Errorf("ids = %v, count = %d; want nothing", ids, count)
	}
}

func TestListFilterDates(t *testing.T) {
	s := newSearchSeed(t)
	searchOrder(t, s.db, s.ivan, s.product, time.Date(2026, 10, 20, 23, 30, 0, 0, moscow), StatusNew, PaymentCash)
	mid := searchOrder(t, s.db, s.ivan, s.product, time.Date(2026, 10, 21, 10, 0, 0, 0, moscow), StatusNew, PaymentCash)
	searchOrder(t, s.db, s.ivan, s.product, time.Date(2026, 10, 22, 0, 0, 0, 0, moscow), StatusNew, PaymentCash)

	from := time.Date(2026, 10, 21, 0, 0, 0, 0, moscow)
	before := time.Date(2026, 10, 22, 0, 0, 0, 0, moscow)
	ids, count := listIDs(t, s.db, ListFilter{From: &from, Before: &before}, 10, 0)
	if want := []int64{mid}; !reflect.DeepEqual(ids, want) || count != 1 {
		t.Errorf("ids = %v, count = %d; want %v and 1", ids, count, want)
	}
}

func TestListFilterStatusAndPayment(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	searchOrder(t, s.db, s.ivan, s.product, base, StatusPaid, PaymentCash)
	want := searchOrder(t, s.db, s.ivan, s.product, base.Add(time.Hour), StatusPaid, PaymentOnline)
	searchOrder(t, s.db, s.ivan, s.product, base.Add(2*time.Hour), StatusNew, PaymentOnline)

	ids, count := listIDs(t, s.db, ListFilter{Status: StatusPaid, PaymentType: PaymentOnline}, 10, 0)
	if !reflect.DeepEqual(ids, []int64{want}) || count != 1 {
		t.Errorf("ids = %v, count = %d; want [%d] and 1", ids, count, want)
	}
}

func TestListFilterPaging(t *testing.T) {
	s := newSearchSeed(t)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, moscow)
	var all []int64
	for i := 0; i < 5; i++ {
		all = append(all, searchOrder(t, s.db, s.ivan, s.product, base.Add(time.Duration(i)*time.Hour), StatusPaid, PaymentCash))
	}
	searchOrder(t, s.db, s.ivan, s.product, base.Add(10*time.Hour), StatusNew, PaymentCash)

	ids, count := listIDs(t, s.db, ListFilter{Status: StatusPaid}, 2, 2)
	// newest first: all[4], all[3], all[2], all[1], all[0]
	if want := []int64{all[2], all[1]}; !reflect.DeepEqual(ids, want) || count != 5 {
		t.Errorf("ids = %v, count = %d; want %v and 5", ids, count, want)
	}
}

type searchEnv struct {
	router http.Handler
	admin  string
	ivan   []int64
	petr   int64
}

func newSearchEnv(t *testing.T) *searchEnv {
	t.Helper()
	now := time.Date(2026, 10, 22, 15, 0, 0, 0, moscow)
	e := newSvcEnv(t, now)
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(now))
	r := chi.NewRouter()
	NewHandler(e.svc, tokens, defaultOpts()).Routes(r)

	ivan := testutil.InsertUser(t, e.db, "Иван", "+79991234567", auth.RoleUser)
	petr := testutil.InsertUser(t, e.db, "Пётр", "+79997654321", auth.RoleUser)
	admin := testutil.InsertUser(t, e.db, "Админ", "+79990000000", auth.RoleAdmin)
	env := &searchEnv{router: r, admin: issue(t, tokens, admin, auth.RoleAdmin)}
	env.ivan = append(env.ivan,
		searchOrder(t, e.db, ivan, e.productA, time.Date(2026, 10, 20, 23, 30, 0, 0, moscow), StatusNew, PaymentCash),
		searchOrder(t, e.db, ivan, e.productA, time.Date(2026, 10, 21, 10, 0, 0, 0, moscow), StatusNew, PaymentCash),
	)
	env.petr = searchOrder(t, e.db, petr, e.productB, time.Date(2026, 10, 22, 9, 0, 0, 0, moscow), StatusNew, PaymentCash)
	return env
}

func (e *searchEnv) ids(t *testing.T, query string) []int64 {
	t.Helper()
	rec := testutil.DoJSON(t, e.router, http.MethodGet, "/orders"+query, e.admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /orders%s = %d, want 200; body %s", query, rec.Code, rec.Body.String())
	}
	var got listBody
	testutil.Decode(t, rec, &got)
	ids := []int64{}
	for _, o := range got.Orders {
		ids = append(ids, int64(o["id"].(float64)))
	}
	if got.Count != len(ids) {
		t.Errorf("count = %d, orders = %d", got.Count, len(ids))
	}
	return ids
}

func TestListOrdersSearchQuery(t *testing.T) {
	e := newSearchEnv(t)
	got := e.ids(t, "?q="+url.QueryEscape("Иван"))
	if want := []int64{e.ivan[1], e.ivan[0]}; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestListOrdersSearchDates(t *testing.T) {
	e := newSearchEnv(t)
	if got, want := e.ids(t, "?from=2026-10-21&to=2026-10-21"), []int64{e.ivan[1]}; !reflect.DeepEqual(got, want) {
		t.Errorf("from+to: ids = %v, want %v", got, want)
	}
	if got, want := e.ids(t, "?from=2026-10-21"), []int64{e.petr, e.ivan[1]}; !reflect.DeepEqual(got, want) {
		t.Errorf("from only: ids = %v, want %v", got, want)
	}
	if got, want := e.ids(t, "?to=2026-10-20"), []int64{e.ivan[0]}; !reflect.DeepEqual(got, want) {
		t.Errorf("to only: ids = %v, want %v", got, want)
	}
}

func TestListOrdersSearchStatusAndPayment(t *testing.T) {
	e := newSearchEnv(t)
	if got := e.ids(t, "?status=paid"); len(got) != 0 {
		t.Errorf("status=paid: ids = %v, want none", got)
	}
	if got := e.ids(t, "?status=new&paymentType=cash"); len(got) != 3 {
		t.Errorf("status=new&paymentType=cash: ids = %v, want 3", got)
	}
}

func TestListOrdersSearchInvalid(t *testing.T) {
	e := newSearchEnv(t)
	tests := []struct {
		name, query, field, message string
	}{
		{"from format", "from=21.10.2026", "from", "дата в формате ГГГГ-ММ-ДД"},
		{"to month", "to=2026-13-01", "to", "дата в формате ГГГГ-ММ-ДД"},
		{"to before from", "from=2026-10-22&to=2026-10-21", "to", "не раньше даты from"},
		{"status", "status=done", "status", "недопустимое значение"},
		{"payment type", "paymentType=card", "paymentType", "недопустимое значение"},
		{"long q", "q=" + strings.Repeat("я", 101), "q", "не более 100 символов"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/orders?" + tt.query
			want := `{"statusCode":400,"message":"Некорректные параметры поиска","errors":{"` + tt.field + `":"` + tt.message + `"}}`
			rec := testutil.DoJSON(t, e.router, http.MethodGet, path, e.admin, nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("GET %s = %d, want 400", path, rec.Code)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != want {
				t.Errorf("body = %s, want %s", got, want)
			}
		})
	}
}

func TestListOrdersSearchAllErrorsReported(t *testing.T) {
	e := newSearchEnv(t)
	rec := testutil.DoJSON(t, e.router, http.MethodGet, "/orders?from=x&status=done&paymentType=card", e.admin, nil)
	want := `{"statusCode":400,"message":"Некорректные параметры поиска","errors":{"from":"дата в формате ГГГГ-ММ-ДД","paymentType":"недопустимое значение","status":"недопустимое значение"}}`
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusBadRequest || got != want {
		t.Errorf("status = %d, body = %s; want 400 and %s", rec.Code, got, want)
	}
}

func TestListOrdersSearchQueryMaxLength(t *testing.T) {
	e := newSearchEnv(t)
	if got := e.ids(t, "?q="+strings.Repeat("я", 100)); len(got) != 0 {
		t.Errorf("ids = %v, want none", got)
	}
}

func TestListOrdersSearchEmptyParams(t *testing.T) {
	e := newSearchEnv(t)
	got := e.ids(t, "?status=&q=&from=&to=&paymentType=")
	if want := []int64{e.petr, e.ivan[1], e.ivan[0]}; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}
