package content

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/upload"
)

const testBase = "http://localhost:3000"

type fixture struct {
	db      *sqlx.DB
	uploads *upload.Storage
	dir     string
	router  http.Handler
	admin   string
	user    string
}

// newFixture returns a router over the migrated database with an empty content_blocks table. NewDB does not truncate
// that table (later tasks add rows with a migration), so the rows found at the start are put back after the test.
func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	var before []Block
	if err := db.Select(&before, `SELECT `+columns+` FROM content_blocks`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM content_blocks`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM content_blocks`); err != nil {
			t.Errorf("clear content_blocks: %v", err)
		}
		for _, b := range before {
			insertBlock(t, db, b)
		}
	})

	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(now))
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
	NewHandler(db, uploads, tokens).Routes(r)
	return fixture{db: db, uploads: uploads, dir: dir, router: r, admin: admin, user: user}
}

func insertBlock(t *testing.T, db *sqlx.DB, b Block) {
	t.Helper()
	_, err := db.NamedExec(`INSERT INTO content_blocks (key, page, section, sort, label, type, max_length, text_value,
			image_url, image_alt, image_position, updated_at)
		VALUES (:key, :page, :section, :sort, :label, :type, :max_length, :text_value, :image_url, :image_alt,
			:image_position, :updated_at)`, b)
	if err != nil {
		t.Fatalf("insert block %s: %v", b.Key, err)
	}
}

// seed inserts one block of each type: home.title (text, max 20), home.photo (image), hotel.amenities (lines) and
// hotel.intro (richtext).
func (f fixture) seed(t *testing.T) {
	t.Helper()
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	insertBlock(t, f.db, Block{Key: "home.title", Page: "home", Section: "Первый экран", Sort: 1, Label: "Заголовок",
		Type: TypeText, MaxLength: 20, TextValue: "Отель и ресторан", UpdatedAt: at})
	insertBlock(t, f.db, Block{Key: "home.photo", Page: "home", Section: "Первый экран", Sort: 2, Label: "Фото",
		Type: TypeImage, ImageURL: "/images/hotel-building.jpg", ImageAlt: "Здание отеля", UpdatedAt: at})
	insertBlock(t, f.db, Block{Key: "hotel.amenities", Page: "hotel", Section: "Номера", Sort: 1, Label: "Удобства",
		Type: TypeLines, TextValue: "Сплит-система\nSmart TV", UpdatedAt: at})
	insertBlock(t, f.db, Block{Key: "hotel.intro", Page: "hotel", Section: "Номера", Sort: 2, Label: "Вступление",
		Type: TypeRichtext, TextValue: "<p>Привет</p>", UpdatedAt: at})
}

func (f fixture) put(t *testing.T, key string, body any) (int, map[string]any) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/content/blocks/"+key, f.admin, body)
	var out map[string]any
	testutil.Decode(t, rec, &out)
	return rec.Code, out
}

// textValue reads text_value straight from the database.
func (f fixture) textValue(t *testing.T, key string) string {
	t.Helper()
	var v string
	if err := f.db.Get(&v, `SELECT text_value FROM content_blocks WHERE key = $1`, key); err != nil {
		t.Fatal(err)
	}
	return v
}

func wantFieldError(t *testing.T, code int, body map[string]any, field, message string) {
	t.Helper()
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body %v", code, body)
	}
	errs, _ := body["errors"].(map[string]any)
	if errs[field] != message {
		t.Errorf("errors[%q] = %v, want %q (body %v)", field, errs[field], message, body)
	}
}

