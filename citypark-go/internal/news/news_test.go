package news

import (
	"context"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/upload"
)

const testBase = "http://localhost:3000"

var october = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

type fixture struct {
	db      *sqlx.DB
	uploads *upload.Storage
	dir     string
	router  http.Handler
	admin   string
	user    string
}

// newFixture returns a router over the migrated database. NewDB does not truncate `news` (the seed is part of the
// schema), so the rows found at the start are put back after the test.
func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	restoreNews(t, db)

	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(october))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	uploads := upload.NewStorage(dir, testBase)
	r := chi.NewRouter()
	NewHandler(db, uploads, tokens, clock.Fixed(october), 20).Routes(r)
	return fixture{db: db, uploads: uploads, dir: dir, router: r, admin: admin, user: user}
}

func restoreNews(t *testing.T, db *sqlx.DB) {
	t.Helper()
	type row struct {
		Item
		CreatedAt time.Time `db:"created_at"`
	}
	var before []row
	if err := db.Select(&before, `SELECT created_at, `+columns+` FROM news ORDER BY id`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM news`); err != nil {
			t.Errorf("clear news: %v", err)
		}
		for _, r := range before {
			_, err := db.Exec(`INSERT INTO news (id, created_at, updated_at, slug, date, title, short_title, excerpt,
					body, photo_url, photo_alt, photo_position)
				OVERRIDING SYSTEM VALUE VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				r.ID, r.CreatedAt, r.UpdatedAt, r.Slug, r.Date, r.Title, r.ShortTitle, r.Excerpt, r.Body,
				r.PhotoURL, r.PhotoAlt, r.PhotoPosition)
			if err != nil {
				t.Errorf("restore news %s: %v", r.Slug, err)
			}
		}
	})
}

func (f fixture) do(t *testing.T, method, path, token string, body any) (int, string) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, method, path, token, body)
	return rec.Code, rec.Body.String()
}

// call sends a request as the admin and decodes the answer into out (status 2xx) or the error body.
func (f fixture) call(t *testing.T, method, path string, body any, out any) (int, httpx.Error) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, method, path, f.admin, body)
	var bad httpx.Error
	switch {
	case rec.Code == http.StatusNoContent:
	case rec.Code < 300:
		testutil.Decode(t, rec, out)
	default:
		testutil.Decode(t, rec, &bad)
	}
	return rec.Code, bad
}

func (f fixture) create(t *testing.T, body map[string]any) fullResponse {
	t.Helper()
	var got fullResponse
	if code, bad := f.call(t, http.MethodPost, "/news", body, &got); code != http.StatusCreated {
		t.Fatalf("POST /news = %d %+v", code, bad)
	}
	return got
}

func (f fixture) list(t *testing.T, query string) listResponse {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/news"+query, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /news%s = %d, body %s", query, rec.Code, rec.Body)
	}
	var out listResponse
	testutil.Decode(t, rec, &out)
	return out
}

func (f fixture) fileExists(rel string) bool {
	_, err := os.Stat(filepath.Join(f.dir, filepath.FromSlash(strings.TrimPrefix(rel, "/uploads/"))))
	return err == nil
}

func validBody() map[string]any {
	return map[string]any{
		"slug": "steaks-june", "date": "2026-06-01", "title": "Стейки июня", "shortTitle": "Стейки",
		"excerpt": "Новые стейки в меню.", "body": "<p>Текст</p>",
		"photo": map[string]any{"url": "/images/news-steaks.jpg", "alt": "Стейк", "position": "50% 30%"},
	}
}

func with(body map[string]any, key string, value any) map[string]any {
	out := maps.Clone(body)
	out[key] = value
	return out
}

func withPhoto(body map[string]any, key string, value any) map[string]any {
	photo := maps.Clone(body["photo"].(map[string]any))
	photo[key] = value
	return with(body, "photo", photo)
}

