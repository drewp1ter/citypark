package order

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type adminEnv struct {
	*svcEnv
	router     http.Handler
	userToken  string
	adminToken string
	productC   int64 // 25000
	orderID    int64 // one item: productA x 1
}

func newAdminEnv(t *testing.T) *adminEnv {
	t.Helper()
	e := &adminEnv{svcEnv: newSvcEnv(t, friday)}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	r := chi.NewRouter()
	NewHandler(e.svc, tokens, defaultOpts()).Routes(r)
	e.router = r

	userID := testutil.InsertUser(t, e.db, "Иван", "+79991234567", auth.RoleUser)
	adminID := testutil.InsertUser(t, e.db, "Админ", "+79990000000", auth.RoleAdmin)
	e.userToken = issue(t, tokens, userID, auth.RoleUser)
	e.adminToken = issue(t, tokens, adminID, auth.RoleAdmin)
	e.productC = testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, Name: "Салат", PriceKopecks: 25000})

	addr := testutil.InsertAddress(t, e.db, userID, "Сочи", "ул. Ленина, 1")
	id, err := NewStore(e.db).Insert(context.Background(), InsertParams{
		CustomerID:    userID,
		AddressID:     &addr,
		PaymentType:   PaymentCash,
		ItemsHash:     "hash",
		DeliveryPrice: 15000,
		Items:         []InsertItem{{ProductID: e.productA, Amount: 1, OfferedPrice: 59900}},
	})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	e.orderID = id
	return e
}

func (e *adminEnv) do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	rec := testutil.DoJSON(t, e.router, method, path, token, body)
	var got map[string]any
	if rec.Body.Len() > 0 {
		testutil.Decode(t, rec, &got)
	}
	return rec.Code, got
}

func (e *adminEnv) itemsPath() string { return fmt.Sprintf("/orders/%d/items", e.orderID) }

func (e *adminEnv) itemRows(t *testing.T, productID int64) int {
	t.Helper()
	return e.count(t, `SELECT count(*) FROM order_items WHERE order_id = $1 AND product_id = $2`, e.orderID, productID)
}

func TestAddItem(t *testing.T) {
	e := newAdminEnv(t)
	code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": e.productB, "amount": 3})
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %v", code, body)
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %v, want 2", body["items"])
	}
	if n := e.count(t, `SELECT count(*) FROM order_items WHERE order_id = $1`, e.orderID); n != 2 {
		t.Errorf("order_items rows = %d, want 2", n)
	}
	// items are ordered by product id: productA first, productB second
	added := items[1].(map[string]any)
	if added["productId"] != float64(e.productB) || added["amount"] != float64(3) || added["offeredPrice"] != float64(100) {
		t.Errorf("added item = %v, want productB x3 at 100", added)
	}
	if body["total"] != float64(899) { // 599 + 3*100
		t.Errorf("total = %v, want 899", body["total"])
	}
}

func TestAddItemExisting(t *testing.T) {
	e := newAdminEnv(t)
	e.db.MustExec(`UPDATE products SET price_kopecks = 99900 WHERE id = $1`, e.productA)
	code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": e.productA, "amount": 5})
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %v", code, body)
	}
	if n := e.itemRows(t, e.productA); n != 1 {
		t.Errorf("rows for product = %d, want 1", n)
	}
	var amount, price int
	err := e.db.QueryRow(`SELECT amount, offered_price_kopecks FROM order_items WHERE order_id = $1 AND product_id = $2`,
		e.orderID, e.productA).Scan(&amount, &price)
	if err != nil {
		t.Fatalf("select item: %v", err)
	}
	if amount != 5 || price != 59900 {
		t.Errorf("amount = %d, price = %d, want 5 and 59900 (price kept)", amount, price)
	}
}

func TestAddItemUnknownProduct(t *testing.T) {
	e := newAdminEnv(t)
	code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": 999, "amount": 1})
	if code != http.StatusBadRequest || body["message"] != "Продукт не найден." {
		t.Errorf("got %d %v, want 400 Продукт не найден.", code, body)
	}
}

func TestAddItemDeletedProduct(t *testing.T) {
	e := newAdminEnv(t)
	deleted := testutil.InsertProduct(t, e.db, testutil.Product{CategoryID: e.category, Name: "Удалён", PriceKopecks: 100, IsDeleted: true})
	code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": deleted, "amount": 1})
	if code != http.StatusBadRequest || body["message"] != "Продукт не найден." {
		t.Errorf("got %d %v, want 400 Продукт не найден.", code, body)
	}
	if n := e.itemRows(t, deleted); n != 0 {
		t.Errorf("rows for deleted product = %d, want 0", n)
	}
}

