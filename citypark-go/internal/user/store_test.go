package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
)

func TestCreateAndGet(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	created, err := s.Create(ctx, user.CreateParams{
		Name: "Иван", Email: new("ivan@example.com"), Phone: "+79001112233",
		PasswordHash: "hash", Role: "user",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Name != "Иван" || created.PasswordHash != "hash" || created.Role != "user" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	byID, err := s.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	byEmail, err := s.ByEmail(ctx, "ivan@example.com")
	if err != nil {
		t.Fatalf("ByEmail: %v", err)
	}
	byPhone, err := s.ByPhone(ctx, "+79001112233")
	if err != nil {
		t.Fatalf("ByPhone: %v", err)
	}
	for name, got := range map[string]user.User{"ByID": byID, "ByEmail": byEmail, "ByPhone": byPhone} {
		if got.ID != created.ID {
			t.Errorf("%s returned id %d, want %d", name, got.ID, created.ID)
		}
	}

	want := user.Response{ID: created.ID, Name: "Иван", Email: "ivan@example.com", Phone: "+79001112233", Role: "user"}
	if got := byID.Response(); got != want {
		t.Errorf("Response() = %+v, want %+v", got, want)
	}
}

func TestCreateNoEmail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	first, err := s.Create(ctx, user.CreateParams{Name: "A", Phone: "+79000000001", Role: "user"})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if first.Email != nil {
		t.Errorf("Email = %q, want nil", *first.Email)
	}
	if got := first.Response().Email; got != "" {
		t.Errorf("Response().Email = %q, want empty", got)
	}

	if _, err := s.Create(ctx, user.CreateParams{Name: "B", Phone: "+79000000002", Role: "user"}); err != nil {
		t.Fatalf("Create second user without email: %v", err)
	}
}

func TestCreateDuplicatePhone(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	p := user.CreateParams{Name: "A", Phone: "+79000000001", Role: "user"}
	if _, err := s.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	p.Name = "B"
	_, err := s.Create(ctx, p)
	if !store.IsUniqueViolation(err) {
		t.Fatalf("err = %v, want unique violation", err)
	}
}

func TestCreateDuplicateEmail(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	_, err := s.Create(ctx, user.CreateParams{Name: "A", Email: new("a@x.io"), Phone: "+79000000001", Role: "user"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = s.Create(ctx, user.CreateParams{Name: "B", Email: new("a@x.io"), Phone: "+79000000002", Role: "user"})
	if !store.IsUniqueViolation(err) {
		t.Fatalf("err = %v, want unique violation", err)
	}
}

func TestNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	_, errByID := s.ByID(ctx, 999)
	_, errByEmail := s.ByEmail(ctx, "x@x.io")
	_, errByPhone := s.ByPhone(ctx, "+79999999999")
	_, errUpdate := s.Update(ctx, 999, user.UpdateParams{Name: "N", Phone: "+79000000000"})
	errSetHash := s.SetPasswordHash(ctx, 999, "h")

	tests := map[string]error{
		"ByID":            errByID,
		"ByEmail":         errByEmail,
		"ByPhone":         errByPhone,
		"Update":          errUpdate,
		"SetPasswordHash": errSetHash,
	}
	for name, err := range tests {
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("%s: err = %v, want store.ErrNotFound", name, err)
		}
	}
}

func TestUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	created, err := s.Create(ctx, user.CreateParams{
		Name: "Old", Email: new("old@x.io"), Phone: "+79000000001", Role: "user",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := s.Update(ctx, created.ID, user.UpdateParams{
		Name: "New", Email: new("new@x.io"), Phone: "+79000000002",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New" || updated.Email == nil || *updated.Email != "new@x.io" || updated.Phone != "+79000000002" {
		t.Errorf("unexpected updated user: %+v", updated)
	}
	if updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Errorf("UpdatedAt %v is before CreatedAt %v", updated.UpdatedAt, updated.CreatedAt)
	}

	cleared, err := s.Update(ctx, created.ID, user.UpdateParams{Name: "New", Email: nil, Phone: "+79000000002"})
	if err != nil {
		t.Fatalf("Update without email: %v", err)
	}
	if cleared.Email != nil {
		t.Errorf("Email = %q, want nil", *cleared.Email)
	}
	got, err := s.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Email != nil {
		t.Errorf("stored Email = %q, want nil", *got.Email)
	}
}

func TestSetPasswordHash(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	var id int64
	err := db.GetContext(ctx, &id,
		`INSERT INTO users (name, phone, legacy_password_hash) VALUES ('A', '+79000000001', 'old') RETURNING id`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := s.SetPasswordHash(ctx, id, "new-hash"); err != nil {
		t.Fatalf("SetPasswordHash: %v", err)
	}
	got, err := s.ByID(ctx, id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.PasswordHash != "new-hash" {
		t.Errorf("PasswordHash = %q, want %q", got.PasswordHash, "new-hash")
	}
	if got.LegacyPasswordHash != "" {
		t.Errorf("LegacyPasswordHash = %q, want empty", got.LegacyPasswordHash)
	}
}

func TestUpsertCustomer(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	s := user.NewStore(db)

	first, err := s.UpsertCustomer(ctx, "Первый", "+79000000001")
	if err != nil {
		t.Fatalf("UpsertCustomer first: %v", err)
	}
	if first.Role != "user" || first.Email != nil || first.PasswordHash != "" {
		t.Errorf("unexpected new customer: %+v", first)
	}

	second, err := s.UpsertCustomer(ctx, "Второй", "+79000000001")
	if err != nil {
		t.Fatalf("UpsertCustomer second: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("id = %d, want %d", second.ID, first.ID)
	}
	if second.Name != "Второй" {
		t.Errorf("Name = %q, want %q", second.Name, "Второй")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM users`); n != 1 {
		t.Errorf("users rows = %d, want 1", n)
	}
}

func TestStoreInTx(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	errRollback := errors.New("rollback")

	err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
		_, err := user.NewStore(tx).Create(ctx, user.CreateParams{Name: "A", Phone: "+79000000001", Role: "user"})
		if err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("InTx err = %v, want %v", err, errRollback)
	}

	_, err = user.NewStore(db).ByPhone(ctx, "+79000000001")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ByPhone after rollback: err = %v, want store.ErrNotFound", err)
	}
}