func TestSlugify(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"title with year", "Новое меню в Сити Парке 2026", "novoe-menyu-v-siti-parke-2026"},
		{"symbols between words", "Щи & борщ!", "shchi-borshch"},
		{"yo", "Ёлка", "elka"},
		{"only symbols", "!!!", "news"},
		{"empty", "", "news"},
		{"hard and soft signs vanish", "Объём, субъект, мальчик", "obem-subekt-malchik"},
		{"latin and digits stay", "Menu 2026 A-B", "menu-2026-a-b"},
		{"upper case", "ЖЮРИ ЧЯ", "zhyuri-chya"},
		{"long title", strings.Repeat("а", 200), strings.Repeat("a", 80)},
		{"cut leaves no trailing dash", strings.Repeat("а", 79) + " б", strings.Repeat("a", 79)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSeed(t *testing.T) {
	f := newFixture(t)
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/news", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"body"`) {
		t.Errorf("the list has a body key: %s", rec.Body)
	}

	out := f.list(t, "")

	if len(out.News) != 7 || out.Count != 7 {
		t.Fatalf("got %d items, count %d, want 7 and 7", len(out.News), out.Count)
	}
	first := out.News[0]
	if first.Slug != "special-offer-september-2026" || first.Date != "2026-09-07" ||
		first.Title != "Спецпредложение сентября" || first.Excerpt != "Что входит, цена и срок действия." ||
		first.Photo.URL != "" {
		t.Errorf("first = %+v", first)
	}
	menu := out.News[3]
	if menu.Slug != "new-menu-2026" || menu.ShortTitle != "Новое меню 2026" || menu.Photo.URL != "/images/news-menu.jpg" {
		t.Errorf("fourth = %+v", menu)
	}
	if out.News[6].Slug != "lenten-menu-2025" {
		t.Errorf("last = %s", out.News[6].Slug)
	}
}

func TestGetNewsBySlug(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, with(validBody(), "body", "<p>Полный текст</p>"))

	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/news/steaks-june", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got fullResponse
	testutil.Decode(t, rec, &got)
	if got.ID != created.ID || got.Body != "<p>Полный текст</p>" || got.Title != "Стейки июня" || got.Date != "2026-06-01" {
		t.Errorf("got %+v", got)
	}
	if code, body := f.do(t, http.MethodGet, "/news/no-such-news", "", nil); code != http.StatusNotFound {
		t.Errorf("unknown slug: status = %d, body %s", code, body)
	}
}

func TestCreateNews(t *testing.T) {
	f := newFixture(t)

	first := f.create(t, map[string]any{"title": "Новые стейки"})
	second := f.create(t, map[string]any{"title": "Новые стейки"})
	third := f.create(t, map[string]any{"title": "Новые стейки"})

	if first.Slug != "novye-steyki" || second.Slug != "novye-steyki-2" || third.Slug != "novye-steyki-3" {
		t.Errorf("slugs = %q, %q, %q", first.Slug, second.Slug, third.Slug)
	}
	if first.Date != "2026-10-22" {
		t.Errorf("date = %q, want today 2026-10-22", first.Date)
	}
	if first.Body != "" || first.Photo != (photoResponse{}) || first.ID == 0 || first.UpdatedAt.IsZero() {
		t.Errorf("first = %+v", first)
	}
	if got := f.list(t, "").News[0]; got.Slug != "novye-steyki-3" {
		t.Errorf("newest first: got %s", got.Slug)
	}
}

