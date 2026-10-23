package room

import (
	"context"
	"errors"
	"maps"
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
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/upload"
)

const testBase = "http://localhost:3000"

var october = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

// dirStorage is an upload.Storage over a temporary directory that remembers the directory,
// so that tests can check which files exist.
type dirStorage struct {
	*upload.Storage
	dir string
}

func newDirStorage(t *testing.T) dirStorage {
	t.Helper()
	dir := t.TempDir()
	return dirStorage{Storage: upload.NewStorage(dir, testBase), dir: dir}
}

type fixture struct {
	db      *sqlx.DB
	uploads dirStorage
	router  http.Handler
	admin   string
	user    string
}

// newFixture returns a router over the migrated database. NewDB does not truncate `rooms` (the seed is part of
// the schema), so the rows are put back after the test.
func newFixture(t *testing.T) fixture {
	t.Helper()
	db := testutil.NewDB(t)
	restoreRooms(t, db)

	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(october))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	uploads := newDirStorage(t)
	r := chi.NewRouter()
	NewHandler(NewStore(db), uploads.Storage, tokens).Routes(r)
	return fixture{db: db, uploads: uploads, router: r, admin: admin, user: user}
}

func restoreRooms(t *testing.T, db *sqlx.DB) {
	t.Helper()
	before, err := NewStore(db).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, r := range before {
			_, err := db.Exec(`UPDATE rooms SET name = $2, hotel_name = $3, numbers = $4, area = $5, area_detailed = $6,
				capacity = $7, description = $8, price_kopecks = $9, photo_url = $10, photo_alt = $11,
				photo_position = $12, hotel_note = $13, updated_at = $14, summary = $15, gallery = $16 WHERE id = $1`,
				r.ID, r.Name, r.HotelName, r.Numbers, r.Area, r.AreaDetailed, r.Capacity, r.Description,
				int64(r.Price), r.PhotoURL, r.PhotoAlt, r.PhotoPosition, r.HotelNote, r.UpdatedAt, r.Summary, r.Gallery)
			if err != nil {
				t.Errorf("restore room %s: %v", r.ID, err)
			}
		}
	})
}

func (f fixture) do(t *testing.T, method, path, token string, body any) (int, string) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, method, path, token, body)
	return rec.Code, rec.Body.String()
}

func (f fixture) put(t *testing.T, id string, body any) (int, response, httpx.Error) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodPut, "/rooms/"+id, f.admin, body)
	var ok response
	var bad httpx.Error
	if rec.Code == http.StatusOK {
		testutil.Decode(t, rec, &ok)
	} else {
		testutil.Decode(t, rec, &bad)
	}
	return rec.Code, ok, bad
}

func (f fixture) list(t *testing.T) []response {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, http.MethodGet, "/rooms", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /rooms = %d, body %s", rec.Code, rec.Body)
	}
	var out []response
	testutil.Decode(t, rec, &out)
	return out
}

func validBody() map[string]any {
	return map[string]any{
		"name": "Новое имя", "hotelName": "Для отеля", "numbers": "№ 1, 2", "area": "30 м²",
		"areaDetailed": "30 м² + балкон", "capacity": "до 3 гостей",
		"description":   "<p>Описание</p>",
		"pricePerNight": 5200,
		"photo":         map[string]any{"url": "/images/room-single.jpg", "alt": "Номер", "position": "50% 30%"},
		"hotelNote":     "№ 1, 2 · кровать",
	}
}

func withField(body map[string]any, key string, value any) map[string]any {
	out := maps.Clone(body)
	out[key] = value
	return out
}

func withPhoto(body map[string]any, key string, value any) map[string]any {
	photo := maps.Clone(body["photo"].(map[string]any))
	photo[key] = value
	return withField(body, "photo", photo)
}

