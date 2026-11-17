package catalog_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestHiddenProductResponses(t *testing.T) {
	r, db := newCatalogEnv(t)
	category := testutil.InsertCategory(t, db, testutil.Category{})
	id := testutil.InsertProduct(t, db, testutil.Product{CategoryID: category, Name: "Шашлык"})
	for _, hidden := range []bool{false, true, false} {
		if _, err := db.Exec(`UPDATE products SET is_hidden=$2 WHERE id=$1`, id, hidden); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/products", "/products?query=шашлык", "/categories/" + strconv.FormatInt(category, 10) + "/products"} {
			got := getProducts(t, r, path)
			if len(got) != 1 || got[0].IsHidden != hidden || got[0].IsAvailableForOrder == hidden || got[0].IsOrderBlocked {
				t.Fatalf("GET %s, hidden=%v: %+v", path, hidden, got)
			}
		}
		products, err := catalog.NewStore(db).ProductsByIDs(context.Background(), []int64{id})
		if err != nil || len(products) != 1 || products[0].IsHidden != hidden {
			t.Fatalf("ProductsByIDs, hidden=%v: %+v, %v", hidden, products, err)
		}
	}
}