func TestCreateNewsFullBody(t *testing.T) {
	f := newFixture(t)
	body := with(validBody(), "title", "  Стейки июня  ")

	got := f.create(t, body)

	want := fullResponse{
		ID: got.ID, Slug: "steaks-june", Date: "2026-06-01", Title: "Стейки июня", ShortTitle: "Стейки",
		Excerpt: "Новые стейки в меню.", Body: "<p>Текст</p>",
		Photo: photoResponse{URL: "/images/news-steaks.jpg", Alt: "Стейк", Position: "50% 30%"}, UpdatedAt: got.UpdatedAt,
	}
	if got != want {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestCreateNewsUploadedPhoto(t *testing.T) {
	f := newFixture(t)
	path, err := f.uploads.Save(october, []byte("jpg"), "jpg")
	if err != nil {
		t.Fatal(err)
	}

	got := f.create(t, withPhoto(validBody(), "url", testBase+path))

	if got.Photo.URL != testBase+path {
		t.Errorf("photo url = %q, want %q", got.Photo.URL, testBase+path)
	}
	stored, err := NewStore(f.db).GetBySlug(context.Background(), "steaks-june")
	if err != nil {
		t.Fatal(err)
	}
	if stored.PhotoURL != path {
		t.Errorf("stored photo = %q, want %q", stored.PhotoURL, path)
	}
}

func TestCreateNewsSlugTaken(t *testing.T) {
	f := newFixture(t)
	before := f.list(t, "").Count

	code, bad := f.call(t, http.MethodPost, "/news", with(validBody(), "slug", "new-menu-2026"), nil)

	if code != http.StatusConflict || bad.Message != "Новость с таким адресом уже есть" {
		t.Errorf("status = %d, error %+v", code, bad)
	}
	if after := f.list(t, "").Count; after != before {
		t.Errorf("count = %d, want %d", after, before)
	}
}

func TestUpdateNews(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, validBody())
	body := with(with(validBody(), "title", "Другой заголовок"), "body", "<script>x</script><h2>a</h2>")

	var got fullResponse
	code, bad := f.call(t, http.MethodPut, "/news/"+itoa(created.ID), body, &got)

	if code != http.StatusOK {
		t.Fatalf("status = %d %+v", code, bad)
	}
	if got.ID != created.ID || got.Slug != "steaks-june" || got.Title != "Другой заголовок" || got.Body != "<h2>a</h2>" {
		t.Errorf("got %+v", got)
	}
	stored, err := NewStore(f.db).GetBySlug(context.Background(), "steaks-june")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Title != "Другой заголовок" || stored.Body != "<h2>a</h2>" {
		t.Errorf("stored %+v", stored)
	}
}

func TestUpdateNewsGeneratedSlug(t *testing.T) {
	f := newFixture(t)
	a := f.create(t, map[string]any{"title": "Акция"})
	b := f.create(t, map[string]any{"title": "Другая"})

	// An empty slug is made from the title again; the item's own slug is free, another item's is not.
	var same, other fullResponse
	if code, bad := f.call(t, http.MethodPut, "/news/"+itoa(a.ID), map[string]any{"title": "Акция"}, &same); code != 200 {
		t.Fatalf("status = %d %+v", code, bad)
	}
	if code, bad := f.call(t, http.MethodPut, "/news/"+itoa(b.ID), map[string]any{"title": "Акция"}, &other); code != 200 {
		t.Fatalf("status = %d %+v", code, bad)
	}

	if same.Slug != "aktsiya" || other.Slug != "aktsiya-2" {
		t.Errorf("slugs = %q, %q", same.Slug, other.Slug)
	}
}

func TestUpdateNewsSlugTaken(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, validBody())

	code, bad := f.call(t, http.MethodPut, "/news/"+itoa(created.ID), with(validBody(), "slug", "new-menu-2026"), nil)

	if code != http.StatusConflict || bad.Message != "Новость с таким адресом уже есть" {
		t.Errorf("status = %d, error %+v", code, bad)
	}
}

func TestUpdateNewsUnknown(t *testing.T) {
	f := newFixture(t)
	for _, id := range []string{"9999", "0", "abc"} {
		if code, _ := f.call(t, http.MethodPut, "/news/"+id, validBody(), nil); code != http.StatusNotFound {
			t.Errorf("PUT /news/%s = %d, want 404", id, code)
		}
	}
}

func TestUpdateNewsReplacesPhoto(t *testing.T) {
	f := newFixture(t)
	oldPath, _ := f.uploads.Save(october, []byte("old"), "jpg")
	newPath, _ := f.uploads.Save(october, []byte("new"), "jpg")
	created := f.create(t, withPhoto(validBody(), "url", oldPath))

	// Same photo: the file stays.
	if code, bad := f.call(t, http.MethodPut, "/news/"+itoa(created.ID), withPhoto(validBody(), "url", oldPath), &fullResponse{}); code != 200 {
		t.Fatalf("status = %d %+v", code, bad)
	}
	if !f.fileExists(oldPath) {
		t.Fatal("the photo was removed although it did not change")
	}

	// New photo: the old file goes, the new one stays.
	if code, bad := f.call(t, http.MethodPut, "/news/"+itoa(created.ID), withPhoto(validBody(), "url", newPath), &fullResponse{}); code != 200 {
		t.Fatalf("status = %d %+v", code, bad)
	}
	if f.fileExists(oldPath) || !f.fileExists(newPath) {
		t.Errorf("old exists = %v, new exists = %v", f.fileExists(oldPath), f.fileExists(newPath))
	}

	// No photo: the file goes.
	if code, bad := f.call(t, http.MethodPut, "/news/"+itoa(created.ID), withPhoto(validBody(), "url", ""), &fullResponse{}); code != 200 {
		t.Fatalf("status = %d %+v", code, bad)
	}
	if f.fileExists(newPath) {
		t.Error("the photo file stays after the photo was cleared")
	}
}