func TestSeed(t *testing.T) {
	f := newFixture(t)
	rooms := f.list(t)

	var ids []string
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}
	if got, want := strings.Join(ids, ","), "vip,single-balcony,single,double-twin,double-euro"; got != want {
		t.Fatalf("room ids = %s, want %s", got, want)
	}
	vip := rooms[0]
	if vip.PricePerNight.Rubles() != 8000 {
		t.Errorf("vip pricePerNight = %v, want 8000", vip.PricePerNight)
	}
	if !strings.HasPrefix(vip.Description, "<p>") || !strings.HasSuffix(vip.Description, "</p>") {
		t.Errorf("vip description = %q, want it wrapped in <p>", vip.Description)
	}
	if vip.Kind != "vip" || !vip.Balcony || vip.Numbers != "№ 301" || vip.Photo.URL != "/images/room-vip.jpg" {
		t.Errorf("vip = %+v", vip)
	}
	// Descriptions were filled from the old site by migration 00011 (they were empty before).
	if !strings.HasPrefix(rooms[1].Description, "<p>Одноместный номер") || rooms[2].Photo.Position != "right center" {
		t.Errorf("seed rows: single-balcony description %q, single position %q", rooms[1].Description, rooms[2].Photo.Position)
	}
	for i, price := range []float64{8000, 4500, 4000, 4500, 4500} {
		if rooms[i].PricePerNight.Rubles() != price {
			t.Errorf("room %s price = %v, want %v", rooms[i].ID, rooms[i].PricePerNight, price)
		}
	}
}

func TestUpdateRoom(t *testing.T) {
	f := newFixture(t)
	before := f.list(t)[3] // double-twin

	code, got, _ := f.put(t, "double-twin", validBody())
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	want := response{
		ID: "double-twin", Kind: before.Kind, Balcony: before.Balcony,
		Name: "Новое имя", HotelName: "Для отеля", Numbers: "№ 1, 2", Area: "30 м²",
		AreaDetailed: "30 м² + балкон", Capacity: "до 3 гостей", Description: "<p>Описание</p>",
		PricePerNight: 520000,
		Photo:         photoResponse{URL: "/images/room-single.jpg", Alt: "Номер", Position: "50% 30%"},
		Gallery:       []galleryResponse{},
		HotelNote:     "№ 1, 2 · кровать",
	}
	got.UpdatedAt = time.Time{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PUT response\n got %+v\nwant %+v", got, want)
	}

	rooms := f.list(t)
	again := rooms[3]
	if again.ID != "double-twin" || again.UpdatedAt.IsZero() || !again.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updatedAt not refreshed: before %v, after %v", before.UpdatedAt, again.UpdatedAt)
	}
	again.UpdatedAt = time.Time{}
	if !reflect.DeepEqual(again, want) {
		t.Errorf("GET after PUT\n got %+v\nwant %+v", again, want)
	}

	var sort int
	if err := f.db.Get(&sort, `SELECT sort FROM rooms WHERE id = 'double-twin'`); err != nil || sort != 4 {
		t.Errorf("sort = %d, err %v; want 4", sort, err)
	}
	if rooms[0].Name != "VIP-номер" {
		t.Errorf("another room changed: %+v", rooms[0])
	}
}

