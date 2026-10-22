package iiko

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func testTokens() *auth.Tokens {
	return auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)))
}

// blockingSource blocks Fetch until release is closed; entered is closed on the first call.
type blockingSource struct {
	Source
	entered chan struct{}
	release chan struct{}
}

func newBlockingSource(src Source) *blockingSource {
	return &blockingSource{Source: src, entered: make(chan struct{}), release: make(chan struct{})}
}

func (b *blockingSource) Fetch(ctx context.Context) (*Menu, error) {
	close(b.entered)
	<-b.release
	return b.Source.Fetch(ctx)
}

// countingSource counts Fetch calls so tests can prove a sync did not start.
type countingSource struct {
	Source
	calls int
}

func (c *countingSource) Fetch(ctx context.Context) (*Menu, error) {
	c.calls++
	return c.Source.Fetch(ctx)
}

// postUpdate sends POST /iiko/update-products; header "" sends no X-Update-Token.
func postUpdate(t *testing.T, token, header string, src Source) *httptest.ResponseRecorder {
	t.Helper()
	db := testutil.NewDB(t)
	r := chi.NewRouter()
	NewHandler(NewSyncer(db, src), token, testTokens()).Routes(r)

	req := httptest.NewRequest(http.MethodPost, "/iiko/update-products", nil)
	if header != "" {
		req.Header.Set("X-Update-Token", header)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestUpdateDisabled(t *testing.T) {
	src := &countingSource{Source: &fakeSource{n: oneItemMenu()}}

	// With no token configured the endpoint does not exist, whatever the header says.
	rec := postUpdate(t, "", "", src)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if src.calls != 0 {
		t.Errorf("source called %d times, want 0", src.calls)
	}
}

func TestUpdateWrongToken(t *testing.T) {
	src := &countingSource{Source: &fakeSource{n: oneItemMenu()}}

	rec := postUpdate(t, "secret", "other", src)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if src.calls != 0 {
		t.Errorf("source called %d times, want 0", src.calls)
	}
}

func TestUpdateNoHeader(t *testing.T) {
	src := &countingSource{Source: &fakeSource{n: oneItemMenu()}}

	rec := postUpdate(t, "secret", "", src)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if src.calls != 0 {
		t.Errorf("source called %d times, want 0", src.calls)
	}
}

func TestUpdateOK(t *testing.T) {
	src := &countingSource{Source: &fakeSource{n: oneItemMenu()}}

	rec := postUpdate(t, "secret", "secret", src)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var got Result
	testutil.Decode(t, rec, &got)
	if want := (Result{TotalCategories: 1, TotalProducts: 1}); got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}
	if src.calls != 1 {
		t.Errorf("source called %d times, want 1", src.calls)
	}
}

func TestUpdateSourceError(t *testing.T) {
	src := &countingSource{Source: &fakeSource{err: errors.New("iiko is down")}}

	rec := postUpdate(t, "secret", "secret", src)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

type adminEnv struct {
	router http.Handler
	db     *sqlx.DB
	admin  string
	user   string
}

// newAdminEnv wires the iiko routes with src and issues a token for an admin and a plain user.
func newAdminEnv(t *testing.T, src Source, updateToken string) adminEnv {
	t.Helper()
	db := testutil.NewDB(t)
	tokens := testTokens()
	adminID := testutil.InsertUser(t, db, "Admin", "+79000000001", auth.RoleAdmin)
	userID := testutil.InsertUser(t, db, "User", "+79000000002", auth.RoleUser)
	admin, err := tokens.Issue(adminID, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(userID, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	NewHandler(NewSyncer(db, src), updateToken, tokens).Routes(r)
	return adminEnv{router: r, db: db, admin: admin, user: user}
}

func TestAdminSync(t *testing.T) {
	env := newAdminEnv(t, &fakeSource{n: oneItemMenu()}, "")

	rec := testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", env.admin, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var got Result
	testutil.Decode(t, rec, &got)
	if want := (Result{TotalCategories: 1, TotalProducts: 1}); got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}
	if n := testutil.Count(t, env.db, `SELECT count(*) FROM categories WHERE iiko_id IS NOT NULL`); n != 1 {
		t.Errorf("categories in database = %d, want 1", n)
	}
}

func TestAdminSyncNoToken(t *testing.T) {
	env := newAdminEnv(t, &fakeSource{n: oneItemMenu()}, "")

	rec := testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", "", nil)

	assertError(t, rec, http.StatusUnauthorized, "Требуется авторизация")
}

func TestAdminSyncUserRole(t *testing.T) {
	src := &countingSource{Source: &fakeSource{n: oneItemMenu()}}
	env := newAdminEnv(t, src, "")

	rec := testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", env.user, nil)

	assertError(t, rec, http.StatusForbidden, "Недостаточно прав")
	if src.calls != 0 {
		t.Errorf("source called %d times, want 0", src.calls)
	}
}

func TestAdminSyncSourceError(t *testing.T) {
	env := newAdminEnv(t, &fakeSource{err: errors.New("iiko is down")}, "")

	rec := testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", env.admin, nil)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestAdminSyncConflict(t *testing.T) {
	src := newBlockingSource(&fakeSource{n: oneItemMenu()})
	env := newAdminEnv(t, src, "secret")

	first := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		first <- testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", env.admin, nil)
	}()
	<-src.entered

	rec := testutil.DoJSON(t, env.router, http.MethodPost, "/iiko/sync", env.admin, nil)
	assertError(t, rec, http.StatusConflict, "Обновление меню уже выполняется")

	req := httptest.NewRequest(http.MethodPost, "/iiko/update-products", nil)
	req.Header.Set("X-Update-Token", "secret")
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	assertError(t, rec, http.StatusConflict, "Обновление меню уже выполняется")

	close(src.release)
	if got := <-first; got.Code != http.StatusOK {
		t.Errorf("first sync status = %d, want 200; body %s", got.Code, got.Body)
	}
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, status, rec.Body)
	}
	var body struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	}
	testutil.Decode(t, rec, &body)
	if body.StatusCode != status || body.Message != message {
		t.Errorf("body = %+v, want statusCode %d and message %q", body, status, message)
	}
}
