# Task 45: Image uploads

## Goal
An admin uploads a JPEG, PNG or WebP image (`POST /uploads`) and gets its URL back. Files are stored on a local disk
directory and served at `/uploads/...`. Package `upload` also gives the content tasks (48, 49, 50) the helpers
to store photo URLs and to delete replaced files.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/app/app.go`, `internal/config/config.go`, `internal/httpx/httpx.go`.

## Background
Decided by the owner (`../docs/adr/0001-admin-panel.md`): local disk, not S3; the site's `next/image` resizes, so the
server keeps only the original. Uploading is a separate step: the admin uploads, gets a URL, and sends that URL in the
form it saves. Photos that the site ships itself (`/images/room-vip.jpg` in `citypark-modern/public`) stay valid photo
URLs, so seeded content can point at them until someone uploads a new photo.

## Files to create
- `internal/upload/storage.go`, `internal/upload/handler.go`, `internal/upload/upload_test.go`

## Files to change
- `internal/config/config.go` + test: `Upload struct{ Dir, PublicBaseURL string }`, field `Upload Upload`;
  `UPLOAD_DIR` (default `./uploads`), `PUBLIC_BASE_URL` (default `http://localhost:3000`, trailing `/` removed).
- `.env.example`: both, with the comments `# where uploaded images are stored (a persistent volume in production)` and
  `# public origin of this server; uploaded image URLs start with it`.
- `internal/app/app.go`: build `upload.NewStorage(cfg.Upload.Dir, cfg.Upload.PublicBaseURL)`; register
  `upload.NewHandler(storage, tokens).Routes(r)` inside `/api/v1` and `storage.Serve()` at `/uploads/*` **outside**
  `/api/v1`; expose the storage as `App.Uploads` for later tasks.
- `.gitignore`: `uploads/`. `docker-compose.yml` and `Dockerfile`: if they run the server, mount a named volume at
  the `UPLOAD_DIR` used there.
- `docs/API.md`, `internal/apidocs/openapi.yaml`: the endpoint. `TestSpecCoversAllRoutes` may need
  `/uploads/*` excluded the way other non-API routes are; read the test.

## Spec

### Storage
```go
// Storage keeps uploaded images in a directory and builds their public URLs.
type Storage struct{ dir, base string }

func NewStorage(dir, publicBaseURL string) *Storage

// Save writes data under dir/YYYY/MM/<32 hex>.<ext> (time from now, random name from crypto/rand)
// and returns the stored path "/uploads/YYYY/MM/<name>.<ext>".
// allow `time.Now()` here
func (s *Storage) Save(now time.Time, data []byte, ext string) (string, error)

// Public turns a stored path into the URL sent to clients: "/uploads/..." → base + path; anything else unchanged.
func (s *Storage) Public(path string) string

// Stored checks a photo URL received from a client and returns what goes into the database.
// Allowed: "" (no photo), "/images/<file>" (a photo shipped with the site), "/uploads/<path>",
// base + "/uploads/<path>". Anything else → ErrBadURL.
func (s *Storage) Stored(url string) (string, error)

// Remove deletes the file of a stored "/uploads/..." path; other paths are ignored. A missing file is not an error.
// Call it after the database transaction that dropped the reference has committed; log errors, do not return them
// to the API user.
func (s *Storage) Remove(path string) error

// Serve serves dir at /uploads/ (no directory listings, Cache-Control: public, max-age=31536000, immutable).
func (s *Storage) Serve() http.Handler

var ErrBadURL = errors.New("bad photo url")
```
`Stored` rejects `..`, `\`, query strings and `//` inside the path. Content tasks map `ErrBadURL` to a 400 field error
`недопустимый адрес фото`.

### Route
| Method | Path | Role | Behaviour | Success |
|---|---|---|---|---|
| POST | `/uploads` | admin | multipart form, field `file` | 201 |

- Limit the body with `http.MaxBytesReader` to 10 MiB + 64 KiB; `ParseMultipartForm(10 << 20)`. Too large →
  413 `Файл больше 10 МБ`.
- No `file` field → 400 `Выберите файл`.
- Detect the type from the content with `http.DetectContentType`: `image/jpeg` → `jpg`, `image/png` → `png`,
  `image/webp` → `webp`; anything else → 400 `Допустимы только JPEG, PNG и WebP`. The client's file name and
  `Content-Type` are ignored.
- Response: `{"url": "http://localhost:3000/uploads/2026/10/3f…a1.jpg"}` (`Public` of the stored path).

## Tests to write
Use `t.TempDir()` as the directory and fixed small images built in the test (`image/png` and `image/jpeg` encoders
for PNG/JPEG; for WebP use the 26-byte RIFF/WEBP header that `DetectContentType` recognises).
- `TestSave`: PNG bytes → path matches `^/uploads/2026/10/[0-9a-f]{32}\.png$` for `now` in October 2026; the file
  exists with the same bytes.
- `TestPublic`: table: `/uploads/a.jpg` → base + path; `/images/a.jpg` → unchanged; `""` → `""`.
- `TestStored`: table: `""`, `/images/room-vip.jpg`, `/uploads/2026/10/x.jpg`, base + `/uploads/2026/10/x.jpg` → ok
  (the last one becomes the relative path); `https://evil.example/x.jpg`, `/uploads/../etc/passwd`, `/images/a.jpg?x`,
  `/other/a.jpg` → `ErrBadURL`.
- `TestRemove`: saved file → removed; missing file → nil; `/images/x.jpg` → nil and nothing touched.
- `TestServe`: a saved file is served with 200, its bytes and the cache header; `/uploads/2026/` → 404.
- Handler (multipart bodies built with `mime/multipart`; admin token): PNG → 201 and the URL resolves through
  `Serve`; text file → 400 `Допустимы только JPEG, PNG и WebP`; no `file` → 400; 11 MiB → 413; no token → 401;
  role `user` → 403.
- Config: defaults and overrides of both variables.

## API documentation
`docs/API.md`: new area `### Uploads` with `POST /uploads` (role admin; request is `multipart/form-data` with
field `file`; notes: "the URL is what you send as a photo URL to the content endpoints; files that are never used are
not cleaned up"). Mention `GET /uploads/...` in a note (static files, outside `/api/v1`).

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of `internal/upload` and `internal/config` is at least 80%,
and with `make run`:
```
curl -s -F file=@some.jpg -H "Authorization: Bearer $ADMIN_TOKEN" localhost:3000/api/v1/uploads   # {"url": ...}
curl -sI <that url>                                                                                  # 200
```

## Out of scope
S3, resizing, cleaning unused uploads, any content endpoint (tasks 48–50).
