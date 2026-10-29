package customer_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/migrations"
)

func TestIdentityStore(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	a := testutil.InsertUser(t, db, "a", "+79990000001", "user")
	b := testutil.InsertUser(t, db, "b", "+79990000002", "user")
	s := customer.NewStore(db)
	one := customer.Identity{UserID: a, Provider: "email", Subject: "a@example.com"}
	if err := s.AddIdentity(ctx, one); err != nil {
		t.Fatal(err)
	}
	if got, err := s.IdentityOwner(ctx, "email", "a@example.com"); err != nil || got != a {
		t.Fatalf("owner = %d, %v", got, err)
	}
	for _, tc := range []struct{ provider, subject string }{{"google", "a@example.com"}, {"email", "missing"}} {
		if _, err := s.IdentityOwner(ctx, tc.provider, tc.subject); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("missing %v: %v", tc, err)
		}
	}
	for _, identity := range []customer.Identity{
		{UserID: b, Provider: "email", Subject: "a@example.com"},
		{UserID: a, Provider: "email", Subject: "other@example.com"},
	} {
		if err := s.AddIdentity(ctx, identity); !store.IsUniqueViolation(err) {
			t.Errorf("duplicate %v: %v", identity, err)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM customer_identities`); n != 1 {
		t.Errorf("rows = %d", n)
	}
	if got, err := s.IdentityOwner(ctx, "email", "a@example.com"); err != nil || got != a {
		t.Errorf("owner after conflicts = %d, %v", got, err)
	}
	if err := s.AddIdentity(ctx, customer.Identity{UserID: b, Provider: "telegram", Subject: "123"}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.IdentityOwner(ctx, "telegram", "123"); err != nil || got != b {
		t.Errorf("telegram owner = %d, %v", got, err)
	}
}

func TestIdentityMigration(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	provider := func() *goose.Provider {
		p, err := goose.NewProvider(goose.DialectPostgres, db.DB, migrations.FS)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := provider().DownTo(ctx, 20); err != nil {
		t.Fatal(err)
	}
	insert := func(name, email, role, password, legacy, phone string, confirmed bool) int64 {
		var id int64
		err := db.Get(&id, `INSERT INTO users (name, email, role, password_hash, legacy_password_hash, phone, is_email_confirmed) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, name, email, role, password, legacy, phone, confirmed)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	password := insert("password", "  Mixed@Example.com  ", "user", "hash", "", "+79990000001", false)
	legacy := insert("legacy", "legacy@example.com", "user", "", "hash", "+79990000002", true)
	insert("admin", "admin@example.com", "admin", "hash", "", "+79990000003", true)
	insert("contact", "contact@example.com", "user", "", "", "+79990000004", false)
	if _, err := provider().UpTo(ctx, 21); err != nil {
		t.Fatal(err)
	}
	s := customer.NewStore(db)
	for _, tc := range []struct {
		subject string
		id      int64
	}{{"mixed@example.com", password}, {"legacy@example.com", legacy}} {
		if got, err := s.IdentityOwner(ctx, "email", tc.subject); err != nil || got != tc.id {
			t.Errorf("%s = %d, %v", tc.subject, got, err)
		}
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM customer_identities`); n != 2 {
		t.Errorf("identity rows = %d", n)
	}
	var confirmed bool
	if err := db.Get(&confirmed, `SELECT is_email_confirmed FROM users WHERE id=$1`, legacy); err != nil || !confirmed {
		t.Errorf("legacy confirmed = %v, %v", confirmed, err)
	}
	if err := db.Get(&confirmed, `SELECT is_email_confirmed FROM users WHERE id=$1`, password); err != nil || confirmed {
		t.Errorf("password confirmed = %v, %v", confirmed, err)
	}
	if _, err := db.Exec(`UPDATE users SET phone=NULL WHERE id=$1`, password); err != nil {
		t.Fatal(err)
	}
	if _, err := provider().DownTo(ctx, 20); err != nil {
		t.Fatal(err)
	}
	var phone string
	if err := db.Get(&phone, `SELECT phone FROM users WHERE id=$1`, password); err != nil || phone != "unassigned:"+itoa(password) {
		t.Errorf("down phone = %q, %v", phone, err)
	}
	if _, err := db.Exec(`UPDATE users SET email=' MIXED@example.com ' WHERE id=$1`, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := provider().UpTo(ctx, 21); err == nil || !strings.Contains(err.Error(), "normalized customer email collision") {
		t.Fatalf("collision error = %v", err)
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM goose_db_version WHERE version_id=21 AND is_applied`); n != 0 {
		t.Errorf("migration applied after collision")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_name='customer_identities'`); n != 0 {
		t.Errorf("identity table survived rollback")
	}
	if _, err := db.Exec(`UPDATE users SET email='legacy@example.com' WHERE id=$1`, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := provider().UpTo(ctx, 21); err != nil {
		t.Fatal(err)
	}
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }
