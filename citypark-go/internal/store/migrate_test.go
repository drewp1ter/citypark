package store

import (
	"context"
	"sort"
	"testing"

	"github.com/jmoiron/sqlx"
)

var schemaTables = []string{
	"addresses", "categories", "category_images", "delivery_prices",
	"order_items", "orders", "product_images", "product_tags",
	"products", "tags", "user_addresses", "users",
}

// existingSchemaTables returns the sorted names of the schema tables that exist.
func existingSchemaTables(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var got []string
	err := db.Select(&got, `SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = ANY($1) ORDER BY table_name`, schemaTables)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	sort.Strings(got)
	return got
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMigrateUpDown(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	// The state before the test is unknown, so the result of this call is ignored.
	_ = MigrateDown(ctx, db)
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := existingSchemaTables(t, db); !equalStrings(got, schemaTables) {
		t.Fatalf("tables after Migrate = %v, want %v", got, schemaTables)
	}

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	if err := MigrateDown(ctx, db); err != nil {
		t.Fatalf("MigrateDown: %v", err)
	}
	if got := existingSchemaTables(t, db); len(got) != 0 {
		t.Fatalf("tables after MigrateDown = %v, want none", got)
	}

	// Leave the database migrated for other packages.
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("final Migrate: %v", err)
	}
}

func TestSchemaConstraints(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM users WHERE phone IN ('+70000000001', '+70000000002')`)
		_, _ = db.Exec(`DELETE FROM categories WHERE name IN ('Шашлыки', 'bad days')`)
	})

	t.Run("duplicate phone", func(t *testing.T) {
		if _, err := db.Exec(`INSERT INTO users (name, phone) VALUES ('a', '+70000000001')`); err != nil {
			t.Fatalf("first insert: %v", err)
		}
		_, err := db.Exec(`INSERT INTO users (name, phone) VALUES ('b', '+70000000001')`)
		if !IsUniqueViolation(err) {
			t.Fatalf("err = %v, want unique violation", err)
		}
	})

	t.Run("bad role", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO users (name, phone, role) VALUES ('c', '+70000000002', 'boss')`)
		if !IsCheckViolation(err) {
			t.Fatalf("err = %v, want check violation", err)
		}
	})

	t.Run("bad sale_days", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO categories (name, sale_days) VALUES ('bad days', '11')`)
		if !IsCheckViolation(err) {
			t.Fatalf("err = %v, want check violation", err)
		}
	})

	t.Run("russian full-text search", func(t *testing.T) {
		if _, err := db.Exec(`INSERT INTO categories (name) VALUES ('Шашлыки')`); err != nil {
			t.Fatalf("insert: %v", err)
		}
		var n int
		err := db.Get(&n, `SELECT count(*) FROM categories WHERE search @@ plainto_tsquery('russian', 'шашлык')`)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if n != 1 {
			t.Fatalf("matches = %d, want 1", n)
		}
	})
}
