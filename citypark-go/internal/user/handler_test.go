package user_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
)

type userEnv struct {
	db     *sqlx.DB
	tokens *auth.Tokens
	router http.Handler
}

func newUserEnv(t *testing.T) userEnv {
	t.Helper()
	db := testutil.NewDB(t)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	user.NewHandler(user.NewStore(db), tokens).Routes(r)
	return userEnv{db: db, tokens: tokens, router: r}
}

func (e userEnv) do(t *testing.T, method string, id int64, body any) *httptest.ResponseRecorder {
	t.Helper()
	token, err := e.tokens.Issue(id, auth.RoleUser)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return testutil.DoJSON(t, e.router, method, "/user", token, body)
}

func (e userEnv) byID(t *testing.T, id int64) user.User {
	t.Helper()
	u, err := user.NewStore(e.db).ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID(%d): %v", id, err)
	}
	return u
}

func TestMe(t *testing.T) {
	e := newUserEnv(t)
	id := testutil.InsertUser(t, e.db, "user", "+79990000001", auth.RoleUser)

	rec := e.do(t, http.MethodGet, id, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	want := fmt.Sprintf(`{"id":%d,"name":"user","email":"","phone":"+79990000001","role":"user"}`, id)
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestMeNoToken(t *testing.T) {
	e := newUserEnv(t)

	rec := testutil.DoJSON(t, e.router, http.MethodGet, "/user", "", nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	want := `{"statusCode":401,"message":"Требуется авторизация"}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestMeBadToken(t *testing.T) {
	e := newUserEnv(t)

	rec := testutil.DoJSON(t, e.router, http.MethodGet, "/user", "bad", nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestMeDeletedUser(t *testing.T) {
	e := newUserEnv(t)

	rec := e.do(t, http.MethodGet, 999, nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body %s", rec.Code, rec.Body)
	}
}

// Named TestUpdateProfile because store_test.go already has TestUpdate.
func TestUpdateProfile(t *testing.T) {
	e := newUserEnv(t)
	id := testutil.InsertUser(t, e.db, "user", "+79990000001", auth.RoleUser)

	rec := e.do(t, http.MethodPut, id, map[string]any{
		"name":  "Новое имя",
		"email": "New@Example.com",
		"phone": "8 999 000 00 02",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	want := fmt.Sprintf(`{"id":%d,"name":"Новое имя","email":"new@example.com","phone":"+79990000002","role":"user"}`, id)
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}

	u := e.byID(t, id)
	if u.Name != "Новое имя" || u.Phone != "+79990000002" || u.Email == nil || *u.Email != "new@example.com" {
		t.Errorf("stored user = %+v", u)
	}
}

func TestUpdateKeepsPhone(t *testing.T) {
	e := newUserEnv(t)
	id := testutil.InsertUser(t, e.db, "user", "+79990000001", auth.RoleUser)

	rec := e.do(t, http.MethodPut, id, map[string]any{"name": "Другое имя"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	u := e.byID(t, id)
	if u.Phone != "+79990000001" || u.Name != "Другое имя" {
		t.Errorf("stored user = %+v, want phone kept and name changed", u)
	}
}

func TestUpdateRemovesEmail(t *testing.T) {
	e := newUserEnv(t)
	id := testutil.InsertUser(t, e.db, "user", "+79990000001", auth.RoleUser)
	if _, err := e.db.Exec(`UPDATE users SET email = 'old@example.com' WHERE id = $1`, id); err != nil {
		t.Fatalf("set email: %v", err)
	}

	rec := e.do(t, http.MethodPut, id, map[string]any{"name": "user", "email": ""})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if u := e.byID(t, id); u.Email != nil {
		t.Errorf("email = %q, want NULL", *u.Email)
	}
}

func TestUpdateValidation(t *testing.T) {
	e := newUserEnv(t)
	id := testutil.InsertUser(t, e.db, "user", "+79990000001", auth.RoleUser)

	rec := e.do(t, http.MethodPut, id, map[string]any{"name": ""})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body)
	}
	if _, keys := errorKeys(t, rec); !keys["name"] {
		t.Errorf("errors = %v, want key name", keys)
	}
}

func TestUpdateConflict(t *testing.T) {
	e := newUserEnv(t)
	first := testutil.InsertUser(t, e.db, "first", "+79990000001", auth.RoleUser)
	testutil.InsertUser(t, e.db, "second", "+79990000002", auth.RoleUser)

	rec := e.do(t, http.MethodPut, first, map[string]any{"name": "first", "phone": "+79990000002"})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body %s", rec.Code, rec.Body)
	}
	if msg, _ := errorKeys(t, rec); msg != "email или телефон уже используется" {
		t.Errorf("message = %q", msg)
	}
}

func TestUpdateDeletedUser(t *testing.T) {
	e := newUserEnv(t)

	rec := e.do(t, http.MethodPut, 999, map[string]any{"name": "ghost"})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body %s", rec.Code, rec.Body)
	}
}

func adminParams() user.AdminParams {
	return user.AdminParams{Email: " Admin@Example.com ", Password: "admin-password", Phone: "8 999 111 22 33"}
}

func TestEnsureAdminCreates(t *testing.T) {
	db := testutil.NewDB(t)
	s := user.NewStore(db)
	ctx := context.Background()

	created, err := user.EnsureAdmin(ctx, s, adminParams())
	if err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true")
	}

	u, err := s.ByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("ByEmail: %v", err)
	}
	if u.Role != auth.RoleAdmin || u.Name != "admin" || u.Phone != "+79991112233" {
		t.Errorf("admin = %+v", u)
	}
	if !auth.CheckPassword(u.PasswordHash, "admin-password") {
		t.Error("stored hash does not accept the admin password")
	}
}

func TestEnsureAdminExists(t *testing.T) {
	db := testutil.NewDB(t)
	s := user.NewStore(db)
	ctx := context.Background()

	if _, err := user.EnsureAdmin(ctx, s, adminParams()); err != nil {
		t.Fatalf("first EnsureAdmin: %v", err)
	}
	created, err := user.EnsureAdmin(ctx, s, adminParams())
	if err != nil {
		t.Fatalf("second EnsureAdmin: %v", err)
	}
	if created {
		t.Error("created = true on the second call, want false")
	}
	if n := testutil.Count(t, db, `SELECT count(*) FROM users`); n != 1 {
		t.Errorf("users = %d, want 1", n)
	}
}

func TestEnsureAdminExistingPhoneUntouched(t *testing.T) {
	db := testutil.NewDB(t)
	s := user.NewStore(db)
	id := testutil.InsertUser(t, db, "customer", "+79991112233", auth.RoleUser)

	created, err := user.EnsureAdmin(context.Background(), s, adminParams())
	if err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	if created {
		t.Error("created = true, want false when the phone is taken")
	}
	u, err := s.ByID(context.Background(), id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if u.Role != auth.RoleUser || u.Name != "customer" {
		t.Errorf("existing user was changed: %+v", u)
	}
}

func TestEnsureAdminBadPhone(t *testing.T) {
	db := testutil.NewDB(t)
	p := adminParams()
	p.Phone = "123"

	created, err := user.EnsureAdmin(context.Background(), user.NewStore(db), p)

	if err == nil {
		t.Fatal("err = nil, want an error")
	}
	if created {
		t.Error("created = true, want false")
	}
}

func TestEnsureAdminStoreError(t *testing.T) {
	db := testutil.NewDB(t)
	s := user.NewStore(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	created, err := user.EnsureAdmin(ctx, s, adminParams())

	if err == nil || store.IsUniqueViolation(err) {
		t.Fatalf("err = %v, want a context error", err)
	}
	if created {
		t.Error("created = true, want false")
	}
}
