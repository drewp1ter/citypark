// Package testutil holds helpers shared by the tests of every feature package.
// It imports only the standard library, sqlx and internal/store.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const defaultTestDSN = "postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable"

// NewDB returns a connection to the test database with all migrations applied and all tables empty.
// DSN: environment variable TEST_DB_DSN, default
// "postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable".
// If the database cannot be reached the test fails with t.Fatalf and a hint to run `make db-test-up`.
func NewDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open test database (run `make db-test-up` or set TEST_DB_DSN): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	_, err = db.ExecContext(ctx, `TRUNCATE users, addresses, user_addresses, menus, categories, category_images,
		products, product_images, tags, product_tags, delivery_prices, orders, order_items
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate test database: %v", err)
	}
	return db
}
