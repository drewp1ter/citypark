package iiko

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type fakeSource struct {
	n   *Menu
	err error
}

func (f *fakeSource) Fetch(context.Context) (*Menu, error) { return f.n, f.err }

func price(v float64) *float64 { return &v }

// item is a one-size item; rubles < 0 means the size has no price.
func item(productID, name string, rubles, grams float64) MenuItem {
	size := ItemSizePrice{IsDefault: true, Weight: grams}
	if rubles >= 0 {
		size.Price = price(rubles)
	}
	return MenuItem{ProductID: productID, Name: name, SizePrices: []ItemSizePrice{size}}
}

// baseMenu has two groups: "g-mangal" with one priced and one priceless item,
// and "g-khinkali" with one priced item.
func baseMenu() *Menu {
	return &Menu{
		ID:   "menu-1",
		Name: "Основное меню",
		ItemsGroups: []ItemsGroup{
			{ID: "g-mangal", Name: " Мангал ", Items: []MenuItem{
				item("p-shashlik", "Шашлык", 600, 1000),
				item("p-noprice", "Без цены", -1, 100),
			}},
			{ID: "g-khinkali", Name: "Хинкали", Items: []MenuItem{
				item("p-khinkali", "Хинкали", 50, 80),
			}},
		},
		Products: []MenuProduct{
			{ID: "p-shashlik", MeasureUnit: "кг"},
			{ID: "p-khinkali", MeasureUnit: "порц"},
		},
	}
}

// oneItemMenu has one group with one priced item.
func oneItemMenu() *Menu {
	return &Menu{
		ID:   "menu-1",
		Name: "Меню",
		ItemsGroups: []ItemsGroup{
			{ID: "g-1", Name: "Группа", Items: []MenuItem{item("p-1", "Товар", 100, 500)}},
		},
	}
}

func runSync(t *testing.T, db *sqlx.DB, m *Menu) Result {
	t.Helper()
	res, err := NewSyncer(db, &fakeSource{n: m}).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	return res
}

type categoryRow struct {
	ID          int64  `db:"id"`
	MenuID      *int64 `db:"menu_id"`
	Name        string `db:"name"`
	Description string `db:"description"`
	SortOrder   int    `db:"sort_order"`
	IsDeleted   bool   `db:"is_deleted"`
	SaleStart   *int   `db:"sale_start"`
	SaleEnd     *int   `db:"sale_end"`
	SaleDays    string `db:"sale_days"`
	IsHidden    bool   `db:"is_hidden"`
}

type productRow struct {
	ID           int64   `db:"id"`
	CategoryID   int64   `db:"category_id"`
	Name         string  `db:"name"`
	Description  string  `db:"description"`
	MeasureUnit  string  `db:"measure_unit"`
	Weight       float64 `db:"weight"`
	SortOrder    int     `db:"sort_order"`
	Fats         float64 `db:"fats"`
	Proteins     float64 `db:"proteins"`
	Carbs        float64 `db:"carbohydrates"`
	Calories     float64 `db:"calories"`
	PriceKopecks int64   `db:"price_kopecks"`
	IsDeleted    bool    `db:"is_deleted"`
	SaleStart    *int    `db:"sale_start"`
	SaleEnd      *int    `db:"sale_end"`
	SaleDays     string  `db:"sale_days"`
}

func getCategory(t *testing.T, db *sqlx.DB, iikoID string) categoryRow {
	t.Helper()
	var c categoryRow
	err := db.GetContext(context.Background(), &c,
		`SELECT id, menu_id, name, description, sort_order, is_deleted, sale_start, sale_end, sale_days, is_hidden
		 FROM categories WHERE iiko_id = $1`, iikoID)
	if err != nil {
		t.Fatalf("select category %s: %v", iikoID, err)
	}
	return c
}

func getProduct(t *testing.T, db *sqlx.DB, iikoID string) productRow {
	t.Helper()
	var p productRow
	err := db.GetContext(context.Background(), &p,
		`SELECT id, category_id, name, description, measure_unit, weight, sort_order, fats, proteins,
		        carbohydrates, calories, price_kopecks, is_deleted, sale_start, sale_end, sale_days
		 FROM products WHERE iiko_id = $1`, iikoID)
	if err != nil {
		t.Fatalf("select product %s: %v", iikoID, err)
	}
	return p
}

