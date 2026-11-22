package hotelbooking

import (
	"context"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/migrations"
	"github.com/pressly/goose/v3"
)

func TestBookingMigration(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewContentDB(t)
	provider, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Down(ctx); err != nil {
		t.Fatal(err)
	}
	var present bool
	if err = db.Get(&present, `SELECT to_regclass('hotel_booking_quotes') IS NOT NULL`); err != nil || present {
		t.Fatalf("rollback %v %v", present, err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	// Public content is replaceable. A saved quote must survive the room content
	// row being deleted/reinserted by the existing production restore command.
	store := NewStore(db)
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	q := QuoteResponse{randomID(), now.Add(10 * time.Minute), "single", "2026-12-01", "2026-12-03", 1, true, PublicPrice{"RUB", 2, 400000, 800000}}
	if err = store.saveQuote(ctx, q, "hash", now); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM rooms WHERE id='single'`); err != nil {
		t.Fatal(err)
	}
	saved, err := store.quote(ctx, q.QuoteID, "hash")
	if err != nil || saved.Nightly != 400000 {
		t.Fatalf("quote lost %+v %v", saved, err)
	}
	// A store built on a caller transaction participates in that transaction.
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = NewStore(tx).transaction(ctx, func(s *Store) error { return s.limit(ctx, "test", "hash", now, time.Minute, 1) }); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM hotel_booking_rate_limits`); err != nil || count != 0 {
		t.Fatalf("rollback count %d %v", count, err)
	}
}
