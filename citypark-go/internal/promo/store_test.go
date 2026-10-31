package promo

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func input() Input { return Input{Code: " test_10 ", Percentage: 10, Enabled: true, Target: "all"} }
func create(t *testing.T, db *sqlx.DB, in Input) Campaign {
	t.Helper()
	var c Campaign
	if err := store.InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		var err error
		c, err = NewStore(tx).Create(context.Background(), in)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCampaignRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewDB(t)
	s := NewStore(db)
	empty, err := s.List(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty: %v %v", empty, err)
	}
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	p := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat, PriceKopecks: 100})
	in := input()
	in.Target = "selected"
	in.ProductIDs = []int64{p}
	in.CategoryIDs = []int64{cat}
	min, cap := money.Kopecks(500), money.Kopecks(100)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	in.MinimumEligibleSubtotal = &min
	in.MaximumDiscount = &cap
	in.StartsAt = &start
	in.EndsAt = &end
	c := create(t, db, in)
	want, _ := normalize(in)
	if c.StartsAt != nil {
		if !c.StartsAt.Equal(*want.StartsAt) || !c.EndsAt.Equal(*want.EndsAt) {
			t.Fatal("dates changed")
		}
		want.StartsAt = c.StartsAt
		want.EndsAt = c.EndsAt
	}
	if !reflect.DeepEqual(c.Input, want) {
		t.Fatalf("got %#v want %#v", c.Input, want)
	}
	got, err := s.ByCode(ctx, " test_10 ")
	if err != nil || !reflect.DeepEqual(got, c) {
		t.Fatalf("read: %#v %v", got, err)
	}
	in = input()
	in.Enabled = false
	in.Percentage = 99
	err = store.InTx(ctx, db, func(tx *sqlx.Tx) error { var e error; c, e = NewStore(tx).Update(ctx, c.ID, in); return e })
	if err != nil {
		t.Fatal(err)
	}
	want, _ = normalize(in)
	if !reflect.DeepEqual(c.Input, want) {
		t.Fatalf("update: %#v", c)
	}
	other := input()
	other.Code = "second"
	c2 := create(t, db, other)
	list, err := s.List(ctx)
	if err != nil || len(list) != 2 || list[0].ID != c2.ID || list[1].Enabled {
		t.Fatalf("list: %v %v", list, err)
	}
	for _, read := range []func() (Campaign, error){func() (Campaign, error) { return s.ByID(ctx, -1) }, func() (Campaign, error) { return s.ByCode(ctx, "missing") }} {
		if _, e := read(); !errors.Is(e, store.ErrNotFound) {
			t.Fatalf("missing: %v", e)
		}
	}
	err = store.InTx(ctx, db, func(tx *sqlx.Tx) error { _, e := NewStore(tx).Update(ctx, -1, in); return e })
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
}
func TestCampaignValidation(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	zero := money.Kopecks(0)
	neg := money.Kopecks(-1)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []func(*Input){func(i *Input) { i.Code = "ab" }, func(i *Input) { i.Code = strings.Repeat("A", 33) }, func(i *Input) { i.Code = "ТЕСТ" }, func(i *Input) { i.Code = "a b" }, func(i *Input) { i.Percentage = 0 }, func(i *Input) { i.Percentage = 100 }, func(i *Input) { i.MinimumEligibleSubtotal = &zero }, func(i *Input) { i.MaximumDiscount = &neg }, func(i *Input) { i.StartsAt = &now; i.EndsAt = &now }, func(i *Input) { i.Target = "bad" }, func(i *Input) { i.Target = "selected" }, func(i *Input) { i.ProductIDs = []int64{1} }}
	for n, change := range cases {
		in := input()
		change(&in)
		for _, update := range []bool{false, true} {
			err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
				if update {
					_, e := NewStore(tx).Update(ctx, 1, in)
					return e
				}
				_, e := NewStore(tx).Create(ctx, in)
				return e
			})
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Error() == "" {
				t.Fatalf("case %d: %v", n, err)
			}
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM promo_codes`); n != 0 {
		t.Fatal(n)
	}
}
func TestCampaignUniqueCode(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := create(t, db, input())
	err := store.InTx(ctx, db, func(tx *sqlx.Tx) error { _, e := NewStore(tx).Create(ctx, input()); return e })
	if !store.IsUniqueViolation(err) {
		t.Fatal(err)
	}
	in := input()
	in.Code = "changed"
	err = store.InTx(ctx, db, func(tx *sqlx.Tx) error { _, e := NewStore(tx).Update(ctx, c.ID, in); return e })
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal(err)
	}
	got, _ := NewStore(db).ByID(ctx, c.ID)
	if got.Code != c.Code {
		t.Fatal(got)
	}
}
func TestCampaignTargets(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	cat := testutil.InsertCategory(t, db, testutil.Category{})
	p := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat})
	in := input()
	in.Target = "selected"
	in.ProductIDs = []int64{p, p}
	in.CategoryIDs = []int64{cat, cat}
	c := create(t, db, in)
	if len(c.ProductIDs) != 1 || len(c.CategoryIDs) != 1 {
		t.Fatal(c)
	}
	for _, category := range []bool{false, true} {
		bad := in
		if category {
			bad.CategoryIDs = []int64{999999}
		} else {
			bad.ProductIDs = []int64{999999}
		}
		for _, update := range []bool{false, true} {
			err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
				if update {
					bad.Code = in.Code
					_, e := NewStore(tx).Update(ctx, c.ID, bad)
					return e
				}
				bad.Code = "other"
				_, e := NewStore(tx).Create(ctx, bad)
				return e
			})
			if !store.IsFKViolation(err) {
				t.Fatal(err)
			}
		}
		got, e := NewStore(db).ByID(ctx, c.ID)
		if e != nil || !reflect.DeepEqual(got.Input, c.Input) {
			t.Fatalf("rollback %v %v", got, e)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM promo_codes`); n != 1 {
		t.Fatal(n)
	}
	if _, e := db.ExecContext(ctx, `DELETE FROM products WHERE id=$1`, p); e != nil {
		t.Fatal(e)
	}
	got, e := NewStore(db).ByID(ctx, c.ID)
	if e != nil || got.Target != "selected" || len(got.ProductIDs) != 0 {
		t.Fatalf("delete: %v %v", got, e)
	}
}
func TestCampaignLocking(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := create(t, db, input())
	reader, e := db.BeginTxx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer reader.Rollback()
	if _, e = NewStore(reader).ByCode(ctx, c.Code); e != nil {
		t.Fatal(e)
	}
	writer, e := db.BeginTxx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer writer.Rollback()
	// A PostgreSQL lock timeout proves the conflicting row lock without sleeps.
	if _, e = writer.ExecContext(ctx, `SET LOCAL lock_timeout='100ms'`); e != nil {
		t.Fatal(e)
	}
	in := input()
	in.Percentage = 20
	if _, e = NewStore(writer).Update(ctx, c.ID, in); e == nil {
		t.Fatal("writer passed shared pricing lock")
	}
	writer.Rollback()
	reader.Commit()
	if e = store.InTx(ctx, db, func(tx *sqlx.Tx) error { _, e := NewStore(tx).Update(ctx, c.ID, in); return e }); e != nil {
		t.Fatal(e)
	}
	got, e := NewStore(db).ByID(ctx, c.ID)
	if e != nil || got.Percentage != 20 {
		t.Fatalf("after unlock: %v %v", got, e)
	}
}
func TestDatabaseErrors(t *testing.T) {
	db := testutil.NewDB(t)
	db.Close()
	s := NewStore(db)
	ctx := context.Background()
	if _, e := s.List(ctx); e == nil {
		t.Fatal("list")
	}
	if _, e := s.ByID(ctx, 1); e == nil {
		t.Fatal("read")
	}
	if _, e := s.Create(ctx, input()); e == nil {
		t.Fatal("create")
	}
	if _, e := s.Update(ctx, 1, input()); e == nil {
		t.Fatal("update")
	}
}