func TestListBlocks(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/content/blocks", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var all []map[string]any
	testutil.Decode(t, rec, &all)
	var keys []string
	for _, b := range all {
		keys = append(keys, b["key"].(string))
	}
	if want := []string{"home.title", "home.photo", "hotel.amenities", "hotel.intro"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}

	title := all[0]
	if title["value"] != "Отель и ресторан" || title["type"] != "text" || title["maxLength"] != float64(20) ||
		title["image"] != nil || title["section"] != "Первый экран" || title["label"] != "Заголовок" ||
		title["page"] != "home" {
		t.Errorf("text block = %v", title)
	}
	photo := all[1]
	wantImage := map[string]any{"url": "/images/hotel-building.jpg", "alt": "Здание отеля", "position": ""}
	if v, has := photo["value"]; !has || v != nil || !reflect.DeepEqual(photo["image"], wantImage) ||
		photo["maxLength"] != float64(0) {
		t.Errorf("image block = %v", photo)
	}
	if !reflect.DeepEqual(all[2]["value"], []any{"Сплит-система", "Smart TV"}) || all[2]["image"] != nil {
		t.Errorf("lines block = %v", all[2])
	}
	if all[3]["value"] != "<p>Привет</p>" || all[3]["maxLength"] != float64(2000) {
		t.Errorf("richtext block = %v", all[3])
	}

	rec = testutil.DoJSON(t, f.router, http.MethodGet, "/content/blocks?page=hotel", "", nil)
	var hotel []map[string]any
	testutil.Decode(t, rec, &hotel)
	if len(hotel) != 2 || hotel[0]["key"] != "hotel.amenities" || hotel[1]["key"] != "hotel.intro" {
		t.Errorf("?page=hotel = %v", hotel)
	}

	rec = testutil.DoJSON(t, f.router, http.MethodGet, "/content/blocks?page=none", "", nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("?page=none = %d %q, want 200 []", rec.Code, rec.Body)
	}
}

func TestListBlocksEmptyLinesAndUploadURL(t *testing.T) {
	f := newFixture(t)
	at := time.Now()
	insertBlock(t, f.db, Block{Key: "a.lines", Page: "a", Label: "L", Type: TypeLines, UpdatedAt: at})
	insertBlock(t, f.db, Block{Key: "a.photo", Page: "a", Sort: 1, Label: "P", Type: TypeImage,
		ImageURL: "/uploads/2026/10/x.jpg", UpdatedAt: at})

	var all []map[string]any
	testutil.Decode(t, testutil.DoJSON(t, f.router, http.MethodGet, "/content/blocks", "", nil), &all)
	if !reflect.DeepEqual(all[0]["value"], []any{}) {
		t.Errorf("empty lines value = %#v, want []", all[0]["value"])
	}
	img := all[1]["image"].(map[string]any)
	if img["url"] != testBase+"/uploads/2026/10/x.jpg" {
		t.Errorf("image url = %v, want the public URL", img["url"])
	}
}

func TestPutText(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	code, body := f.put(t, "home.title", map[string]any{"value": "  Новый заголовок  "})
	if code != http.StatusOK || body["value"] != "Новый заголовок" || body["key"] != "home.title" {
		t.Fatalf("put = %d %v", code, body)
	}
	if got := f.textValue(t, "home.title"); got != "Новый заголовок" {
		t.Errorf("stored = %q", got)
	}
	if body["updatedAt"] == "2026-10-01T10:00:00Z" {
		t.Errorf("updatedAt was not refreshed: %v", body["updatedAt"])
	}

	code, body = f.put(t, "home.title", map[string]any{"value": strings.Repeat("я", 21)})
	wantFieldError(t, code, body, "value", "не более 20 символов")
	code, body = f.put(t, "home.title", map[string]any{"value": strings.Repeat("я", 20)})
	if code != http.StatusOK {
		t.Errorf("20 characters: status %d %v", code, body)
	}

	for _, v := range []string{"a\nb", "a\r\nb"} {
		code, body = f.put(t, "home.title", map[string]any{"value": v})
		wantFieldError(t, code, body, "value", "не должно содержать переносов строк")
	}
	if got := f.textValue(t, "home.title"); got != strings.Repeat("я", 20) {
		t.Errorf("a rejected value changed the row: %q", got)
	}
}

func TestPutTextDefaultLength(t *testing.T) {
	f := newFixture(t)
	insertBlock(t, f.db, Block{Key: "a.t", Page: "a", Label: "T", Type: TypeText, UpdatedAt: time.Now()})

	code, body := f.put(t, "a.t", map[string]any{"value": strings.Repeat("x", 300)})
	if code != http.StatusOK || body["maxLength"] != float64(300) {
		t.Errorf("300 characters: %d %v", code, body)
	}
	code, body = f.put(t, "a.t", map[string]any{"value": strings.Repeat("x", 301)})
	wantFieldError(t, code, body, "value", "не более 300 символов")
}

