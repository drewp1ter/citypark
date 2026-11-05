package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type fakeNotifier struct{ pending, paid []Order }

func (f *fakeNotifier) OrderPending(_ context.Context, o Order) { f.pending = append(f.pending, o) }
func (f *fakeNotifier) OrderPaid(_ context.Context, o Order)    { f.paid = append(f.paid, o) }

type fakePayments struct {
	requests []PaymentRequest
	form     PaymentForm
	valid    bool
}

func (f *fakePayments) Form(r PaymentRequest) PaymentForm {
	f.requests = append(f.requests, r)
	return f.form
}

func (f *fakePayments) Verify(PaymentCallback) bool { return f.valid }

type svcEnv struct {
	db       *sqlx.DB
	svc      *Service
	notifier *fakeNotifier
	payments *fakePayments
	category int64
	productA int64 // 59900
	productB int64 // 10000
}

// friday is Friday 2024-05-24 15:00.
var friday = time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC)

func newSvcEnv(t *testing.T, now time.Time) *svcEnv {
	t.Helper()
	db := testutil.NewDB(t)
	var settingsData []byte
	if err := db.Get(&settingsData, `SELECT data FROM site_settings WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		restore, err := sqlx.Connect("pgx", os.Getenv("TEST_DB_DSN"))
		if err != nil {
			t.Error(err)
			return
		}
		defer restore.Close()
		if _, err := restore.Exec(`INSERT INTO site_settings (id, data) VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET data=EXCLUDED.data`, settingsData); err != nil {
			t.Error(err)
		}
	})
	db.MustExec(`UPDATE site_settings SET data=jsonb_set(data::jsonb, '{promoCodesEnabled}', 'true') WHERE id=1`)
	e := &svcEnv{db: db, notifier: &fakeNotifier{}, payments: &fakePayments{form: PaymentForm{Amount: "749.00", Signature: "sig", URL: "https://pay.test/?x"}}}
	e.category = testutil.InsertCategory(t, db, testutil.Category{Name: "Мангал"})
	e.productA = testutil.InsertProduct(t, db, testutil.Product{CategoryID: e.category, Name: "Шашлык", PriceKopecks: 59900})
	e.productB = testutil.InsertProduct(t, db, testutil.Product{CategoryID: e.category, Name: "Лаваш", PriceKopecks: 10000})
	testutil.InsertDeliveryPrice(t, db, "Сочи", 15000)
	e.svc = NewService(db, clock.Fixed(now), e.notifier, e.payments)
	e.svc.SetPaymentHours(defaultOpts().Hours)
	e.svc.ConfigurePromo("00000000-0000-4000-8000-000000000001")
	return e
}

func (e *svcEnv) input(withAddress bool) CreateInput {
	in := CreateInput{
		GuestCheckoutID: "test-checkout",
		Phone:           "8 (999) 111-22-33",
		Name:            "  Пётр ",
		Comment:         " без лука ",
		Items:           []ItemInput{{ProductID: e.productA, Amount: 1}},
	}
	if withAddress {
		in.Address = &address.Input{City: "Сочи", Address: "Ленина, 1"}
	}
	return in
}

func (e *svcEnv) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	return testutil.Count(t, e.db, query, args...)
}

func (e *svcEnv) order(t *testing.T, id int64) Order {
	t.Helper()
	o, err := NewStore(e.db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID(%d): %v", id, err)
	}
	return o
}

func wantHTTPError(t *testing.T, err error, status int, message string) {
	t.Helper()
	var he *httpx.Error
	if !errors.As(err, &he) {
		t.Fatalf("err = %v, want *httpx.Error", err)
	}
	if he.Status != status || he.Message != message {
		t.Errorf("error = %d %q, want %d %q", he.Status, he.Message, status, message)
	}
}

func TestCreatePickup(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(false))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := e.count(t, `SELECT count(*) FROM orders WHERE id = $1 AND address_id IS NULL`, res.OrderID); n != 1 {
		t.Errorf("order with NULL address_id = %d, want 1", n)
	}
	o := e.order(t, res.OrderID)
	if o.DeliveryPrice != 0 || o.Address != nil {
		t.Errorf("DeliveryPrice = %d, Address = %+v; want 0 and nil", o.DeliveryPrice, o.Address)
	}
}

func TestCreateExistingCustomer(t *testing.T) {
	e := newSvcEnv(t, friday)
	id := testutil.InsertUser(t, e.db, "Старое имя", "+79991112233", "user")

	res, err := e.svc.Create(context.Background(), e.input(false))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := e.count(t, `SELECT count(*) FROM users`); n != 1 {
		t.Errorf("users = %d, want 1", n)
	}
	o := e.order(t, res.OrderID)
	if o.Customer.ID != id || o.Customer.Name != "Пётр" {
		t.Errorf("Customer = %+v, want id %d and name Пётр", o.Customer, id)
	}
}

func TestCreateMergesDuplicateItems(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(false)
	in.Items = []ItemInput{{ProductID: e.productA, Amount: 1}, {ProductID: e.productB, Amount: 1}, {ProductID: e.productA, Amount: 2}}

	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	o := e.order(t, res.OrderID)
	if len(o.Items) != 2 {
		t.Fatalf("Items = %+v, want 2", o.Items)
	}
	amounts := map[int64]int{}
	for _, it := range o.Items {
		amounts[it.ProductID] = it.Amount
	}
	if amounts[e.productA] != 3 || amounts[e.productB] != 1 {
		t.Errorf("amounts = %v, want A=3 B=1", amounts)
	}
	wantHash := ItemsHash([]ItemInput{{ProductID: e.productA, Amount: 3}, {ProductID: e.productB, Amount: 1}})
	if o.ItemsHash != wantHash {
		t.Errorf("ItemsHash = %q, want %q", o.ItemsHash, wantHash)
	}
}

func TestCreateBadPhone(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(false)
	in.Phone = "123"
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Неверный номер телефона.")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateUnknownCity(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(true)
	in.Address.City = "Атлантида"
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Указано не существующее имя населенного пункта.")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateUnknownProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(true)
	in.Items = []ItemInput{{ProductID: e.productA, Amount: 1}, {ProductID: 999, Amount: 1}}
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Продукт не найден.")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
	if n := e.count(t, `SELECT count(*) FROM users`); n != 0 {
		t.Errorf("users = %d, want 0 (rolled back)", n)
	}
}

func TestCreateDeletedProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	deleted := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, Name: "Удалён", PriceKopecks: 100, IsDeleted: true})
	in := e.input(false)
	in.Items = []ItemInput{{ProductID: deleted, Amount: 1}}
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Продукт не найден.")
}

func TestCreateCategoryClosedDay(t *testing.T) {
	saturday := time.Date(2024, 5, 25, 15, 0, 0, 0, time.UTC)
	e := newSvcEnv(t, saturday)
	if _, err := e.db.Exec(`UPDATE categories SET sale_days = '1111100' WHERE id = $1`, e.category); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable,
		`Возможность заказа "Мангал" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт`)
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateCategoryClosedTime(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC))
	if _, err := e.db.Exec(`UPDATE categories SET sale_start = 540, sale_end = 1260 WHERE id = $1`, e.category); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Мангал" ограничено с 9:00 до 21:00.`)
}

