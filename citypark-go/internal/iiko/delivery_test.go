package iiko

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// deliveryServer fakes iiko: token, deliveries/create and commands/status.
// It records the last request body of each endpoint.
type deliveryServer struct {
	mu       sync.Mutex
	cnt      *counter
	create   func(w http.ResponseWriter, n int)
	status   func(w http.ResponseWriter, n int)
	bodies   map[string]map[string]any
	authSeen string
}

func (s *deliveryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := s.cnt.inc(r.URL.Path)
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.mu.Lock()
	s.bodies[r.URL.Path] = body
	if r.URL.Path == "/1/deliveries/create" {
		s.authSeen = r.Header.Get("Authorization")
	}
	s.mu.Unlock()

	switch r.URL.Path {
	case "/v2/access_token":
		w.Write([]byte(`{"token":"t1"}`))
	case "/1/deliveries/create":
		s.create(w, n)
	case "/1/commands/status":
		s.status(w, n)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *deliveryServer) body(path string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bodies[path]
}

func newDeliveryClient(t *testing.T, create, status func(w http.ResponseWriter, n int)) (*Client, *deliveryServer) {
	t.Helper()
	s := &deliveryServer{cnt: newCounter(), create: create, status: status, bodies: map[string]map[string]any{}}
	c := newTestClient(t, s)
	c.pollEvery = time.Millisecond
	return c, s
}

func reply(body string) func(w http.ResponseWriter, n int) {
	return func(w http.ResponseWriter, _ int) { w.Write([]byte(body)) }
}

func testOrder() DeliveryOrder {
	return DeliveryOrder{
		ID:               "guid-1",
		ExternalNumber:   "42",
		Phone:            "+79001234567",
		OrderServiceType: ServiceDeliveryByClient,
		Customer:         DeliveryCustomer{Type: "one-time", Name: "Ivan"},
		Items:            []DeliveryItem{{Type: "Product", ProductID: "p-1", Amount: 2, Price: 150.5}},
		Payments: []DeliveryPayment{{
			PaymentTypeKind: "External", Sum: 301, PaymentTypeID: "pt-1", IsProcessedExternally: true,
		}},
	}
}

func TestCreateDeliverySuccess(t *testing.T) {
	c, s := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"iiko-1","creationStatus":"Success"}}`), nil)

	got, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err != nil {
		t.Fatal(err)
	}
	if got != (DeliveryResult{ID: "iiko-1"}) {
		t.Errorf("result = %+v", got)
	}

	body := s.body("/1/deliveries/create")
	if body["organizationId"] != "org-1" || body["terminalGroupId"] != "tg-1" {
		t.Errorf("ids in body = %v, %v", body["organizationId"], body["terminalGroupId"])
	}
	if settings, _ := body["createOrderSettings"].(map[string]any); settings["transportToFrontTimeout"] != float64(180) {
		t.Errorf("createOrderSettings = %v", body["createOrderSettings"])
	}
	order, _ := body["order"].(map[string]any)
	if order["id"] != "guid-1" || order["phone"] != "+79001234567" {
		t.Errorf("order = %v", order)
	}
	if cust, _ := order["customer"].(map[string]any); cust["type"] != "one-time" {
		t.Errorf("customer = %v", order["customer"])
	}
	items, _ := order["items"].([]any)
	item, _ := items[0].(map[string]any)
	if item["type"] != "Product" || item["productId"] != "p-1" || item["amount"] != float64(2) || item["price"] != 150.5 {
		t.Errorf("item = %v", item)
	}
	pays, _ := order["payments"].([]any)
	pay, _ := pays[0].(map[string]any)
	if pay["paymentTypeKind"] != "External" || pay["sum"] != float64(301) ||
		pay["paymentTypeId"] != "pt-1" || pay["isProcessedExternally"] != true {
		t.Errorf("payment = %v", pay)
	}
	if _, ok := order["deliveryPoint"]; ok {
		t.Errorf("deliveryPoint is sent for a nil point: %v", order["deliveryPoint"])
	}
	if s.authSeen != "Bearer t1" {
		t.Errorf("Authorization = %q", s.authSeen)
	}
}

func TestCreateDeliveryWithAddress(t *testing.T) {
	c, s := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"iiko-1","creationStatus":"Success"}}`), nil)
	o := testOrder()
	o.OrderServiceType = ServiceDeliveryByCourier
	o.DeliveryPoint = &DeliveryPoint{Address: DeliveryAddress{
		Type: "legacy", Street: DeliveryStreet{Name: "Ленина", City: "Белореченск"}, House: "5",
	}}

	if _, err := c.CreateDelivery(context.Background(), "tg-1", o); err != nil {
		t.Fatal(err)
	}
	order, _ := s.body("/1/deliveries/create")["order"].(map[string]any)
	point, _ := order["deliveryPoint"].(map[string]any)
	addr, _ := point["address"].(map[string]any)
	street, _ := addr["street"].(map[string]any)
	if addr["type"] != "legacy" || addr["house"] != "5" || street["name"] != "Ленина" || street["city"] != "Белореченск" {
		t.Errorf("address = %v", addr)
	}
	if order["orderServiceType"] != ServiceDeliveryByCourier {
		t.Errorf("orderServiceType = %v", order["orderServiceType"])
	}
}

