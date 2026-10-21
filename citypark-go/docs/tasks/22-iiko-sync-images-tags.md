# Task 22: iiko sync — images, tags, endpoint

## Goal
Finish the sync: images and tags. Add `POST /iiko/update-products`. Test against the three recorded iiko responses.

## Depends on
21.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `internal/iiko/types.go`.

## Files to change
- `internal/iiko/sync.go`: call the new steps inside the transaction, after `syncProducts`.

## Files to create
- `internal/iiko/sync_extra.go` (images and tags)
- `internal/iiko/handler.go`
- `internal/iiko/fixtures_test.go`, `internal/iiko/handler_test.go`

Never open `internal/iiko/testdata/response-*.json` in an editor: each is over 200 KB. Tests read them with `os.ReadFile` and `json.Unmarshal`.

## Spec

### Images
For categories (kept groups only) and products (kept products only):
1. Collect `url → owner id` from `ImageLinks`, skipping the literal value `IMAGE_UPLOAD_ERROR`. If the same url appears twice, the last owner wins (the `url` column is unique).
2. Upsert each pair:
```sql
INSERT INTO product_images (product_id, url, type) VALUES ($1, $2, 'from_iiko')
ON CONFLICT (url) DO UPDATE SET product_id = EXCLUDED.product_id, type = 'from_iiko', updated_at = now()
```
3. Delete iiko images that are no longer present. `urls` must be a non-nil slice, also when empty:
```sql
DELETE FROM product_images WHERE type = 'from_iiko' AND NOT (url = ANY($1))
```
The same three steps for `category_images` with `category_id`. Images with another `type` are never touched.

### Tags
1. `DELETE FROM product_tags` (all links are rebuilt on every sync).
2. For each kept product and each string in its `Tags`: lower-case it, split with the regular expression `[\s-]+`, remove every `"` and `#` from each piece, trim spaces, drop empty pieces. Each remaining piece is a tag of that product. Example: `"\"Грузинская кухня\""` → `грузинская`, `кухня`.
3. Upsert each distinct tag once and remember its id:
```sql
INSERT INTO tags (tag) VALUES ($1) ON CONFLICT (tag) DO UPDATE SET tag = EXCLUDED.tag RETURNING id
```
4. Insert each distinct (product, tag) pair:
   `INSERT INTO product_tags (product_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`.
5. Remove unused tags:
   `DELETE FROM tags WHERE NOT EXISTS (SELECT 1 FROM product_tags pt WHERE pt.tag_id = tags.id)`.

### `handler.go`
```go
type Handler struct { /* syncer *Syncer, token string */ }

// token is the value of IIKO_UPDATE_TOKEN; "" disables the endpoint.
func NewHandler(s *Syncer, token string) *Handler
func (h *Handler) Routes(r chi.Router)
```
`POST /iiko/update-products`:
1. If the configured token is empty → `httpx.NotFound()`.
2. Compare the header `X-Update-Token` with the token using `subtle.ConstantTimeCompare`; different → `httpx.Unauthorized()`.
3. `syncer.Sync(r.Context())`; an error is returned as is (500).
4. 200 with the `Result`: `{"totalCategories": 2, "totalProducts": 27}`.

## Tests to write

### `fixtures_test.go`
Helper `loadFixture(t, n) *Nomenclature` reading `testdata/response-<n>.json`. One test, `TestSyncFixtures`, with a fake source whose menu is replaced between syncs, marker `#`, and one database for all three steps. After each `Sync`, check with `testutil.Count`:

| After syncing | Categories not deleted | Products not deleted, by category name | `category_images` | `product_images` | `tags` | `product_tags` |
|---|---|---|---|---|---|---|
| response 1 | 2 | `Грузинская кухня` 9, `Мангал` 18 | 1 | 20 | 6 | 52 |
| response 2 | 3 | `Грузинская кухня` 9, `Мангал` 18, `Пицца` 1 | 1 | 20 | 6 | 54 |
| response 3 | 2 | `Грузинская кухня` 0, `Мангал` 18, `Пицца` 1 | 0 | 11 | 4 | 36 |

Also check: after response 1 `Result` is `{2, 27}`; after response 3 the `categories` table still has 3 rows and `products` 28 rows (soft delete), and the 6 tags after response 1 are `грузинская, кухня, мангал, овощи, мясо, рыба`.

Count query for products by category name:
```sql
SELECT count(*) FROM products p JOIN categories c ON c.id = p.category_id
WHERE c.name = $1 AND NOT p.is_deleted
```

### Unit tests (small hand-built menus)
- `TestImagesSkipUploadError`: links `["IMAGE_UPLOAD_ERROR", "https://x/a.jpg"]` → one image.
- `TestImagesKeepManual`: a `product_images` row with `type = 'manual'` survives a sync that has no images.
- `TestImagesDuplicateURL`: two products with the same url → one row, owned by the later product, no error.
- `TestTagsSplit`: tags `["\"Грузинская кухня\"", "#Острое-мясо", "  "]` → tags `грузинская, кухня, острое, мясо`.
- `TestTagsRemovedWhenUnused`: second sync without tags → `tags` and `product_tags` are empty.

### `handler_test.go`
- `TestUpdateDisabled`: token `""` → 404.
- `TestUpdateWrongToken` and `TestUpdateNoHeader`: 401, and the fake source was not called.
- `TestUpdateOK`: right header → 200 and body `{"totalCategories":1,"totalProducts":1}` for a one-group, one-product menu.
- `TestUpdateSourceError`: the source fails → 500.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Scheduling the sync. Changing the fixtures.
