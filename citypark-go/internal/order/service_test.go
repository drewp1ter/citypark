package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const cashURL = "http://cash"

type fakeNotifier struct{ created, paid, confirmed []Order }

func (f *fakeNotifier) OrderCreated(_ context.Context, o Order) { f.created = append(f.created, o) }
func (f *fakeNotifier) OrderPaid(_ context.Context, o Order)    { f.paid = append(f.paid, o) }
func (f *fakeNotifier) OrderConfirmed(_ context.Context, o Order) {
	f.confirmed = append(f.confirmed, o)
}

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
	e := &svcEnv{db: db, notifier: &fakeNotifier{}, payments: &fakePayments{form: PaymentForm{Amount: "749.00", Signature: "sig"}}}
	e.category = testutil.InsertCategory(t, db, testutil.Category{Name: "Мангал"})
	e.productA = testutil.InsertProduct(t, db, testutil.Product{CategoryID: e.category, Name: "Шашлык", PriceKopecks: 59900})
	e.productB = testutil.InsertProduct(t, db, testutil.Product{CategoryID: e.category, Name: "Лаваш", PriceKopecks: 10000})
	testutil.InsertDeliveryPrice(t, db, "Сочи", 15000)
	e.svc = NewService(db, clock.Fixed(now), e.notifier, e.payments, Options{ThankYouCash: cashURL})
	return e
}

func (e *svcEnv) input(paymentType string, withAddress bool) CreateInput {
	in := CreateInput{
		Phone:       "8 (999) 111-22-33",
		Name:        "  Пётр ",
		Comment:     " без лука ",
		PaymentType: paymentType,
		Items:       []ItemInput{{ProductID: e.productA, Amount: 1}},
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

func TestCreateCash(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(PaymentCash, true))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.OrderID == 0 || res.RedirectURL != cashURL {
		t.Errorf("result = %+v, want order id and redirect %q", res, cashURL)
	}
	if res.Amount != "" || res.MntSignature != "" {
		t.Errorf("result has payment fields: %+v", res)
	}

	if n := e.count(t, `SELECT count(*) FROM orders`); n != 1 {
		t.Fatalf("orders = %d, want 1", n)
	}
	o := e.order(t, res.OrderID)
	if o.Status != StatusNew {
		t.Errorf("Status = %q, want new", o.Status)
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

	if len(e.notifier.created) != 1 || e.notifier.created[0].ID != o.ID {
		t.Errorf("OrderCreated calls = %+v, want one for order %d", e.notifier.created, o.ID)
	}
	if len(e.payments.requests) != 0 {
		t.Errorf("payments called: %+v", e.payments.requests)
	}
	if n := e.count(t, `SELECT count(*) FROM user_addresses WHERE user_id = $1`, o.Customer.ID); n != 1 {
		t.Errorf("user_addresses links = %d, want 1", n)
	}
}

func TestCreatePickup(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
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

	res, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
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
	in := e.input(PaymentCash, false)
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
	in := e.input(PaymentCash, false)
	in.Phone = "123"
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Неверный номер телефона.")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateUnknownCity(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(PaymentCash, true)
	in.Address.City = "Атлантида"
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusBadRequest, "Указано не существующее имя населенного пункта.")
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
}

func TestCreateUnknownProduct(t *testing.T) {
	e := newSvcEnv(t, friday)
	in := e.input(PaymentCash, true)
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
	in := e.input(PaymentCash, false)
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
	_, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
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
	_, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Мангал" ограничено с 9:00 до 21:00.`)
}

func TestCreateProductClosedDay(t *testing.T) {
	saturday := time.Date(2024, 5, 25, 15, 0, 0, 0, time.UTC)
	e := newSvcEnv(t, saturday)
	if _, err := e.db.Exec(`UPDATE products SET sale_days = '1111100' WHERE id = $1`, e.productA); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
	wantHTTPError(t, err, http.StatusServiceUnavailable,
		`Возможность заказа "Шашлык" доступна только в следующие дни недели: Пн, Вт, Ср, Чт, Пт`)
}

func TestCreateProductClosedTime(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC))
	if _, err := e.db.Exec(`UPDATE products SET sale_start = 600, sale_end = 1200 WHERE id = $1`, e.productA); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.Create(context.Background(), e.input(PaymentCash, false))
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Шашлык" ограничено с 10:00 до 20:00.`)
}

func TestCreateFirstClosedProductWins(t *testing.T) {
	e := newSvcEnv(t, time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC))
	if _, err := e.db.Exec(`UPDATE products SET sale_start = 600, sale_end = 1200 WHERE id = ANY($1)`,
		[]int64{e.productA, e.productB}); err != nil {
		t.Fatal(err)
	}
	in := e.input(PaymentCash, false)
	in.Items = []ItemInput{{ProductID: e.productB, Amount: 1}, {ProductID: e.productA, Amount: 1}}
	_, err := e.svc.Create(context.Background(), in)
	wantHTTPError(t, err, http.StatusServiceUnavailable, `Время заказа "Лаваш" ограничено с 10:00 до 20:00.`)
}

func TestCreateOnline(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(PaymentOnline, true))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.OrderID == 0 || res.Amount != "749.00" || res.MntSignature != "sig" || res.RedirectURL != "" {
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
	if len(e.notifier.created) != 0 {
		t.Errorf("notifier called: %+v", e.notifier.created)
	}
}

func TestCreateOnlinePickup(t *testing.T) {
	e := newSvcEnv(t, friday)
	res, err := e.svc.Create(context.Background(), e.input(PaymentOnline, false))
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
	first, err := e.svc.Create(context.Background(), e.input(PaymentOnline, true))
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second, err := e.svc.Create(context.Background(), e.input(PaymentOnline, true))
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

func TestCreateOnlineNotReusedForCash(t *testing.T) {
	e := newSvcEnv(t, friday)
	first, err := e.svc.Create(context.Background(), e.input(PaymentCash, true))
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second, err := e.svc.Create(context.Background(), e.input(PaymentCash, true))
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if first.OrderID == second.OrderID {
		t.Errorf("OrderIDs are both %d, want different", first.OrderID)
	}
	if n := e.count(t, `SELECT count(*) FROM orders`); n != 2 {
		t.Errorf("orders = %d, want 2", n)
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
	cash := jsonKeys(t, CreateResult{OrderID: 1, RedirectURL: "http://x"})
	if want := []string{"orderId", "redirectUrl"}; !slices.Equal(cash, want) {
		t.Errorf("cash keys = %v, want %v", cash, want)
	}
	online := jsonKeys(t, CreateResult{OrderID: 1, Amount: "1.00", MntSignature: "s"})
	if want := []string{"amount", "mntSignature", "orderId"}; !slices.Equal(online, want) {
		t.Errorf("online keys = %v, want %v", online, want)
	}
}