func TestAddItemUnknownOrder(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, "/orders/999/items", e.adminToken, map[string]any{"productId": e.productB, "amount": 1})
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestAddItemValidation(t *testing.T) {
	e := newAdminEnv(t)
	code, body := e.do(t, http.MethodPost, e.itemsPath(), e.adminToken, map[string]any{"productId": e.productB, "amount": 0})
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	errs, _ := body["errors"].(map[string]any)
	if _, ok := errs["amount"]; !ok {
		t.Errorf("errors = %v, want a key amount", body["errors"])
	}
}

func TestAddItemBadOrderID(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, "/orders/abc/items", e.adminToken, map[string]any{"productId": e.productB, "amount": 1})
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestAddItemForbidden(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, e.itemsPath(), e.userToken, map[string]any{"productId": e.productB, "amount": 1})
	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
	if n := e.itemRows(t, e.productB); n != 0 {
		t.Errorf("rows for product = %d, want 0", n)
	}
}

func TestAddItemUnauthorized(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, e.itemsPath(), "", map[string]any{"productId": e.productB, "amount": 1})
	if code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", code)
	}
}

func TestRemoveItem(t *testing.T) {
	e := newAdminEnv(t)
	path := fmt.Sprintf("/orders/%d/items/%d", e.orderID, e.productA)
	for i := 0; i < 2; i++ { // the second call removes nothing and still succeeds
		code, _ := e.do(t, http.MethodDelete, path, e.adminToken, nil)
		if code != http.StatusNoContent {
			t.Fatalf("call %d: status = %d, want 204", i+1, code)
		}
	}
	if n := e.itemRows(t, e.productA); n != 0 {
		t.Errorf("rows = %d, want 0", n)
	}
}

func TestRemoveItemBadProductID(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodDelete, fmt.Sprintf("/orders/%d/items/abc", e.orderID), e.adminToken, nil)
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestRemoveItemForbidden(t *testing.T) {
	e := newAdminEnv(t)
	path := fmt.Sprintf("/orders/%d/items/%d", e.orderID, e.productA)
	code, _ := e.do(t, http.MethodDelete, path, e.userToken, nil)
	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
	if n := e.itemRows(t, e.productA); n != 1 {
		t.Errorf("rows = %d, want 1", n)
	}
}

func TestDeleteOrder(t *testing.T) {
	e := newAdminEnv(t)
	users := e.count(t, `SELECT count(*) FROM users`)
	addresses := e.count(t, `SELECT count(*) FROM addresses`)
	path := fmt.Sprintf("/orders/%d", e.orderID)
	for i := 0; i < 2; i++ {
		code, _ := e.do(t, http.MethodDelete, path, e.adminToken, nil)
		if code != http.StatusNoContent {
			t.Fatalf("call %d: status = %d, want 204", i+1, code)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM orders WHERE id = $1`, e.orderID); n != 0 {
		t.Errorf("order rows = %d, want 0", n)
	}
	if n := e.count(t, `SELECT count(*) FROM order_items WHERE order_id = $1`, e.orderID); n != 0 {
		t.Errorf("item rows = %d, want 0", n)
	}
	if n := e.count(t, `SELECT count(*) FROM users`); n != users {
		t.Errorf("users = %d, want %d", n, users)
	}
	if n := e.count(t, `SELECT count(*) FROM addresses`); n != addresses {
		t.Errorf("addresses = %d, want %d", n, addresses)
	}
}

func TestDeleteOrderForbidden(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodDelete, fmt.Sprintf("/orders/%d", e.orderID), e.userToken, nil)
	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
	if n := e.count(t, `SELECT count(*) FROM orders WHERE id = $1`, e.orderID); n != 1 {
		t.Errorf("order rows = %d, want 1", n)
	}
}

func TestConfirm(t *testing.T) {
	e := newAdminEnv(t)
	code, body := e.do(t, http.MethodPost, fmt.Sprintf("/orders/%d/confirm", e.orderID), e.adminToken, nil)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %v", code, body)
	}
	if body["status"] != StatusConfirmed {
		t.Errorf("body status = %v, want confirmed", body["status"])
	}
	if got := e.order(t, e.orderID).Status; got != StatusConfirmed {
		t.Errorf("stored status = %q, want confirmed", got)
	}
	if len(e.notifier.confirmed) != 1 || e.notifier.confirmed[0].ID != e.orderID {
		t.Errorf("OrderConfirmed calls = %v, want one for order %d", e.notifier.confirmed, e.orderID)
	}
}

func TestConfirmMissing(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, "/orders/999/confirm", e.adminToken, nil)
	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
	if len(e.notifier.confirmed) != 0 {
		t.Errorf("OrderConfirmed was called %d times, want 0", len(e.notifier.confirmed))
	}
}

func TestConfirmForbidden(t *testing.T) {
	e := newAdminEnv(t)
	code, _ := e.do(t, http.MethodPost, fmt.Sprintf("/orders/%d/confirm", e.orderID), e.userToken, nil)
	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
	if got := e.order(t, e.orderID).Status; got != StatusNew {
		t.Errorf("status = %q, want unchanged new", got)
	}
}
