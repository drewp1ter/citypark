package customer_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/jmoiron/sqlx"
)

type fakeMailer struct {
	err  error
	sent []string
}

func (m *fakeMailer) Send(_ context.Context, to, subject, body string) error {
	m.sent = append(m.sent, to+"|"+subject+"|"+body)
	return m.err
}

func TestEmailTokens(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	a := testutil.InsertUser(t, db, "a", "+79990000001", "user")
	b := testutil.InsertUser(t, db, "b", "+79990000002", "user")
	m := &fakeMailer{}
	s := customer.NewService(db, clock.Fixed(now))
	s.ConfigureEmail(m, "https://site.example", nil, nil)
	for _, tc := range []struct {
		purpose string
		life    time.Duration
	}{{"verify", 24 * time.Hour}, {"reset", time.Hour}, {"change-email", 24 * time.Hour}} {
		raw, err := s.IssueEmailToken(ctx, a, tc.purpose, "  A@Example.com  ")
		if err != nil {
			t.Fatal(err)
		}
		hash, err := customer.HashEmailToken(raw)
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			Hash    []byte    `db:"token_hash"`
			Email   string    `db:"email"`
			Expires time.Time `db:"expires_at"`
		}
		if err := db.Get(&saved, `SELECT token_hash,email,expires_at FROM customer_email_tokens WHERE token_hash=$1`, hash); err != nil {
			t.Fatal(err)
		}
		if saved.Email != "a@example.com" || !saved.Expires.Equal(now.Add(tc.life)) || string(saved.Hash) != string(hash) || strings.Contains(fmt.Sprint(saved.Hash), raw) {
			t.Fatalf("saved: %+v", saved)
		}
		if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash, "other", now); !errors.Is(err, customer.ErrInvalidEmailToken) {
			t.Fatalf("wrong purpose: %v", err)
		}
		if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash, tc.purpose, now.Add(tc.life)); !errors.Is(err, customer.ErrInvalidEmailToken) {
			t.Fatalf("expiry: %v", err)
		}
		if got, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash, tc.purpose, now.Add(tc.life-time.Nanosecond)); err != nil || got.UserID != a || got.Email != "a@example.com" {
			t.Fatalf("consume: %+v %v", got, err)
		}
		if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash, tc.purpose, now); !errors.Is(err, customer.ErrInvalidEmailToken) {
			t.Fatalf("reuse: %v", err)
		}
	}
	raw, err := s.IssueEmailToken(ctx, b, "verify", "b@example.com")
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.IssueEmailToken(ctx, a, "verify", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.IssueEmailToken(ctx, a, "verify", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	oldHash, _ := customer.HashEmailToken(first)
	if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, oldHash, "verify", now); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatalf("resend: %v", err)
	}
	secondHash, _ := customer.HashEmailToken(second)
	if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, secondHash, "verify", now); err != nil {
		t.Fatal(err)
	}
	otherHash, _ := customer.HashEmailToken(raw)
	if got, err := customer.NewStore(db).ConsumeEmailToken(ctx, otherHash, "verify", now); err != nil || got.UserID != b {
		t.Fatalf("other account: %+v %v", got, err)
	}
	var logs bytes.Buffer
	s.ConfigureEmail(m, "https://site.example", nil, slog.New(slog.NewTextHandler(&logs, nil)))
	m.err = errors.New("secret password")
	failed, err := s.IssueEmailToken(ctx, a, "reset", "a@example.com")
	if failed != "" || !errors.Is(err, customer.ErrEmailUnavailable) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("send failure: %q %v", failed, err)
	}
	var active int
	if err := db.Get(&active, `SELECT count(*) FROM customer_email_tokens WHERE user_id=$1 AND purpose='reset' AND consumed_at IS NULL`, a); err != nil || active != 0 {
		t.Fatalf("failed delivery left %d active tokens: %v", active, err)
	}
	if logText := logs.String(); !strings.Contains(logText, "purpose=reset") || strings.Contains(logText, "secret") || strings.Contains(logText, "a@example.com") || strings.Contains(logText, "token=") {
		t.Fatalf("unsafe delivery log: %s", logText)
	}
	for _, row := range m.sent {
		if strings.Contains(row, "A@Example") {
			t.Fatalf("not normalized: %s", row)
		}
	}
	if _, err := customer.HashEmailToken("bad"); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatal(err)
	}
	if _, err := s.IssueEmailToken(ctx, a, "bad", "a@example.com"); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatal(err)
	}
}

func TestEmailTokenTransaction(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	id := testutil.InsertUser(t, db, "a", "+79990000001", "user")
	raw := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
	hash := sha256.Sum256([]byte(strings.Repeat("x", 32)))
	_ = raw
	if err := customer.NewStore(db).InsertEmailToken(ctx, hash[:], customer.EmailToken{UserID: id, Purpose: "verify", Email: "a@example.com"}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
		v, e := customer.NewStore(tx).ConsumeEmailToken(ctx, hash[:], "verify", now)
		if e != nil || v.UserID != id {
			return fmt.Errorf("consume: %+v %v", v, e)
		}
		_, e = tx.ExecContext(ctx, `UPDATE users SET missing_column=1 WHERE id=$1`, id)
		return e
	})
	if err == nil {
		t.Fatal("expected rollback")
	}
	if err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
		_, e := customer.NewStore(tx).ConsumeEmailToken(ctx, hash[:], "verify", now)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash[:], "verify", now); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatal(err)
	}
}

func TestEmailTokenConcurrentConsume(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	id := testutil.InsertUser(t, db, "a", "+79990000001", "user")
	hash := sha256.Sum256([]byte("proof"))
	if err := customer.NewStore(db).InsertEmailToken(ctx, hash[:], customer.EmailToken{UserID: id, Purpose: "reset", Email: "a@example.com"}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := customer.NewStore(db).ConsumeEmailToken(ctx, hash[:], "reset", now)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, customer.ErrInvalidEmailToken) {
			invalid++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatalf("success=%d invalid=%d", success, invalid)
	}
}
