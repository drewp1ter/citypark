package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/go-chi/chi/v5"
)

const (
	bodyUnauthorized = `{"statusCode":401,"message":"Требуется авторизация"}`
	bodyForbidden    = `{"statusCode":403,"message":"Недостаточно прав"}`
)

// middlewareRouter has GET /x behind RequireRole(role); the handler echoes the identity.
func middlewareRouter(role string) (http.Handler, *Tokens) {
	tokens := newTokens("secret", t0)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(tokens.RequireRole(role))
		r.Get("/x", func(w http.ResponseWriter, r *http.Request) {
			id, ok := FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, httpx.Unauthorized())
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]any{"userId": id.UserID, "role": id.Role})
		})
	})
	return r, tokens
}

func get(h http.Handler, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func checkResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != body {
		t.Errorf("body = %s, want %s", got, body)
	}
}

func TestAllows(t *testing.T) {
	tests := []struct {
		have, need string
		want       bool
	}{
		{RoleUser, RoleUser, true},
		{RoleAdmin, RoleUser, true},
		{RoleUser, RoleAdmin, false},
		{RoleAdmin, RoleAdmin, true},
		{"", RoleUser, false},
		{RoleAdmin, "unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.have+"->"+tt.need, func(t *testing.T) {
			if got := Allows(tt.have, tt.need); got != tt.want {
				t.Errorf("Allows(%q, %q) = %v, want %v", tt.have, tt.need, got, tt.want)
			}
		})
	}
}

func TestRequireRoleNoHeader(t *testing.T) {
	h, _ := middlewareRouter(RoleUser)
	checkResponse(t, get(h, ""), http.StatusUnauthorized, bodyUnauthorized)
}

func TestRequireRoleBadScheme(t *testing.T) {
	h, _ := middlewareRouter(RoleUser)
	for _, header := range []string{"Token abc", "Bearer", "Bearer ", "bearer abc"} {
		t.Run(header, func(t *testing.T) {
			checkResponse(t, get(h, header), http.StatusUnauthorized, bodyUnauthorized)
		})
	}
}

func TestRequireRoleBadToken(t *testing.T) {
	h, _ := middlewareRouter(RoleUser)
	checkResponse(t, get(h, "Bearer garbage"), http.StatusUnauthorized, bodyUnauthorized)
}

func TestRequireRoleUserOnUserRoute(t *testing.T) {
	h, tokens := middlewareRouter(RoleUser)
	token, err := tokens.Issue(42, RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	checkResponse(t, get(h, "Bearer "+token), http.StatusOK, `{"role":"user","userId":42}`)
}

func TestRequireRoleAdminOnUserRoute(t *testing.T) {
	h, tokens := middlewareRouter(RoleUser)
	token, err := tokens.Issue(7, RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	checkResponse(t, get(h, "Bearer "+token), http.StatusOK, `{"role":"admin","userId":7}`)
}

func TestRequireRoleUserOnAdminRoute(t *testing.T) {
	h, tokens := middlewareRouter(RoleAdmin)
	token, err := tokens.Issue(42, RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	checkResponse(t, get(h, "Bearer "+token), http.StatusForbidden, bodyForbidden)
}

func TestRequireRoleAdminOnAdminRoute(t *testing.T) {
	h, tokens := middlewareRouter(RoleAdmin)
	token, err := tokens.Issue(7, RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	checkResponse(t, get(h, "Bearer "+token), http.StatusOK, `{"role":"admin","userId":7}`)
}

func TestFromContextEmpty(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("FromContext on an empty context returned ok == true")
	}
}

func TestNewContextRoundTrip(t *testing.T) {
	want := Identity{UserID: 5, Role: RoleUser}
	got, ok := FromContext(NewContext(context.Background(), want))
	if !ok || got != want {
		t.Errorf("FromContext = %+v, %v; want %+v, true", got, ok, want)
	}
}
