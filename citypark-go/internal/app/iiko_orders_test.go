package app_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/app"
	"github.com/drewp1ter/citypark-go/internal/iiko"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const alertChat = "333"

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// fakeCreator records every DeliveryOrder and answers from a script. When the script runs out
// the last answer repeats.
type fakeCreator struct {
	mu      sync.Mutex
	got     []iiko.DeliveryOrder
	groups  []string
	answers []error // nil = success
}

func (f *fakeCreator) CreateDelivery(_ context.Context, group string, o iiko.DeliveryOrder) (iiko.DeliveryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, o)
	f.groups = append(f.groups, group)
	i := min(len(f.got), len(f.answers)) - 1
	if i >= 0 && f.answers[i] != nil {
		return iiko.DeliveryResult{}, f.answers[i]
	}
	return iiko.DeliveryResult{ID: "iiko-order-1"}, nil
}

func (f *fakeCreator) calls() []iiko.DeliveryOrder {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]iiko.DeliveryOrder(nil), f.got...)
}

func iikoSettings() app.IikoOrderSettings {
	return app.IikoOrderSettings{
		TerminalGroupID: "tg", PaymentTypeID: "pt", PaymentTypeKind: "Card",
		City: "Белореченск", Street: "Гоголя", House: "61",
		Delays: []time.Duration{0, 0},
	}
}

func fullOrder() order.Order {
	return order.Order{
		ID:            7,
		Comment:       "Без лука",
		DeliveryTime:  "18:30",
		Total:         79000,
		DeliveryPrice: 15000,
		Customer:      order.Customer{ID: 3, Name: "Иван", Phone: "+79991234567"},
		Address:       &order.Address{ID: 1, City: "Белореченск", Address: "ул. Мира, 5"},
		Items: []order.Item{
			{ProductID: 1, Amount: 2, OfferedPrice: 29000, IikoID: "iiko-a"},
			{ProductID: 2, Amount: 1, OfferedPrice: 21000, IikoID: "iiko-b"},
		},
	}
}

func TestDeliveryOrder(t *testing.T) {
	pickup := fullOrder()
	pickup.Address, pickup.DeliveryTime, pickup.DeliveryPrice, pickup.Comment = nil, "", 0, ""
	noID := fullOrder()
	noID.Items = []order.Item{{ProductID: 3, Amount: 1, OfferedPrice: 100}}

	tests := []struct {
		name    string
		in      order.Order
		want    iiko.DeliveryOrder
		wantErr string
	}{
		{
			name: "delivery",
			in:   fullOrder(),
			want: iiko.DeliveryOrder{
				ID: "guid", ExternalNumber: "7", Phone: "+79991234567",
				OrderServiceType: iiko.ServiceDeliveryByCourier,
				DeliveryPoint: &iiko.DeliveryPoint{
					Address: iiko.DeliveryAddress{Type: "legacy", Street: iiko.DeliveryStreet{Name: "Гоголя", City: "Белореченск"}, House: "61"},
					Comment: "Белореченск, ул. Мира, 5",
				},
				Comment:  "Заказ с сайта №7, оплачен онлайн\nКо времени: 18:30\nАдрес: Белореченск, ул. Мира, 5\nДоставка: 150 ₽ (оплачена)\nКомментарий: Без лука",
				Customer: iiko.DeliveryCustomer{Type: "one-time", Name: "Иван"},
				Items: []iiko.DeliveryItem{
					{Type: "Product", ProductID: "iiko-a", Amount: 2, Price: 290},
					{Type: "Product", ProductID: "iiko-b", Amount: 1, Price: 210},
				},
				Payments: []iiko.DeliveryPayment{{PaymentTypeKind: "Card", Sum: 790, PaymentTypeID: "pt", IsProcessedExternally: true}},
			},
		},
		{
			name: "pickup",
			in:   pickup,
			want: iiko.DeliveryOrder{
				ID: "guid", ExternalNumber: "7", Phone: "+79991234567",
				OrderServiceType: iiko.ServiceDeliveryByClient,
				Comment:          "Заказ с сайта №7, оплачен онлайн",
				Customer:         iiko.DeliveryCustomer{Type: "one-time", Name: "Иван"},
				Items: []iiko.DeliveryItem{
					{Type: "Product", ProductID: "iiko-a", Amount: 2, Price: 290},
					{Type: "Product", ProductID: "iiko-b", Amount: 1, Price: 210},
				},
				Payments: []iiko.DeliveryPayment{{PaymentTypeKind: "Card", Sum: 790, PaymentTypeID: "pt", IsProcessedExternally: true}},
			},
		},
		{name: "item without iiko id", in: noID, wantErr: "product 3 has no iiko id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := app.DeliveryOrderForTest(tc.in, iikoSettings(), "guid")
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("deliveryOrder: %v", err)
			}
			assertDeliveryOrder(t, got, tc.want)
		})
	}
}

