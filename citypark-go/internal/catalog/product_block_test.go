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
	"testing"
	"time"
)

func TestProductOrderingBlock(t *testing.T) {
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
	path := "/products/" + strconv.FormatInt(id, 10) + "/ordering-block"
	for _, tt := range []struct {
		name, token, path string
		body              any
		status            int
	}{
		{"unauthenticated", "", path, map[string]bool{"isOrderBlocked": true}, 401},
		{"role", user, path, map[string]bool{"isOrderBlocked": true}, 403},
		{"missing", admin, "/products/999999/ordering-block", map[string]bool{"isOrderBlocked": true}, 404},
		{"invalid id", admin, "/products/no/ordering-block", map[string]bool{"isOrderBlocked": true}, 404},
		{"missing flag", admin, path, map[string]any{}, 400},
		{"null", admin, path, map[string]any{"isOrderBlocked": nil}, 400},
		{"wrong type", admin, path, map[string]any{"isOrderBlocked": "true"}, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := testutil.DoJSON(t, r, http.MethodPut, tt.path, tt.token, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
	for _, blocked := range []bool{true, true, false} {
		rec := testutil.DoJSON(t, r, http.MethodPut, path, admin, map[string]bool{"isOrderBlocked": blocked})
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		var v catalog.ProductView
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		if v.IsOrderBlocked != blocked || v.IsAvailableForOrder == blocked {
			t.Fatalf("%+v", v)
		}
		list, err := catalog.NewStore(db).ProductsByCategory(t.Context(), cat)
		if err != nil || len(list) != 1 || list[0].IsOrderBlocked != blocked {
			t.Fatalf("%+v %v", list, err)
		}
	}
	if _, err := db.Exec(`UPDATE categories SET sale_days='0000001' WHERE id=$1`, cat); err != nil {
		t.Fatal(err)
	}
	list, err := catalog.NewStore(db).ProductsByIDs(t.Context(), []int64{id})
	if err != nil || len(list) != 1 || list[0].View(c.Now()).IsAvailableForOrder {
		t.Fatalf("schedule lost: %+v %v", list, err)
	}
	if _, err := db.Exec(`UPDATE products SET is_deleted=true WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, r, http.MethodPut, path, admin, map[string]bool{"isOrderBlocked": true})
	if rec.Code != 404 {
		t.Fatalf("deleted: %d %s", rec.Code, rec.Body)
	}

}
