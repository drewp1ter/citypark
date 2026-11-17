package catalog_test

import (
	"context"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/migrations"
	"github.com/pressly/goose/v3"
)

func TestProductHiddenMigration(t *testing.T) {
	db := testutil.NewDB(t)
	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := provider.DownTo(ctx, 37); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := provider.Up(ctx); err != nil {
			t.Error(err)
		}
	})
	category := testutil.InsertCategory(t, db, testutil.Category{})
	id := testutil.InsertProduct(t, db, testutil.Product{CategoryID: category})
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	var hidden bool
	if err := db.Get(&hidden, `SELECT is_hidden FROM products WHERE id=$1`, id); err != nil || hidden {
		t.Fatalf("existing product default: hidden=%v, err=%v", hidden, err)
	}
	if _, err := db.Exec(`UPDATE products SET is_hidden=NULL WHERE id=$1`, id); err == nil {
		t.Fatal("is_hidden accepted NULL")
	}
	if _, err := provider.DownTo(ctx, 37); err != nil {
		t.Fatal(err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM products WHERE id=$1`, id); n != 1 {
		t.Fatal("rollback lost product")
	}
}
