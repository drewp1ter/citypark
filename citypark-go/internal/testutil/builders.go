package testutil

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
)

// InsertUser creates a user with a NULL email and empty password. role is "user" or "admin".
func InsertUser(t *testing.T, db *sqlx.DB, name, phone, role string) int64 {
	t.Helper()
	var id int64
	err := db.GetContext(context.Background(), &id,
		`INSERT INTO users (name, phone, role) VALUES ($1, $2, $3) RETURNING id`,
		name, phone, role)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

type Category struct {
	Name      string // default "Категория" when empty
	SortOrder int
	IsDeleted bool
	SaleStart *int // minutes from midnight
	SaleEnd   *int
	SaleDays  string // "" or 7 chars of 0/1
	IsHidden  bool
}

func InsertCategory(t *testing.T, db *sqlx.DB, c Category) int64 {
	t.Helper()
	if c.Name == "" {
		c.Name = "Категория"
	}
	var id int64
	err := db.GetContext(context.Background(), &id,
		`INSERT INTO categories (name, sort_order, is_deleted, sale_start, sale_end, sale_days, is_hidden)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		c.Name, c.SortOrder, c.IsDeleted, c.SaleStart, c.SaleEnd, c.SaleDays, c.IsHidden)
	if err != nil {
		t.Fatalf("insert category: %v", err)
	}
	return id
}

type Product struct {
	CategoryID   int64
	Name         string // default "Продукт" when empty
	PriceKopecks int64
	SortOrder    int
	IsDeleted    bool
	SaleStart    *int
	SaleEnd      *int
	SaleDays     string
}

func InsertProduct(t *testing.T, db *sqlx.DB, p Product) int64 {
	t.Helper()
	if p.Name == "" {
		p.Name = "Продукт"
	}
	var id int64
	err := db.GetContext(context.Background(), &id,
		`INSERT INTO products (category_id, name, price_kopecks, sort_order, is_deleted, sale_start, sale_end, sale_days)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		p.CategoryID, p.Name, p.PriceKopecks, p.SortOrder, p.IsDeleted, p.SaleStart, p.SaleEnd, p.SaleDays)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	return id
}

func InsertDeliveryPrice(t *testing.T, db *sqlx.DB, name string, priceKopecks int64) int64 {
	t.Helper()
	var id int64
	err := db.GetContext(context.Background(), &id,
		`INSERT INTO delivery_prices (name, price_kopecks) VALUES ($1, $2) RETURNING id`,
		name, priceKopecks)
	if err != nil {
		t.Fatalf("insert delivery price: %v", err)
	}
	return id
}

// InsertAddress creates an address. When userID is not 0 it also links it in user_addresses.
func InsertAddress(t *testing.T, db *sqlx.DB, userID int64, city, address string) int64 {
	t.Helper()
	ctx := context.Background()
	var id int64
	err := db.GetContext(ctx, &id,
		`INSERT INTO addresses (city, address) VALUES ($1, $2) RETURNING id`,
		city, address)
	if err != nil {
		t.Fatalf("insert address: %v", err)
	}
	if userID != 0 {
		_, err := db.ExecContext(ctx,
			`INSERT INTO user_addresses (user_id, address_id) VALUES ($1, $2)`,
			userID, id)
		if err != nil {
			t.Fatalf("link address to user: %v", err)
		}
	}
	return id
}

// Count runs a "SELECT count(*) ..." query and returns the number.
func Count(t *testing.T, db *sqlx.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.GetContext(context.Background(), &n, query, args...); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func IntPtr(v int) *int {
	return &v
}
