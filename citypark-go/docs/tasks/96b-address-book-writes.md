# Task 96b: Transactional saved-address creation and editing

## Goal
Create/deduplicate and edit per-user saved addresses without modifying another customer or historical orders.

## Depends on
Backend 96. Do not deploy before 98.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/address/book.go`
- `internal/address/book_store.go`
- `internal/store/store.go`

## Files to create
- `internal/address/book_service.go`
- `internal/address/book_writes_test.go`

## Files to change
- `internal/address/book_store.go` — transaction-bound insert/update helpers.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
Export:
```go
type Service struct { db *sqlx.DB }
func NewService(db *sqlx.DB) *Service
type InvalidFieldsError struct { Fields map[string]string }
func (e *InvalidFieldsError) Error() string // "invalid address fields"; no input text.
var ErrDuplicate = errors.New("saved address already exists")
var ErrOwnerUnavailable = errors.New("address owner unavailable")
func (s *Service) CreateEntry(ctx context.Context, userID int64, in Fields) (Entry, bool, error)
func (s *Service) UpdateEntry(ctx context.Context, userID, entryID int64, in Fields) (Entry, error)
```
The bool is `created`. Clean and validate with task 96's helpers before writes; invalid fields return InvalidFieldsError. Every operation uses `store.InTx`, then locks its owner before reading/writing the address book:
```sql
SELECT id FROM users WHERE id=$1 AND role='user' AND merged_into_user_id IS NULL FOR UPDATE;
```
Missing/ineligible owners return ErrOwnerUnavailable, distinguishable from a missing entry's store.ErrNotFound. Use `NewStore(tx)` throughout the transaction; propagate context/SQL/commit errors and roll back partial writes. Private lock/helper names are local choices.

Create: list this owner's entries under the lock, compare DuplicateKey(city,address), and return the lowest-ID matching entry with `created=false`, preserving all fields, default and timestamps. This includes legacy entries; return them unchanged with needsReview=true. Otherwise call existing FindOrCreate(city,FormatAddress(in)) and insert:
```sql
INSERT INTO user_addresses(user_id,address_id,name,street,apartment,entrance,is_default)
VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id;
```
`$7` is true only when the book was empty, not whenever no default exists. Read the complete Entry back in the same transaction. Return created=true only after commit. No unique normalized-key migration: old normalized duplicates remain visible.

Update: look up `(userID,entryID)` under the owner lock before changing data. Compare the proposed key to every other entry; return ErrDuplicate on collision without changing either entry. Reuse/create the immutable shared address and execute:
```sql
UPDATE user_addresses SET address_id=$3,name=$4,street=$5,apartment=$6,entrance=$7,updated_at=now()
WHERE user_id=$1 AND id=$2;
```
Keep stable entry ID, created_at and default flag. Updating a legacy entry explicitly replaces its null street with validated structured input, clearing needsReview. A same-entry unchanged address is allowed. Do not delete the old shared row or update `addresses.city/address`; do not rewrite orders.

Matching structured creates do not silently repair/overwrite a legacy duplicate. The caller can explicitly edit it. Names are optional; an empty name remains empty. Validation limits and Russian field messages are exactly task 96's FieldErrors; errors crossing the API are mapped in task 97.

## Tests to write
- `TestCreateEntry`: first creates default, second does not; deleting/default absence is not assumed to mean empty.
- `TestCreateEntryDuplicate`: repeated and normalized equivalent input → same stable ID, created=false, original name/default/timestamps; different apartment/entrance → distinct entries.
- `TestCreateLegacyDuplicate`: duplicate legacy text → unchanged review-needed entry; existing normalized duplicates choose smallest ID.
- `TestUpdateEntry`: stable ID, created_at/default preserved; new full text and structured values read back.
- `TestUpdateLegacyEntry`: explicit valid edit clears needsReview without parsing old text.
- `TestUpdateEntryConflict`: equivalent other entry → ErrDuplicate and no rows/metadata changed.
- `TestBookOwnerIsolation`: two users share one address; editing one preserves the other's data and existing order snapshots.
- `TestBookWriteValidation`: all FieldErrors plus missing/merged/staff owner and missing/foreign entry → errors, no writes.
- `TestBookConcurrentCreates`: goroutines inside one test, same owner/key → one association and one default; do not use t.Parallel.
- `TestBookWriteRollback`: failure after shared-address creation (test transaction/constraint failure), canceled context and DB failure → no orphan write from the failed transaction, generic error with no address contents.

Use testutil.NewDB and inspect persisted rows, not only returned values. Preserve existing tests.

## API documentation
No endpoint changes in this task.

## Done when
Load `.env.development`; `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test -p 1 ./...`, `go test -p 1 -cover ./internal/address/...` pass (at least 80%). Never start the database. Tick 96b and commit `task 96b: create and edit saved addresses`.

## Out of scope
Default/deletion/legacy-write transaction integration (96c), API (97), account merge (98), deployment.