func TestDeleteNews(t *testing.T) {
	f := newFixture(t)
	path, _ := f.uploads.Save(october, []byte("jpg"), "jpg")
	created := f.create(t, withPhoto(validBody(), "url", path))
	before := f.list(t, "").Count

	code, _ := f.call(t, http.MethodDelete, "/news/"+itoa(created.ID), nil, nil)

	if code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", code)
	}
	out := f.list(t, "")
	if out.Count != before-1 {
		t.Errorf("count = %d, want %d", out.Count, before-1)
	}
	for _, it := range out.News {
		if it.ID == created.ID {
			t.Errorf("item %d is still in the list", it.ID)
		}
	}
	if f.fileExists(path) {
		t.Error("the uploaded photo file was not removed")
	}
	if code, _ := f.call(t, http.MethodDelete, "/news/"+itoa(created.ID), nil, nil); code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", code)
	}
	if code, _ := f.call(t, http.MethodDelete, "/news/abc", nil, nil); code != http.StatusNotFound {
		t.Errorf("bad id = %d, want 404", code)
	}
}

func TestDeleteNewsKeepsSitePhoto(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, validBody())

	if code, _ := f.call(t, http.MethodDelete, "/news/"+itoa(created.ID), nil, nil); code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", code)
	}
}

func TestNewsInvalid(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name  string
		body  map[string]any
		field string
		want  string
	}{
		{"no title", map[string]any{"slug": "x"}, "title", "поле не должно быть пустым"},
		{"blank title", map[string]any{"title": "   "}, "title", "поле не должно быть пустым"},
		{"long title", map[string]any{"title": strings.Repeat("я", 201)}, "title", "значение должно быть не больше 200"},
		{"long short title", with(validBody(), "shortTitle", strings.Repeat("я", 101)), "shortTitle", "значение должно быть не больше 100"},
		{"long excerpt", with(validBody(), "excerpt", strings.Repeat("я", 301)), "excerpt", "значение должно быть не больше 300"},
		{"date format", with(validBody(), "date", "07.09.2026"), "date", "дата в формате ГГГГ-ММ-ДД"},
		{"date not real", with(validBody(), "date", "2026-02-30"), "date", "дата в формате ГГГГ-ММ-ДД"},
		{"russian slug", with(validBody(), "slug", "Новость"), "slug", "только латиница, цифры и дефисы"},
		{"slug with double dash", with(validBody(), "slug", "a--b"), "slug", "только латиница, цифры и дефисы"},
		{"long slug", with(validBody(), "slug", strings.Repeat("a", 101)), "slug", "значение должно быть не больше 100"},
		{"foreign photo", withPhoto(validBody(), "url", "https://evil.example/a.jpg"), "photo.url", "недопустимый адрес фото"},
		{"long alt", withPhoto(validBody(), "alt", strings.Repeat("я", 201)), "photo.alt", "значение должно быть не больше 200"},
		{"position format", withPhoto(validBody(), "position", "right center"), "photo.position", "формат «50% 30%»"},
		{"position over 100", withPhoto(validBody(), "position", "50% 101%"), "photo.position", "формат «50% 30%»"},
		{"body too long", with(validBody(), "body", "<p>"+strings.Repeat("я", 50001)+"</p>"), "body", "не более 50000 символов"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range []struct{ method, path string }{{http.MethodPost, "/news"}, {http.MethodPut, "/news/1"}} {
				code, bad := f.call(t, req.method, req.path, tt.body, nil)
				if code != http.StatusBadRequest {
					t.Fatalf("%s %s = %d, want 400", req.method, req.path, code)
				}
				if bad.Fields[tt.field] != tt.want {
					t.Errorf("%s errors = %v, want %s: %q", req.method, bad.Fields, tt.field, tt.want)
				}
			}
		})
	}
	if n := f.list(t, "").Count; n != 7 {
		t.Errorf("count = %d, a rejected request changed the data", n)
	}
}

func TestNewsInvalidReportsAllFields(t *testing.T) {
	f := newFixture(t)
	body := map[string]any{"date": "x", "slug": "Ы", "photo": map[string]any{"url": "ftp://x", "position": "?"}}

	code, bad := f.call(t, http.MethodPost, "/news", body, nil)

	if code != http.StatusBadRequest || bad.Message != validationMessage {
		t.Fatalf("status = %d, error %+v", code, bad)
	}
	for _, field := range []string{"title", "date", "slug", "photo.url", "photo.position"} {
		if bad.Fields[field] == "" {
			t.Errorf("no error for %s: %v", field, bad.Fields)
		}
	}
}