func TestPutLines(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	code, body := f.put(t, "hotel.amenities", map[string]any{"value": []string{"a", " ", "b "}})
	if code != http.StatusOK || !reflect.DeepEqual(body["value"], []any{"a", "b"}) {
		t.Fatalf("put = %d %v", code, body)
	}
	if got := f.textValue(t, "hotel.amenities"); got != "a\nb" {
		t.Errorf("stored = %q, want a\\nb", got)
	}

	code, body = f.put(t, "hotel.amenities", map[string]any{"value": []string{}})
	wantFieldError(t, code, body, "value", "от 1 до 50 строк")
	code, body = f.put(t, "hotel.amenities", map[string]any{"value": []string{" ", ""}})
	wantFieldError(t, code, body, "value", "от 1 до 50 строк")

	fifty := make([]string, 50)
	for i := range fifty {
		fifty[i] = "строка"
	}
	if code, body = f.put(t, "hotel.amenities", map[string]any{"value": fifty}); code != http.StatusOK {
		t.Errorf("50 lines: %d %v", code, body)
	}
	code, body = f.put(t, "hotel.amenities", map[string]any{"value": append(fifty, "ещё")})
	wantFieldError(t, code, body, "value", "от 1 до 50 строк")

	code, body = f.put(t, "hotel.amenities", map[string]any{"value": []string{strings.Repeat("я", 201)}})
	wantFieldError(t, code, body, "value", "каждая строка не более 200 символов")
	code, body = f.put(t, "hotel.amenities", map[string]any{"value": []string{"a\nb"}})
	wantFieldError(t, code, body, "value", "строки не должны содержать переносов")
	code, body = f.put(t, "hotel.amenities", map[string]any{"value": []string{"Штраф {{price}}"}})
	wantFieldError(t, code, body, "value", "неизвестная подстановка {{price}}")
}

func TestPutRichtext(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	code, body := f.put(t, "hotel.intro", map[string]any{"value": "<h2>x</h2><p>y</p>"})
	if code != http.StatusOK || body["value"] != "x<p>y</p>" {
		t.Fatalf("put = %d %v", code, body)
	}
	if got := f.textValue(t, "hotel.intro"); got != "x<p>y</p>" {
		t.Errorf("stored = %q", got)
	}

	code, body = f.put(t, "hotel.intro", map[string]any{"value": "<p>" + strings.Repeat("я", 2001) + "</p>"})
	wantFieldError(t, code, body, "value", "не более 2000 символов")
	// Tags do not count towards the limit.
	code, body = f.put(t, "hotel.intro", map[string]any{"value": "<p><strong>" + strings.Repeat("я", 2000) + "</strong></p>"})
	if code != http.StatusOK {
		t.Errorf("2000 visible characters: %d %v", code, body)
	}
	code, body = f.put(t, "hotel.intro", map[string]any{"value": "<p>{{nope}}</p>"})
	wantFieldError(t, code, body, "value", "неизвестная подстановка {{nope}}")
}

func TestPutPlaceholders(t *testing.T) {
	f := newFixture(t)
	f.seed(t)
	insertBlock(t, f.db, Block{Key: "hotel.rule", Page: "hotel", Label: "R", Type: TypeText, UpdatedAt: time.Now()})

	code, body := f.put(t, "hotel.rule", map[string]any{"value": "Штраф {{smokingFine}}"})
	if code != http.StatusOK || body["value"] != "Штраф {{smokingFine}}" {
		t.Fatalf("put = %d %v", code, body)
	}
	if got := f.textValue(t, "hotel.rule"); got != "Штраф {{smokingFine}}" {
		t.Errorf("stored = %q, want unchanged", got)
	}
	// Spaces inside the braces are allowed and kept as typed; every listed name works.
	typed := "{{ minRoomPrice }} {{lateCheckoutPerHour}} {{breakfastDelivery}}"
	if code, body = f.put(t, "hotel.rule", map[string]any{"value": typed}); code != http.StatusOK || body["value"] != typed {
		t.Errorf("all placeholders: %d %v", code, body)
	}

	code, body = f.put(t, "hotel.rule", map[string]any{"value": "{{price}}"})
	wantFieldError(t, code, body, "value", "неизвестная подстановка {{price}}")
	code, body = f.put(t, "hotel.rule", map[string]any{"value": "{{smokingFine}} и {{ other }}"})
	wantFieldError(t, code, body, "value", "неизвестная подстановка {{other}}")
	if got := f.textValue(t, "hotel.rule"); got != typed {
		t.Errorf("a rejected value changed the row: %q", got)
	}
}

