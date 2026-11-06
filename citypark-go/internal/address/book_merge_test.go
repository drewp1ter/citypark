package address

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/jmoiron/sqlx"
)

func mergeBooks(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		var owners []int64
		if err := tx.Select(&owners, `SELECT id FROM users WHERE id=ANY($1) ORDER BY id FOR UPDATE`, []int64{1, 2}); err != nil {
			return err
		}
		return NewStore(tx).MergeEntries(context.Background(), 1, 2)
	}); err != nil {
		t.Fatal(err)
	}
	if len(bookEntries(t, db, 1)) != 0 {
		t.Fatal("stranded source associations")
	}
}

func TestMergeEntriesUnique(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Home", City: "City", Street: "Street 1", Apartment: "2", Entrance: "3"})
	b := createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Work", City: "City", Street: "Street 2"})
	mergeBooks(t, db)
	got := bookEntries(t, db, 2)
	a.IsDefault = false
	got[0].UpdatedAt = a.UpdatedAt
	if !reflect.DeepEqual(got, []Entry{a, b}) {
		t.Fatalf("metadata: %+v", got)
	}
}

func TestMergeEntriesDuplicate(t *testing.T) {
	db, s := bookFixture(t)
	createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Source", City: " CITY ", Street: "STREET  1", Apartment: "2"})
	b := createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Destination", City: "city", Street: "street 1", Apartment: "2"})
	c := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "city", Street: "street 1", Apartment: "3"})
	mergeBooks(t, db)
	got := bookEntries(t, db, 2)
	c.UpdatedAt = got[1].UpdatedAt
	if !reflect.DeepEqual(got, []Entry{b, c}) {
		t.Fatalf("duplicate resolution: %+v", got)
	}
}

func TestMergeEntriesDefaults(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		dest, source, duplicate bool
	}{
		{"destination wins", true, true, false}, {"source adopted", false, true, false},
		{"equivalent adopted", false, true, true}, {"neither", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, s := bookFixture(t)
			a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: "Street 1"})
			street := "Street 2"
			if tc.duplicate {
				street = "Street 1"
			}
			b := createBookEntry(t, s, 2, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: street})
			bookExec(t, db, `UPDATE user_addresses SET is_default=false`)
			if tc.dest {
				bookExec(t, db, `UPDATE user_addresses SET is_default=true WHERE id=$1`, b.ID)
			}
			if tc.source {
				bookExec(t, db, `UPDATE user_addresses SET is_default=true WHERE id=$1`, a.ID)
			}
			mergeBooks(t, db)
			want := int64(0)
			if tc.dest || (tc.source && tc.duplicate) {
				want = b.ID
			} else if tc.source {
				want = a.ID
			}
			count := 0
			for _, e := range bookEntries(t, db, 2) {
				if e.IsDefault {
					count++
					if e.ID != want {
						t.Fatal(e)
					}
				}
			}
			if (want == 0 && count != 0) || (want != 0 && count != 1) {
				t.Fatalf("defaults: %d", count)
			}
		})
	}
}

func legacyMergeEntry(t *testing.T, db *sqlx.DB, owner int64, city, raw, name string, def bool) int64 {
	t.Helper()
	a, err := NewStore(db).FindOrCreate(context.Background(), city, raw)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.Get(&id, `INSERT INTO user_addresses(user_id,address_id,name,is_default) VALUES($1,$2,$3,$4) RETURNING id`, owner, a.ID, name, def); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMergeEntriesLegacy(t *testing.T) {
	db, _ := bookFixture(t)
	a := legacyMergeEntry(t, db, 1, "City", "raw / unparsed", "Legacy", true)
	b := legacyMergeEntry(t, db, 2, "CITY", "OTHER", "First", false)
	c := legacyMergeEntry(t, db, 2, "city", " other ", "Second", false)
	before := bookEntries(t, db, 1)[0]
	mergeBooks(t, db)
	got := bookEntries(t, db, 2)
	if len(got) != 3 || got[0].ID != a || got[1].ID != b || got[2].ID != c {
		t.Fatal(got)
	}
	before.UpdatedAt = got[0].UpdatedAt
	if !reflect.DeepEqual(before, got[0]) {
		t.Fatalf("legacy changed: %+v", got[0])
	}
}

func TestMergeEntriesDestinationDuplicates(t *testing.T) {
	db, _ := bookFixture(t)
	legacyMergeEntry(t, db, 1, "city", " RAW ", "Source", true)
	first := legacyMergeEntry(t, db, 2, "City", "raw", "First", false)
	second := legacyMergeEntry(t, db, 2, "CITY", " RAW ", "Second", false)
	mergeBooks(t, db)
	got := bookEntries(t, db, 2)
	if len(got) != 2 || got[0].ID != first || !got[0].IsDefault || got[0].Name != "First" || got[1].ID != second || got[1].IsDefault {
		t.Fatal(got)
	}
}

func TestMergeEntriesSourceDuplicates(t *testing.T) {
	db, _ := bookFixture(t)
	a := legacyMergeEntry(t, db, 1, "City", "raw", "First", false)
	legacyMergeEntry(t, db, 1, "CITY", " RAW ", "Second", true)
	mergeBooks(t, db)
	got := bookEntries(t, db, 2)
	if len(got) != 1 || got[0].ID != a || got[0].Name != "First" || !got[0].IsDefault {
		t.Fatal(got)
	}
}

func TestMergeEntriesRollback(t *testing.T) {
	db, s := bookFixture(t)
	createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: "Street 1"})
	bad := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: "Street 2"})
	before := bookEntries(t, db, 1)
	bookExec(t, db, `CREATE FUNCTION fail_address_merge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.id=`+strconv.FormatInt(bad.ID, 10)+` AND NEW.user_id=2 THEN RAISE EXCEPTION 'forced move failure'; END IF; RETURN NEW; END $$`)
	bookExec(t, db, `CREATE TRIGGER fail_address_merge BEFORE UPDATE ON user_addresses FOR EACH ROW EXECUTE FUNCTION fail_address_merge()`)
	defer bookExec(t, db, `DROP FUNCTION fail_address_merge() CASCADE`)
	err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error { return NewStore(tx).MergeEntries(context.Background(), 1, 2) })
	if err == nil || !reflect.DeepEqual(before, bookEntries(t, db, 1)) || len(bookEntries(t, db, 2)) != 0 {
		t.Fatal("rollback failed", err)
	}
}