func TestPickSize(t *testing.T) {
	tests := []struct {
		name   string
		sizes  []ItemSizePrice
		wantOK bool
		want   string // sku of the chosen size
	}{
		{"no sizes", nil, false, ""},
		{"default without price", []ItemSizePrice{{IsDefault: true, SKU: "a"}}, false, ""},
		{"default is second", []ItemSizePrice{
			{SKU: "a", Price: price(1)}, {IsDefault: true, SKU: "b", Price: price(2)},
		}, true, "b"},
		{"no default takes first", []ItemSizePrice{
			{SKU: "a", Price: price(1)}, {SKU: "b", Price: price(2)},
		}, true, "a"},
		{"first without price", []ItemSizePrice{
			{SKU: "a"}, {SKU: "b", Price: price(2)},
		}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pickSize(tt.sizes)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.SKU != tt.want {
				t.Errorf("size = %q, want %q", got.SKU, tt.want)
			}
		})
	}
}

func TestSyncCreatesMenu(t *testing.T) {
	db := testutil.NewDB(t)

	runSync(t, db, baseMenu())
	if n := testutil.Count(t, db, `SELECT count(*) FROM menus WHERE iiko_id = 'menu-1' AND name = 'Основное меню'`); n != 1 {
		t.Errorf("menus rows = %d, want 1", n)
	}

	m := baseMenu()
	m.Name = "Новое имя"
	runSync(t, db, m)
	if n := testutil.Count(t, db, `SELECT count(*) FROM menus`); n != 1 {
		t.Errorf("menus rows = %d, want 1", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM menus WHERE name = 'Новое имя'`); n != 1 {
		t.Errorf("menu name was not updated")
	}
}

func TestSyncCategories(t *testing.T) {
	db := testutil.NewDB(t)

	runSync(t, db, baseMenu())

	var menuID int64
	if err := db.Get(&menuID, `SELECT id FROM menus WHERE iiko_id = 'menu-1'`); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		iikoID, name string
		order        int
	}{{"g-mangal", "Мангал", 0}, {"g-khinkali", "Хинкали", 1}} {
		c := getCategory(t, db, tt.iikoID)
		if c.MenuID == nil || *c.MenuID != menuID {
			t.Errorf("%s: menu_id = %v, want %d", tt.iikoID, c.MenuID, menuID)
		}
		if c.Name != tt.name || c.SortOrder != tt.order || c.Description != "" || c.IsDeleted {
			t.Errorf("%s: category = %+v, want name %q, sort_order %d, empty description", tt.iikoID, c, tt.name, tt.order)
		}
		if c.SaleStart != nil || c.SaleEnd != nil || c.SaleDays != "" {
			t.Errorf("%s: sale window = %v/%v/%q, want NULL/NULL/''", tt.iikoID, c.SaleStart, c.SaleEnd, c.SaleDays)
		}
	}
}

func TestSyncHiddenCategories(t *testing.T) {
	db := testutil.NewDB(t)

	m := baseMenu()
	m.ItemsGroups[0].IsHidden = true
	res := runSync(t, db, m)
	if res.TotalCategories != 2 {
		t.Errorf("TotalCategories = %d, want 2", res.TotalCategories)
	}

	a, b := getCategory(t, db, "g-mangal"), getCategory(t, db, "g-khinkali")
	if !a.IsHidden || a.IsDeleted {
		t.Errorf("hidden group: is_hidden = %v, is_deleted = %v, want true/false", a.IsHidden, a.IsDeleted)
	}
	if b.IsHidden {
		t.Errorf("visible group: is_hidden = true, want false")
	}
	if p := getProduct(t, db, "p-shashlik"); p.IsDeleted {
		t.Errorf("product of a hidden group is deleted: %+v", p)
	}

	m.ItemsGroups[0].IsHidden = false
	runSync(t, db, m)
	if getCategory(t, db, "g-mangal").IsHidden {
		t.Errorf("group no longer hidden: is_hidden = true, want false")
	}
}

func TestSyncCategoryWithoutSellableItems(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	m.ItemsGroups = append(m.ItemsGroups, ItemsGroup{ID: "g-empty", Name: "Пусто"})

	res := runSync(t, db, m)

	if res.TotalCategories != 3 {
		t.Errorf("TotalCategories = %d, want 3", res.TotalCategories)
	}
	if c := getCategory(t, db, "g-empty"); c.SortOrder != 2 {
		t.Errorf("sort_order = %d, want 2", c.SortOrder)
	}
}

func TestSyncProducts(t *testing.T) {
	db := testutil.NewDB(t)

	res := runSync(t, db, baseMenu())

	if res != (Result{TotalCategories: 2, TotalProducts: 2}) {
		t.Errorf("Result = %+v, want 2 categories and 2 products", res)
	}
	p := getProduct(t, db, "p-shashlik")
	if p.PriceKopecks != 60000 || p.Weight != 1.0 || p.MeasureUnit != "кг" || p.SortOrder != 0 {
		t.Errorf("product = %+v, want price 60000, weight 1, unit кг, sort_order 0", p)
	}
	if p.CategoryID != getCategory(t, db, "g-mangal").ID {
		t.Errorf("category_id = %d, want the category of g-mangal", p.CategoryID)
	}
	if p.Name != "Шашлык" || p.Description != "" {
		t.Errorf("name/description = %q/%q", p.Name, p.Description)
	}
	if p.Fats != 0 || p.Proteins != 0 || p.Carbs != 0 || p.Calories != 0 || p.IsDeleted {
		t.Errorf("product = %+v, want zero nutrition and not deleted", p)
	}

	k := getProduct(t, db, "p-khinkali")
	if k.Weight != 0.08 || k.PriceKopecks != 5000 || k.SortOrder != 0 || k.MeasureUnit != "порц" {
		t.Errorf("khinkali = %+v", k)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM products WHERE iiko_id = 'p-noprice'`); n != 0 {
		t.Errorf("priceless item was saved")
	}
}

func TestSyncProductSortOrderIsItemIndex(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	// The priceless item keeps its index: the next item is j = 2.
	m.ItemsGroups[0].Items = append(m.ItemsGroups[0].Items, item("p-third", "Третий", 10, 0))

	runSync(t, db, m)

	if got := getProduct(t, db, "p-third").SortOrder; got != 2 {
		t.Errorf("sort_order = %d, want 2", got)
	}
}

func TestSyncUnknownMeasureUnit(t *testing.T) {
	db := testutil.NewDB(t)
	m := oneItemMenu() // no Products entries

	runSync(t, db, m)

	if got := getProduct(t, db, "p-1").MeasureUnit; got != "" {
		t.Errorf("measure_unit = %q, want empty", got)
	}
}

func TestSyncDuplicateProductLaterWins(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	m.ItemsGroups[1].Items = append(m.ItemsGroups[1].Items, item("p-shashlik", "Шашлык-2", 700, 500))

	res := runSync(t, db, m)

	if res.TotalProducts != 2 {
		t.Errorf("TotalProducts = %d, want 2", res.TotalProducts)
	}
	p := getProduct(t, db, "p-shashlik")
	if p.Name != "Шашлык-2" || p.PriceKopecks != 70000 || p.CategoryID != getCategory(t, db, "g-khinkali").ID {
		t.Errorf("product = %+v, want the later row", p)
	}
}

func TestSyncNutrition(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	m.ItemsGroups[0].Items[0].SizePrices[0].Nutritions = Nutritions{Fats: 20, Proteins: 10, Carbs: 30, Energy: 40}

	runSync(t, db, m)

	p := getProduct(t, db, "p-shashlik")
	if p.Fats != 20 || p.Proteins != 10 || p.Carbs != 30 || p.Calories != 40 {
		t.Errorf("product = %+v, want fats 20, proteins 10, carbohydrates 30, calories 40", p)
	}
}

func productImages(t *testing.T, db *sqlx.DB) map[string]int64 {
	t.Helper()
	var rows []struct {
		URL       string `db:"url"`
		ProductID int64  `db:"product_id"`
	}
	if err := db.Select(&rows, `SELECT url, product_id FROM product_images WHERE type = 'from_iiko'`); err != nil {
		t.Fatal(err)
	}
	got := make(map[string]int64, len(rows))
	for _, r := range rows {
		got[r.URL] = r.ProductID
	}
	return got
}

func TestSyncProductImages(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = "http://i/shashlik"
	m.ItemsGroups[0].Items[1].SizePrices[0].ButtonImageURL = "http://i/noprice" // not for sale

	runSync(t, db, m)

	got := productImages(t, db)
	if len(got) != 1 || got["http://i/shashlik"] != getProduct(t, db, "p-shashlik").ID {
		t.Errorf("images = %v, want only http://i/shashlik of p-shashlik", got)
	}

	// The url moves to another product; the old one is gone from the menu.
	m.ItemsGroups[0].Items[0].SizePrices[0].ButtonImageURL = ""
	m.ItemsGroups[1].Items[0].SizePrices[0].ButtonImageURL = "http://i/khinkali"
	runSync(t, db, m)

	got = productImages(t, db)
	if len(got) != 1 || got["http://i/khinkali"] != getProduct(t, db, "p-khinkali").ID {
		t.Errorf("images = %v, want only http://i/khinkali of p-khinkali", got)
	}
}

func menuWithDescription(desc string) *Menu {
	m := baseMenu()
	for gi := range m.ItemsGroups {
		for ii := range m.ItemsGroups[gi].Items {
			if m.ItemsGroups[gi].Items[ii].ProductID == "p-shashlik" {
				m.ItemsGroups[gi].Items[ii].Description = desc
			}
		}
	}
	return m
}

func TestSyncProductDescription(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "Свекла, чеснок", "Свекла, чеснок"},
		{"trimmed", "  Свекла  \n", "Свекла"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			runSync(t, db, menuWithDescription(tt.in))
			if got := getProduct(t, db, "p-shashlik").Description; got != tt.want {
				t.Errorf("description = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSyncProductDescriptionUpdated(t *testing.T) {
	db := testutil.NewDB(t)
	for _, want := range []string{"Старое", "Новое", ""} {
		runSync(t, db, menuWithDescription(want))
		if got := getProduct(t, db, "p-shashlik").Description; got != want {
			t.Errorf("description = %q, want %q", got, want)
		}
	}
}

func TestSyncClearsOldData(t *testing.T) {
	db := testutil.NewDB(t)
	// An iiko category and product as the old nomenclature sync left them.
	var catID, prodID int64
	if err := db.Get(&catID, `INSERT INTO categories (iiko_id, name, description, sale_start, sale_end, sale_days)
		VALUES ('g-mangal', 'Старая', 'Описание', 600, 1200, '1111100') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&prodID, `INSERT INTO products (iiko_id, category_id, name, description, fats, proteins,
		carbohydrates, calories, price_kopecks, sale_start, sale_end, sale_days)
		VALUES ('p-shashlik', $1, 'Старый', 'Вкусно', 1, 2, 3, 4, 100, 600, 1200, '1111100') RETURNING id`, catID); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO category_images (category_id, url, type) VALUES (` + strconv.FormatInt(catID, 10) + `, 'http://i/c1', 'from_iiko')`,
		`INSERT INTO category_images (category_id, url, type) VALUES (` + strconv.FormatInt(catID, 10) + `, 'http://i/c2', 'manual')`,
		`INSERT INTO product_images (product_id, url, type) VALUES (` + strconv.FormatInt(prodID, 10) + `, 'http://i/p1', 'from_iiko')`,
		`INSERT INTO product_images (product_id, url, type) VALUES (` + strconv.FormatInt(prodID, 10) + `, 'http://i/p2', 'manual')`,
		`INSERT INTO tags (tag) VALUES ('острое')`,
		`INSERT INTO product_tags (product_id, tag_id) SELECT ` + strconv.FormatInt(prodID, 10) + `, id FROM tags`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}

	runSync(t, db, baseMenu())

	c := getCategory(t, db, "g-mangal")
	if c.Description != "" || c.SaleStart != nil || c.SaleEnd != nil || c.SaleDays != "" {
		t.Errorf("category = %+v, want description and sale window cleared", c)
	}
	p := getProduct(t, db, "p-shashlik")
	if p.Description != "" || p.Fats != 0 || p.Proteins != 0 || p.Carbs != 0 || p.Calories != 0 ||
		p.SaleStart != nil || p.SaleEnd != nil || p.SaleDays != "" {
		t.Errorf("product = %+v, want description replaced from iiko (empty), nutrition and sale window cleared", p)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM category_images WHERE type = 'from_iiko'`); n != 0 {
		t.Errorf("from_iiko category images = %d, want 0", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE type = 'from_iiko'`); n != 0 {
		t.Errorf("from_iiko product images = %d, want 0", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM category_images WHERE type = 'manual'`); n != 1 {
		t.Errorf("manual category images = %d, want 1", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_images WHERE type = 'manual'`); n != 1 {
		t.Errorf("manual product images = %d, want 1", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM product_tags`); n != 0 {
		t.Errorf("product_tags rows = %d, want 0", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM tags`); n != 0 {
		t.Errorf("tags rows = %d, want 0", n)
	}
}

func TestSyncSoftDeletesMissing(t *testing.T) {
	db := testutil.NewDB(t)
	var manualID int64
	if err := db.Get(&manualID, `INSERT INTO categories (name) VALUES ('Ручная') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	runSync(t, db, baseMenu())

	m := baseMenu()
	m.ItemsGroups = m.ItemsGroups[:1]                   // drop g-khinkali with p-khinkali
	m.ItemsGroups[0].Items = m.ItemsGroups[0].Items[1:] // drop p-shashlik
	res := runSync(t, db, m)

	if res != (Result{TotalCategories: 1, TotalProducts: 0}) {
		t.Errorf("Result = %+v", res)
	}
	if !getCategory(t, db, "g-khinkali").IsDeleted {
		t.Error("missing category is not deleted")
	}
	if getCategory(t, db, "g-mangal").IsDeleted {
		t.Error("kept category is deleted")
	}
	for _, id := range []string{"p-shashlik", "p-khinkali"} {
		if !getProduct(t, db, id).IsDeleted {
			t.Errorf("product %s is not deleted", id)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM categories WHERE id = $1 AND NOT is_deleted`, manualID); n != 1 {
		t.Error("category without iiko_id was touched")
	}

	// A returning group is restored.
	runSync(t, db, baseMenu())
	if getCategory(t, db, "g-khinkali").IsDeleted || getProduct(t, db, "p-shashlik").IsDeleted {
		t.Error("rows were not restored")
	}
}

func TestSyncFetchError(t *testing.T) {
	db := testutil.NewDB(t)
	boom := errors.New("iiko is down")

	_, err := NewSyncer(db, &fakeSource{err: boom}).Sync(context.Background())

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	for _, table := range []string{"menus", "categories", "products"} {
		if n := testutil.Count(t, db, `SELECT count(*) FROM `+table); n != 0 {
			t.Errorf("%s rows = %d, want 0", table, n)
		}
	}
}

func TestSyncRollsBackOnError(t *testing.T) {
	db := testutil.NewDB(t)
	m := baseMenu()
	// A negative weight violates products.weight CHECK (weight >= 0) after the menu and categories were written.
	m.ItemsGroups[1].Items[0].SizePrices[0].Weight = -1

	_, err := NewSyncer(db, &fakeSource{n: m}).Sync(context.Background())

	if err == nil {
		t.Fatal("want error")
	}
	for _, table := range []string{"menus", "categories", "products"} {
		if n := testutil.Count(t, db, `SELECT count(*) FROM `+table); n != 0 {
			t.Errorf("%s rows = %d, want 0 after rollback", table, n)
		}
	}
}

func TestSyncRunningReturnsErr(t *testing.T) {
	db := testutil.NewDB(t)
	src := newBlockingSource(&fakeSource{n: oneItemMenu()})
	syncer := NewSyncer(db, src)

	done := make(chan error, 1)
	go func() {
		_, err := syncer.Sync(context.Background())
		done <- err
	}()
	<-src.entered

	if _, err := syncer.Sync(context.Background()); !errors.Is(err, ErrSyncRunning) {
		t.Errorf("second Sync error = %v, want ErrSyncRunning", err)
	}

	close(src.release)
	if err := <-done; err != nil {
		t.Errorf("first Sync error = %v, want nil", err)
	}
}

func TestSyncUnlocksAfterError(t *testing.T) {
	db := testutil.NewDB(t)
	src := &switchSource{err: errors.New("iiko is down")}
	syncer := NewSyncer(db, src)

	if _, err := syncer.Sync(context.Background()); err == nil {
		t.Fatal("first Sync error = nil, want an error")
	}

	src.err, src.n = nil, oneItemMenu()
	res, err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("second Sync error = %v, want nil", err)
	}
	if want := (Result{TotalCategories: 1, TotalProducts: 1}); res != want {
		t.Errorf("result = %+v, want %+v", res, want)
	}
}

// switchSource is a fakeSource whose answer a test can change between calls.
type switchSource struct {
	n   *Menu
	err error
}

func (s *switchSource) Fetch(context.Context) (*Menu, error) { return s.n, s.err }
