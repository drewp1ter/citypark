package iiko

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

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
	NewHandler(NewSyncer(db, src), token).Routes(r)

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
