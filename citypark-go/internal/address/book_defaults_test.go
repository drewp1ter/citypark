package address

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func TestSetDefault(t *testing.T) {
	db, s := bookFixture(t)
	ctx := context.Background()
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	for _, id := range []int64{a.ID, a.ID, b.ID, b.ID} {
		e, err := s.SetDefault(ctx, 1, id)
		if err != nil || !e.IsDefault || e.ID != id {
			t.Fatal(e, err)
		}
		if n := testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=1 AND is_default`); n != 1 {
			t.Fatal(n)
		}
		got, err := NewStore(db).EntryByID(ctx, 1, id)
		if err != nil || got != e {
			t.Fatal(got, err)
		}
	}
	legacy, err := s.CreateLegacy(ctx, 1, "Город", "Старый")
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Get(&id, `SELECT id FROM user_addresses WHERE user_id=1 AND address_id=$1`, legacy.ID); err != nil {
		t.Fatal(err)
	}
	e, err := s.SetDefault(ctx, 1, id)
	if err != nil || !e.NeedsReview || !e.IsDefault {
		t.Fatal(e, err)
	}
}
func TestSetDefaultForeign(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	before := bookEntries(t, db, 1)
	for _, id := range []int64{b.ID, 999} {
		if _, err := s.SetDefault(context.Background(), 1, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatal(err)
		}
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) || got[0] != a {
		t.Fatal(got)
	}
}
func TestDeleteEntry(t *testing.T) {
	db, s := bookFixture(t)
	ctx := context.Background()
	in := Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"}
	a := createBookEntry(t, s, 1, in)
	b := createBookEntry(t, s, 2, in)
	c := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	bookExec(t, db, `INSERT INTO orders(customer_id,address_id,items_hash,delivery_city,delivery_address) VALUES(1,$1,'hash','Город','Дом')`, a.AddressID)
	for _, id := range []int64{c.ID, a.ID} {
		if err := s.DeleteEntry(ctx, 1, id); err != nil {
			t.Fatal(err)
		}
	}
	if len(bookEntries(t, db, 1)) != 0 || bookEntries(t, db, 2)[0] != b {
		t.Fatal("ownership changed")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM addresses`) != 2 || testutil.Count(t, db, `SELECT count(*) FROM orders WHERE address_id=$1 AND delivery_city='Город' AND delivery_address='Дом'`, a.AddressID) != 1 {
		t.Fatal("history changed")
	}
	for _, id := range []int64{a.ID, b.ID, 999} {
		if err := s.DeleteEntry(ctx, 1, id); !errors.Is(err, store.ErrNotFound) {
			t.Fatal(err)
		}
	}
}
func TestConcurrentDefaults(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	var wg sync.WaitGroup
	errs := make(chan error, 21)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := a.ID
			if i%2 == 0 {
				id = b.ID
			}
			_, err := s.SetDefault(context.Background(), 1, id)
			errs <- err
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); errs <- s.DeleteEntry(context.Background(), 1, b.ID) }()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			t.Fatal(err)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=1 AND is_default`); n > 1 {
		t.Fatal(n)
	}
	if got := bookEntries(t, db, 1); len(got) != 1 || got[0].ID != a.ID {
		t.Fatal(got)
	}
}
func legacyRouter(db *sqlx.DB) (http.Handler, *auth.Tokens) {
	tokens := auth.NewTokens("test", time.Hour, clock.Fixed(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	NewHandler(NewStore(db), tokens).WithService(NewService(db)).Routes(r)
	return r, tokens
}
func TestLegacyBookCreate(t *testing.T) {
	db, s := bookFixture(t)
	r, tokens := legacyRouter(db)
	token, err := tokens.Issue(1, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, r, http.MethodPost, "/addresses", token, map[string]string{"city": " Город ", "address": " Дом "})
	if rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body)
	}
	var body map[string]any
	testutil.Decode(t, rec, &body)
	if len(body) != 5 || body["city"] != "Город" || body["address"] != "Дом" || body["createdAt"] == nil || body["updatedAt"] == nil {
		t.Fatal(body)
	}
	a := bookEntries(t, db, 1)[0]
	if !a.IsDefault || !a.NeedsReview || int64(body["id"].(float64)) != a.AddressID {
		t.Fatal(a)
	}
	if _, err := s.UpdateEntry(context.Background(), 1, a.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Домой", City: "Город", Street: "Дом"}); err != nil {
		t.Fatal(err)
	}
	before := bookEntries(t, db, 1)
	rec = testutil.DoJSON(t, r, http.MethodPost, "/addresses", token, map[string]string{"city": "Город", "address": "Дом"})
	if rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body)
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if _, err := s.CreateLegacy(context.Background(), 1, "Город", "дом"); err != nil {
		t.Fatal(err)
	}
	if got := bookEntries(t, db, 1); len(got) != 2 || got[1].IsDefault || !got[1].NeedsReview {
		t.Fatal(got)
	}
	if _, err := s.CreateLegacy(context.Background(), 1, " ", " "); err == nil {
		t.Fatal("invalid accepted")
	}
}
func TestLegacyBookDelete(t *testing.T) {
	db, s := bookFixture(t)
	bookExec(t, db, `SELECT setval(pg_get_serial_sequence('user_addresses','id'),100,false)`)
	ctx := context.Background()
	createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Другой"})
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	c := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	if c.ID == c.AddressID {
		t.Fatal("fixture needs distinct IDs")
	}
	r, tokens := legacyRouter(db)
	token, _ := tokens.Issue(1, auth.RoleUser)
	// An association ID must not be interpreted as one by the legacy route.
	rec := testutil.DoJSON(t, r, http.MethodDelete, "/addresses/"+strconv.FormatInt(c.ID, 10), token, nil)
	if rec.Code != 204 || len(bookEntries(t, db, 1)) != 2 {
		t.Fatal(rec.Code, rec.Body)
	}
	for _, id := range []int64{a.AddressID, a.AddressID, 999} {
		rec := testutil.DoJSON(t, r, http.MethodDelete, "/addresses/"+strconv.FormatInt(id, 10), token, nil)
		if rec.Code != 204 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	if got := bookEntries(t, db, 1); len(got) != 1 || got[0].ID != c.ID || got[0].IsDefault {
		t.Fatal(got)
	}
	if got, err := NewStore(db).EntryByID(ctx, 2, b.ID); err != nil || got != b {
		t.Fatal(got, err)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM addresses WHERE id=$1`, a.AddressID) != 1 {
		t.Fatal("shared address deleted")
	}
}
func TestLegacyBookPermissions(t *testing.T) {
	db, _ := bookFixture(t)
	bookExec(t, db, `INSERT INTO users(id,name,phone,role) VALUES(3,'Staff','3','admin')`)
	r, tokens := legacyRouter(db)
	for _, tc := range []struct {
		id   int64
		role string
	}{{1, auth.RoleUser}, {3, auth.RoleAdmin}} {
		token, _ := tokens.Issue(tc.id, tc.role)
		rec := testutil.DoJSON(t, r, http.MethodPost, "/addresses", token, map[string]string{"city": "Город", "address": "Дом"})
		if rec.Code != 201 {
			t.Fatal(rec.Code, rec.Body)
		}
		if got := bookEntries(t, db, tc.id); len(got) != 1 || !got[0].IsDefault {
			t.Fatal(got)
		}
		var a Address
		testutil.Decode(t, rec, &a)
		rec = testutil.DoJSON(t, r, http.MethodDelete, "/addresses/"+strconv.FormatInt(a.ID, 10), token, nil)
		if rec.Code != 204 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	for _, token := range []string{"", "invalid"} {
		for _, method := range []string{http.MethodPost, http.MethodDelete} {
			path := "/addresses"
			if method == http.MethodDelete {
				path += "/1"
			}
			if rec := testutil.DoJSON(t, r, method, path, token, map[string]string{"city": "Город", "address": "Дом"}); rec.Code != 401 {
				t.Fatal(rec.Code)
			}
		}
	}
}
func TestBookDefaultFailures(t *testing.T) {
	db, s := bookFixture(t)
	ctx := context.Background()
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	before := bookEntries(t, db, 1)
	operations := func(s *Service, ctx context.Context, owner int64) []error {
		_, set := s.SetDefault(ctx, owner, b.ID)
		del := s.DeleteEntry(ctx, owner, b.ID)
		_, create := s.CreateLegacy(ctx, owner, "Город", "Частный адрес")
		legacyDel := s.DeleteLegacy(ctx, owner, a.AddressID)
		return []error{set, del, create, legacyDel}
	}
	bookExec(t, db, `INSERT INTO users(id,name,phone,merged_into_user_id) VALUES(3,'Merged','3',2)`)
	for _, owner := range []int64{3, 999} {
		for _, err := range operations(s, ctx, owner) {
			if !errors.Is(err, ErrOwnerUnavailable) {
				t.Fatal(err)
			}
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	for _, err := range operations(s, canceled, 1) {
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	// Fail after clearing the old default: the transaction must restore it.
	bookExec(t, db, `ALTER TABLE user_addresses ADD CONSTRAINT test_default_failure CHECK(NOT is_default OR id<>`+strconv.FormatInt(b.ID, 10)+`)`)
	if _, err := s.SetDefault(ctx, 1, b.ID); err == nil {
		t.Fatal("failure expected")
	}
	bookExec(t, db, `ALTER TABLE user_addresses DROP CONSTRAINT test_default_failure`)
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	broken, err := sqlx.Open("pgx", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = broken.Close()
	for _, err := range operations(NewService(broken), ctx, 1) {
		if err == nil || strings.Contains(err.Error(), "Частный адрес") {
			t.Fatal(err)
		}
	}
}

func TestBookDefaultMutationRollback(t *testing.T) {
	db, s := bookFixture(t)
	ctx := context.Background()
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом"})
	b := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "Город", Street: "Дом 2"})
	before := bookEntries(t, db, 1)
	bookExec(t, db, `CREATE FUNCTION test_book_defaults_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'address mutation failed'; END $$;
 CREATE TRIGGER test_book_defaults_failure BEFORE INSERT OR UPDATE OR DELETE ON user_addresses FOR EACH ROW EXECUTE FUNCTION test_book_defaults_failure()`)
	t.Cleanup(func() {
		bookExec(t, db, `DROP TRIGGER test_book_defaults_failure ON user_addresses; DROP FUNCTION test_book_defaults_failure()`)
	})
	if _, err := s.SetDefault(ctx, 1, b.ID); err == nil {
		t.Fatal("set accepted")
	}
	if err := s.DeleteEntry(ctx, 1, a.ID); err == nil {
		t.Fatal("delete accepted")
	}
	if _, err := s.CreateLegacy(ctx, 1, "Город", "Частный адрес"); err == nil || strings.Contains(err.Error(), "Частный адрес") {
		t.Fatal(err)
	}
	if err := s.DeleteLegacy(ctx, 1, a.AddressID); err == nil {
		t.Fatal("legacy delete accepted")
	}
	if got := bookEntries(t, db, 1); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM addresses`); n != 2 {
		t.Fatal("orphan address", n)
	}
}
