package catalog_test

import (
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

// getProducts requests path, expects 200 and returns the decoded list.
func getProducts(t *testing.T, h http.Handler, path string) []catalog.ProductView {
	t.Helper()
	rec := testutil.DoJSON(t, h, http.MethodGet, path, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, body = %s", path, rec.Code, rec.Body)
	}
	var got []catalog.ProductView
	testutil.Decode(t, rec, &got)
	return got
}

func viewNames(list []catalog.ProductView) []string {
	names := make([]string, len(list))
	for i, p := range list {
		names[i] = p.Name
	}
	return names
}

func TestProductsByCategoryEndpoint(t *testing.T) {
	r, db := newCatalogEnv(t)

	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Мангал"})
	other := testutil.InsertCategory(t, db, testutil.Category{Name: "Салаты", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Второй", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Первый", SortOrder: 0})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Удалённый", SortOrder: 0, IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: other, Name: "Чужой"})

	got := getProducts(t, r, "/categories/"+strconv.FormatInt(cat, 10)+"/products")

	if want := []string{"Первый", "Второй"}; !reflect.DeepEqual(viewNames(got), want) {
		t.Fatalf("names = %v, want %v", viewNames(got), want)
	}
	for _, p := range got {
		if p.CategoryID != cat {
			t.Errorf("product %q categoryId = %d, want %d", p.Name, p.CategoryID, cat)
		}
	}
}

func TestProductsByCategoryUnknown(t *testing.T) {
	r, _ := newCatalogEnv(t)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/999/products", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestProductsByCategoryBadID(t *testing.T) {
	r, _ := newCatalogEnv(t)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/abc/products", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestProductKeys(t *testing.T) {
	r, db := newCatalogEnv(t)

	cat := testutil.InsertCategory(t, db, testutil.Category{})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык", PriceKopecks: 32000})

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/"+strconv.FormatInt(cat, 10)+"/products", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var raw []map[string]any
	testutil.Decode(t, rec, &raw)
	if len(raw) != 1 {
		t.Fatalf("len = %d, body = %s", len(raw), rec.Body)
	}
	keys := make([]string, 0, len(raw[0]))
	for k := range raw[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{
		"calories", "carbohydrates", "categoryId", "description", "endTime", "fats", "flags", "id",
		"images", "isAvailableForOrder", "measureUnit", "name", "price", "proteins", "salesDays",
		"startTime", "weight",
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if raw[0]["price"] != float64(320) {
		t.Errorf("price = %v, want 320", raw[0]["price"])
	}
	if imgs, ok := raw[0]["images"].([]any); !ok || len(imgs) != 0 {
		t.Errorf("images = %#v, want []", raw[0]["images"])
	}
}

func TestSearchByNames(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Хачапури", SortOrder: 0})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Салат", SortOrder: 2})

	got := getProducts(t, r, "/products?query=хачапури+салат")

	want := []string{"Хачапури", "Салат"}
	if !reflect.DeepEqual(viewNames(got), want) {
		t.Fatalf("names = %v, want %v", viewNames(got), want)
	}
}

func TestSearchUpperCase(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Салат", SortOrder: 1})

	got := getProducts(t, r, "/products?query=%D0%A8%D0%90%D0%A8%D0%9B%D0%AB%D0%9A")

	if want := []string{"Шашлык"}; !reflect.DeepEqual(viewNames(got), want) {
		t.Fatalf("names = %v, want %v", viewNames(got), want)
	}
}

func TestSearchNoQuery(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Хачапури", SortOrder: 0})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Салат", SortOrder: 2})

	for _, path := range []string{"/products", "/products?query=", "/products?query=+%20"} {
		got := getProducts(t, r, path)
		want := []string{"Хачапури", "Шашлык", "Салат"}
		if !reflect.DeepEqual(viewNames(got), want) {
			t.Errorf("GET %s: names = %v, want %v", path, viewNames(got), want)
		}
	}
}

func TestSearchPaging(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Хачапури", SortOrder: 0})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Салат", SortOrder: 2})

	got := getProducts(t, r, "/products?page=2&pageSize=2")
	if want := []string{"Салат"}; !reflect.DeepEqual(viewNames(got), want) {
		t.Fatalf("page 2: names = %v, want %v", viewNames(got), want)
	}

	rec := testutil.DoJSON(t, r, http.MethodGet, "/products?pageSize=0", "", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("pageSize=0: status = %d, want 400", rec.Code)
	}
}

func TestSearchNoMatch(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык"})

	rec := testutil.DoJSON(t, r, http.MethodGet, "/products?query=zzz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestSearchManyWords(t *testing.T) {
	r, db := newCatalogEnv(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Кухня"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык"})

	// 25 words; only the first 20 are used, so the 21st-word match must not count and
	// the request must still succeed.
	q := ""
	for i := 0; i < 20; i++ {
		q += "zzz+"
	}
	q += "шашлык"

	got := getProducts(t, r, "/products?query="+q)
	if len(got) != 0 {
		t.Fatalf("names = %v, want none (word 21 is ignored)", viewNames(got))
	}
}

func TestProductAvailability(t *testing.T) {
	r, db := newCatalogEnv(t)

	weekend := testutil.InsertCategory(t, db, testutil.Category{Name: "Выходные", SaleDays: "0000011"})
	weekday := testutil.InsertCategory(t, db, testutil.Category{Name: "Будни", SortOrder: 1, SaleDays: "1111100"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: weekend, Name: "Воскресный"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: weekday, Name: "Обычный"})

	got := getProducts(t, r, "/categories/"+strconv.FormatInt(weekend, 10)+"/products")
	if len(got) != 1 || got[0].IsAvailableForOrder {
		t.Fatalf("weekend product on Friday must not be available: %+v", got)
	}

	got = getProducts(t, r, "/categories/"+strconv.FormatInt(weekday, 10)+"/products")
	if len(got) != 1 || !got[0].IsAvailableForOrder {
		t.Fatalf("weekday product on Friday must be available: %+v", got)
	}
}
