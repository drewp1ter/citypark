# Task 97: Structured customer address-book API

## Goal
Expose saved-entry CRUD/default operations with explicit ownership, validated structured fields and documented responses.

## Depends on
Backend 96c. Feature release also requires 98.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/address/book_service.go`
- `internal/customer/session.go`
- `internal/app/app.go`

## Files to create
- `internal/address/book_handler.go`
- `internal/address/book_handler_test.go`
- `internal/app/customer_addresses_test.go`

## Files to change
- `internal/app/app.go` — register new handler.
- `docs/API.md`, `internal/apidocs/openapi.yaml` — all five routes and models.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
```go
type BookHandler struct { /* injected store, service, access middleware */ }
func NewBookHandler(s *Store, service *Service, access func(http.Handler) http.Handler) *BookHandler
func (h *BookHandler) Routes(r chi.Router)
```
Wire `adapter.Required(false)`; additionally require auth.Identity.Role == user in the handler group (403 for authenticated non-user reaching it). Do not alter the adapter. With customer auth enabled, web and Mini App cs_ sessions work, revoked/expired/legacy customer JWT and staff JWT return the adapter's 401. With it disabled, legacy user JWT works; cs_ returns 503. Derive owner only from auth.FromContext. All outcomes, including middleware errors, are `Cache-Control: private, no-store`.

| Route | Success | Behavior |
| --- | --- | --- |
| GET `/customer/addresses` | 200 | `{ "addresses": [] }`, ascending saved ID |
| POST `/customer/addresses` | 201 new / 200 duplicate | `{ "entry": Entry, "created": true/false }` |
| PUT `/customer/addresses/{id}` | 200 | `{ "entry": Entry }`; replace editable fields, preserve ID/default |
| PUT `/customer/addresses/{id}/default` | 200 | `{ "entry": Entry }`; no request body required |
| DELETE `/customer/addresses/{id}` | 204 | no body; absent/foreign entry 404 |

POST/PUT input (all optional fields default to empty strings):
```json
{"name":"Дом","city":"Белореченск","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3"}
```
Do not accept default changes through this body; default is a separate operation. Strictly allow only these five fields (reject unknown fields, including userId/id/isDefault), reject null/scalar/array/multiple JSON documents and non-string fields. Enforce the same 1 MiB body bound as httpx: read through http.MaxBytesReader, reject a read error or json.Valid=false (including trailing documents), reset r.Body to that bounded buffer, then use httpx.DecodeJSON into an object of json.RawMessage values and validate keys/types before building Fields. This validates the complete document without a second decoder losing buffered trailing bytes. Do not modify shared decoding for unrelated APIs. Use validate.Struct on cleaned Fields; return task 96 FieldErrors verbatim instead of inconsistent generic field translations.

Entry schema example (timestamps illustrative; documentation examples must be captured from tests):
```json
{"id":7,"name":"Дом","city":"Белореченск","address":"ул. Гоголя, 61, кв. 12, подъезд 2, этаж 3","street":"ул. Гоголя, 61","apartment":"12","entrance":"2, этаж 3","isDefault":true,"needsReview":false,"createdAt":"2026-11-05T12:00:00Z","updatedAt":"2026-11-05T12:00:00Z"}
```
Do not expose AddressID, userID or credentials. Legacy entries have street="", needsReview=true, full address unchanged.

Error contract:
- 400 malformed/type/unknown-field/null/extra-document/body-too-large: `Некорректный JSON`.
- 400 invalid fields: `Проверьте адрес`, with `errors` from FieldErrors.
- 401: `Требуется авторизация`; 403: `Недостаточно прав`.
- 404 invalid/nonpositive/overflow ID or missing/foreign entry: `Не найдено` (httpx.PathID behavior).
- 409 edit collision: `Такой адрес уже сохранён`, `errors.street` with the same text.
- 503 disabled customer session flow: `Вход временно недоступен` (adapter behavior).
- 500 unexpected failure: `Внутренняя ошибка сервера`; error/log capture must not include submitted fields.

Old `/addresses` routes retain their status/schema/shared IDs. Map ErrOwnerUnavailable after an authenticated account merge to 401, not a guest fallback. Map store.ErrNotFound for missing-entry operations to 404.

## Tests to write
- `TestBookAPIList`: empty/structured/legacy entries, exact envelope/fields, stable IDs and cache header.
- `TestBookAPICreate`: 201 then normalized duplicate 200, created flag, old metadata preserved; DB rows checked.
- `TestBookAPIUpdate`: owned edit/default/delete and stable IDs; independent owner and order unchanged.
- `TestBookAPIValidation`: each FieldErrors case and invalid JSON class → exact status/message/field map.
- `TestBookAPIIDs`: invalid/overflow/missing/foreign IDs for every ID route; no other-owner data leaked.
- `TestBookAPIConflict`: duplicate edit 409 and unchanged rows/default.
- `TestBookAPIAuth`: web/mini, no credential, expired/revoked, legacy JWT with flag on/off, staff and unknown role; exact adapter status plus cache header.
- `TestBookAPIFailures`: store failure → generic 500; request text absent from captured logs/monitoring.
- `TestAppCustomerAddresses`: real app wiring serves new routes and legacy routes with their distinct ID semantics.

## API documentation
Add all five routes, request/Entry/envelope schemas, auth feature-flag matrix and every error above to docs/API.md and OpenAPI. State that address storage does not prove delivery eligibility and duplicate creation does not overwrite fields. Legacy endpoints keep existing sections.

## Done when
Load `.env.development`; gofmt/build/vet/full serial tests pass; coverage of address/app/apidocs and any other changed internal package is at least 80%. API/OpenAPI examples agree with test output. Tick 97 and commit `task 97: expose customer saved addresses`.

## Out of scope
Bulk import endpoint, delivery-price changes, order creation changes, account merge (98), frontend, deployment.
