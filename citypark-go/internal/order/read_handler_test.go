package order

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type readEnv struct {
	*svcEnv
	router     http.Handler
	userToken  string
	otherToken string
	adminToken string
	userID     int64
	userOrders []int64 // oldest first
	otherOrder int64
}

func newReadEnv(t *testing.T) *readEnv {
	t.Helper()
	e := &readEnv{svcEnv: newSvcEnv(t, friday)}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	r := chi.NewRouter()
	NewHandler(e.svc, tokens, defaultOpts()).Routes(r)
	e.router = r

	e.userID = testutil.InsertUser(t, e.db, "Иван", "+79991234567", auth.RoleUser)
	otherID := testutil.InsertUser(t, e.db, "Пётр", "+79997654321", auth.RoleUser)
	adminID := testutil.InsertUser(t, e.db, "Админ", "+79990000000", auth.RoleAdmin)
	e.userToken = issue(t, tokens, e.userID, auth.RoleUser)
	e.otherToken = issue(t, tokens, otherID, auth.RoleUser)
	e.adminToken = issue(t, tokens, adminID, auth.RoleAdmin)

	addr := testutil.InsertAddress(t, e.db, e.userID, "Сочи", "ул. Ленина, 1")
	e.userOrders = append(e.userOrders,
		e.insert(t, e.userID, &addr, "first", InsertItem{ProductID: e.productA, Amount: 1, OfferedPrice: 59900}),
		e.insert(t, e.userID, nil, "second",
			InsertItem{ProductID: e.productA, Amount: 1, OfferedPrice: 59900},
			InsertItem{ProductID: e.productB, Amount: 2, OfferedPrice: 10000}),
	)
	e.otherOrder = e.insert(t, otherID, nil, "other", InsertItem{ProductID: e.productB, Amount: 1, OfferedPrice: 10000})
	return e
}

func issue(t *testing.T, tokens *auth.Tokens, id int64, role string) string {
	t.Helper()
	tok, err := tokens.Issue(id, role)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return tok
}