func TestUpdateRoomTrimsAndKeepsKopecks(t *testing.T) {
	f := newFixture(t)
	body := withField(withField(validBody(), "name", "  Номер  "), "pricePerNight", 4500.5)
	body = withPhoto(body, "alt", "  Альт ")

	code, got, _ := f.put(t, "single", body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if got.Name != "Номер" || got.Photo.Alt != "Альт" || got.PricePerNight != 450050 {
		t.Errorf("response = %+v", got)
	}
	var kopecks int64
	if err := f.db.Get(&kopecks, `SELECT price_kopecks FROM rooms WHERE id = 'single'`); err != nil || kopecks != 450050 {
		t.Errorf("price_kopecks = %d, err %v; want 450050", kopecks, err)
	}
}

func TestUpdateRoomClearsOptionalFields(t *testing.T) {
	f := newFixture(t)
	body := withPhoto(validBody(), "position", "")
	for _, k := range []string{"hotelName", "areaDetailed", "capacity", "description", "hotelNote"} {
		body = withField(body, k, "")
	}
	code, got, _ := f.put(t, "vip", body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if got.HotelName != "" || got.Capacity != "" || got.Description != "" || got.Photo.Position != "" || got.HotelNote != "" {
		t.Errorf("optional fields not cleared: %+v", got)
	}
}

func TestUpdateRoomSanitizes(t *testing.T) {
	f := newFixture(t)
	body := withField(validBody(), "description", `<h2>x</h2><p onclick="a">y</p>`)

	code, got, _ := f.put(t, "vip", body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if got.Description != "x<p>y</p>" {
		t.Errorf("response description = %q, want %q", got.Description, "x<p>y</p>")
	}
	var stored string
	if err := f.db.Get(&stored, `SELECT description FROM rooms WHERE id = 'vip'`); err != nil || stored != "x<p>y</p>" {
		t.Errorf("stored description = %q, err %v", stored, err)
	}
}

func TestUpdateRoomDescriptionLimitCountsVisibleText(t *testing.T) {
	f := newFixture(t)
	// 1000 visible letters inside markup: the tags do not count.
	body := withField(validBody(), "description", "<p><strong>"+strings.Repeat("а", 1000)+"</strong></p>")
	if code, _, bad := f.put(t, "vip", body); code != http.StatusOK {
		t.Fatalf("1000 letters: status = %d, %+v", code, bad)
	}
}

func TestUpdateRoomPhotoReplaced(t *testing.T) {
	f := newFixture(t)
	save := func() string {
		t.Helper()
		p, err := f.uploads.Save(october, []byte("jpeg"), "jpg")
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	exists := func(p string) bool {
		_, err := os.Stat(filepath.Join(f.uploads.dir, filepath.FromSlash(strings.TrimPrefix(p, "/uploads/"))))
		return err == nil
	}
	a, b := save(), save()

	if code, _, _ := f.put(t, "vip", withPhoto(validBody(), "url", a)); code != http.StatusOK {
		t.Fatalf("PUT a: status = %d", code)
	}
	if !exists(a) {
		t.Fatal("the current photo file is gone")
	}

	if code, got, _ := f.put(t, "vip", withPhoto(validBody(), "url", b)); code != http.StatusOK || got.Photo.URL != testBase+b {
		t.Fatalf("PUT b: status = %d, photo url %q, want %q", code, got.Photo.URL, testBase+b)
	}
	if exists(a) {
		t.Error("the replaced photo file was not removed")
	}
	if !exists(b) {
		t.Fatal("the new photo file was removed")
	}

	// The same URL (here in its public form) keeps the file.
	if code, _, _ := f.put(t, "vip", withPhoto(validBody(), "url", testBase+b)); code != http.StatusOK {
		t.Fatalf("PUT b again: status = %d", code)
	}
	if !exists(b) {
		t.Error("the file was removed although the photo did not change")
	}
	var stored string
	if err := f.db.Get(&stored, `SELECT photo_url FROM rooms WHERE id = 'vip'`); err != nil || stored != b {
		t.Errorf("stored photo_url = %q, err %v; want %q", stored, err, b)
	}

	// A site image replaces an upload: the upload goes, nothing is deleted for the site image.
	if code, _, _ := f.put(t, "vip", withPhoto(validBody(), "url", "/images/room-vip.jpg")); code != http.StatusOK {
		t.Fatalf("PUT site image: status = %d", code)
	}
	if exists(b) {
		t.Error("the upload was not removed when a site image replaced it")
	}
}

func TestUpdateRoomInvalid(t *testing.T) {
	long := strings.Repeat("а", 101)
	tests := []struct {
		name  string
		body  map[string]any
		field string
		msg   string // "" = only the field is checked
	}{
		{"empty name", withField(validBody(), "name", ""), "name", "поле не должно быть пустым"},
		{"blank name", withField(validBody(), "name", "   "), "name", "поле не должно быть пустым"},
		{"long name", withField(validBody(), "name", long), "name", "значение должно быть не больше 100"},
		{"empty numbers", withField(validBody(), "numbers", ""), "numbers", "поле не должно быть пустым"},
		{"empty area", withField(validBody(), "area", ""), "area", "поле не должно быть пустым"},
		{"long hotelName", withField(validBody(), "hotelName", long), "hotelName", ""},
		{"long areaDetailed", withField(validBody(), "areaDetailed", long), "areaDetailed", ""},
		{"long capacity", withField(validBody(), "capacity", long), "capacity", ""},
		{"long hotelNote", withField(validBody(), "hotelNote", long), "hotelNote", ""},
		{"price 0", withField(validBody(), "pricePerNight", 0), "pricePerNight", ""},
		{"price negative", withField(validBody(), "pricePerNight", -5), "pricePerNight", ""},
		{"price too high", withField(validBody(), "pricePerNight", 1000001), "pricePerNight", ""},
		{"price 3 decimals", withField(validBody(), "pricePerNight", 10.555), "pricePerNight", "не более 2 знаков после запятой"},
		{"no price", withField(validBody(), "pricePerNight", nil), "pricePerNight", ""},
		{"foreign photo url", withPhoto(validBody(), "url", "https://evil.example/a.jpg"), "photo.url", "недопустимый адрес фото"},
		{"traversal photo url", withPhoto(validBody(), "url", "/uploads/../secret"), "photo.url", "недопустимый адрес фото"},
		{"empty photo url", withPhoto(validBody(), "url", ""), "photo.url", "поле не должно быть пустым"},
		{"no photo", withField(validBody(), "photo", nil), "photo.url", "поле не должно быть пустым"},
		{"long photo alt", withPhoto(validBody(), "alt", strings.Repeat("а", 201)), "photo.alt", ""},
		{"position word", withPhoto(validBody(), "position", "top"), "photo.position", "формат «50% 30%»"},
		{"position without percent", withPhoto(validBody(), "position", "50 30"), "photo.position", "формат «50% 30%»"},
		{"position over 100", withPhoto(validBody(), "position", "50% 101%"), "photo.position", "формат «50% 30%»"},
		{"position too long", withPhoto(validBody(), "position", strings.Repeat("5", 101)), "photo.position", ""},
		{"long description", withField(validBody(), "description", strings.Repeat("а", 1001)), "description", "не более 1000 символов"},
	}
	f := newFixture(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, bad := f.put(t, "vip", tt.body)
			if code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", code)
			}
			got, ok := bad.Fields[tt.field]
			if !ok {
				t.Fatalf("errors = %v, want field %q", bad.Fields, tt.field)
			}
			if tt.msg != "" && got != tt.msg {
				t.Errorf("errors[%s] = %q, want %q", tt.field, got, tt.msg)
			}
			if bad.Message != "Ошибка валидации, пожалуйста проверьте введенные вами данные" {
				t.Errorf("message = %q", bad.Message)
			}
		})
	}

	var name string
	if err := f.db.Get(&name, `SELECT name FROM rooms WHERE id = 'vip'`); err != nil || name != "VIP-номер" {
		t.Errorf("a rejected request changed the room: name = %q, err %v", name, err)
	}
}

func TestUpdateRoomReportsAllFields(t *testing.T) {
	f := newFixture(t)
	body := withPhoto(withPhoto(withField(validBody(), "name", ""), "url", "https://evil.example/a.jpg"), "position", "top")
	body = withField(body, "description", strings.Repeat("а", 1001))

	code, _, bad := f.put(t, "vip", body)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	for _, field := range []string{"name", "photo.url", "photo.position", "description"} {
		if _, ok := bad.Fields[field]; !ok {
			t.Errorf("errors = %v, missing %q", bad.Fields, field)
		}
	}
}

func TestUpdateRoomMalformedJSON(t *testing.T) {
	f := newFixture(t)
	code, body := f.do(t, http.MethodPut, "/rooms/vip", f.admin, "{not json")
	if code != http.StatusBadRequest || !strings.Contains(body, "Некорректный JSON") {
		t.Errorf("status = %d, body %s; want 400 Некорректный JSON", code, body)
	}
}

func TestUpdateRoomNotFound(t *testing.T) {
	f := newFixture(t)
	code, _, bad := f.put(t, "penthouse", validBody())
	if code != http.StatusNotFound || bad.Message != "Не найдено" {
		t.Errorf("status = %d, message %q; want 404 Не найдено", code, bad.Message)
	}
}

func TestUpdateRoomAuth(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"garbage token", "nope", http.StatusUnauthorized},
		{"user role", f.user, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if code, body := f.do(t, http.MethodPut, "/rooms/vip", tt.token, validBody()); code != tt.want {
				t.Errorf("status = %d, want %d, body %s", code, tt.want, body)
			}
		})
	}
	if rooms := f.list(t); rooms[0].Name != "VIP-номер" {
		t.Errorf("a rejected request changed the room: %+v", rooms[0])
	}
}

