package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
)

const defaultTestDSN = "postgres://postgres:postgres@127.0.0.1:5433/test?sslmode=disable"

// openTestDB returns a single-connection pool, so TEMP tables stay visible to
// every statement of the test.
func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}
	db, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open test database (run `make db-test-up` or set TEST_DB_DSN): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	return db
}

func createItems(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TEMP TABLE items (id int PRIMARY KEY, n int CHECK (n > 0))`); err != nil {
		t.Fatalf("create temp table: %v", err)
	}
}

func countItems(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT count(*) FROM items`); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestOpenBadDSN(t *testing.T) {
	db, err := Open(context.Background(), "postgres://postgres:postgres@127.0.0.1:1/test?sslmode=disable")
	if err == nil {
		_ = db.Close()
		t.Fatal("expected an error for an unreachable database")
	}
}

func TestOpenMalformedDSN(t *testing.T) {
	db, err := Open(context.Background(), "postgres://%zz")
	if err == nil {
		_ = db.Close()
		t.Fatal("expected an error for a malformed DSN")
	}
}

func TestInTxCommit(t *testing.T) {
	db := openTestDB(t)
	createItems(t, db)

	err := InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO items (id, n) VALUES (1, 5)`)
		return err
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if got := countItems(t, db); got != 1 {
		t.Fatalf("rows after commit = %d, want 1", got)
	}
}

func TestInTxRollback(t *testing.T) {
	db := openTestDB(t)
	createItems(t, db)
	want := errors.New("boom")

	err := InTx(context.Background(), db, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`INSERT INTO items (id, n) VALUES (1, 5)`); err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if got := countItems(t, db); got != 0 {
		t.Fatalf("rows after rollback = %d, want 0", got)
	}
}

func TestInTxPanic(t *testing.T) {
	db := openTestDB(t)
	createItems(t, db)

	func() {
		defer func() {
			if r := recover(); r != "kaboom" {
				t.Fatalf("recovered %v, want kaboom", r)
			}
		}()
		_ = InTx(context.Background(), db, func(tx *sqlx.Tx) error {
			if _, err := tx.Exec(`INSERT INTO items (id, n) VALUES (1, 5)`); err != nil {
				t.Fatalf("insert: %v", err)
			}
			panic("kaboom")
		})
		t.Fatal("InTx returned, want panic")
	}()

	if got := countItems(t, db); got != 0 {
		t.Fatalf("rows after panic = %d, want 0", got)
	}
}

func TestInTxBeginError(t *testing.T) {
	db := openTestDB(t)
	called := false
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := InTx(ctx, db, func(tx *sqlx.Tx) error {
		called = true
		return nil
	})
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if called {
		t.Fatal("fn must not run when the transaction cannot begin")
	}
}

func TestNotFound(t *testing.T) {
	db := openTestDB(t)
	createItems(t, db)

	var n int
	err := NotFound(db.GetContext(context.Background(), &n, `SELECT n FROM items WHERE id = 42`))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	other := errors.New("other")
	if got := NotFound(other); got != other {
		t.Fatalf("NotFound(other) = %v, want the same error", got)
	}
	if got := NotFound(nil); got != nil {
		t.Fatalf("NotFound(nil) = %v, want nil", got)
	}
}

func TestViolations(t *testing.T) {
	db := openTestDB(t)
	createItems(t, db)
	if _, err := db.Exec(`CREATE TEMP TABLE refs (item_id int REFERENCES items (id))`); err != nil {
		t.Fatalf("create refs: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO items (id, n) VALUES (1, 5)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, uniqueErr := db.Exec(`INSERT INTO items (id, n) VALUES (1, 6)`)
	_, checkErr := db.Exec(`INSERT INTO items (id, n) VALUES (2, 0)`)
	_, fkErr := db.Exec(`INSERT INTO refs (item_id) VALUES (99)`)

	checks := []struct {
		name string
		fn   func(error) bool
		hit  error
		miss []error
	}{
		{"unique", IsUniqueViolation, uniqueErr, []error{checkErr, fkErr}},
		{"check", IsCheckViolation, checkErr, []error{uniqueErr, fkErr}},
		{"fk", IsFKViolation, fkErr, []error{uniqueErr, checkErr}},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !c.fn(c.hit) {
				t.Errorf("want true for %v", c.hit)
			}
			for _, e := range c.miss {
				if c.fn(e) {
					t.Errorf("want false for %v", e)
				}
			}
			if c.fn(nil) {
				t.Error("want false for nil")
			}
			if c.fn(errors.New("x")) {
				t.Error("want false for a plain error")
			}
		})
	}
}
