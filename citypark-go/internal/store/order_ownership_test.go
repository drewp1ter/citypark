package store

import (
	"context"
	"reflect"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/drewp1ter/citypark-go/migrations"
)

func TestOrderOwnershipMigration(t *testing.T) {
	db := openTestDB(t) // One connection keeps search_path scoped to this test.
	ctx := context.Background()
	exec := func(query string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	// Test an actual upgrade with historical data, without resetting public data.
	exec(`CREATE SCHEMA order_ownership_migration_test`)
	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `SET search_path TO public`); err != nil {
			t.Error(err)
		}
		if _, err := db.ExecContext(ctx, `DROP SCHEMA order_ownership_migration_test CASCADE`); err != nil {
			t.Error(err)
		}
	})
	exec(`SET search_path TO order_ownership_migration_test`)
	p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 18); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO users (id, name, phone, email, password_hash, legacy_password_hash, role) VALUES
		(1, 'Password', '+70000000001', 'password@example.com', 'bcrypt', '', 'user'),
		(2, 'Legacy', '+70000000002', NULL, '', 'legacy', 'user'),
		(3, 'Guest', '+70000000003', NULL, '', '', 'user'),
		(4, 'Email only', '+70000000004', 'email@example.com', '', '', 'user'),
		(5, 'Admin', '+70000000005', NULL, 'bcrypt', '', 'admin'),
		(6, 'Lost address', '+70000000006', NULL, '', '', 'user')`)
	exec(`INSERT INTO addresses (id, city, address) VALUES
		(1, 'Белореченск', 'Гоголя, 61'), (2, 'Москва', 'Удалённый адрес')`)
	exec(`INSERT INTO orders (id, customer_id, address_id, items_hash, status) VALUES
		(1, 1, 1, 'one', 'paid'), (2, 2, NULL, 'two', 'confirmed'),
		(3, 3, 1, 'three', 'pending'), (4, 4, NULL, 'four', 'canceled'),
		(5, 5, 1, 'five', 'new'), (6, 6, 2, 'six', 'paid')`)
	exec(`DELETE FROM addresses WHERE id = 2`)
	if _, err := p.Up(ctx); err != nil {
		t.Fatal(err)
	}

	type snapshot struct {
		ID        int64   `db:"id"`
		AccountID *int64  `db:"account_id"`
		Name      *string `db:"contact_name"`
		Phone     *string `db:"contact_phone"`
		City      *string `db:"delivery_city"`
		Address   *string `db:"delivery_address"`
		Status    string  `db:"status"`
	}
	const selectSnapshots = `SELECT id, account_id, contact_name, contact_phone,
		delivery_city, delivery_address, status FROM orders WHERE id <= 6 ORDER BY id`
	var got []snapshot
	if err := db.SelectContext(ctx, &got, selectSnapshots); err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("orders after upgrade = %d, want 6", len(got))
	}
	names := []string{"Password", "Legacy", "Guest", "Email only", "Admin", "Lost address"}
	phones := []string{"+70000000001", "+70000000002", "+70000000003", "+70000000004", "+70000000005", "+70000000006"}
	statuses := []string{"paid", "confirmed", "pending", "canceled", "new", "paid"}
	for i, row := range got {
		if row.ID != int64(i+1) || row.Name == nil || *row.Name != names[i] ||
			row.Phone == nil || *row.Phone != phones[i] || row.Status != statuses[i] {
			t.Errorf("order %d: %+v", i+1, row)
		}
		if i < 2 {
			if row.AccountID == nil || *row.AccountID != row.ID {
				t.Errorf("credential-bearing order %d owner = %v", row.ID, row.AccountID)
			}
		} else if row.AccountID != nil {
			t.Errorf("unclaimed order %d owner = %d", row.ID, *row.AccountID)
		}
		if i == 0 || i == 2 || i == 4 {
			if row.City == nil || *row.City != "Белореченск" || row.Address == nil || *row.Address != "Гоголя, 61" {
				t.Errorf("delivery order %d snapshot = %+v", row.ID, row)
			}
		} else if row.City != nil || row.Address != nil {
			t.Errorf("pickup/lost address order %d snapshot = %+v", row.ID, row)
		}
	}

	exec(`UPDATE users SET name = 'Changed', phone = '+70000000007', updated_at = now() WHERE id = 1`)
	exec(`UPDATE addresses SET city = 'Changed', address = 'Changed', updated_at = now() WHERE id = 1`)
	exec(`DELETE FROM addresses WHERE id = 1`)
	var after []snapshot
	if err := db.SelectContext(ctx, &after, selectSnapshots); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, after) {
		t.Errorf("snapshots changed after source edits/deletion: before %+v, after %+v", got, after)
	}
	exec(`INSERT INTO orders (id, customer_id, items_hash) VALUES (7, 1, 'new guest')`)
	var fresh snapshot
	if err := db.GetContext(ctx, &fresh, `SELECT id, account_id, contact_name, contact_phone,
		delivery_city, delivery_address, status FROM orders WHERE id = 7`); err != nil {
		t.Fatal(err)
	}
	if fresh.AccountID != nil || fresh.Name != nil || fresh.Phone != nil || fresh.City != nil || fresh.Address != nil {
		t.Errorf("new legacy insert must remain unclaimed with unknown snapshots: %+v", fresh)
	}
	if _, err := db.ExecContext(ctx, `UPDATE orders SET account_id = 999999, updated_at = now() WHERE id = 7`); !IsFKViolation(err) {
		t.Fatalf("invalid owner error = %v, want FK violation", err)
	}
	var indexCount int
	if err := db.GetContext(ctx, &indexCount, `SELECT count(*) FROM pg_indexes
		WHERE schemaname = current_schema() AND indexname = 'orders_account_id_idx'`); err != nil || indexCount != 1 {
		t.Fatalf("account index count = %d, error = %v", indexCount, err)
	}
	if _, err := p.DownTo(ctx, 18); err != nil {
		t.Fatal(err)
	}
	var columns, orders int
	if err := db.GetContext(ctx, &columns, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'orders'
		AND column_name IN ('account_id', 'contact_name', 'contact_phone', 'delivery_city', 'delivery_address')`); err != nil || columns != 0 {
		t.Fatalf("new columns after rollback = %d, error = %v", columns, err)
	}
	if err := db.GetContext(ctx, &orders, `SELECT count(*) FROM orders`); err != nil || orders != 7 {
		t.Fatalf("orders after rollback = %d, error = %v", orders, err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("upgrade after rollback: %v", err)
	}
}
