package address

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func bookFixture(t *testing.T) (*sqlx.DB, *Service) {
	t.Helper()
	db := testutil.NewDB(t)
	if _, err := db.Exec(`INSERT INTO users(id,name,phone) VALUES(1,'One','1'),(2,'Two','2')`); err != nil {
		t.Fatal(err)
	}
	return db, NewService(db)
}
func createBookEntry(t *testing.T, s *Service, user int64, in Fields) Entry {
	t.Helper()
	e, created, err := s.CreateEntry(context.Background(), user, in)
	if err != nil || !created {
		t.Fatalf("create: %+v %v %v", e, created, err)
	}
	return e
}
func bookExec(t *testing.T, db *sqlx.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
func bookEntries(t *testing.T, db *sqlx.DB, user int64) []Entry {
	t.Helper()
	e, err := NewStore(db).ListEntries(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestCreateEntry(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: " Город ", Street: " Дом ", Name: " "})
	if !a.IsDefault || a.NeedsReview || a.Name != "" || a.City != "Город" || a.Street != "Дом" {
		t.Fatal(a)
	}
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	if b.IsDefault {
		t.Fatal(b)
	}
	bookExec(t, db, `UPDATE user_addresses SET is_default=false,updated_at=now()`)
	c := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 3"})
	if c.IsDefault {
		t.Fatal(c)
	}
	bookExec(t, db, `DELETE FROM user_addresses WHERE id=$1`, a.ID)
	d := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 4"})
	if d.IsDefault || len(bookEntries(t, db, 1)) != 3 {
		t.Fatal(d)
	}
}
func TestCreateEntryDuplicate(t *testing.T) {
	db, s := bookFixture(t)
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Домой", City: "Й город", Street: "Дом", Apartment: "1", Entrance: "2"}
	a := createBookEntry(t, s, 1, in)
	for _, candidate := range []Fields{in, {RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Другое", City: "и\u0306\u00a0ГОРОД", Street: "ДОМ", Apartment: "1", Entrance: "2"}} {
		e, created, err := s.CreateEntry(context.Background(), 1, candidate)
		if err != nil || created || e != a {
			t.Fatal(e, created, err)
		}
	}
	in.Apartment = "2"
	createBookEntry(t, s, 1, in)
	in.Entrance = "3"
	createBookEntry(t, s, 1, in)
	if got := bookEntries(t, db, 1); len(got) != 3 || got[0] != a {
		t.Fatal(got)
	}
}
func TestCreateLegacyDuplicate(t *testing.T) {
	db, s := bookFixture(t)
	bookExec(t, db, `INSERT INTO addresses(id,city,address) VALUES(1,'Город','Дом'),(2,' ГОРОД ',' ДОМ '); INSERT INTO user_addresses(user_id,address_id) VALUES(1,1),(1,2)`)
	before := bookEntries(t, db, 1)
	e, created, err := s.CreateEntry(context.Background(), 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Новое", City: "город", Street: "дом"})
	if err != nil || created || e != before[0] || !e.NeedsReview {
		t.Fatal(e, created, err)
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
}
func TestUpdateEntry(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: " Работа ", City: "Новый", Street: "Улица 2", Apartment: "5", Entrance: "3"}
	b, err := s.UpdateEntry(context.Background(), 1, a.ID, in)
	if err != nil || b.ID != a.ID || b.CreatedAt != a.CreatedAt || !b.IsDefault || b.Name != "Работа" || b.Address != "Улица 2, кв. 5, подъезд 3" || b.City != "Новый" || b.UpdatedAt.Before(a.UpdatedAt) {
		t.Fatal(b, err)
	}
	if got := bookEntries(t, db, 1); len(got) != 1 || got[0] != b {
		t.Fatal(got)
	}
	c, err := s.UpdateEntry(context.Background(), 1, b.ID, in)
	if err != nil || c.AddressID != b.AddressID {
		t.Fatal(c, err)
	}
}
func TestUpdateLegacyEntry(t *testing.T) {
	db, s := bookFixture(t)
	bookExec(t, db, `INSERT INTO addresses(id,city,address) VALUES(1,'Город','неструктурированный текст'); SELECT setval(pg_get_serial_sequence('addresses','id'),1); INSERT INTO user_addresses(user_id,address_id) VALUES(1,1)`)
	a := bookEntries(t, db, 1)[0]
	b, err := s.UpdateEntry(context.Background(), 1, a.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 3"})
	if err != nil || b.NeedsReview || b.IsDefault || b.ID != a.ID || b.CreatedAt != a.CreatedAt || b.Street != "Дом 3" {
		t.Fatal(b, err)
	}
	if got := bookEntries(t, db, 1); len(got) != 1 || got[0] != b {
		t.Fatal(got)
	}
}
func TestUpdateEntryConflict(t *testing.T) {
	db, s := bookFixture(t)
	createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	before := bookEntries(t, db, 1)
	_, err := s.UpdateEntry(context.Background(), 1, b.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Коллизия", City: " ГОРОД ", Street: "ДОМ"})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatal(err)
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 2 {
		t.Fatal(n)
	}
}
func TestBookOwnerIsolation(t *testing.T) {
	db, s := bookFixture(t)
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"}
	a := createBookEntry(t, s, 1, in)
	b := createBookEntry(t, s, 2, in)
	if a.AddressID != b.AddressID {
		t.Fatal(a, b)
	}
	bookExec(t, db, `INSERT INTO orders(customer_id,account_id,address_id,items_hash,delivery_city,delivery_address) VALUES(1,1,$1,'hash','Город','Дом')`, a.AddressID)
	if _, err := s.UpdateEntry(context.Background(), 1, a.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Новый", Street: "Дом 3"}); err != nil {
		t.Fatal(err)
	}
	if got := bookEntries(t, db, 2); len(got) != 1 || got[0] != b {
		t.Fatal(got)
	}
	var snapshot struct {
		City    string `db:"delivery_city"`
		Address string `db:"delivery_address"`
		ID      int64  `db:"address_id"`
	}
	if err := db.Get(&snapshot, `SELECT delivery_city,delivery_address,address_id FROM orders`); err != nil || snapshot.City != "Город" || snapshot.Address != "Дом" || snapshot.ID != a.AddressID {
		t.Fatal(snapshot, err)
	}
	var text string
	if err := db.Get(&text, `SELECT address FROM addresses WHERE id=$1`, a.AddressID); err != nil || text != "Дом" {
		t.Fatal(text, err)
	}
}
func TestBookWriteValidation(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	before := bookEntries(t, db, 1)
	inputs := []Fields{{}, {City: " ", Street: " \u00a0 "}, {Name: strings.Repeat("я", 41), City: strings.Repeat("я", 81), Street: strings.Repeat("я", 121), Apartment: strings.Repeat("я", 21), Entrance: strings.Repeat("я", 21)}}
	for _, in := range inputs {
		want := FieldErrors(CleanFields(in))
		_, _, err := s.CreateEntry(context.Background(), 1, in)
		var bad *InvalidFieldsError
		if !errors.As(err, &bad) || !reflect.DeepEqual(bad.Fields, want) || bad.Error() != "invalid address fields" {
			t.Fatal(err)
		}
		_, err = s.UpdateEntry(context.Background(), 1, a.ID, in)
		if !errors.As(err, &bad) || !reflect.DeepEqual(bad.Fields, want) {
			t.Fatal(err)
		}
	}
	bookExec(t, db, `INSERT INTO users(id,name,phone,role) VALUES(3,'Staff','3','admin'),(4,'Merged','4','user'); UPDATE users SET merged_into_user_id=2,updated_at=now() WHERE id=4`)
	valid := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"}
	for _, user := range []int64{99, 3, 4} {
		if _, _, err := s.CreateEntry(context.Background(), user, valid); !errors.Is(err, ErrOwnerUnavailable) {
			t.Fatal(err)
		}
		if _, err := s.UpdateEntry(context.Background(), user, a.ID, valid); !errors.Is(err, ErrOwnerUnavailable) {
			t.Fatal(err)
		}
	}
	for _, tc := range [][2]int64{{1, 999}, {2, a.ID}} {
		if _, err := s.UpdateEntry(context.Background(), tc[0], tc[1], valid); !errors.Is(err, store.ErrNotFound) {
			t.Fatal(err)
		}
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Fatal(n)
	}
}
func TestBookConcurrentCreates(t *testing.T) {
	db, s := bookFixture(t)
	const count = 12
	var wg sync.WaitGroup
	entries := make([]Entry, count)
	created := make([]bool, count)
	errs := make([]error, count)
	start := make(chan struct{})
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			entries[i], created[i], errs[i] = s.CreateEntry(context.Background(), 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
		}()
	}
	close(start)
	wg.Wait()
	n := 0
	for i := range entries {
		if errs[i] != nil || entries[i] != entries[0] {
			t.Fatal(entries[i], errs[i])
		}
		if created[i] {
			n++
		}
	}
	if n != 1 || !entries[0].IsDefault || len(bookEntries(t, db, 1)) != 1 || testutil.Count(t, db, `SELECT count(*) FROM addresses`) != 1 {
		t.Fatal(n)
	}
}
func TestBookWriteRollback(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	before := bookEntries(t, db, 1)
	bookExec(t, db, `ALTER TABLE user_addresses ADD CONSTRAINT test_book_write_failure CHECK(name <> 'fail')`)
	t.Cleanup(func() { bookExec(t, db, `ALTER TABLE user_addresses DROP CONSTRAINT test_book_write_failure`) })
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "fail", City: "Частный город", Street: "Частная улица"}
	if e, created, err := s.CreateEntry(context.Background(), 1, in); err == nil || created || e.ID != 0 || strings.Contains(err.Error(), in.Street) {
		t.Fatal(e, created, err)
	}
	if e, err := s.UpdateEntry(context.Background(), 1, a.ID, in); err == nil || e.ID != 0 || strings.Contains(err.Error(), in.Street) {
		t.Fatal(e, err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Fatal(n)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.CreateEntry(ctx, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.UpdateEntry(ctx, 1, a.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	broken, err := sqlx.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = broken.Close()
	svc := NewService(broken)
	if _, _, err := svc.CreateEntry(context.Background(), 1, in); err == nil {
		t.Fatal("closed DB accepted")
	}
	if _, err := svc.UpdateEntry(context.Background(), 1, a.ID, in); err == nil {
		t.Fatal("closed DB accepted")
	}
}

func TestBookCommitRollback(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	before := bookEntries(t, db, 1)
	bookExec(t, db, `CREATE FUNCTION test_book_commit_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'saved address commit failed'; END $$;
 CREATE CONSTRAINT TRIGGER test_book_commit_failure AFTER INSERT OR UPDATE ON user_addresses DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION test_book_commit_failure()`)
	t.Cleanup(func() {
		bookExec(t, db, `DROP TRIGGER test_book_commit_failure ON user_addresses; DROP FUNCTION test_book_commit_failure()`)
	})
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Новый дом"}
	if entry, created, err := s.CreateEntry(context.Background(), 1, in); err == nil || created || entry.ID != 0 {
		t.Fatal(entry, created, err)
	}
	if entry, err := s.UpdateEntry(context.Background(), 1, a.ID, in); err == nil || entry.ID != 0 {
		t.Fatal(entry, err)
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 1 {
		t.Fatalf("orphan addresses after commit failure: %d", n)
	}
}
