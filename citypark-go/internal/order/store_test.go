package order

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

type seeded struct {
	userID, addressID, categoryID, productA, productB int64
}

func seed(t *testing.T, db *sqlx.DB) seeded {
	t.Helper()
	var s seeded
	s.userID = testutil.InsertUser(t, db, "Иван", "+79990000000", "user")
	s.categoryID = testutil.InsertCategory(t, db, testutil.Category{Name: "Пицца"})
	s.productA = testutil.InsertProduct(t, db, testutil.Product{CategoryID: s.categoryID, Name: "Маргарита", PriceKopecks: 59900})
	s.productB = testutil.InsertProduct(t, db, testutil.Product{CategoryID: s.categoryID, Name: "Кола", PriceKopecks: 10000})
	s.addressID = testutil.InsertAddress(t, db, s.userID, "Москва", "Тверская, 1")
	return s
}

func insertOrder(t *testing.T, db *sqlx.DB, p InsertParams) int64 {
	t.Helper()
	id, err := NewStore(db).Insert(context.Background(), p)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}
	return id
}

func (s seeded) params(withAddress bool) InsertParams {
	p := InsertParams{
		CustomerID:    s.userID,
		Comment:       "позвонить",
		PaymentType:   PaymentCash,
		ItemsHash:     ItemsHash([]ItemInput{{ProductID: s.productA, Amount: 2}, {ProductID: s.productB, Amount: 1}}),
		DeliveryPrice: money.Kopecks(15000),
		Items: []InsertItem{
			{ProductID: s.productA, Amount: 2, OfferedPrice: 59900},
			{ProductID: s.productB, Amount: 1, OfferedPrice: 10000},
		},
	}
	if withAddress {
		p.AddressID = &s.addressID
	}
	return p
}

func TestItemsHash(t *testing.T) {
	a := ItemsHash([]ItemInput{{ProductID: 1, Amount: 2}, {ProductID: 5, Amount: 1}})
	b := ItemsHash([]ItemInput{{ProductID: 5, Amount: 1}, {ProductID: 1, Amount: 2}})
	c := ItemsHash([]ItemInput{{ProductID: 1, Amount: 3}, {ProductID: 5, Amount: 1}})

	if a != b {
		t.Errorf("input order changed the hash: %s != %s", a, b)
	}
	if a == c {
		t.Errorf("different amounts gave the same hash: %s", a)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a) {
		t.Errorf("hash %q is not 64 hex chars", a)
	}
}

func TestItemsHashDoesNotReorderInput(t *testing.T) {
	items := []ItemInput{{ProductID: 5, Amount: 1}, {ProductID: 1, Amount: 2}}
	ItemsHash(items)
	if items[0].ProductID != 5 || items[1].ProductID != 1 {
		t.Errorf("input was reordered: %+v", items)
	}
}

func TestInsertAndByID(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))

	got, err := NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}

	if got.ID != id || got.Status != StatusNew || got.Comment != "позвонить" || got.PaymentType != PaymentCash {
		t.Errorf("order fields: %+v", got)
	}
	if got.Customer != (Customer{ID: sd.userID, Name: "Иван", Phone: "+79990000000"}) {
		t.Errorf("customer: %+v", got.Customer)
	}
	if got.Address == nil || *got.Address != (Address{ID: sd.addressID, City: "Москва", Address: "Тверская, 1"}) {
		t.Errorf("address: %+v", got.Address)
	}
	if got.Total != 129800 {
		t.Errorf("Total = %d, want 129800", got.Total)
	}
	if got.DeliveryPrice != 15000 {
		t.Errorf("DeliveryPrice = %d, want 15000", got.DeliveryPrice)
	}
	if got.ItemsHash != sd.params(true).ItemsHash || got.PaymentID != "" {
		t.Errorf("hash/payment id: %q %q", got.ItemsHash, got.PaymentID)
	}
	if len(got.Items) != 2 {
		t.Fatalf("items = %+v, want 2", got.Items)
	}
	sort.Slice(got.Items, func(i, j int) bool { return got.Items[i].ProductID < got.Items[j].ProductID })
	want := []Item{
		{OrderID: id, ProductID: sd.productA, Name: "Маргарита", CategoryName: "Пицца", Amount: 2, OfferedPrice: 59900},
		{OrderID: id, ProductID: sd.productB, Name: "Кола", CategoryName: "Пицца", Amount: 1, OfferedPrice: 10000},
	}
	for i := range want {
		if got.Items[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got.Items[i], want[i])
		}
	}
}

