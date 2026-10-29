package customer_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestCustomerSessions(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	id := testutil.InsertUser(t, db, "Ivan", "+79990000001", "user")
	admin := testutil.InsertUser(t, db, "Admin", "+79990000002", "admin")
	s := customer.NewService(db, clock.Fixed(now))
	if _, _, err := s.CreateSession(ctx, admin, "password", "web"); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("admin session: %v", err)
	}
	if _, _, err := s.CreateSession(ctx, id, "telegram", "web"); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("surface: %v", err)
	}
	token, exp, err := s.CreateSession(ctx, id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, "cs_") || exp != now.Add(30*24*time.Hour) {
		t.Fatalf("token or expiry: %q %v", token, exp)
	}
	other, _, err := s.CreateSession(ctx, id, "google", "web")
	if err != nil || other == token {
		t.Fatalf("random token: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token[3:])
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	var saved []byte
	if err := db.Get(&saved, `SELECT token_hash FROM customer_sessions WHERE user_id=$1 AND method='password'`, id); err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(hash[:]) || string(saved) == token {
		t.Fatal("token hash was not persisted")
	}
	for _, bad := range []string{"", "cs_", "cs_" + strings.Repeat("A", 43), "Bearer " + token, "cs_" + strings.Repeat("!", 43)} {
		if _, err := s.Authenticate(ctx, bad); !errors.Is(err, customer.ErrInvalidSession) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if v, err := s.Authenticate(ctx, token); err != nil || v.UserID != id || v.Method != "password" || v.Surface != "web" {
		t.Fatalf("authenticate: %+v %v", v, err)
	}
	atBoundary := customer.NewService(db, clock.Fixed(exp))
	if _, err := atBoundary.Authenticate(ctx, token); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("expiry: %v", err)
	}
	if err := s.Revoke(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, token); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("revoked: %v", err)
	}
	if err := customer.NewStore(db).RevokeAllSessions(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, other); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("all revoked: %v", err)
	}
	fresh, _, err := s.CreateSession(ctx, id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, fresh); !errors.Is(err, customer.ErrInvalidSession) {
		t.Fatalf("deleted: %v", err)
	}
}