func TestNewsMalformedJSON(t *testing.T) {
	f := newFixture(t)
	if code, _ := f.call(t, http.MethodPost, "/news", "not an object", nil); code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

func TestNewsPaging(t *testing.T) {
	f := newFixture(t)

	out := f.list(t, "?page=2&pageSize=5")

	if len(out.News) != 2 || out.Count != 7 {
		t.Fatalf("got %d items, count %d, want 2 and 7", len(out.News), out.Count)
	}
	if out.News[0].Slug != "updated-menu-2025" || out.News[1].Slug != "lenten-menu-2025" {
		t.Errorf("page 2 = %s, %s", out.News[0].Slug, out.News[1].Slug)
	}
	empty := f.list(t, "?page=9&pageSize=5")
	if empty.News == nil || len(empty.News) != 0 || empty.Count != 7 {
		t.Errorf("past the end = %+v", empty)
	}
	if code, _ := f.do(t, http.MethodGet, "/news?page=0", "", nil); code != http.StatusBadRequest {
		t.Errorf("page=0: status = %d, want 400", code)
	}
}

func TestNewsListOrder(t *testing.T) {
	f := newFixture(t)
	a := f.create(t, with(validBody(), "slug", "same-day-a"))
	b := f.create(t, with(validBody(), "slug", "same-day-b"))
	newest := f.create(t, with(with(validBody(), "slug", "future"), "date", "2027-01-01"))

	out := f.list(t, "")

	got := []int64{out.News[0].ID, out.News[1].ID}
	if got[0] != newest.ID || out.News[1].Slug != "special-offer-september-2026" {
		t.Errorf("order = %v", got)
	}
	idx := map[int64]int{}
	for i, it := range out.News {
		idx[it.ID] = i
	}
	if idx[b.ID] > idx[a.ID] {
		t.Errorf("same date: id %d must come before id %d", b.ID, a.ID)
	}
}

func TestNewsAuth(t *testing.T) {
	f := newFixture(t)
	routes := []struct{ method, path string }{
		{http.MethodPost, "/news"},
		{http.MethodPut, "/news/1"},
		{http.MethodDelete, "/news/1"},
	}
	tokens := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"garbage token", "nope", http.StatusUnauthorized},
		{"user role", f.user, http.StatusForbidden},
	}
	for _, rt := range routes {
		for _, tk := range tokens {
			t.Run(rt.method+" "+rt.path+" "+tk.name, func(t *testing.T) {
				if code, body := f.do(t, rt.method, rt.path, tk.token, validBody()); code != tk.want {
					t.Errorf("status = %d, want %d, body %s", code, tk.want, body)
				}
			})
		}
	}
	if n := f.list(t, "").Count; n != 7 {
		t.Errorf("count = %d, a rejected request changed the data", n)
	}
	for _, path := range []string{"/news", "/news/new-menu-2026"} {
		if code, _ := f.do(t, http.MethodGet, path, "", nil); code != http.StatusOK {
			t.Errorf("GET %s without token = %d, want 200", path, code)
		}
	}
}

func TestPublicUploadedPhotoURL(t *testing.T) {
	f := newFixture(t)
	path, _ := f.uploads.Save(october, []byte("jpg"), "jpg")
	f.create(t, withPhoto(validBody(), "url", path))

	out := f.list(t, "")

	found := false
	for _, it := range out.News {
		if it.Slug == "steaks-june" {
			found = true
			if it.Photo.URL != testBase+path {
				t.Errorf("photo url = %q, want %q", it.Photo.URL, testBase+path)
			}
		}
	}
	if !found {
		t.Error("created item is not in the list")
	}
}

func TestStoreNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	restoreNews(t, db)
	s := NewStore(db)
	ctx := context.Background()

	if _, err := s.GetBySlug(ctx, "nope"); err != store.ErrNotFound {
		t.Errorf("GetBySlug err = %v", err)
	}
	if _, _, err := s.Update(ctx, 9999, Input{Title: "x", Date: october}); err != store.ErrNotFound {
		t.Errorf("Update err = %v", err)
	}
	if _, err := s.Delete(ctx, 9999); err != store.ErrNotFound {
		t.Errorf("Delete err = %v", err)
	}
}

func TestValidPosition(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true}, {"0% 0%", true}, {"50% 30%", true}, {"100% 100%", true},
		{"101% 0%", false}, {"0% 101%", false}, {"50%30%", false}, {"center", false}, {"-5% 10%", false},
	}
	for _, tt := range tests {
		if got := validPosition(tt.in); got != tt.want {
			t.Errorf("validPosition(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
