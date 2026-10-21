package catalog_test

import (
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

// newCatalogEnv returns the catalog routes and an empty database.
// The clock is Friday 2024-05-24 15:00.
func newCatalogEnv(t *testing.T) (http.Handler, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2024, 5, 24, 15, 0, 0, 0, time.UTC))
	r := chi.NewRouter()
	catalog.NewHandler(catalog.NewStore(db), c, 20).Routes(r)
	return r, db
}

func TestListCategoriesEndpoint(t *testing.T) {
	r, db := newCatalogEnv(t)

	testutil.InsertCategory(t, db, testutil.Category{Name: "Второй", SortOrder: 1})
	testutil.InsertCategory(t, db, testutil.Category{Name: "Первый", SortOrder: 0})
	testutil.InsertCategory(t, db, testutil.Category{Name: "Третий", SortOrder: 2})
	testutil.InsertCategory(t, db, testutil.Category{Name: "Удалённый", SortOrder: 0, IsDeleted: true})

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var got []catalog.CategoryView
	testutil.Decode(t, rec, &got)
	names := make([]string, len(got))
	for i, v := range got {
		names[i] = v.Name
	}
	want := []string{"Первый", "Второй", "Третий"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestListCategoriesEmptyEndpoint(t *testing.T) {
	r, _ := newCatalogEnv(t)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestListCategoriesAvailability(t *testing.T) {
	r, db := newCatalogEnv(t)

	testutil.InsertCategory(t, db, testutil.Category{
		Name: "Будни", SortOrder: 0,
		SaleStart: testutil.IntPtr(9 * 60), SaleEnd: testutil.IntPtr(21 * 60), SaleDays: "1111100",
	})
	testutil.InsertCategory(t, db, testutil.Category{
		Name: "Выходные", SortOrder: 1,
		SaleStart: testutil.IntPtr(9 * 60), SaleEnd: testutil.IntPtr(21 * 60), SaleDays: "0000011",
	})

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var got []catalog.CategoryView
	testutil.Decode(t, rec, &got)
	if len(got) != 2 {
		t.Fatalf("len = %d, body = %s", len(got), rec.Body)
	}
	if !got[0].IsAvailableForOrder {
		t.Errorf("weekday category on Friday 15:00 must be available: %+v", got[0])
	}
	if got[1].IsAvailableForOrder {
		t.Errorf("weekend category on Friday must not be available: %+v", got[1])
	}
	if got[0].StartTime != "9:00" || got[0].EndTime != "21:00" {
		t.Errorf("times = %q-%q, want 9:00-21:00", got[0].StartTime, got[0].EndTime)
	}
}

func TestGetCategory(t *testing.T) {
	r, db := newCatalogEnv(t)

	id := testutil.InsertCategory(t, db, testutil.Category{
		Name:      "Мангал",
		SaleStart: testutil.IntPtr(9 * 60), SaleEnd: testutil.IntPtr(21 * 60), SaleDays: "1111100",
	})
	insertCategoryImage(t, db, id, "https://example.com/a.jpg")

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/"+strconv.FormatInt(id, 10), "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	var raw map[string]any
	testutil.Decode(t, rec, &raw)
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	wantKeys := []string{"description", "endTime", "id", "images", "isAvailableForOrder", "name", "salesDays", "startTime"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("keys = %v, want %v", keys, wantKeys)
	}

	var got catalog.CategoryView
	testutil.Decode(t, rec, &got)
	if got.ID != id || got.Name != "Мангал" || !got.IsAvailableForOrder {
		t.Fatalf("view = %+v", got)
	}
	if len(got.Images) != 1 || got.Images[0].URL != "https://example.com/a.jpg" {
		t.Fatalf("images = %+v", got.Images)
	}
}

func TestGetCategoryMissing(t *testing.T) {
	r, _ := newCatalogEnv(t)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/999", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	want := `{"statusCode":404,"message":"Не найдено"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestGetCategoryDeleted(t *testing.T) {
	r, db := newCatalogEnv(t)

	id := testutil.InsertCategory(t, db, testutil.Category{Name: "Удалённая", IsDeleted: true})

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/"+strconv.FormatInt(id, 10), "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetCategoryBadID(t *testing.T) {
	r, _ := newCatalogEnv(t)

	rec := testutil.DoJSON(t, r, http.MethodGet, "/categories/abc", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
