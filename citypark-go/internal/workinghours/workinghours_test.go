package workinghours

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const (
	topMessage  = "Ошибка валидации, пожалуйста проверьте введенные вами данные"
	hhmmMessage = "Укажите время в формате ЧЧ:ММ"
)

type fixture struct {
	db     *sqlx.DB
	router http.Handler
	admin  string
	user   string
}

// newFixture returns a router over the migrated database. NewDB does not truncate working_hours (the migration
// seeds it), so the row found at the start is put back after the test.
func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	var row struct {
		Start string `db:"start_time"`
		End   string `db:"end_time"`
	}
	if err := db.Get(&row, `SELECT start_time, end_time FROM working_hours WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := db.Exec(`INSERT INTO working_hours (id, start_time, end_time) VALUES (1, $1, $2)
			ON CONFLICT (id) DO UPDATE SET start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time`,
			row.Start, row.End)
		if err != nil {
			t.Errorf("restore working_hours: %v", err)
		}
	})

	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(now))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	NewHandler(NewStore(db), tokens).Routes(r)
	return fixture{db: db, router: r, admin: admin, user: user}
}

type body struct {
	Start     string    `json:"start"`
	End       string    `json:"end"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (f fixture) get(t *testing.T) body {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/working-hours", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body %s", rec.Code, rec.Body)
	}
	var b body
	testutil.Decode(t, rec, &b)
	return b
}

func TestSeed(t *testing.T) {
	f := newFixture(t)
	b := f.get(t)
	if b.Start != "09:00" || b.End != "22:00" {
		t.Errorf("hours = %s-%s, want 09:00-22:00", b.Start, b.End)
	}
	if b.UpdatedAt.IsZero() {
		t.Error("updatedAt is zero")
	}
}

func TestPut(t *testing.T) {
	f := newFixture(t)
	before := f.get(t)
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.admin,
		map[string]any{"start": "10:00", "end": "23:00", "updatedAt": "2000-01-01T00:00:00Z"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %s", rec.Code, rec.Body)
	}
	var out body
	testutil.Decode(t, rec, &out)
	if out.Start != "10:00" || out.End != "23:00" {
		t.Errorf("PUT body = %+v", out)
	}
	after := f.get(t)
	if after.Start != "10:00" || after.End != "23:00" {
		t.Errorf("GET after PUT = %+v", after)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updatedAt %v not after %v", after.UpdatedAt, before.UpdatedAt)
	}
	if !out.UpdatedAt.Equal(after.UpdatedAt) {
		t.Errorf("PUT updatedAt %v != GET updatedAt %v", out.UpdatedAt, after.UpdatedAt)
	}
}

func TestPutTrims(t *testing.T) {
	f := newFixture(t)
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.admin,
		map[string]any{"start": " 10:00 ", "end": " 21:30\t"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body %s", rec.Code, rec.Body)
	}
	if b := f.get(t); b.Start != "10:00" || b.End != "21:30" {
		t.Errorf("stored = %+v", b)
	}
}

func TestPutInvalid(t *testing.T) {
	tests := []struct {
		name       string
		start, end string
		field, msg string
	}{
		{"start without zero", "9:00", "22:00", "start", hhmmMessage},
		{"end hour 25", "09:00", "25:00", "end", hhmmMessage},
		{"start empty", "", "22:00", "start", "поле не должно быть пустым"},
		{"equal", "12:00", "12:00", "end", endBeforeStart},
		{"reversed", "13:00", "12:00", "end", endBeforeStart},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.admin,
				map[string]any{"start": tt.start, "end": tt.end})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body %s", rec.Code, rec.Body)
			}
			var he struct {
				Message string            `json:"message"`
				Errors  map[string]string `json:"errors"`
			}
			testutil.Decode(t, rec, &he)
			if he.Message != topMessage {
				t.Errorf("message = %q", he.Message)
			}
			if got := he.Errors[tt.field]; got != tt.msg {
				t.Errorf("errors.%s = %q, want %q (all: %v)", tt.field, got, tt.msg, he.Errors)
			}
			if b := f.get(t); b.Start != "09:00" || b.End != "22:00" {
				t.Errorf("row changed to %+v", b)
			}
		})
	}
}

func TestPutBadJSON(t *testing.T) {
	f := newFixture(t)
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.admin, "not an object")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body %s", rec.Code, rec.Body)
	}
}

func TestPutAuth(t *testing.T) {
	f := newFixture(t)
	payload := map[string]any{"start": "10:00", "end": "23:00"}
	if rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", "", payload); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", rec.Code)
	}
	if rec := testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.user, payload); rec.Code != http.StatusForbidden {
		t.Errorf("user role: status = %d, want 403", rec.Code)
	}
	if b := f.get(t); b.Start != "09:00" {
		t.Errorf("row changed to %+v", b)
	}
}

func TestMissingRow(t *testing.T) {
	f := newFixture(t) // its cleanup puts the row back
	if _, err := f.db.Exec(`DELETE FROM working_hours`); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/working-hours", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET status = %d, want 404", rec.Code)
	}
	rec = testutil.DoJSON(t, f.router, http.MethodPut, "/working-hours", f.admin,
		map[string]any{"start": "10:00", "end": "23:00"})
	if rec.Code != http.StatusNotFound {
		t.Errorf("PUT status = %d, want 404", rec.Code)
	}
	if _, _, err := NewStore(f.db).Get(context.Background()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestDBErrors(t *testing.T) {
	f := newFixture(t)
	// A cancelled context makes every query fail with an error that is not ErrNotFound.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewStore(f.db)
	if _, _, err := s.Get(ctx); err == nil || errors.Is(err, store.ErrNotFound) {
		t.Errorf("Get error = %v, want a database error", err)
	}
	if _, err := s.Put(ctx, Hours{Start: "10:00", End: "11:00"}); err == nil || errors.Is(err, store.ErrNotFound) {
		t.Errorf("Put error = %v, want a database error", err)
	}
}

func TestHours(t *testing.T) {
	h := Hours{Start: "09:00", End: "22:00"}
	contains := []struct {
		hhmm string
		want bool
	}{{"08:59", false}, {"09:00", true}, {"18:30", true}, {"22:00", true}, {"22:01", false}}
	for _, tt := range contains {
		if got := h.Contains(tt.hhmm); got != tt.want {
			t.Errorf("Contains(%q) = %v, want %v", tt.hhmm, got, tt.want)
		}
	}

	openAt := []struct {
		hour, min int
		want      bool
	}{{8, 59, false}, {9, 0, true}, {21, 59, true}, {22, 0, false}}
	for _, tt := range openAt {
		at := time.Date(2026, 10, 28, tt.hour, tt.min, 0, 0, time.UTC)
		if got := h.OpenAt(at); got != tt.want {
			t.Errorf("OpenAt(%02d:%02d) = %v, want %v", tt.hour, tt.min, got, tt.want)
		}
	}
	if (Hours{Start: "abc", End: "22:00"}).OpenAt(time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)) {
		t.Error("OpenAt with an unparsable start = true, want false")
	}
	if (Hours{Start: "09:00", End: "abc"}).OpenAt(time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)) {
		t.Error("OpenAt with an unparsable end = true, want false")
	}

	display := []struct{ in, want string }{{"09:00", "9:00"}, {"00:30", "0:30"}, {"22:00", "22:00"}, {"", ""}}
	for _, tt := range display {
		if got := Display(tt.in); got != tt.want {
			t.Errorf("Display(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
