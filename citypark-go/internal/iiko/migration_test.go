package iiko

import (
	"testing"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestMigrationMenus(t *testing.T) {
	db := testutil.NewDB(t)

	var menuID int64
	if err := db.QueryRow(`INSERT INTO menus (iiko_id, name) VALUES ('94764', 'test') RETURNING id`).Scan(&menuID); err != nil {
		t.Fatal(err)
	}
	var withMenu, withoutMenu int64
	if err := db.QueryRow(`INSERT INTO categories (name, menu_id) VALUES ('A', $1) RETURNING id`, menuID).Scan(&withMenu); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO categories (name) VALUES ('B') RETURNING id`).Scan(&withoutMenu); err != nil {
		t.Fatal(err)
	}

	var got *int64
	if err := db.Get(&got, `SELECT menu_id FROM categories WHERE id = $1`, withMenu); err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != menuID {
		t.Errorf("menu_id = %v, want %d", got, menuID)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM categories WHERE id = $1 AND menu_id IS NULL`, withoutMenu); n != 1 {
		t.Errorf("category without menu: menu_id IS NULL rows = %d, want 1", n)
	}
}
