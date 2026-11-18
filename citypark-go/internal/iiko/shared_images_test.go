package iiko

import (
	"context"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/migrations"
	"github.com/pressly/goose/v3"
)

func TestSyncSharedProductImages(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	const shared = "https://images.test/shared.PNG"
	m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = shared
	m.ItemsGroups[1].Items[0].SizePrices[0].ButtonImageURL = shared
	runSync(t, db, m)
	first := getProduct(t, db, "p-shashlik").ID
	second := getProduct(t, db, "p-khinkali").ID
	var firstImage, secondImage int64
	if err := db.Get(&firstImage, `SELECT id FROM product_images WHERE product_id=$1 AND url=$2`, first, shared); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&secondImage, `SELECT id FROM product_images WHERE product_id=$1 AND url=$2`, second, shared); err != nil {
		t.Fatal(err)
	}
	if firstImage == secondImage {
		t.Fatal("shared image must have a separate association for each dish")
	}
	// Repeated syncs must preserve association IDs and not create duplicates.
	runSync(t, db, m)
	for _, pair := range [][2]int64{{first, firstImage}, {second, secondImage}} {
		if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE product_id=$1 AND id=$2 AND url=$3`, pair[0], pair[1], shared); n != 1 {
			t.Fatal("repeated sync changed image association")
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images`); n != 2 {
		t.Fatalf("got %d images, want 2", n)
	}
	// Both associations reach the catalog read used by admin and public clients.
	products, err := catalog.NewStore(db).SearchProducts(context.Background(), nil, 500, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("got %d products, want 2", len(products))
	}
	for _, p := range products {
		if len(p.Images) != 1 || p.Images[0].URL != shared {
			t.Fatalf("product %d images: %+v", p.ID, p.Images)
		}
	}
	// A manual image may use the same URL, and must survive sync cleanup.
	manual := testutil.InsertProduct(t, db, testutil.Product{CategoryID: getProduct(t, db, "p-shashlik").CategoryID})
	db.MustExec(`INSERT INTO product_images (product_id,url,type) VALUES ($1,$2,'manual')`, manual, shared)
	// One dish removes its image while the other keeps that same URL.
	m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = ""
	runSync(t, db, m)
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE product_id=$1`, first); n != 0 {
		t.Fatal("stale association retained")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE id=$1 AND product_id=$2 AND url=$3`, secondImage, second, shared); n != 1 {
		t.Fatal("other dish lost shared image")
	}
	// Changing the remaining dish's URL must remove its previous association.
	m.ItemsGroups[1].Items[0].SizePrices[0].ButtonImageURL = "https://images.test/new.PNG"
	runSync(t, db, m)
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE type='from_iiko' AND url=$1`, shared); n != 0 {
		t.Fatal("old URL retained")
	}
	m.ItemsGroups = nil
	runSync(t, db, m)
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE type='from_iiko'`); n != 0 {
		t.Fatal("empty menu retained iiko images")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE product_id=$1 AND url=$2 AND type='manual'`, manual, shared); n != 1 {
		t.Fatal("manual image removed")
	}
}

func TestSharedProductImagesMigration(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// A failed assertion must not leave shared URLs blocking later migration tests.
		if _, err := db.Exec(`DELETE FROM product_images`); err != nil {
			t.Error(err)
		}
		if _, err := provider.Up(ctx); err != nil {
			t.Error(err)
		}
	})
	if _, err := provider.DownTo(ctx, 38); err != nil {
		t.Fatal(err)
	}
	category := testutil.InsertCategory(t, db, testutil.Category{})
	first := testutil.InsertProduct(t, db, testutil.Product{CategoryID: category})
	second := testutil.InsertProduct(t, db, testutil.Product{CategoryID: category})
	const shared = "https://images.test/shared.PNG"
	db.MustExec(`INSERT INTO product_images (product_id,url,type) VALUES ($1,$2,'from_iiko')`, first, shared)
	var originalID int64
	if err := db.Get(&originalID, `SELECT id FROM product_images WHERE product_id=$1`, first); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE id=$1 AND product_id=$2 AND url=$3`, originalID, first, shared); n != 1 {
		t.Fatal("migration lost original image")
	}
	db.MustExec(`INSERT INTO product_images (product_id,url,type) VALUES ($1,$2,'from_iiko')`, second, shared)
	if _, err := db.Exec(`INSERT INTO product_images (product_id,url) VALUES ($1,$2)`, first, shared); err == nil {
		t.Fatal("duplicate product/URL pair accepted")
	}
	if _, err := provider.DownTo(ctx, 38); err == nil {
		t.Fatal("rollback must refuse shared URLs without deleting them")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE url=$1`, shared); n != 2 {
		t.Fatal("failed rollback lost associations")
	}
	db.MustExec(`DELETE FROM product_images WHERE product_id=$1`, second)
	if _, err := provider.DownTo(ctx, 38); err != nil {
		t.Fatal(err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE id=$1`, originalID); n != 1 {
		t.Fatal("rollback lost original image")
	}
	if _, err := db.Exec(`INSERT INTO product_images (product_id,url) VALUES ($1,$2)`, second, shared); err == nil {
		t.Fatal("rollback did not restore URL uniqueness")
	}
}

func TestSyncImageErrorsRollBack(t *testing.T) {
	for _, tc := range []struct{ name, table, event, message string }{
		{"category cleanup", "category_images", "DELETE", "delete stale category_images"},
		{"product upsert", "product_images", "INSERT", "upsert product_images"},
		{"product cleanup", "product_images", "DELETE", "delete stale product_images"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			m := oneItemMenu()
			m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = "https://images.test/original.PNG"
			runSync(t, db, m)
			p := getProduct(t, db, "p-1")
			db.MustExec(`INSERT INTO category_images (category_id,url,type) VALUES ($1,'https://images.test/category.PNG','from_iiko')`, p.CategoryID)
			tx, err := db.BeginTxx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			_, err = tx.Exec(`CREATE FUNCTION pg_temp.reject_image_sync() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'image sync test failure'; END $$`)
			if err != nil {
				t.Fatal(err)
			}
			// Table and event are fixed test literals, never external input.
			_, err = tx.Exec(`CREATE TRIGGER reject_image_sync BEFORE ` + tc.event + ` ON ` + tc.table + ` FOR EACH STATEMENT EXECUTE FUNCTION pg_temp.reject_image_sync()`)
			if err != nil {
				t.Fatal(err)
			}
			m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = "https://images.test/new.PNG"
			err = syncImages(context.Background(), tx, m, map[string]int64{"p-1": p.ID})
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v, want %s error", err, tc.message)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if n := testutil.Count(t, db, `SELECT count(*) FROM category_images WHERE type='from_iiko'`); n != 1 {
				t.Fatal("failed sync lost category image")
			}
			if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE url='https://images.test/original.PNG'`); n != 1 {
				t.Fatal("failed sync lost product image")
			}
			if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE url='https://images.test/new.PNG'`); n != 0 {
				t.Fatal("failed sync retained partial write")
			}
		})
	}
}