func assertDeliveryOrder(t *testing.T, got, want iiko.DeliveryOrder) {
	t.Helper()
	gp, wp := got.DeliveryPoint, want.DeliveryPoint
	got.DeliveryPoint, want.DeliveryPoint = nil, nil
	if (gp == nil) != (wp == nil) || (gp != nil && *gp != *wp) {
		t.Errorf("DeliveryPoint = %+v, want %+v", gp, wp)
	}
	if got.ID != want.ID || got.ExternalNumber != want.ExternalNumber || got.Phone != want.Phone ||
		got.OrderServiceType != want.OrderServiceType || got.Comment != want.Comment || got.Customer != want.Customer {
		t.Errorf("order = %+v, want %+v", got, want)
	}
	if len(got.Items) != len(want.Items) {
		t.Fatalf("items = %+v, want %+v", got.Items, want.Items)
	}
	for i := range got.Items {
		if got.Items[i] != want.Items[i] {
			t.Errorf("item %d = %+v, want %+v", i, got.Items[i], want.Items[i])
		}
	}
	if len(got.Payments) != 1 || got.Payments[0] != want.Payments[0] {
		t.Errorf("payments = %+v, want %+v", got.Payments, want.Payments)
	}
}

// iikoFixture inserts a paid-looking online order whose products have (or lack) iiko ids.
func iikoFixture(t *testing.T, db *sqlx.DB, iikoIDs ...string) order.Order {
	t.Helper()
	userID := testutil.InsertUser(t, db, "Иван", "+79990000000", "user")
	catID := testutil.InsertCategory(t, db, testutil.Category{Name: "Пицца"})
	addrID := testutil.InsertAddress(t, db, userID, "Белореченск", "ул. Мира, 5")
	p := order.InsertParams{CustomerID: userID, AddressID: &addrID, DeliveryPrice: 15000}
	for i, id := range iikoIDs {
		pid := testutil.InsertProduct(t, db, testutil.Product{CategoryID: catID, Name: "П" + string(rune('A'+i)), PriceKopecks: 10000, IikoID: id})
		p.Items = append(p.Items, order.InsertItem{ProductID: pid, Amount: 1, OfferedPrice: 10000})
	}
	id, err := order.NewStore(db).Insert(context.Background(), p)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}
	o, err := order.NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	return o
}

func runIikoOrders(t *testing.T, db *sqlx.DB, s app.IikoOrderSettings, o order.Order, answers ...error) (*fakeCreator, *fakeSender) {
	t.Helper()
	creator := &fakeCreator{answers: answers}
	sender := &fakeSender{}
	n := app.NewIikoOrders(creator, db, sender, alertChat, s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.OrderPaid(context.Background(), o)
	n.Wait()
	return creator, sender
}

func savedIikoID(t *testing.T, db *sqlx.DB, id int64) string {
	t.Helper()
	o, err := order.NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	return o.IikoOrderID
}

func TestIikoOrdersSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a", "iiko-b")
	creator, sender := runIikoOrders(t, db, iikoSettings(), o)

	calls := creator.calls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if creator.groups[0] != "tg" {
		t.Errorf("terminal group = %q, want tg", creator.groups[0])
	}
	if !uuidV4.MatchString(calls[0].ID) {
		t.Errorf("order id %q is not a UUID v4", calls[0].ID)
	}
	if got := savedIikoID(t, db, o.ID); got != "iiko-order-1" {
		t.Errorf("iiko_order_id = %q, want iiko-order-1", got)
	}
	if len(sender.calls()) != 0 {
		t.Errorf("alerts = %v, want none", sender.calls())
	}
}

func TestIikoOrdersRetry(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	fail := &iiko.StatusError{Endpoint: "1/deliveries/create", Status: 500}
	creator, sender := runIikoOrders(t, db, iikoSettings(), o, fail, fail, nil)

	calls := creator.calls()
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(calls))
	}
	if calls[0].ID != calls[1].ID || calls[1].ID != calls[2].ID {
		t.Errorf("ids differ between retries: %q %q %q", calls[0].ID, calls[1].ID, calls[2].ID)
	}
	if got := savedIikoID(t, db, o.ID); got != "iiko-order-1" {
		t.Errorf("iiko_order_id = %q", got)
	}
	if len(sender.calls()) != 0 {
		t.Errorf("alerts = %v, want none", sender.calls())
	}
}

func TestIikoOrdersGivesUp(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	fail := &iiko.StatusError{Endpoint: "1/deliveries/create", Status: 500}
	creator, sender := runIikoOrders(t, db, iikoSettings(), o, fail)

	if n := len(creator.calls()); n != 3 {
		t.Errorf("calls = %d, want 3", n)
	}
	want := []sentMessage{{alertChat, "Заказ №" + strconv.FormatInt(o.ID, 10) + " не создан в iiko: " + fail.Error() + ". Внесите его в iikoFront вручную."}}
	assertSent(t, sender.calls(), want)
	if got := savedIikoID(t, db, o.ID); got != "" {
		t.Errorf("iiko_order_id = %q, want empty", got)
	}
}

