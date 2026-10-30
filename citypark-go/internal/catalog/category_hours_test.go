package catalog_test

import (
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

func TestCategoryHours(t *testing.T) {
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC))
	tokens := auth.NewTokens("test", time.Hour, c)
	admin, _ := tokens.Issue(1, auth.RoleAdmin)
	user, _ := tokens.Issue(2, auth.RoleUser)
	r := chi.NewRouter()
	h := catalog.NewHandler(catalog.NewStore(db), c, 20)
	h.Routes(r)
	h.AdminRoutes(r, tokens)
	id := testutil.InsertCategory(t, db, testutil.Category{Name: "Обеды"})
	path := "/categories/" + strconv.FormatInt(id, 10) + "/ordering-hours"
	for _, tt := range []struct {
		name, token, path string
		body              any
		status            int
	}{
		{"unauthenticated", "", path, map[string]string{"days": "1111100", "startTime": "9:00", "endTime": "14:00"}, 401},
		{"user", user, path, map[string]string{"days": "1111100", "startTime": "9:00", "endTime": "14:00"}, 403},
		{"missing", admin, "/categories/999999/ordering-hours", map[string]string{"days": "1111100", "startTime": "", "endTime": ""}, 404},
		{"invalid days", admin, path, map[string]string{"days": "0000000", "startTime": "", "endTime": ""}, 400},
		{"invalid mask", admin, path, map[string]string{"days": "abcdefg", "startTime": "", "endTime": ""}, 400},
		{"missing fields", admin, path, map[string]string{}, 400},
		{"bad clocks", admin, path, map[string]string{"days": "1111100", "startTime": "foo", "endTime": "25:00"}, 400},
		{"one empty", admin, path, map[string]string{"days": "1111100", "startTime": "", "endTime": "14:00"}, 400},
		{"reversed", admin, path, map[string]string{"days": "1111100", "startTime": "15:00", "endTime": "14:00"}, 400},
		{"equal", admin, path, map[string]string{"days": "1111100", "startTime": "14:00", "endTime": "14:00"}, 400},
		{"valid", admin, path, map[string]string{"days": "1111100", "startTime": "9:00", "endTime": "14:00"}, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := testutil.DoJSON(t, r, http.MethodPut, tt.path, tt.token, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
		})
	}
	got, err := catalog.NewStore(db).CategoryByID(t.Context(), id)
	if err != nil || got.SaleDays != "1111100" || got.SaleStart == nil || *got.SaleStart != 540 || got.View(c.Now()).IsAvailableForOrder {
		t.Fatalf("category=%+v err=%v", got, err)
	}
	for _, end := range []string{"24:00", ""} {
		start := "9:00"
		if end == "" {
			start = ""
		}
		rec := testutil.DoJSON(t, r, http.MethodPut, path, admin, map[string]string{"days": "1111100", "startTime": start, "endTime": end})
		if rec.Code != 200 {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
	}
	got, err = catalog.NewStore(db).CategoryByID(t.Context(), id)
	if err != nil || got.SaleStart != nil || got.SaleEnd != nil || !got.View(c.Now()).IsAvailableForOrder {
		t.Fatalf("category=%+v err=%v", got, err)
	}
}