func TestByIDPickup(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(false))

	got, err := NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Address != nil {
		t.Errorf("Address = %+v, want nil", got.Address)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["address"]) != "null" {
		t.Errorf(`"address" = %s, want null`, m["address"])
	}
}

func TestByIDNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	_, err := NewStore(db).ByID(context.Background(), 12345)
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("err = %v, want store.ErrNotFound", err)
	}
}

func TestInsertBadProduct(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	p := sd.params(true)
	p.Items = append(p.Items, InsertItem{ProductID: 999999, Amount: 1, OfferedPrice: 100})

	err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		_, err := NewStore(tx).Insert(context.Background(), p)
		return err
	})
	if !store.IsFKViolation(err) {
		t.Fatalf("err = %v, want FK violation", err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM orders`); n != 0 {
		t.Errorf("orders rows = %d, want 0 after rollback", n)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM order_items`); n != 0 {
		t.Errorf("order_items rows = %d, want 0 after rollback", n)
	}
}

func TestInsertBadCustomer(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	p := sd.params(true)
	p.CustomerID = 999999

	_, err := NewStore(db).Insert(context.Background(), p)
	if !store.IsFKViolation(err) {
		t.Errorf("err = %v, want FK violation", err)
	}
}

func TestInsertZeroAmount(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	p := sd.params(true)
	p.Items[0].Amount = 0

	err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		_, err := NewStore(tx).Insert(context.Background(), p)
		return err
	})
	if !store.IsCheckViolation(err) {
		t.Errorf("err = %v, want check violation", err)
	}
}

func TestOrderJSON(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))
	got, err := NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"address", "comment", "createdAt", "customer", "deliveryPrice", "id", "items",
		"paymentType", "status", "total", "updatedAt"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
	if m["total"] != float64(1298) {
		t.Errorf("total = %v, want 1298", m["total"])
	}
	if m["deliveryPrice"] != float64(150) {
		t.Errorf("deliveryPrice = %v, want 150", m["deliveryPrice"])
	}
}

func TestDeletedProductStillListed(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	id := insertOrder(t, db, sd.params(true))

	if _, err := db.Exec(`UPDATE products SET is_deleted = true`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE categories SET is_deleted = true`); err != nil {
		t.Fatal(err)
	}

	got, err := NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if len(got.Items) != 2 || got.Total != 129800 {
		t.Errorf("items = %+v, total = %d", got.Items, got.Total)
	}
}

func TestLoadItemsManyOrders(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	first := insertOrder(t, db, sd.params(true))
	p := sd.params(false)
	p.Items = p.Items[:1]
	second := insertOrder(t, db, p)

	s := NewStore(db)
	orders := []Order{{ID: second, Items: []Item{}}, {ID: first, Items: []Item{}}}
	if err := s.loadItems(context.Background(), orders); err != nil {
		t.Fatalf("loadItems: %v", err)
	}
	if len(orders[0].Items) != 1 || orders[0].Total != 119800 {
		t.Errorf("second order: items %d, total %d", len(orders[0].Items), orders[0].Total)
	}
	if len(orders[1].Items) != 2 || orders[1].Total != 129800 {
		t.Errorf("first order: items %d, total %d", len(orders[1].Items), orders[1].Total)
	}
}
