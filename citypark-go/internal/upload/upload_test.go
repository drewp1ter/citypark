package upload

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

const testBase = "http://localhost:3000"

var october = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// webpBytes is the 26-byte RIFF/WEBP header that http.DetectContentType recognises.
func webpBytes() []byte {
	return []byte("RIFF\x1a\x00\x00\x00WEBPVP8 \x0e\x00\x00\x00\x00\x00\x00\x00\x00\x00")
}

func newStorage(t *testing.T) *Storage {
	t.Helper()
	return NewStorage(t.TempDir(), testBase)
}

func TestSave(t *testing.T) {
	s := newStorage(t)
	data := pngBytes(t)

	path, err := s.Save(october, data, "png")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^/uploads/2026/10/[0-9a-f]{32}\.png$`).MatchString(path) {
		t.Fatalf("path = %q", path)
	}
	got, err := os.ReadFile(filepath.Join(s.dir, filepath.FromSlash(strings.TrimPrefix(path, "/uploads/"))))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Error("stored bytes differ from the saved bytes")
	}

	other, err := s.Save(october, data, "png")
	if err != nil {
		t.Fatal(err)
	}
	if other == path {
		t.Error("two saves returned the same path")
	}
}

func TestSaveSingleDigitMonth(t *testing.T) {
	s := newStorage(t)
	path, err := s.Save(time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC), []byte("x"), "jpg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, "/uploads/2027/03/") {
		t.Errorf("path = %q, want month 03", path)
	}
}

func TestSaveErrors(t *testing.T) {
	// The base directory is a regular file, so MkdirAll must fail.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStorage(file, testBase).Save(october, []byte("x"), "png"); err == nil {
		t.Error("Save into a file path: want an error")
	}

	// The target directory exists but is read-only, so WriteFile must fail.
	dir := t.TempDir()
	month := filepath.Join(dir, "2026", "10")
	if err := os.MkdirAll(month, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(month, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(month, 0o755) })
	if os.Geteuid() != 0 {
		if _, err := NewStorage(dir, testBase).Save(october, []byte("x"), "png"); err == nil {
			t.Error("Save into a read-only directory: want an error")
		}
	}
}

func TestPublic(t *testing.T) {
	s := newStorage(t)
	tests := []struct{ in, want string }{
		{"/uploads/a.jpg", testBase + "/uploads/a.jpg"},
		{"/images/a.jpg", "/images/a.jpg"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := s.Public(tt.in); got != tt.want {
			t.Errorf("Public(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNewStorageTrimsBaseSlash(t *testing.T) {
	s := NewStorage(t.TempDir(), testBase+"/")
	if got := s.Public("/uploads/a.jpg"); got != testBase+"/uploads/a.jpg" {
		t.Errorf("Public = %q", got)
	}
}

func TestStored(t *testing.T) {
	s := newStorage(t)
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"/images/room-vip.jpg", "/images/room-vip.jpg", false},
		{"/uploads/2026/10/x.jpg", "/uploads/2026/10/x.jpg", false},
		{testBase + "/uploads/2026/10/x.jpg", "/uploads/2026/10/x.jpg", false},
		{"https://evil.example/x.jpg", "", true},
		{"https://evil.example/uploads/2026/10/x.jpg", "", true},
		{"/uploads/../etc/passwd", "", true},
		{"/images/../uploads/x.jpg", "", true},
		{`/uploads/a\b.jpg`, "", true},
		{"/images/a.jpg?x", "", true},
		{"/images/a.jpg#top", "", true},
		{"/uploads/2026//x.jpg", "", true},
		{"/uploads/", "", true},
		{"/uploads/2026/", "", true},
		{"/images/", "", true},
		{"/other/a.jpg", "", true},
		{testBase + "/images/a.jpg", "", true},
	}
	for _, tt := range tests {
		got, err := s.Stored(tt.in)
		if tt.wantErr {
			if !errors.Is(err, ErrBadURL) {
				t.Errorf("Stored(%q) error = %v, want ErrBadURL", tt.in, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("Stored(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}

func TestRemove(t *testing.T) {
	s := newStorage(t)

	path, err := s.Save(october, []byte("x"), "png")
	if err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(s.dir, filepath.FromSlash(strings.TrimPrefix(path, "/uploads/")))
	if err := s.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file still exists after Remove: %v", err)
	}

	if err := s.Remove(path); err != nil {
		t.Errorf("Remove of a missing file = %v, want nil", err)
	}

	// A file that must not be touched by a non-upload path.
	keep := filepath.Join(s.dir, "images", "x.jpg")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("/images/x.jpg"); err != nil {
		t.Errorf("Remove(/images/x.jpg) = %v, want nil", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("file under /images was touched: %v", err)
	}

	if err := s.Remove("/uploads/../outside.jpg"); !errors.Is(err, ErrBadURL) {
		t.Errorf("Remove with .. = %v, want ErrBadURL", err)
	}

	// A non-empty directory cannot be removed with os.Remove: the error is returned.
	if err := os.WriteFile(filepath.Join(s.dir, "2026", "10", "f.png"), []byte("x"), 0o644); err == nil {
		if err := s.Remove("/uploads/2026"); err == nil {
			t.Error("Remove of a non-empty directory: want an error")
		}
	}
}

func serve(s *Storage, method, path string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Get("/uploads/*", s.Serve().ServeHTTP)
	r.Head("/uploads/*", s.Serve().ServeHTTP)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestServe(t *testing.T) {
	s := newStorage(t)
	data := pngBytes(t)
	path, err := s.Save(october, data, "png")
	if err != nil {
		t.Fatal(err)
	}

	rec := serve(s, http.MethodGet, path)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Error("served bytes differ from the saved bytes")
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}

	head := serve(s, http.MethodHead, path)
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Errorf("HEAD = %d with %d body bytes, want 200 and none", head.Code, head.Body.Len())
	}

	for _, p := range []string{"/uploads/2026/", "/uploads/2026/10", "/uploads/", "/uploads/2026/10/missing.png"} {
		rec := serve(s, http.MethodGet, p)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", p, rec.Code)
		}
		if rec.Header().Get("Cache-Control") != "" {
			t.Errorf("GET %s: a 404 must not be cached, got %q", p, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestServeRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStorage(filepath.Join(root, "uploads"), testBase)
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Call the handler directly: the router would clean nothing, so ".." reaches Serve as is.
	req := httptest.NewRequest(http.MethodGet, "/uploads/x", nil)
	req.URL.Path = "/uploads/../secret.txt"
	rec := httptest.NewRecorder()
	s.Serve().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("status = %d, body = %q; want 404 without the secret", rec.Code, rec.Body.String())
	}

	req.URL.Path = "/other/x.png"
	rec = httptest.NewRecorder()
	s.Serve().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("path outside /uploads/: status = %d, want 404", rec.Code)
	}
}

// --- handler ---

type fixture struct {
	storage *Storage
	router  chi.Router
	admin   string
	user    string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)))
	admin, err := tokens.Issue(1, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	user, err := tokens.Issue(2, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	s := newStorage(t)
	r := chi.NewRouter()
	NewHandler(s, tokens).Routes(r)
	r.Get("/uploads/*", s.Serve().ServeHTTP)
	return fixture{storage: s, router: r, admin: admin, user: user}
}

// post sends a multipart form with one part named field; field == "" sends a form without parts.
func (f fixture) post(t *testing.T, token, field, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if field != "" {
		part, err := mw.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/uploads", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	var e httpx.Error
	testutil.Decode(t, rec, &e)
	if rec.Code != status || e.Message != message {
		t.Errorf("got %d %q, want %d %q", rec.Code, e.Message, status, message)
	}
}

func TestUploadImages(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name    string
		content []byte
		ext     string
		ctype   string
	}{
		{"png", pngBytes(t), "png", "image/png"},
		{"jpeg", jpegBytes(t), "jpg", "image/jpeg"},
		{"webp", webpBytes(), "webp", "image/webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The client's file name is ignored: the extension comes from the content.
			rec := f.post(t, f.admin, "file", "photo.txt", tt.content)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
			}
			var resp struct{ URL string }
			testutil.Decode(t, rec, &resp)
			re := regexp.MustCompile(`^http://localhost:3000(/uploads/\d{4}/\d{2}/[0-9a-f]{32}\.` + tt.ext + `)$`)
			m := re.FindStringSubmatch(resp.URL)
			if m == nil {
				t.Fatalf("url = %q", resp.URL)
			}

			got := serve(f.storage, http.MethodGet, m[1])
			if got.Code != http.StatusOK || !bytes.Equal(got.Body.Bytes(), tt.content) {
				t.Errorf("GET %s = %d, bytes equal: %v", m[1], got.Code, bytes.Equal(got.Body.Bytes(), tt.content))
			}
			if got.Header().Get("Content-Type") != tt.ctype {
				t.Errorf("Content-Type = %q, want %q", got.Header().Get("Content-Type"), tt.ctype)
			}
		})
	}
}