func TestCreateProductClosedDay(t *testing.T) {
	saturday := time.Date(2024, 5, 25, 15, 0, 0, 0, time.UTC)
	e := newSvcEnv(t, saturday)
	if _, err := e.db.Exec(`UPDATE products SET sale_days = '1111100' WHERE id = $1`, e.productA); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable,
		`Возможность заказа "Шашлык" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт`)
}

func TestCreateProductClosedTime(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC))
	if _, err := e.db.Exec(`UPDATE products SET sale_start = 600, sale_end = 1200 WHERE id = $1`, e.productA); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Шашлык" ограничено с 10:00 до 20:00.`)
}

func TestCreateFirstClosedProductWins(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC))
	if _, err := e.db.Exec(`UPDATE products SET sale_start = 600, sale_end = 1200 WHERE id = ANY($1)`,
		[]int64{e.productA, e.productB}); err != nil {
		t.Fatal(err)
	}
	in := e.input(false)
	in.Items = []ItemInput{{ProductID: e.productB, Amount: 1}, {ProductID: e.productA, Amount: 1}}
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Лаваш" ограничено с 10:00 до 20:00.`)
}

func TestCreateOnline(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.OrderID == 0 || res.RedirectURL != "https://pay.test/?x" {
		t.Errorf("result = %+v", res)
	}
	o := e.order(t, res.OrderID)
	if o.Status != StatusPending || o.PaymentID != "" {
		t.Errorf("Status = %q, PaymentID = %q; want pending and empty", o.Status, o.PaymentID)
	}
	want := []PaymentRequest{{OrderID: res.OrderID, Amount: 74900}}
	if !slices.Equal(e.payments.requests, want) {
		t.Errorf("payment requests = %+v, want %+v", e.payments.requests, want)
	}
	if o.DeliveryPrice != 15000 {
		t.Errorf("DeliveryPrice = %d, want 15000", o.DeliveryPrice)
	}
	if o.Customer.Name != "Пётр" || o.Customer.Phone != "+79991112233" || o.Comment != "без лука" {
		t.Errorf("customer/comment = %+v / %q", o.Customer, o.Comment)
	}
	if o.Address == nil || o.Address.City != "Сочи" || o.Address.Address != "Ленина, 1" {
		t.Errorf("Address = %+v", o.Address)
	}
	if len(o.Items) != 1 || o.Items[0].ProductID != e.productA || o.Items[0].Amount != 1 || o.Items[0].OfferedPrice != 59900 {
		t.Errorf("Items = %+v", o.Items)
	}
	if n := e.count(t, `SELECT count(*) FROM user_addresses WHERE user_id = $1`, o.Customer.ID); n != 0 {
		t.Errorf("user_addresses links = %d, want 0", n)
	}
}

