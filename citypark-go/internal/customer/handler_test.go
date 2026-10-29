package customer_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/go-chi/chi/v5"
)

func TestCustomerAuthenticationRoutes(t *testing.T) {
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC))
	tokens := auth.NewTokens("secret", time.Hour, c)
	sessions := customer.NewService(db, c)
	id := testutil.InsertUser(t, db, "Иван", "+79990000001", "user")
	admin := testutil.InsertUser(t, db, "Admin", "+79990000002", "admin")
	token, _, err := sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := tokens.Issue(id, "user")
	if err != nil {
		t.Fatal(err)
	}
	staff, err := tokens.Issue(admin, "admin")
	if err != nil {
		t.Fatal(err)
	}
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: true}
	r := chi.NewRouter()
	customer.NewHandler(user.NewStore(db), sessions, adapter).Routes(r)
	user.NewHandler(user.NewStore(db), tokens, adapter.Required(true)).Routes(r)
	address.NewHandler(address.NewStore(db), tokens, adapter.Required(true)).Routes(r)
	for _, tc := range []struct {
		path, header string
		want         int
	}{
		{"/customer/me", "Bearer " + token, 200}, {"/customer/me", "", 401}, {"/customer/me", "Bearer " + legacy, 401}, {"/customer/me", "Bearer " + staff, 401},
		{"/user", "Bearer " + token, 200}, {"/user", "Bearer " + legacy, 401}, {"/user", "Bearer " + staff, 200},
		{"/addresses", "Bearer " + token, 200}, {"/addresses", "Bearer " + legacy, 401}, {"/addresses", "Bearer " + staff, 200},
		{"/customer/me", "Basic abc", 401}, {"/customer/me", "Bearer ", 401},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.header != "" {
			req.Header.Add("Authorization", tc.header)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s %q = %d, want %d: %s", tc.path, tc.header[:min(len(tc.header), 8)], rec.Code, tc.want, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("%s cache header = %q", tc.path, rec.Header().Get("Cache-Control"))
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/customer/me", nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("repeated header = %d", rec.Code)
	}
	adapter.Enabled = false
	rec = testutil.DoJSON(t, r, http.MethodGet, "/customer/me", token, nil)
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "Вход временно недоступен") {
		t.Errorf("disabled = %d %s", rec.Code, rec.Body.String())
	}
	rec = testutil.DoJSON(t, r, http.MethodGet, "/user", legacy, nil)
	if rec.Code != 200 {
		t.Errorf("legacy disabled = %d", rec.Code)
	}
}

func TestCustomerProfile(t *testing.T) {
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC))
	tokens := auth.NewTokens("secret", time.Hour, c)
	sessions := customer.NewService(db, c)
	id := testutil.InsertUser(t, db, "Иван", "+79990000001", "user")
	testutil.InsertUser(t, db, "Other", "+79990000002", "user")
	token, _, err := sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: true}
	r := chi.NewRouter()
	customer.NewHandler(user.NewStore(db), sessions, adapter).Routes(r)
	get := testutil.DoJSON(t, r, http.MethodGet, "/customer/me", token, nil)
	want := fmt.Sprintf(`{"id":%d,"name":"Иван","email":"","phone":"+79990000001","role":"user"}`, id)
	if strings.TrimSpace(get.Body.String()) != want {
		t.Errorf("profile = %s", get.Body.String())
	}
	for _, tc := range []struct {
		body any
		want int
	}{
		{map[string]any{"name": "Новое имя", "phone": "8 999 000 00 03"}, 200},
		{map[string]any{"name": "Без телефона"}, 200},
		{map[string]any{"name": "Без телефона", "phone": ""}, 200},
		{map[string]any{"name": "Новое имя", "phone": "+79990000002"}, 409},
		{map[string]any{"name": "Новое имя", "email": "x@y.com"}, 400},
		{map[string]any{"name": "Новое имя", "role": "admin"}, 400},
		{map[string]any{"name": "", "phone": "bad"}, 400},
	} {
		rec := testutil.DoJSON(t, r, http.MethodPut, "/customer/me", token, tc.body)
		if rec.Code != tc.want {
			t.Errorf("update %v = %d %s", tc.body, rec.Code, rec.Body.String())
		}
		if tc.want == 200 && tc.body.(map[string]any)["phone"] == "8 999 000 00 03" {
			if !strings.Contains(rec.Body.String(), `"phone":"+79990000003"`) {
				t.Errorf("normalized profile = %s", rec.Body.String())
			}
		}
		if rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Error("cache header")
		}
	}
	cleared := testutil.DoJSON(t, r, http.MethodGet, "/customer/me", token, nil)
	if !strings.Contains(cleared.Body.String(), `"phone":""`) {
		t.Errorf("phone not cleared: %s", cleared.Body.String())
	}
	rec := testutil.DoJSON(t, r, http.MethodPost, "/auth/customer/logout", token, nil)
	if rec.Code != 204 {
		t.Errorf("logout = %d %s", rec.Code, rec.Body.String())
	}
	rec = testutil.DoJSON(t, r, http.MethodGet, "/customer/me", token, nil)
	if rec.Code != 401 {
		t.Errorf("revoked = %d", rec.Code)
	}
}