func TestListIsPublic(t *testing.T) {
	f := newFixture(t)
	if code, _ := f.do(t, http.MethodGet, "/rooms", "", nil); code != http.StatusOK {
		t.Errorf("GET /rooms without a token = %d, want 200", code)
	}
}

func TestPublicPhotoURL(t *testing.T) {
	f := newFixture(t)
	path, err := f.uploads.Save(october, []byte("jpeg"), "jpg")
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := f.put(t, "single", withPhoto(validBody(), "url", path)); code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if got := f.list(t)[2].Photo.URL; got != testBase+path {
		t.Errorf("GET photo.url = %q, want %q", got, testBase+path)
	}
}

func TestStoreGet(t *testing.T) {
	db := testutil.NewDB(t)
	s := NewStore(db)
	ctx := context.Background()

	r, err := s.Get(ctx, "single-balcony")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "Одноместный с балконом" || r.Sort != 2 || r.Kind != "single" || !r.Balcony || r.Price != 450000 {
		t.Errorf("Get = %+v", r)
	}
	if _, err := s.Get(ctx, "penthouse"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Get unknown id: err = %v, want ErrNotFound", err)
	}
}

func TestStoreUpdateJoinsTransaction(t *testing.T) {
	db := testutil.NewDB(t)
	restoreRooms(t, db)
	ctx := context.Background()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	current, err := NewStore(db).Get(ctx, "vip")
	if err != nil {
		t.Fatal(err)
	}
	current.Name = "В транзакции"
	current.PhotoURL = "/images/other.jpg"
	saved, old, err := NewStore(tx).Update(ctx, current)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "В транзакции" || saved.PhotoURL != "/images/other.jpg" || old.PhotoURL != "/images/room-vip.jpg" || len(old.Gallery) != 3 {
		t.Errorf("Update = %+v, old photo %q", saved, old)
	}
	if _, _, err := NewStore(tx).Update(ctx, Room{ID: "penthouse", Price: 100}); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Update unknown id in a tx: err = %v, want ErrNotFound", err)
	}
	_ = tx.Rollback()

	after, err := NewStore(db).Get(ctx, "vip")
	if err != nil {
		t.Fatal(err)
	}
	if after.Name != "VIP-номер" {
		t.Errorf("the update escaped a rolled back transaction: name = %q", after.Name)
	}
}

func TestStoreUpdateNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	_, old, err := NewStore(db).Update(context.Background(), Room{ID: "penthouse", Price: 100})
	if !errors.Is(err, store.ErrNotFound) || old.PhotoURL != "" || old.Gallery != nil {
		t.Errorf("Update unknown id: old %q, err %v; want ErrNotFound", old, err)
	}
}

func TestHasKopecksPrecision(t *testing.T) {
	tests := []struct {
		in   float64
		want bool
	}{
		{8000, true}, {4500.5, true}, {4500.55, true}, {0.01, true}, {19.99, true}, {1000000, true},
		{10.555, false}, {0.001, false}, {4500.501, false},
	}
	for _, tt := range tests {
		if got := hasKopecksPrecision(tt.in); got != tt.want {
			t.Errorf("hasKopecksPrecision(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestValidPosition(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true}, {"50% 30%", true}, {"0% 0%", true}, {"100% 100%", true}, {"5% 7%", true},
		{"101% 50%", false}, {"50% 101%", false}, {"1000% 5%", false}, {"top", false},
		{"center 70%", false}, {"right center", false}, {"50%30%", false}, {"50% 30", false}, {" 50% 30%", false},
	}
	for _, tt := range tests {
		if got := validPosition(tt.in); got != tt.want {
			t.Errorf("validPosition(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

const vipSummary = "Гостиная и спальня, расширенная комплектация, балкон над кронами парка."

func TestRoomSummarySeed(t *testing.T) {
	f := newFixture(t)
	byID := map[string]response{}
	for _, r := range f.list(t) {
		byID[r.ID] = r
	}
	if got := byID["vip"].Summary; got != vipSummary {
		t.Errorf("vip summary = %q, want %q", got, vipSummary)
	}
	if got := byID["single"].Summary; got != "" {
		t.Errorf("single summary = %q, want empty", got)
	}
}

func TestUpdateRoomSummary(t *testing.T) {
	f := newFixture(t)
	code, got, _ := f.put(t, "vip", withField(validBody(), "summary", "Кратко о номере"))
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	if got.Summary != "Кратко о номере" {
		t.Errorf("response summary = %q", got.Summary)
	}
	var stored string
	if err := f.db.Get(&stored, `SELECT summary FROM rooms WHERE id = 'vip'`); err != nil || stored != "Кратко о номере" {
		t.Errorf("stored summary = %q, err %v", stored, err)
	}
}

func TestUpdateRoomSummaryTooLong(t *testing.T) {
	f := newFixture(t)
	// 200 Cyrillic letters are 400 bytes: this passes only if the validator counts runes.
	code, _, _ := f.put(t, "vip", withField(validBody(), "summary", strings.Repeat("я", 200)))
	if code != http.StatusOK {
		t.Fatalf("200 characters: status = %d, want 200", code)
	}
	if _, err := f.db.Exec(`UPDATE rooms SET summary = $1 WHERE id = 'vip'`, vipSummary); err != nil {
		t.Fatal(err)
	}

	code, _, bad := f.put(t, "vip", withField(validBody(), "summary", strings.Repeat("я", 201)))
	if code != http.StatusBadRequest {
		t.Fatalf("201 characters: status = %d, want 400", code)
	}
	if _, ok := bad.Fields["summary"]; !ok {
		t.Errorf("errors = %v, want field summary", bad.Fields)
	}
	var stored string
	if err := f.db.Get(&stored, `SELECT summary FROM rooms WHERE id = 'vip'`); err != nil || stored != vipSummary {
		t.Errorf("a rejected request changed the summary: %q, err %v", stored, err)
	}
}

func TestUpdateRoomSummaryMissing(t *testing.T) {
	f := newFixture(t)
	code, got, _ := f.put(t, "vip", validBody())
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	var stored string
	if err := f.db.Get(&stored, `SELECT summary FROM rooms WHERE id = 'vip'`); err != nil || stored != "" || got.Summary != "" {
		t.Errorf("stored summary = %q, response %q, err %v; want empty", stored, got.Summary, err)
	}
}

func uploadFile(t *testing.T, f fixture) string {
	t.Helper()
	p, err := f.uploads.Save(october, []byte("jpeg"), "jpg")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func fileExists(f fixture, p string) bool {
	_, err := os.Stat(filepath.Join(f.uploads.dir, filepath.FromSlash(strings.TrimPrefix(p, "/uploads/"))))
	return err == nil
}

func gallery(items ...[2]string) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{"url": it[0], "alt": it[1]})
	}
	return out
}

func storedGallery(t *testing.T, f fixture, id string) string {
	t.Helper()
	var raw string
	if err := f.db.Get(&raw, `SELECT gallery::text FROM rooms WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRoomGallerySeed(t *testing.T) {
	f := newFixture(t)
	byID := map[string]response{}
	for _, r := range f.list(t) {
		byID[r.ID] = r
	}
	want := []galleryResponse{
		{"/images/room-vip-living.jpg", "VIP-номер: гостиная"},
		{"/images/room-vip-bathroom.jpg", "VIP-номер: санузел"},
		{"/images/room-vip-balcony.jpg", "VIP-номер: балкон"},
	}
	if got := byID["vip"].Gallery; !reflect.DeepEqual(got, want) {
		t.Errorf("vip gallery = %+v, want %+v", got, want)
	}
	if got := byID["single"].Gallery; got == nil || len(got) != 0 {
		t.Errorf("single gallery = %#v, want empty non-nil", got)
	}
	_, body := f.do(t, http.MethodGet, "/rooms", "", nil)
	if strings.Count(body, `"gallery":[]`) != 4 || strings.Contains(body, `"gallery":null`) {
		t.Errorf("raw JSON must have 4 empty galleries and no null: %s", body)
	}
}

func TestUpdateRoomGallery(t *testing.T) {
	f := newFixture(t)
	a, b := uploadFile(t, f), uploadFile(t, f)
	body := withField(validBody(), "gallery", gallery([2]string{b, "  Второе "}, [2]string{"/images/room-vip.jpg", "Первое"}))

	code, got, _ := f.put(t, "single", body)
	if code != http.StatusOK {
		t.Fatalf("PUT status = %d", code)
	}
	want := []galleryResponse{{testBase + b, "Второе"}, {"/images/room-vip.jpg", "Первое"}}
	if !reflect.DeepEqual(got.Gallery, want) {
		t.Errorf("response gallery = %+v, want %+v", got.Gallery, want)
	}
	wantRaw := `[{"alt": "Второе", "url": "` + b + `"}, {"alt": "Первое", "url": "/images/room-vip.jpg"}]`
	if raw := storedGallery(t, f, "single"); raw != wantRaw {
		t.Errorf("stored gallery = %s, want %s", raw, wantRaw)
	}
	if again := f.list(t)[2].Gallery; !reflect.DeepEqual(again, want) {
		t.Errorf("GET gallery = %+v, want %+v", again, want)
	}
	if !fileExists(f, a) {
		t.Error("an unrelated upload was removed")
	}
}

func TestUpdateRoomGalleryMissing(t *testing.T) {
	f := newFixture(t)
	for _, g := range []any{nil, "missing"} {
		body := validBody()
		if g != "missing" {
			body["gallery"] = g
		}
		code, got, _ := f.put(t, "vip", body)
		if code != http.StatusOK {
			t.Fatalf("PUT status = %d", code)
		}
		if got.Gallery == nil || len(got.Gallery) != 0 {
			t.Errorf("response gallery = %#v, want empty non-nil", got.Gallery)
		}
		if raw := storedGallery(t, f, "vip"); raw != "[]" {
			t.Errorf("stored gallery = %s, want []", raw)
		}
	}
}

func TestUpdateRoomGalleryTooMany(t *testing.T) {
	f := newFixture(t)
	items := make([][2]string, 13)
	for i := range items {
		items[i] = [2]string{"/images/room-vip.jpg", "x"}
	}
	code, _, bad := f.put(t, "vip", withField(validBody(), "gallery", gallery(items...)))
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	if _, ok := bad.Fields["gallery"]; !ok {
		t.Errorf("errors = %v, want field gallery", bad.Fields)
	}
	if n := len(f.list(t)[0].Gallery); n != 3 {
		t.Errorf("a rejected request changed the gallery: %d items", n)
	}

	items = items[:12]
	if code, _, bad := f.put(t, "vip", withField(validBody(), "gallery", gallery(items...))); code != http.StatusOK {
		t.Errorf("12 items: status = %d, %+v", code, bad)
	}
}

func TestUpdateRoomGalleryBadURL(t *testing.T) {
	f := newFixture(t)
	body := withField(validBody(), "gallery", gallery(
		[2]string{"https://evil.example/a.jpg", "x"},
		[2]string{"/images/room-vip.jpg", strings.Repeat("я", 201)},
		[2]string{"", "x"},
		[2]string{"/images/ok.jpg", "x"},
	))
	code, _, bad := f.put(t, "vip", body)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	want := map[string]string{
		"gallery[0].url": "недопустимый адрес фото",
		"gallery[2].url": "поле не должно быть пустым",
	}
	for k, v := range want {
		if bad.Fields[k] != v {
			t.Errorf("errors[%s] = %q, want %q (all: %v)", k, bad.Fields[k], v, bad.Fields)
		}
	}
	if _, ok := bad.Fields["gallery[1].alt"]; !ok {
		t.Errorf("errors = %v, want gallery[1].alt", bad.Fields)
	}
	if _, ok := bad.Fields["gallery[3].url"]; ok {
		t.Errorf("valid item reported: %v", bad.Fields)
	}
	if n := len(f.list(t)[0].Gallery); n != 3 {
		t.Errorf("a rejected request changed the gallery: %d items", n)
	}
}

func TestUpdateRoomGalleryRemovesFiles(t *testing.T) {
	f := newFixture(t)
	a, b := uploadFile(t, f), uploadFile(t, f)

	if code, _, bad := f.put(t, "vip", withField(validBody(), "gallery", gallery([2]string{a, ""}, [2]string{b, ""}))); code != http.StatusOK {
		t.Fatalf("PUT [a, b]: status = %d, %+v", code, bad)
	}
	if !fileExists(f, a) || !fileExists(f, b) {
		t.Fatal("gallery files are gone after saving them")
	}

	if code, _, _ := f.put(t, "vip", withField(validBody(), "gallery", gallery([2]string{b, ""}))); code != http.StatusOK {
		t.Fatalf("PUT [b]: status = %d", code)
	}
	if fileExists(f, a) {
		t.Error("the file dropped from the gallery was not removed")
	}
	if !fileExists(f, b) {
		t.Fatal("the file kept in the gallery was removed")
	}

	body := withField(withPhoto(validBody(), "url", b), "gallery", gallery())
	if code, _, _ := f.put(t, "vip", body); code != http.StatusOK {
		t.Fatalf("PUT photo = b: status = %d", code)
	}
	if !fileExists(f, b) {
		t.Error("the file moved from the gallery to the photo was removed")
	}

	// The old photo upload goes when the gallery takes a site image instead, and a site path is never removed.
	body = withField(validBody(), "gallery", gallery([2]string{"/images/room-vip.jpg", ""}))
	if code, _, _ := f.put(t, "vip", body); code != http.StatusOK {
		t.Fatalf("PUT site image: status = %d", code)
	}
	if fileExists(f, b) {
		t.Error("the old photo upload was not removed")
	}
}

func TestGalleryValue(t *testing.T) {
	var nilGallery Gallery
	v, err := nilGallery.Value()
	if err != nil || v != "[]" {
		t.Errorf("nil Value = %v, %v; want []", v, err)
	}
	v, err = Gallery{{URL: "/a", Alt: "б"}}.Value()
	if err != nil || v != `[{"url":"/a","alt":"б"}]` {
		t.Errorf("Value = %v, %v", v, err)
	}
}

func TestGalleryScan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		want    Gallery
		wantErr bool
	}{
		{"bytes", []byte(`[{"url":"/a","alt":"x"}]`), Gallery{{"/a", "x"}}, false},
		{"string", `[{"url":"/a","alt":"x"}]`, Gallery{{"/a", "x"}}, false},
		{"json null", []byte(`null`), Gallery{}, false},
		{"sql null", nil, Gallery{}, false},
		{"bad json", []byte(`{`), nil, true},
		{"bad type", 42, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Gallery
			err := g.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(g, tt.want) {
				t.Errorf("Scan = %#v, want %#v", g, tt.want)
			}
		})
	}
}
