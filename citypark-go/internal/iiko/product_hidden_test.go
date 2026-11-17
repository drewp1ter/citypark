package iiko

import (
	"testing"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestSyncHiddenProducts(t *testing.T) {
	db := testutil.NewDB(t)
	m := oneItemMenu()
	m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = "https://images.test/dish.jpg"
	runSync(t, db, m)
	id := getProduct(t, db, "p-1").ID
	db.MustExec(`UPDATE products SET full_description='<p>Описание</p>', is_order_blocked=true WHERE id=$1`, id)
	for _, hidden := range []bool{true, false} {
		m.ItemsGroups[0].Items[0].IsHidden = hidden
		if res := runSync(t, db, m); res.TotalProducts != 1 {
			t.Fatalf("hidden=%v: sync result %+v", hidden, res)
		}
		var row struct {
			ID          int64  `db:"id"`
			Hidden      bool   `db:"is_hidden"`
			Deleted     bool   `db:"is_deleted"`
			Blocked     bool   `db:"is_order_blocked"`
			Description string `db:"full_description"`
		}
		if err := db.Get(&row, `SELECT id, is_hidden, is_deleted, is_order_blocked, full_description FROM products WHERE iiko_id='p-1'`); err != nil {
			t.Fatal(err)
		}
		if row.ID != id || row.Hidden != hidden || row.Deleted || !row.Blocked || row.Description != "<p>Описание</p>" {
			t.Fatalf("hidden=%v: %+v", hidden, row)
		}
		if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE product_id=$1 AND url='https://images.test/dish.jpg' AND type='from_iiko'`, id); n != 1 {
			t.Fatal("visibility transition lost the iiko image")
		}
	}
	m.ItemsGroups[0].Items[0].IsHidden = true
	runSync(t, db, m)
	m.ItemsGroups[0].Items = nil
	runSync(t, db, m)
	if !getProduct(t, db, "p-1").IsDeleted {
		t.Fatal("missing hidden product was not soft-deleted")
	}
	m.ItemsGroups[0].Items = []MenuItem{item("p-1", "Товар", 100, 500)}
	runSync(t, db, m)
	if p := getProduct(t, db, "p-1"); p.ID != id || p.IsDeleted {
		t.Fatalf("reappeared product: %+v", p)
	}
}
