package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestNewDBIsEmpty(t *testing.T) {
	db := NewDB(t)
	if id := InsertUser(t, db, "Иван", "+79000000001", "user"); id != 1 {
		t.Fatalf("first user id = %d, want 1", id)
	}

	db = NewDB(t)
	if n := Count(t, db, `SELECT count(*) FROM users`); n != 0 {
		t.Fatalf("users after NewDB = %d, want 0", n)
	}
	if id := InsertUser(t, db, "Пётр", "+79000000002", "admin"); id != 1 {
		t.Fatalf("user id after reset = %d, want 1", id)
	}
}

func TestBuilders(t *testing.T) {
	db := NewDB(t)

	userID := InsertUser(t, db, "Иван", "+79000000001", "user")
	categoryID := InsertCategory(t, db, Category{SaleStart: IntPtr(540)})
	productID := InsertProduct(t, db, Product{CategoryID: categoryID, PriceKopecks: 59900})
	deliveryID := InsertDeliveryPrice(t, db, "Центр", 15000)
	addressID := InsertAddress(t, db, userID, "Москва", "Тверская, 1")
	InsertAddress(t, db, 0, "Москва", "Арбат, 2")

	for _, id := range []int64{userID, categoryID, productID, deliveryID, addressID} {
		if id != 1 {
			t.Errorf("id = %d, want 1", id)
		}
	}
	tests := []struct {
		query string
		want  int
	}{
		{`SELECT count(*) FROM users`, 1},
		{`SELECT count(*) FROM categories WHERE name = 'Категория' AND sale_start = 540 AND sale_end IS NULL`, 1},
		{`SELECT count(*) FROM products WHERE name = 'Продукт' AND price_kopecks = 59900`, 1},
		{`SELECT count(*) FROM delivery_prices`, 1},
		{`SELECT count(*) FROM addresses`, 2},
		{`SELECT count(*) FROM user_addresses`, 1},
	}
	for _, tt := range tests {
		if got := Count(t, db, tt.query); got != tt.want {
			t.Errorf("%s = %d, want %d", tt.query, got, tt.want)
		}
	}
	if got := Count(t, db, `SELECT count(*) FROM users WHERE email IS NULL AND role = $1`, "user"); got != 1 {
		t.Errorf("user with NULL email and role user = %d, want 1", got)
	}
}

func TestDoJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	type echo struct {
		Method      string `json:"method"`
		Auth        string `json:"auth"`
		ContentType string `json:"contentType"`
		Body        string `json:"body"`
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(echo{
			Method:      r.Method,
			Auth:        r.Header.Get("Authorization"),
			ContentType: r.Header.Get("Content-Type"),
			Body:        string(body),
		})
	})

	rec := DoJSON(t, h, http.MethodPost, "/x", "tok", payload{Name: "борщ"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var got echo
	Decode(t, rec, &got)
	if got.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.Method)
	}
	if got.Auth != "Bearer tok" {
		t.Errorf("authorization = %q, want %q", got.Auth, "Bearer tok")
	}
	if got.ContentType != "application/json" {
		t.Errorf("content type = %q, want application/json", got.ContentType)
	}
	var p payload
	if err := json.Unmarshal([]byte(got.Body), &p); err != nil || p.Name != "борщ" {
		t.Errorf("body = %q (err %v), want name борщ", got.Body, err)
	}

	rec = DoJSON(t, h, http.MethodGet, "/x", "", nil)
	got = echo{}
	Decode(t, rec, &got)
	if got.Auth != "" || got.ContentType != "" || got.Body != "" {
		t.Errorf("empty request echoed %+v, want no auth, content type or body", got)
	}
}
