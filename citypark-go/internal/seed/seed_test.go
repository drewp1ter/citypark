package seed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/order"
	"github.com/drewp1ter/citypark-go/internal/seed"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestRun(t *testing.T) {
	db := testutil.NewDB(t)
	got, err := seed.Run(context.Background(), db)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := seed.Summary{Users: 1, Categories: 4, Products: 16, DeliveryPrices: 3, Orders: 2}
	if got != want {
		t.Fatalf("summary = %+v, want %+v", got, want)
	}

	counts := []struct {
		query string
		want  int
	}{
		{`SELECT count(*) FROM users`, 1},
		{`SELECT count(*) FROM categories`, 4},
		{`SELECT count(*) FROM products`, 16},
		{`SELECT count(*) FROM delivery_prices`, 3},
		{`SELECT count(*) FROM orders`, 2},
		{`SELECT count(*) FROM product_images`, 16},
		{`SELECT count(*) FROM tags`, 3},
		{`SELECT count(*) FROM addresses`, 1},
		{`SELECT count(*) FROM products p WHERE
			(SELECT count(*) FROM product_images i WHERE i.product_id = p.id AND i.type = 'seed') <> 1`, 0},
	}
	for _, c := range counts {
		if n := testutil.Count(t, db, c.query); n != c.want {
			t.Errorf("%s = %d, want %d", c.query, n, c.want)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_tags`); n < 6 {
		t.Errorf("product_tags = %d, want at least 6", n)
	}
	// every tag is linked to at least two products
	q := `SELECT count(*) FROM tags t WHERE (SELECT count(*) FROM product_tags pt WHERE pt.tag_id = t.id) < 2`
	if n := testutil.Count(t, db, q); n != 0 {
		t.Errorf("%d tags are linked to fewer than two products", n)
	}
}

func TestRunTwice(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := seed.Run(context.Background(), db); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	sum, err := seed.Run(context.Background(), db)
	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Fatalf("second Run error = %v, want ErrNotEmpty", err)
	}
	if sum != (seed.Summary{}) {
		t.Errorf("second Run summary = %+v, want zero", sum)
	}
	for table, want := range map[string]int{"users": 1, "categories": 4, "products": 16, "delivery_prices": 3, "orders": 2} {
		if n := testutil.Count(t, db, `SELECT count(*) FROM `+table); n != want {
			t.Errorf("%s count = %d after second Run, want %d", table, n, want)
		}
	}
}

func TestSeedUserCanSignIn(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := seed.Run(context.Background(), db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var hash string
	err := db.GetContext(context.Background(), &hash,
		`SELECT password_hash FROM users WHERE phone = '+79990000001'`)
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if !auth.CheckPassword(hash, "password123") {
		t.Error("CheckPassword(password_hash, \"password123\") = false")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Error("CheckPassword accepted a wrong password")
	}
}

func TestSeedOrdersLoad(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := seed.Run(context.Background(), db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var userID int64
	if err := db.GetContext(context.Background(), &userID, `SELECT id FROM users`); err != nil {
		t.Fatalf("load user id: %v", err)
	}
	orders, total, err := order.NewStore(db).ListByCustomer(context.Background(), userID, 10, 0)
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if total != 2 || len(orders) != 2 {
		t.Fatalf("total = %d, len = %d, want 2 and 2", total, len(orders))
	}
	withAddress := 0
	for _, o := range orders {
		if len(o.Items) == 0 {
			t.Errorf("order %d has no items", o.ID)
		}
		if o.Total <= 0 {
			t.Errorf("order %d total = %d, want > 0", o.ID, o.Total)
		}
		if o.ItemsHash == "" {
			t.Errorf("order %d has an empty items hash", o.ID)
		}
		if o.Address != nil {
			withAddress++
			if o.DeliveryPrice != 15000 {
				t.Errorf("delivery price = %d, want 15000", o.DeliveryPrice)
			}
			if len(o.Items) != 2 {
				t.Errorf("delivery order has %d items, want 2", len(o.Items))
			}
		} else if len(o.Items) != 1 {
			t.Errorf("pickup order has %d items, want 1", len(o.Items))
		}
	}
	if withAddress != 1 {
		t.Errorf("%d orders have an address, want 1", withAddress)
	}
}

func TestSeedSaleWindows(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := seed.Run(context.Background(), db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	tests := []struct {
		name       string
		start, end int
		days       string
	}{
		{"Завтраки", 540, 720, "1111100"},
		{"Пицца", 720, 1380, ""},
	}
	for _, tc := range tests {
		var row struct {
			Start int    `db:"sale_start"`
			End   int    `db:"sale_end"`
			Days  string `db:"sale_days"`
		}
		err := db.GetContext(context.Background(), &row,
			`SELECT sale_start, sale_end, sale_days FROM categories WHERE name = $1`, tc.name)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if row.Start != tc.start || row.End != tc.end || row.Days != tc.days {
			t.Errorf("%s = %+v, want start %d end %d days %q", tc.name, row, tc.start, tc.end, tc.days)
		}
	}
}