func TestCreateDeliveryErrorStatus(t *testing.T) {
	tests := []struct {
		name, info, want string
	}{
		{"message", `{"code":"C","message":"M","description":"D"}`, "M"},
		{"description", `{"code":"C","description":"D"}`, "D"},
		{"code", `{"code":"C"}`, "C"},
		{"empty info", `{}`, "unknown error"},
		{"no info", ``, "unknown error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"Error"}}`
			if tt.info != "" {
				body = `{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"Error","errorInfo":` + tt.info + `}}`
			}
			c, _ := newDeliveryClient(t, reply(body), nil)

			_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
			var de *DeliveryError
			if !errors.As(err, &de) {
				t.Fatalf("err = %v, want *DeliveryError", err)
			}
			if de.Reason != tt.want {
				t.Errorf("reason = %q, want %q", de.Reason, tt.want)
			}
			if want := "iiko: delivery not created: " + tt.want; err.Error() != want {
				t.Errorf("err = %q, want %q", err, want)
			}
			if !Permanent(err) {
				t.Error("Permanent = false, want true")
			}
		})
	}
}

func TestCreateDeliveryPolls(t *testing.T) {
	c, s := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"iiko-7","creationStatus":"InProgress"}}`),
		func(w http.ResponseWriter, n int) {
			if n == 1 {
				w.Write([]byte(`{"state":"InProgress"}`))
				return
			}
			w.Write([]byte(`{"state":"Success"}`))
		})

	got, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "iiko-7" {
		t.Errorf("id = %q, want iiko-7", got.ID)
	}
	if n := s.cnt.get("/1/commands/status"); n != 2 {
		t.Errorf("status calls = %d, want 2", n)
	}
	body := s.body("/1/commands/status")
	if len(body) != 2 || body["organizationId"] != "org-1" || body["correlationId"] != "c1" {
		t.Errorf("status body = %v", body)
	}
}

func TestCreateDeliveryPollError(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`),
		reply(`{"state":"Error","errorReason":"Product not found"}`))

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	var de *DeliveryError
	if !errors.As(err, &de) || de.Reason != "Product not found" {
		t.Fatalf("err = %v, want DeliveryError{Product not found}", err)
	}
}

func TestCreateDeliveryPollErrorWithoutReason(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`),
		reply(`{"state":"Error"}`))

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	var de *DeliveryError
	if !errors.As(err, &de) || de.Reason != "unknown error" {
		t.Fatalf("err = %v, want DeliveryError{unknown error}", err)
	}
}

func TestCreateDeliveryPollUnknownState(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`),
		reply(`{"state":"Strange"}`))

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err == nil || !strings.Contains(err.Error(), "Strange") {
		t.Fatalf("err = %v, want a message with Strange", err)
	}
	if Permanent(err) {
		t.Error("Permanent = true for an unknown state")
	}
}

func TestCreateDeliveryPollRequestError(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`),
		func(w http.ResponseWriter, _ int) { w.WriteHeader(http.StatusInternalServerError) })

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	var se *StatusError
	if !errors.As(err, &se) || se.Status != 500 {
		t.Fatalf("err = %v, want *StatusError 500", err)
	}
}

func TestCreateDeliveryPending(t *testing.T) {
	c, s := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`),
		reply(`{"state":"InProgress"}`))
	c.pollTimes = 3

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if !errors.Is(err, ErrDeliveryPending) {
		t.Fatalf("err = %v, want ErrDeliveryPending", err)
	}
	if n := s.cnt.get("/1/commands/status"); n != 3 {
		t.Errorf("status calls = %d, want 3", n)
	}
	if Permanent(err) {
		t.Error("Permanent = true for a pending delivery")
	}
}

func TestCreateDeliveryUnknownStatus(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"Weird"}}`), nil)

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err == nil || !strings.Contains(err.Error(), "Weird") {
		t.Fatalf("err = %v, want a message with Weird", err)
	}
}

func TestCreateDeliveryBadRequest(t *testing.T) {
	c, _ := newDeliveryClient(t, func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorDescription":"Invalid phone"}`))
	}, nil)

	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	var se *StatusError
	if !errors.As(err, &se) || se.Status != 400 {
		t.Fatalf("err = %v, want *StatusError 400", err)
	}
	if want := "iiko: 1/deliveries/create: unexpected status 400: Invalid phone"; err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
	if !Permanent(err) {
		t.Error("Permanent(400) = false, want true")
	}
}

func TestPermanent(t *testing.T) {
	c, _ := newDeliveryClient(t, func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusInternalServerError)
	}, nil)
	_, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err == nil || Permanent(err) {
		t.Errorf("Permanent(%v) = true, want false for a 500", err)
	}
	if Permanent(nil) {
		t.Error("Permanent(nil) = true")
	}
	if Permanent(errors.New("x")) {
		t.Error("Permanent(plain error) = true")
	}
}

func TestCreateDeliveryContextCanceled(t *testing.T) {
	c, _ := newDeliveryClient(t,
		reply(`{"correlationId":"c1","orderInfo":{"id":"x","creationStatus":"InProgress"}}`), nil)
	c.pollEvery = time.Hour
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := c.CreateDelivery(ctx, "tg-1", testOrder())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
}

func TestCreateDeliveryRefreshesToken(t *testing.T) {
	c, s := newDeliveryClient(t, func(w http.ResponseWriter, n int) {
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"correlationId":"c1","orderInfo":{"id":"iiko-2","creationStatus":"Success"}}`))
	}, nil)

	got, err := c.CreateDelivery(context.Background(), "tg-1", testOrder())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "iiko-2" {
		t.Errorf("id = %q", got.ID)
	}
	if n := s.cnt.get("/v2/access_token"); n != 2 {
		t.Errorf("access_token calls = %d, want 2", n)
	}
	if n := s.cnt.get("/1/deliveries/create"); n != 2 {
		t.Errorf("create calls = %d, want 2", n)
	}
}
