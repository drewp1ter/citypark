package catalog_test

import (
	"encoding/json"
	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProductFullDescription(t *testing.T) {
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC))
	tokens := auth.NewTokens("test", time.Hour, c)
	admin, _ := tokens.Issue(1, auth.RoleAdmin)
	user, _ := tokens.Issue(2, auth.RoleUser)
	r := chi.NewRouter()
	h := catalog.NewHandler(catalog.NewStore(db), c, 20)
	h.Routes(r)
	h.AdminRoutes(r, tokens)
	cat := testutil.InsertCategory(t, db, testutil.Category{Name: "Обеды"})
	id := testutil.InsertProduct(t, db, testutil.Product{CategoryID: cat})
	path := "/products/" + strconv.FormatInt(id, 10) + "/full-description"
	for _, tt := range []struct {
		name, token, path string
		body              any
		status            int
	}{
		{"unauthenticated", "", path, map[string]string{"fullDescription": "<p>Текст</p>"}, 401},
		{"role", user, path, map[string]string{"fullDescription": "<p>Текст</p>"}, 403},
		{"missing", admin, "/products/999999/full-description", map[string]string{"fullDescription": "<p>Текст</p>"}, 404},
		{"invalid id", admin, "/products/no/full-description", map[string]string{"fullDescription": "<p>Текст</p>"}, 404},
		{"missing flag", admin, path, map[string]any{}, 400},
		{"null", admin, path, map[string]any{"fullDescription": nil}, 400},
		{"too long", admin, path, map[string]string{"fullDescription": strings.Repeat("я", 10001)}, 400},
		{"wrong type", admin, path, map[string]any{"fullDescription": true}, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := testutil.DoJSON(t, r, http.MethodPut, tt.path, tt.token, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
	for _, tt := range []struct{ input, want string }{
		{"<p>Блюдо <strong>вкусное</strong></p><script>alert(1)</script>", "<p>Блюдо <strong>вкусное</strong></p>"},
		{"<p><br></p>", ""}, {"<p>&nbsp;</p>", ""}, {"", ""},
	} {
		rec := testutil.DoJSON(t, r, http.MethodPut, path, admin, map[string]string{"fullDescription": tt.input})
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		var v catalog.ProductView
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v.FullDescription != tt.want {
			t.Fatalf("got %q want %q", v.FullDescription, tt.want)
		}
		list, err := catalog.NewStore(db).ProductsByCategory(t.Context(), cat)
		if err != nil || len(list) != 1 || list[0].FullDescription != tt.want {
			t.Fatalf("%+v %v", list, err)
		}
		public := testutil.DoJSON(t, r, http.MethodGet, "/products", "", nil)
		if public.Code != 200 || !strings.Contains(public.Body.String(), `"fullDescription":`) {
			t.Fatalf("%d %s", public.Code, public.Body)
		}
	}
	if _, err := db.Exec(`UPDATE products SET is_deleted=true WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, r, http.MethodPut, path, admin, map[string]string{"fullDescription": ""})
	if rec.Code != 404 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}