func TestCreateNotifiesPending(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(e.notifier.pending) != 1 {
		t.Fatalf("OrderPending calls = %d, want 1", len(e.notifier.pending))
	}
	if got := e.notifier.pending[0]; got.ID != res.OrderID || got.Status != StatusPending {
		t.Errorf("pending order = id %d status %q, want id %d status pending", got.ID, got.Status, res.OrderID)
	}
}

func TestCreateOnlineDuplicateNoEvent(t *testing.T) {
	e := newSvcEnv(t, friday)
	for i := 0; i < 2; i++ {
		if _, err := e.svc.Create(context.Background(), e.input(true)); err != nil {
			t.Fatalf("Create #%d: %v", i+1, err)
		}
	}
	if len(e.notifier.pending) != 1 {
		t.Errorf("OrderPending calls = %d, want 1 (a reused duplicate does not fire)", len(e.notifier.pending))
	}
}

func TestCreateOnlinePickup(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(false))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	want := []PaymentRequest{{OrderID: res.OrderID, Amount: money.Kopecks(59900)}}
	if !slices.Equal(e.payments.requests, want) {
		t.Errorf("payment requests = %+v, want %+v", e.payments.requests, want)
	}
}

func TestCreateOnlineReuse(t *testing.T) {
	e := newSvcEnv(t, friday)
	first, err := e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second, err := e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if first.OrderID != second.OrderID {
		t.Errorf("OrderIDs = %d and %d, want the same", first.OrderID, second.OrderID)
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 1 {
		t.Errorf("orders = %d, want 1", n)
	}
	want := []PaymentRequest{{OrderID: first.OrderID, Amount: 74900}, {OrderID: first.OrderID, Amount: 74900}}
	if !slices.Equal(e.payments.requests, want) {
		t.Errorf("payment requests = %+v, want %+v", e.payments.requests, want)
	}
	if o := e.order(t, first.OrderID); o.Status != StatusPending {
		t.Errorf("Status = %q, want pending", o.Status)
	}
}

func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func TestCreateResultJSON(t *testing.T) {
	got := jsonKeys(t, CreateResult{OrderID: 1, RedirectURL: "https://pay"})
	if want := []string{"orderId", "redirectUrl"}; !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestDeliveryCost(t *testing.T) {
	tests := []struct {
		name     string
		dp       deliveryprice.DeliveryPrice
		subtotal money.Kopecks
		want     money.Kopecks
	}{
		{"below the threshold", deliveryprice.DeliveryPrice{Price: 10000, FreeFrom: 50000}, 49999, 10000},
		{"at the threshold", deliveryprice.DeliveryPrice{Price: 10000, FreeFrom: 50000}, 50000, 0},
		{"above the threshold", deliveryprice.DeliveryPrice{Price: 10000, FreeFrom: 50000}, 60000, 0},
		{"never free", deliveryprice.DeliveryPrice{Price: 25000, FreeFrom: 0}, 500000, 25000},
		{"pickup", deliveryprice.DeliveryPrice{}, 100000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryCost(tt.dp, tt.subtotal); got != tt.want {
				t.Errorf("deliveryCost = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCreateFreeDelivery(t *testing.T) {
	e := newSvcEnv(t, friday)
	if _, err := e.db.Exec(`INSERT INTO delivery_prices (name, price_kopecks, free_from_kopecks)
		VALUES ('Белореченск', 10000, 50000)`); err != nil {
		t.Fatalf("insert delivery price: %v", err)
	}
	product := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, Name: "Пицца", PriceKopecks: 25000})
	for _, tt := range []struct {
		amount int
		want   money.Kopecks
	}{{2, 0}, {1, 10000}} {
		in := e.input(true)
		in.Address.City = "Белореченск"
		in.Items = []ItemInput{{ProductID: product, Amount: tt.amount}}
		res, err := e.svc.Create(context.Background(), in)
		if err != nil {
			t.Fatalf("Create x%d: %v", tt.amount, err)
		}
		if got := e.order(t, res.OrderID).DeliveryPrice; got != tt.want {
			t.Errorf("x%d: DeliveryPrice = %d, want %d", tt.amount, got, tt.want)
		}
	}
}

func TestCreateDeliveryTime(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(true)
	in.DeliveryTime = "18:30"
	res, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := e.order(t, res.OrderID).DeliveryTime; got != "18:30" {
		t.Errorf("DeliveryTime = %q, want 18:30", got)
	}

	res, err = e.svc.Create(context.Background(), e.input(true))
	if err != nil {
		t.Fatalf("Create without time: %v", err)
	}
	if got := e.order(t, res.OrderID).DeliveryTime; got != "" {
		t.Errorf("DeliveryTime = %q, want empty", got)
	}
}

func TestCreateDuplicateOtherTime(t *testing.T) {
	e := newSvcEnv(t, friday)
	create := func(deliveryTime string) int64 {
		in := e.input(true)
		in.DeliveryTime = deliveryTime
		res, err := e.svc.Create(context.Background(), in)
		if err != nil {
			t.Fatalf("Create(%q): %v", deliveryTime, err)
		}
		return res.OrderID
	}
	first := create("18:30")
	if other := create("19:00"); other == first {
		t.Errorf("another time reused order %d, want a new order", first)
	}
	if same := create("18:30"); same != first {
		t.Errorf("same time gave order %d, want %d", same, first)
	}
}

func TestCreateBlockedProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	if _, err := e.db.Exec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, e.productA); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, "Заказ блюда «Шашлык» временно недоступен.")
	if n := e.count(t, "SELECT count(*) FROM orders"); n != 0 {
		t.Fatalf("orders=%d", n)
	}
}
func TestDuplicateBlockedProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	if _, err := e.svc.Create(context.Background(), e.input(false)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`UPDATE products SET is_order_blocked=true WHERE id=$1`, e.productA); err != nil {
		t.Fatal(err)
	}
	calls := len(e.payments.requests)
	_, err := e.svc.Create(context.Background(), e.input(false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, "Заказ блюда «Шашлык» временно недоступен.")
	if len(e.payments.requests) != calls || e.count(t, "SELECT count(*) FROM orders") != 1 {
		t.Fatal("retry changed orders/payment")
	}
}

func (f *fakePayments) CheckResponse(id, code, amount string) moneta.CheckResponse {
	return moneta.New("100", "secret", false).CheckResponse(id, code, amount)
}