func (e *readEnv) insert(t *testing.T, customer int64, addr *int64, comment string, items ...InsertItem) int64 {
	t.Helper()
	var delivery money.Kopecks
	if addr != nil {
		delivery = 15000
	}
	id, err := NewStore(e.db).Insert(context.Background(), InsertParams{
		CustomerID:    customer,
		AddressID:     addr,
		Comment:       comment,
		PaymentType:   PaymentCash,
		ItemsHash:     comment,
		DeliveryPrice: delivery,
		Items:         items,
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	return id
}

type listBody struct {
	Orders []map[string]any `json:"orders"`
	Count  int              `json:"count"`
}

func (e *readEnv) getList(t *testing.T, path, token string) listBody {
	t.Helper()
	rec := testutil.DoJSON(t, e.router, http.MethodGet, path, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200; body %s", path, rec.Code, rec.Body.String())
	}
	var got listBody
	testutil.Decode(t, rec, &got)
	return got
}

func wantBody(t *testing.T, e *readEnv, method, path, token string, status int, body string) {
	t.Helper()
	rec := testutil.DoJSON(t, e.router, method, path, token, nil)
	if rec.Code != status {
		t.Fatalf("%s %s = %d, want %d; body %s", method, path, rec.Code, status, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != body {
		t.Errorf("body = %s, want %s", got, body)
	}
}

const (
	unauthorizedBody = `{"statusCode":401,"message":"Требуется авторизация"}`
	forbiddenBody    = `{"statusCode":403,"message":"Недостаточно прав"}`
)

func TestListMine(t *testing.T) {
	e := newReadEnv(t)
	got := e.getList(t, "/orders/my", e.userToken)
	if got.Count != 2 || len(got.Orders) != 2 {
		t.Fatalf("count = %d, orders = %d, want 2 and 2", got.Count, len(got.Orders))
	}
	newest, oldest := e.userOrders[1], e.userOrders[0]
	if id := int64(got.Orders[0]["id"].(float64)); id != newest {
		t.Errorf("first id = %d, want newest %d", id, newest)
	}
	if id := int64(got.Orders[1]["id"].(float64)); id != oldest {
		t.Errorf("second id = %d, want oldest %d", id, oldest)
	}
	items, _ := got.Orders[0]["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("newest items = %v, want 2", got.Orders[0]["items"])
	}
	if first := items[0].(map[string]any); first["name"] != "Шашлык" || first["categoryName"] != "Мангал" {
		t.Errorf("first item = %v", first)
	}
	if got.Orders[0]["total"] != float64(799) {
		t.Errorf("total = %v, want 799", got.Orders[0]["total"])
	}
	addr, _ := got.Orders[1]["address"].(map[string]any)
	if addr["city"] != "Сочи" || addr["address"] != "ул. Ленина, 1" {
		t.Errorf("oldest address = %v", got.Orders[1]["address"])
	}
	if got.Orders[0]["address"] != nil {
		t.Errorf("pickup address = %v, want null", got.Orders[0]["address"])
	}
}

func TestListMinePaging(t *testing.T) {
	e := newReadEnv(t)
	got := e.getList(t, "/orders/my?pageSize=1&page=2", e.userToken)
	if got.Count != 2 {
		t.Errorf("count = %d, want 2", got.Count)
	}
	if len(got.Orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(got.Orders))
	}
	if id := int64(got.Orders[0]["id"].(float64)); id != e.userOrders[0] {
		t.Errorf("id = %d, want the older order %d", id, e.userOrders[0])
	}
}

func TestListMineEmpty(t *testing.T) {
	e := newReadEnv(t)
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	lonely := testutil.InsertUser(t, e.db, "Одинокий", "+79995555555", auth.RoleUser)
	wantBody(t, e, http.MethodGet, "/orders/my", issue(t, tokens, lonely, auth.RoleUser),
		http.StatusOK, `{"orders":[],"count":0}`)
}

func TestListMineUnauthorized(t *testing.T) {
	e := newReadEnv(t)
	wantBody(t, e, http.MethodGet, "/orders/my", "", http.StatusUnauthorized, unauthorizedBody)
}

func TestGetOwn(t *testing.T) {
	e := newReadEnv(t)
	rec := testutil.DoJSON(t, e.router, http.MethodGet, fmt.Sprintf("/orders/%d", e.userOrders[0]), e.userToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	testutil.Decode(t, rec, &got)
	wantKeys(t, got, "id", "createdAt", "updatedAt", "status", "comment", "paymentType",
		"total", "deliveryPrice", "customer", "address", "items")
	if got["comment"] != "first" || got["status"] != StatusNew || got["deliveryPrice"] != float64(150) {
		t.Errorf("order = %v", got)
	}
}

func TestGetOthers(t *testing.T) {
	e := newReadEnv(t)
	path := fmt.Sprintf("/orders/%d", e.otherOrder)
	wantBody(t, e, http.MethodGet, path, e.userToken, http.StatusNotFound, `{"statusCode":404,"message":"Не найдено"}`)
}

func TestGetAsAdmin(t *testing.T) {
	e := newReadEnv(t)
	rec := testutil.DoJSON(t, e.router, http.MethodGet, fmt.Sprintf("/orders/%d", e.otherOrder), e.adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	var got struct {
		ID       int64    `json:"id"`
		Customer Customer `json:"customer"`
	}
	testutil.Decode(t, rec, &got)
	if got.ID != e.otherOrder || got.Customer.Name != "Пётр" {
		t.Errorf("order = %+v, want id %d of customer Пётр", got, e.otherOrder)
	}
}

func TestGetMissing(t *testing.T) {
	e := newReadEnv(t)
	wantBody(t, e, http.MethodGet, "/orders/9999", e.userToken, http.StatusNotFound, `{"statusCode":404,"message":"Не найдено"}`)
	wantBody(t, e, http.MethodGet, "/orders/abc", e.userToken, http.StatusNotFound, `{"statusCode":404,"message":"Не найдено"}`)
}

func TestGetUnauthorized(t *testing.T) {
	e := newReadEnv(t)
	path := fmt.Sprintf("/orders/%d", e.userOrders[0])
	wantBody(t, e, http.MethodGet, path, "", http.StatusUnauthorized, unauthorizedBody)
}

func TestListAll(t *testing.T) {
	e := newReadEnv(t)
	got := e.getList(t, "/orders", e.adminToken)
	if got.Count != 3 || len(got.Orders) != 3 {
		t.Fatalf("count = %d, orders = %d, want 3 and 3", got.Count, len(got.Orders))
	}
	for i, o := range got.Orders {
		c, _ := o["customer"].(map[string]any)
		if c["name"] == nil || c["name"] == "" {
			t.Errorf("order %d has no customer name: %v", i, o)
		}
	}
	if id := int64(got.Orders[0]["id"].(float64)); id != e.otherOrder {
		t.Errorf("first id = %d, want newest %d", id, e.otherOrder)
	}
}

func TestListAllForbidden(t *testing.T) {
	e := newReadEnv(t)
	wantBody(t, e, http.MethodGet, "/orders", e.userToken, http.StatusForbidden, forbiddenBody)
}

func TestListAllUnauthorized(t *testing.T) {
	e := newReadEnv(t)
	wantBody(t, e, http.MethodGet, "/orders", "", http.StatusUnauthorized, unauthorizedBody)
}

func TestListBadPaging(t *testing.T) {
	e := newReadEnv(t)
	for _, path := range []string{"/orders?page=0", "/orders/my?page=0"} {
		token := e.adminToken
		if strings.Contains(path, "/my") {
			token = e.userToken
		}
		rec := testutil.DoJSON(t, e.router, http.MethodGet, path, token, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400; body %s", path, rec.Code, rec.Body.String())
		}
	}
}