func TestMergeEntriesConcurrentWrite(t *testing.T) {
	db, s := bookFixture(t)
	a := createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: "Street 1"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var owners []int64
	if err := tx.SelectContext(ctx, &owners, `SELECT id FROM users WHERE id=ANY($1) ORDER BY id FOR UPDATE`, []int64{1, 2}); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := s.UpdateEntry(ctx, 1, a.ID, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", Name: "Changed", City: "City", Street: "Street 1"})
		result <- err
	}()
	// The mutation cannot pass the owner lock held by this transaction.
	select {
	case err := <-result:
		t.Fatalf("write bypassed lock: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET merged_into_user_id=2 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err := NewStore(tx).MergeEntries(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, ErrOwnerUnavailable) {
		t.Fatal(err)
	}
	if len(bookEntries(t, db, 1)) != 0 || bookEntries(t, db, 2)[0].ID != a.ID {
		t.Fatal("lost entry")
	}
}

func TestMergeEntriesInvalid(t *testing.T) {
	db, _ := bookFixture(t)
	for _, ids := range [][2]int64{{1, 1}, {0, 2}, {1, -1}, {1, 99999}} {
		if err := NewStore(db).MergeEntries(context.Background(), ids[0], ids[1]); err == nil {
			t.Fatal(ids)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewStore(db).MergeEntries(ctx, 1, 2); err == nil {
		t.Fatal("canceled merge succeeded")
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	if err := NewStore(tx).MergeEntries(context.Background(), 1, 2); err == nil {
		t.Fatal("closed transaction succeeded")
	}
}

// Fault injection verifies each database failure aborts the transaction, including
// a trigger that silently suppresses a mutation instead of reporting an SQL error.
type mergeFaultDB struct {
	store.DBTX
	selectCall, execCall, failSelect, failExec int
	missing                                    bool
}

func (f *mergeFaultDB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	f.selectCall++
	if f.selectCall == f.failSelect {
		return errors.New("forced read failure")
	}
	return f.DBTX.SelectContext(ctx, dest, query, args...)
}
func (f *mergeFaultDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	f.execCall++
	if f.execCall == f.failExec {
		if f.missing {
			return mergeResult{}, nil
		}
		return nil, errors.New("forced mutation failure")
	}
	return f.DBTX.ExecContext(ctx, query, args...)
}

type mergeResult struct{ fail bool }

func (r mergeResult) LastInsertId() (int64, error) { return 0, errors.New("unsupported") }
func (r mergeResult) RowsAffected() (int64, error) {
	if r.fail {
		return 0, errors.New("forced result failure")
	}
	return 0, nil
}

func TestMergeEntriesDatabaseFailures(t *testing.T) {
	db, s := bookFixture(t)
	createBookEntry(t, s, 1, Fields{RecipientName: "Иван", RecipientPhone: "+79991234567", City: "City", Street: "Street 1"})
	legacyMergeEntry(t, db, 1, "CITY", " STREET 1 ", "Duplicate", false)
	before := bookEntries(t, db, 1)
	for _, tc := range []struct {
		read, write int
		missing     bool
	}{
		{1, 0, false}, {2, 0, false}, {3, 0, false},
		{0, 1, false}, {0, 2, false}, {0, 3, false}, {0, 4, false},
		{0, 1, true}, {0, 2, true}, {0, 3, true}, {0, 4, true},
	} {
		err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error {
			return NewStore(&mergeFaultDB{DBTX: tx, failSelect: tc.read, failExec: tc.write, missing: tc.missing}).MergeEntries(context.Background(), 1, 2)
		})
		if err == nil || !reflect.DeepEqual(before, bookEntries(t, db, 1)) || len(bookEntries(t, db, 2)) != 0 {
			t.Fatalf("partial merge: %+v %v", tc, err)
		}
	}
}