func TestPutImage(t *testing.T) {
	f := newFixture(t)
	f.seed(t)
	oldPath, err := f.uploads.Save(time.Now(), []byte("old"), "jpg")
	if err != nil {
		t.Fatal(err)
	}
	newPath, err := f.uploads.Save(time.Now(), []byte("new"), "jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE content_blocks SET image_url = $1 WHERE key = 'home.photo'`, oldPath); err != nil {
		t.Fatal(err)
	}

	code, body := f.put(t, "home.photo", map[string]any{"image": map[string]any{
		"url": testBase + newPath, "alt": " Новое фото ", "position": "50% 30%"}})
	if code != http.StatusOK {
		t.Fatalf("put = %d %v", code, body)
	}
	wantImage := map[string]any{"url": testBase + newPath, "alt": "Новое фото", "position": "50% 30%"}
	if v, has := body["value"]; !has || v != nil || !reflect.DeepEqual(body["image"], wantImage) {
		t.Errorf("response = %v", body)
	}
	var row Block
	if err := f.db.Get(&row, `SELECT `+columns+` FROM content_blocks WHERE key = 'home.photo'`); err != nil {
		t.Fatal(err)
	}
	if row.ImageURL != newPath || row.ImageAlt != "Новое фото" || row.ImagePosition != "50% 30%" {
		t.Errorf("row = %+v", row)
	}
	if _, err := os.Stat(filepath.Join(f.dir, strings.TrimPrefix(oldPath, "/uploads/"))); !os.IsNotExist(err) {
		t.Errorf("the replaced file still exists (stat error %v)", err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, strings.TrimPrefix(newPath, "/uploads/"))); err != nil {
		t.Errorf("the new file is gone: %v", err)
	}

	// The same photo again keeps its file.
	code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": newPath, "alt": "x"}})
	if code != http.StatusOK {
		t.Fatalf("same photo: %d %v", code, body)
	}
	if _, err := os.Stat(filepath.Join(f.dir, strings.TrimPrefix(newPath, "/uploads/"))); err != nil {
		t.Errorf("the unchanged file was removed: %v", err)
	}

	// A site photo replaces an uploaded one; the shipped file name is not touched.
	code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": "/images/a.jpg"}})
	if code != http.StatusOK {
		t.Fatalf("site photo: %d %v", code, body)
	}
	if _, err := os.Stat(filepath.Join(f.dir, strings.TrimPrefix(newPath, "/uploads/"))); !os.IsNotExist(err) {
		t.Errorf("the replaced upload still exists (stat error %v)", err)
	}

	code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": "https://x.y/a.jpg"}})
	wantFieldError(t, code, body, "image.url", "недопустимый адрес фото")
	if got := f.imageURL(t); got != "/images/a.jpg" {
		t.Errorf("a rejected photo changed the row: %q", got)
	}
}

func (f fixture) imageURL(t *testing.T) string {
	t.Helper()
	var u string
	if err := f.db.Get(&u, `SELECT image_url FROM content_blocks WHERE key = 'home.photo'`); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestPutImageInvalidFields(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	code, body := f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": "  ", "alt": "a"}})
	wantFieldError(t, code, body, "image.url", "поле не должно быть пустым")

	code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{
		"url": "/images/a.jpg", "alt": strings.Repeat("я", 201), "position": "101% 5%"}})
	wantFieldError(t, code, body, "image.alt", "не более 200 символов")
	wantFieldError(t, code, body, "image.position", "формат «50% 30%»")

	code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": "/images/a.jpg", "position": "left"}})
	wantFieldError(t, code, body, "image.position", "формат «50% 30%»")
	if code, body = f.put(t, "home.photo", map[string]any{"image": map[string]any{"url": "/images/a.jpg", "position": "100% 0%"}}); code != http.StatusOK {
		t.Errorf("edge position: %d %v", code, body)
	}
}

