# Task 96c: Address defaults, deletion and legacy write compatibility

## Goal
Finish address mutations and ensure old `/addresses` writes obey the same owner locking and default invariants.

## Depends on
Backend 96b. Do not deploy before 98.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/address/book_service.go`
- `internal/address/handler.go`
- `internal/address/address_test.go`

## Files to create
- `internal/address/book_defaults.go`
- `internal/address/book_defaults_test.go`

## Files to change
- `internal/address/handler.go` — optional service-backed legacy mutation path, keeping old constructor callers valid.
- `internal/app/app.go` — supply the service to the legacy handler.
- `docs/API.md` — document legacy default effect without altering legacy shape/IDs/statuses.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Export:
```go
func (s *Service) SetDefault(ctx context.Context, userID, entryID int64) (Entry, error)
func (s *Service) DeleteEntry(ctx context.Context, userID, entryID int64) error
func (s *Service) CreateLegacy(ctx context.Context, userID int64, city, address string) (Address, error)
func (s *Service) DeleteLegacy(ctx context.Context, userID, addressID int64) error
func (h *Handler) WithService(s *Service) *Handler
```
All service mutations use InTx and the user-row lock from 96b. SetDefault verifies ownership first, then executes in this order (avoid partial-index violations):
```sql
UPDATE user_addresses SET is_default=false,updated_at=now() WHERE user_id=$1 AND is_default AND id<>$2;
UPDATE user_addresses SET is_default=true,updated_at=now() WHERE user_id=$1 AND id=$2 AND NOT is_default;
```
Return the freshly read Entry. Repeating the same selection succeeds. Missing/foreign IDs return store.ErrNotFound with no default cleared. Legacy entries needing review may be default but must not auto-fill checkout until corrected.

DeleteEntry verifies ownership and then executes:
```sql
DELETE FROM user_addresses WHERE user_id=$1 AND id=$2;
```
Missing/foreign IDs return store.ErrNotFound (the new API uses 404). Deleting the default leaves no default. Never delete shared addresses or orders.

CreateLegacy retains existing trimming/validation, FindOrCreate semantics and response ID. Under the lock, inspect whether the book was empty and execute:
```sql
INSERT INTO user_addresses(user_id,address_id,is_default) VALUES($1,$2,$3) ON CONFLICT DO NOTHING;
```
`$3` is true only for an empty book. Do not normalize legacy matching differently, overwrite structured metadata or convert an existing association back to legacy. New legacy links use null street and need review.

DeleteLegacy uses shared address ID, not association ID:
```sql
DELETE FROM user_addresses WHERE user_id=$1 AND address_id=$2;
```
It retains 204 for absent links and leaves no default if the deleted link was default. Existing low-level Store.LinkUser/UnlinkUser remain transaction primitives for fixtures; all production HTTP mutations must use Service. WithService configures this in app wiring; existing standalone tests can still construct the old handler. Add service-backed handler tests so production behavior cannot regress to the fallback.

Keep legacy role behavior. The owner lock's role restriction from 96b must not accidentally revoke existing legacy staff access: legacy methods use `SELECT id FROM users WHERE id=$1 AND merged_into_user_id IS NULL FOR UPDATE`; new customer methods retain role='user'. Audit auth.Allows rather than inventing a different legacy policy.

## Tests to write
- `TestSetDefault`: first/repeated/second selections → exactly one default; returned entry matches DB.
- `TestSetDefaultForeign`: missing/foreign selection leaves old default untouched.
- `TestDeleteEntry`: delete own/default/nondefault; shared row, other owner and order snapshots survive; missing/foreign returns not found.
- `TestConcurrentDefaults`: competing sets and deletion under owner lock → at most one default, no partial state after rollback.
- `TestLegacyBookCreate`: existing wire shape/status/shared ID preserved; first new link default, duplicate does not alter metadata, new legacy entries need review.
- `TestLegacyBookDelete`: shared-ID deletion, absent link 204, no replacement default; entry IDs are not confused with shared IDs.
- `TestLegacyBookPermissions`: existing customer/staff behavior unchanged; absent/bad auth unchanged.
- `TestBookDefaultFailures`: missing/merged owner, canceled context and DB failures roll back; no raw address in errors.

## API documentation
Update the existing POST/DELETE `/addresses` notes in docs/API.md. No field/status/schema change; existing OpenAPI remains accurate. The structured routes belong to 97.

## Done when
Load `.env.development`; formatting/build/vet/full serial suite pass; `go test -p 1 -cover ./internal/address/... ./internal/app/...` reports at least 80% per package. Never start a database. Tick 96c and commit `task 96c: manage address defaults and legacy writes`.

## Out of scope
New HTTP routes, metadata-aware account merging, frontend, deployment.