func TestIikoOrdersPermanent(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	creator, sender := runIikoOrders(t, db, iikoSettings(), o, &iiko.DeliveryError{Reason: "Product not found"})

	if n := len(creator.calls()); n != 1 {
		t.Errorf("calls = %d, want 1", n)
	}
	want := []sentMessage{{alertChat, "Заказ №" + strconv.FormatInt(o.ID, 10) + " не создан в iiko: iiko: delivery not created: Product not found. Внесите его в iikoFront вручную."}}
	assertSent(t, sender.calls(), want)
}

func TestIikoOrdersPending(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	creator, sender := runIikoOrders(t, db, iikoSettings(), o, iiko.ErrDeliveryPending)

	if n := len(creator.calls()); n != 1 {
		t.Errorf("calls = %d, want 1", n)
	}
	want := []sentMessage{{alertChat, "Заказ №" + strconv.FormatInt(o.ID, 10) + " отправлен в iiko, но создание не подтверждено. Проверьте его в iikoFront."}}
	assertSent(t, sender.calls(), want)
}

func TestIikoOrdersMissingIikoID(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a", "")
	creator, sender := runIikoOrders(t, db, iikoSettings(), o)

	if n := len(creator.calls()); n != 0 {
		t.Errorf("calls = %d, want 0", n)
	}
	calls := sender.calls()
	if len(calls) != 1 || calls[0].ChatID != alertChat || !regexp.MustCompile(`^Заказ №\d+ не создан в iiko: product \d+ has no iiko id\. `).MatchString(calls[0].Text) {
		t.Errorf("alerts = %v", calls)
	}
}

func TestIikoOrdersAlertSendError(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	creator := &fakeCreator{answers: []error{&iiko.DeliveryError{Reason: "x"}}}
	sender := &fakeSender{err: errors.New("telegram down")}
	n := app.NewIikoOrders(creator, db, sender, alertChat, iikoSettings(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.OrderPaid(context.Background(), o)
	n.Wait() // must not panic or hang
	if len(sender.calls()) != 1 {
		t.Errorf("send attempts = %d, want 1", len(sender.calls()))
	}
}

func TestIikoOrdersEmptyAlertChat(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	creator := &fakeCreator{answers: []error{&iiko.DeliveryError{Reason: "x"}}}
	sender := &fakeSender{}
	n := app.NewIikoOrders(creator, db, sender, "", iikoSettings(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.OrderPaid(context.Background(), o)
	n.Wait()
	if len(sender.calls()) != 0 {
		t.Errorf("alerts = %v, want none", sender.calls())
	}
}

func TestIikoOrdersNotConfigured(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	s := iikoSettings()
	s.TerminalGroupID = ""
	creator, sender := runIikoOrders(t, db, s, o)

	if n := len(creator.calls()); n != 0 {
		t.Errorf("calls = %d, want 0", n)
	}
	if len(sender.calls()) != 0 {
		t.Errorf("alerts = %v, want none", sender.calls())
	}
}

func TestIikoOrdersIgnoresOtherEvents(t *testing.T) {
	db := testutil.NewDB(t)
	o := iikoFixture(t, db, "iiko-a")
	creator := &fakeCreator{}
	n := app.NewIikoOrders(creator, db, &fakeSender{}, alertChat, iikoSettings(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	n.OrderPending(context.Background(), o)
	n.OrderConfirmed(context.Background(), o)
	n.Wait()
	if len(creator.calls()) != 0 {
		t.Errorf("calls = %d, want 0", len(creator.calls()))
	}
}

type recordingNotifier struct {
	name string
	log  *[]string
}

func (r recordingNotifier) OrderPending(context.Context, order.Order) {
	*r.log = append(*r.log, r.name+":pending")
}
func (r recordingNotifier) OrderPaid(context.Context, order.Order) {
	*r.log = append(*r.log, r.name+":paid")
}
func (r recordingNotifier) OrderConfirmed(context.Context, order.Order) {
	*r.log = append(*r.log, r.name+":confirmed")
}

func TestNotifiersFanOut(t *testing.T) {
	var log []string
	ns := app.Notifiers{recordingNotifier{"a", &log}, recordingNotifier{"b", &log}}
	ctx, o := context.Background(), order.Order{}
	ns.OrderPending(ctx, o)
	ns.OrderPaid(ctx, o)
	ns.OrderConfirmed(ctx, o)

	want := []string{"a:pending", "b:pending", "a:paid", "b:paid", "a:confirmed", "b:confirmed"}
	if len(log) != len(want) {
		t.Fatalf("log = %v, want %v", log, want)
	}
	for i := range want {
		if log[i] != want[i] {
			t.Errorf("log = %v, want %v", log, want)
			break
		}
	}
}