func TestPutWrongType(t *testing.T) {
	f := newFixture(t)
	f.seed(t)

	for _, tc := range []struct {
		name, key string
		body      map[string]any
		field     string
	}{
		{"array to text", "home.title", map[string]any{"value": []string{"a"}}, "value"},
		{"null to text", "home.title", map[string]any{"value": nil}, "value"},
		{"no value for text", "home.title", map[string]any{"image": map[string]any{"url": "/images/a.jpg"}}, "value"},
		{"number to text", "home.title", map[string]any{"value": 5}, "value"},
		{"string to lines", "hotel.amenities", map[string]any{"value": "a"}, "value"},
		{"null to lines", "hotel.amenities", map[string]any{"value": nil}, "value"},
		{"number in lines", "hotel.amenities", map[string]any{"value": []any{1}}, "value"},
		{"array to richtext", "hotel.intro", map[string]any{"value": []string{"a"}}, "value"},
		{"string to image", "home.photo", map[string]any{"value": "a"}, "value"},
		{"no image for image", "home.photo", map[string]any{}, "image"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, body := f.put(t, tc.key, tc.body)
			wantFieldError(t, code, body, tc.field, "неверный тип значения")
		})
	}
	if got := f.textValue(t, "home.title"); got != "Отель и ресторан" {
		t.Errorf("a rejected value changed the row: %q", got)
	}
}

func TestPutBadJSON(t *testing.T) {
	f := newFixture(t)
	f.seed(t)
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/content/blocks/home.title", f.admin, "not an object")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Некорректный JSON") {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestPutUnknownKey(t *testing.T) {
	f := newFixture(t)
	f.seed(t)
	code, body := f.put(t, "no.such.key", map[string]any{"value": "x"})
	if code != http.StatusNotFound || body["message"] != "Не найдено" {
		t.Errorf("got %d %v", code, body)
	}
}

func TestPutAuth(t *testing.T) {
	f := newFixture(t)
	f.seed(t)
	body := map[string]any{"value": "x"}

	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/content/blocks/home.title", "", body)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, f.router, http.MethodPut, "/content/blocks/home.title", f.user, body)
	if rec.Code != http.StatusForbidden {
		t.Errorf("user role: %d", rec.Code)
	}
	if got := f.textValue(t, "home.title"); got != "Отель и ресторан" {
		t.Errorf("an unauthorised request changed the row: %q", got)
	}
}

func TestStoreNotFound(t *testing.T) {
	f := newFixture(t)
	s := NewStore(f.db)
	ctx := context.Background()
	if _, err := s.Get(ctx, "no.key"); err != store.ErrNotFound {
		t.Errorf("Get: %v", err)
	}
	if _, err := s.SetText(ctx, "no.key", "x"); err != store.ErrNotFound {
		t.Errorf("SetText: %v", err)
	}
	if _, _, err := s.SetImage(ctx, "no.key", "/images/a.jpg", "", ""); err != store.ErrNotFound {
		t.Errorf("SetImage: %v", err)
	}
}

func TestKeyConstraint(t *testing.T) {
	f := newFixture(t)
	for _, key := range []string{"nodots", "Home.title", "home..title", "home.title "} {
		_, err := f.db.Exec(`INSERT INTO content_blocks (key, page, label, type) VALUES ($1, 'p', 'l', 'text')`, key)
		if !store.IsCheckViolation(err) {
			t.Errorf("key %q: error %v, want a check violation", key, err)
		}
	}
	_, err := f.db.Exec(`INSERT INTO content_blocks (key, page, label, type) VALUES ('a.b-c.d1', 'a', 'l', 'bogus')`)
	if !store.IsCheckViolation(err) {
		t.Errorf("unknown type: error %v, want a check violation", err)
	}
}

func TestValidateValueUnknownType(t *testing.T) {
	if _, err := validateValue(Block{Type: "bogus"}, valueInput{}); err == nil {
		t.Error("an unknown block type was accepted")
	}
}

func TestValidPosition(t *testing.T) {
	for p, want := range map[string]bool{
		"": true, "0% 0%": true, "50% 30%": true, "100% 100%": true,
		"101% 0%": false, "0% 101%": false, "50%30%": false, "50 30": false, "a% b%": false, "1000% 5%": false,
	} {
		if got := validPosition(p); got != want {
			t.Errorf("validPosition(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestPlaceholdersList(t *testing.T) {
	want := []string{"minRoomPrice", "lateCheckoutPerHour", "smokingFine", "breakfastDelivery"}
	if !reflect.DeepEqual(Placeholders, want) {
		t.Errorf("Placeholders = %v", Placeholders)
	}
}
