package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func insertCategoryImage(t *testing.T, db *sqlx.DB, categoryID int64, url string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO category_images (category_id, url, type) VALUES ($1, $2, 'image')`, categoryID, url)
	if err != nil {
		t.Fatalf("insert category image: %v", err)
	}
}

func insertProductImage(t *testing.T, db *sqlx.DB, productID int64, url string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO product_images (product_id, url, type) VALUES ($1, $2, 'image')`, productID, url)
	if err != nil {
		t.Fatalf("insert product image: %v", err)
	}
}

func insertTag(t *testing.T, db *sqlx.DB, productID int64, tag string) {
	t.Helper()
	ctx := context.Background()
	var tagID int64
	err := db.GetContext(ctx, &tagID, `INSERT INTO tags (tag) VALUES ($1) RETURNING id`, tag)
	if err != nil {
		t.Fatalf("insert tag: %v", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2)`, productID, tagID)
	if err != nil {
		t.Fatalf("link tag: %v", err)
	}
}

func productNames(list []catalog.Product) []string {
	names := make([]string, len(list))
	for i, p := range list {
		names[i] = p.Name
	}
	return names
}

func mustEqual(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %v, want %v", what, got, want)
	}
}

func TestListCategories(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	third := testutil.InsertCategory(t, db, testutil.Category{Name: "C", SortOrder: 2})
	second := testutil.InsertCategory(t, db, testutil.Category{Name: "B", SortOrder: 1})
	first := testutil.InsertCategory(t, db, testutil.Category{Name: "A", SortOrder: 0})
	testutil.InsertCategory(t, db, testutil.Category{Name: "Deleted", IsDeleted: true})
	insertCategoryImage(t, db, second, "http://img/b.png")

	list, err := catalog.NewStore(db).ListCategories(ctx)
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("got %d categories, want 3", len(list))
	}
	mustEqual(t, "ids", []int64{list[0].ID, list[1].ID, list[2].ID}, []int64{first, second, third})
	if len(list[1].Images) != 1 || list[1].Images[0].URL != "http://img/b.png" || list[1].Images[0].Type != "image" {
		t.Fatalf("images of second = %+v", list[1].Images)
	}
	if len(list[0].Images) != 0 || len(list[2].Images) != 0 {
		t.Fatalf("other categories must have no images: %+v %+v", list[0].Images, list[2].Images)
	}
}

func TestListCategoriesEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	list, err := catalog.NewStore(db).ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("got %#v, want empty non-nil slice", list)
	}
}

func TestCategoryByID(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	id := testutil.InsertCategory(t, db, testutil.Category{
		Name: "Горячее", SaleStart: testutil.IntPtr(540), SaleEnd: testutil.IntPtr(1260), SaleDays: "1111100",
	})
	deleted := testutil.InsertCategory(t, db, testutil.Category{Name: "Old", IsDeleted: true})
	insertCategoryImage(t, db, id, "http://img/hot.png")

	c, err := s.CategoryByID(ctx, id)
	if err != nil {
		t.Fatalf("CategoryByID: %v", err)
	}
	if c.ID != id || c.Name != "Горячее" || c.SaleDays != "1111100" {
		t.Fatalf("category = %+v", c)
	}
	if c.SaleStart == nil || *c.SaleStart != 540 || c.SaleEnd == nil || *c.SaleEnd != 1260 {
		t.Fatalf("window = %v %v", c.SaleStart, c.SaleEnd)
	}
	if len(c.Images) != 1 || c.Images[0].URL != "http://img/hot.png" {
		t.Fatalf("images = %+v", c.Images)
	}

	if _, err := s.CategoryByID(ctx, deleted); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("deleted: err = %v, want ErrNotFound", err)
	}
	if _, err := s.CategoryByID(ctx, 999999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing: err = %v, want ErrNotFound", err)
	}
}

func TestProductsByCategory(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Горячее"})
	other := testutil.InsertCategory(t, db, testutil.Category{Name: "Другое"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Второй", SortOrder: 2, PriceKopecks: 20000})
	first := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Первый", SortOrder: 1, PriceKopecks: 10000})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Удалён", SortOrder: 0, IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: other, Name: "Чужой"})
	insertProductImage(t, db, first, "http://img/first.png")

	list, err := s.ProductsByCategory(ctx, cat)
	if err != nil {
		t.Fatalf("ProductsByCategory: %v", err)
	}
	mustEqual(t, "names", productNames(list), []string{"Первый", "Второй"})
	if list[0].CategoryName != "Горячее" || list[0].CategoryID != cat {
		t.Fatalf("category fields = %q %d", list[0].CategoryName, list[0].CategoryID)
	}
	if list[0].Price != money.Kopecks(10000) {
		t.Fatalf("price = %d", list[0].Price)
	}
	if len(list[0].Images) != 1 || list[0].Images[0].URL != "http://img/first.png" {
		t.Fatalf("images = %+v", list[0].Images)
	}
	if len(list[1].Images) != 0 {
		t.Fatalf("second product images = %+v", list[1].Images)
	}

	deletedCat := testutil.InsertCategory(t, db, testutil.Category{Name: "Закрыта", IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: deletedCat, Name: "Скрыт"})
	list, err = s.ProductsByCategory(ctx, deletedCat)
	if err != nil {
		t.Fatalf("ProductsByCategory deleted category: %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("deleted category: got %#v, want empty non-nil slice", list)
	}
}

func TestSearchByName(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Хачапури по-аджарски", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Шашлык из баранины", SortOrder: 2})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Салат", SortOrder: 3})

	list, err := s.SearchProducts(ctx, []string{"шашлыки"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "stemmed match", productNames(list), []string{"Шашлык из баранины"})

	list, err = s.SearchProducts(ctx, []string{"хачапури", "салат"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "any word", productNames(list), []string{"Хачапури по-аджарски", "Салат"})
}

func TestSearchByCategoryName(t *testing.T) {
	db := testutil.NewDB(t)
	s := catalog.NewStore(db)
	georgian := testutil.InsertCategory(t, db, testutil.Category{Name: "Грузинская кухня"})
	other := testutil.InsertCategory(t, db, testutil.Category{Name: "Напитки"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: georgian, Name: "Хинкали", SortOrder: 1})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: georgian, Name: "Лобио", SortOrder: 2})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: other, Name: "Чай"})

	list, err := s.SearchProducts(context.Background(), []string{"кухня"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "names", productNames(list), []string{"Хинкали", "Лобио"})
}

func TestSearchByTag(t *testing.T) {
	db := testutil.NewDB(t)
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	spicy := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Харчо"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Лобио"})
	insertTag(t, db, spicy, "острое")

	list, err := s.SearchProducts(context.Background(), []string{"острое"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "names", productNames(list), []string{"Харчо"})
}

func TestSearchEmptyWords(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	for i, name := range []string{"Один", "Два", "Три", "Четыре"} {
		testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: name, SortOrder: i})
	}
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Удалён", IsDeleted: true})

	list, err := s.SearchProducts(ctx, []string{}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "all", productNames(list), []string{"Один", "Два", "Три", "Четыре"})

	list, err = s.SearchProducts(ctx, nil, 2, 1)
	if err != nil {
		t.Fatalf("SearchProducts paged: %v", err)
	}
	mustEqual(t, "page", productNames(list), []string{"Два", "Три"})
}

func TestSearchLoadsImages(t *testing.T) {
	db := testutil.NewDB(t)
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	id := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Хинкали"})
	insertProductImage(t, db, id, "http://img/khinkali.png")

	list, err := catalog.NewStore(db).SearchProducts(context.Background(), []string{"хинкали"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	if len(list) != 1 || len(list[0].Images) != 1 || list[0].Images[0].URL != "http://img/khinkali.png" {
		t.Fatalf("list = %+v", list)
	}
}

func TestSearchExcludesDeleted(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Салаты"})
	deletedCat := testutil.InsertCategory(t, db, testutil.Category{Name: "Салаты старые", IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Цезарь салат"})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Греческий салат", IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: deletedCat, Name: "Столичный салат"})

	list, err := s.SearchProducts(ctx, []string{"салат"}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "with words", productNames(list), []string{"Цезарь салат"})

	list, err = s.SearchProducts(ctx, []string{}, 10, 0)
	if err != nil {
		t.Fatalf("SearchProducts: %v", err)
	}
	mustEqual(t, "without words", productNames(list), []string{"Цезарь салат"})
}

func TestProductsByIDs(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := catalog.NewStore(db)
	cat := testutil.InsertCategory(t, db, testutil.Category{
		Name: "Горячее", SaleStart: testutil.IntPtr(600), SaleEnd: testutil.IntPtr(1200), SaleDays: "1111111",
	})
	a := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "А", PriceKopecks: 100})
	b := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Б", PriceKopecks: 200})
	deleted := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "В", IsDeleted: true})
	testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, Name: "Не запрошен"})
	insertProductImage(t, db, a, "http://img/a.png")

	list, err := s.ProductsByIDs(ctx, []int64{a, b, deleted})
	if err != nil {
		t.Fatalf("ProductsByIDs: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d products, want 2: %v", len(list), productNames(list))
	}
	got := map[string]catalog.Product{}
	for _, p := range list {
		got[p.Name] = p
	}
	pa, pb := got["А"], got["Б"]
	if pa.ID != a || pb.ID != b {
		t.Fatalf("ids = %d %d, want %d %d", pa.ID, pb.ID, a, b)
	}
	if pa.Price != 100 || pb.Price != 200 {
		t.Fatalf("prices = %d %d", pa.Price, pb.Price)
	}
	if pa.CategoryName != "Горячее" || pa.CategorySaleDays != "1111111" ||
		pa.CategorySaleStart == nil || *pa.CategorySaleStart != 600 ||
		pa.CategorySaleEnd == nil || *pa.CategorySaleEnd != 1200 {
		t.Fatalf("category window fields = %+v", pa)
	}
	if pa.Images != nil {
		t.Fatalf("images must not be loaded: %+v", pa.Images)
	}

	list, err = s.ProductsByIDs(ctx, []int64{})
	if err != nil {
		t.Fatalf("ProductsByIDs empty: %v", err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("empty ids: got %#v, want empty non-nil slice", list)
	}
}

func TestCategoryView(t *testing.T) {
	c := catalog.Category{
		ID: 7, Name: "Горячее", Description: "Вкусно",
		SaleStart: testutil.IntPtr(9 * 60), SaleEnd: testutil.IntPtr(21 * 60), SaleDays: "1111100",
	}
	loc := time.UTC
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"friday afternoon", time.Date(2024, 5, 24, 15, 0, 0, 0, loc), true},
		{"saturday afternoon", time.Date(2024, 5, 25, 15, 0, 0, 0, loc), false},
		{"friday early", time.Date(2024, 5, 24, 5, 0, 0, 0, loc), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := c.View(tt.now)
			if v.IsAvailableForOrder != tt.want {
				t.Fatalf("isAvailableForOrder = %v, want %v", v.IsAvailableForOrder, tt.want)
			}
			if v.SalesDays != "Пн, Вт, Ср, Чт, Пт" || v.StartTime != "9:00" || v.EndTime != "21:00" {
				t.Fatalf("view = %+v", v)
			}
		})
	}

	raw, err := json.Marshal(c.View(time.Date(2024, 5, 24, 15, 0, 0, 0, loc)))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	want := []string{"id", "name", "description", "images", "isAvailableForOrder", "salesDays", "startTime", "endTime"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing key %q in %s", k, raw)
		}
	}
	if m["id"] != float64(7) || m["name"] != "Горячее" || m["isAvailableForOrder"] != true {
		t.Fatalf("json = %s", raw)
	}
}

func TestProductView(t *testing.T) {
	now := time.Date(2024, 5, 24, 5, 0, 0, 0, time.UTC)
	p := catalog.Product{
		ID: 3, CategoryID: 7, Name: "Хинкали", Price: 599,
		CategorySaleStart: testutil.IntPtr(9 * 60), CategorySaleEnd: testutil.IntPtr(21 * 60),
		CategorySaleDays: "1111111",
	}
	v := p.View(now)
	if v.IsAvailableForOrder {
		t.Fatalf("product in a closed category must not be available")
	}
	if v.SalesDays != "" || v.StartTime != "" || v.EndTime != "" {
		t.Fatalf("own window must be empty: %+v", v)
	}
	if v.CategoryID != 7 || v.Price != 599 {
		t.Fatalf("view = %+v", v)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(raw)
	for _, want := range []string{`"price":5.99`, `"images":[]`, `"categoryId":7`, `"isAvailableForOrder":false`,
		`"measureUnit":""`, `"salesDays":""`, `"startTime":""`, `"endTime":""`} {
		if !strings.Contains(s, want) {
			t.Fatalf("json %s does not contain %s", s, want)
		}
	}

	open := time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)
	if !p.View(open).IsAvailableForOrder {
		t.Fatalf("product must be available when its category is open")
	}
	p.SaleStart, p.SaleEnd = testutil.IntPtr(13*60), testutil.IntPtr(14*60)
	if p.View(open).IsAvailableForOrder {
		t.Fatalf("product must be unavailable outside its own window")
	}
}

func TestViewKeepsImages(t *testing.T) {
	imgs := []catalog.Image{{ID: 1, URL: "http://img/1.png", Type: "image", ParentID: 9}}
	now := time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)

	cv := catalog.Category{Images: imgs}.View(now)
	pv := catalog.Product{Images: imgs}.View(now)
	mustEqual(t, "category images", cv.Images, imgs)
	mustEqual(t, "product images", pv.Images, imgs)

	raw, err := json.Marshal(pv.Images)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `[{"id":1,"url":"http://img/1.png","type":"image"}]` {
		t.Fatalf("images json = %s", raw)
	}
}