func TestUploadRejectsOtherTypes(t *testing.T) {
	f := newFixture(t)
	for name, content := range map[string][]byte{
		"text":  []byte("just some text"),
		"gif":   []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"html":  []byte("<html><body>hi</body></html>"),
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			// The name claims JPEG, the content decides.
			rec := f.post(t, f.admin, "file", "photo.jpg", content)
			wantError(t, rec, http.StatusBadRequest, "Допустимы только JPEG, PNG и WebP")
		})
	}
}

func TestUploadNoFile(t *testing.T) {
	f := newFixture(t)
	wantError(t, f.post(t, f.admin, "", "", nil), http.StatusBadRequest, "Выберите файл")
	wantError(t, f.post(t, f.admin, "photo", "a.png", pngBytes(t)), http.StatusBadRequest, "Выберите файл")

	// Not a multipart request at all.
	req := httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"file":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.admin)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	wantError(t, rec, http.StatusBadRequest, "Выберите файл")
}

func TestUploadTooLarge(t *testing.T) {
	f := newFixture(t)
	// 11 MiB is over the body limit: the request is cut off while it is read.
	big := append(pngBytes(t), make([]byte, 11<<20)...)
	wantError(t, f.post(t, f.admin, "file", "big.png", big), http.StatusRequestEntityTooLarge, "Файл больше 10 МБ")

	// Over 10 MiB but within the body limit (10 MiB + 64 KiB): the file size check rejects it.
	edge := append(pngBytes(t), make([]byte, 10<<20+1000)...)
	wantError(t, f.post(t, f.admin, "file", "edge.png", edge), http.StatusRequestEntityTooLarge, "Файл больше 10 МБ")

	// Exactly 10 MiB is accepted.
	ok := append(pngBytes(t), make([]byte, 10<<20)...)[:10<<20]
	if rec := f.post(t, f.admin, "file", "ok.png", ok); rec.Code != http.StatusCreated {
		t.Errorf("10 MiB file: status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestUploadAuth(t *testing.T) {
	f := newFixture(t)
	wantError(t, f.post(t, "", "file", "a.png", pngBytes(t)), http.StatusUnauthorized, "Требуется авторизация")
	wantError(t, f.post(t, f.user, "file", "a.png", pngBytes(t)), http.StatusForbidden, "Недостаточно прав")

	entries, err := os.ReadDir(f.storage.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("a rejected request left %d entries on disk", len(entries))
	}
}

func TestUploadSaveFailure(t *testing.T) {
	f := newFixture(t)
	// Make the storage directory a file so that Save fails.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	f.storage.dir = file
	wantError(t, f.post(t, f.admin, "file", "a.png", pngBytes(t)), http.StatusInternalServerError, "Внутренняя ошибка сервера")
}
