# Task 98: Preserve saved addresses when customer accounts merge

## Goal
Make the existing consented account merge retain address metadata and one deterministic default, without changing identity/pairing rules or old orders.

## Depends on
Backend 97. Completes the backend release unit 96–98.

## Read first
- `docs/tasks/CONVENTIONS.md`
- `internal/customer/merge.go`
- `internal/address/book_store.go`
- `internal/customer/merge_test.go`

## Files to create
- `internal/address/book_merge.go`
- `internal/address/book_merge_test.go`

## Files to change
- `internal/customer/merge.go` — replace old association INSERT/DELETE with the transaction-bound helper.
- `internal/customer/merge_test.go` — integration/rollback assertions.
- `docs/API.md` — update account-pairing/merge notes for saved addresses/defaults.
- `docs/tasks/PROGRESS.md` — completion only.

## Spec
```go
// Caller already holds both user-row locks in ascending ID order.
func (s *Store) MergeEntries(ctx context.Context, sourceID, destinationID int64) error
```
Invoke `address.NewStore(tx).MergeEntries` inside MergeAccountsInTx, after its existing user locks and within the same transaction. Do not start a nested transaction or add an address-to-customer import cycle. Address mutations lock their one user first (96b/96c), so merge observes a consistent book. No changes to authentication/consent/session-revocation semantics.

Read both books with ListEntries; take a snapshot of defaults. Destination entries win on DuplicateKey(city,address), including legacy review state/name; choose the smallest destination ID when old duplicates exist. Do not collapse pre-existing destination duplicates. Unique source entries keep their association IDs and metadata. For duplicate source entries, retain only the first source entry by ID unless it matches destination; never overwrite destination metadata. Map the source default to its surviving equivalent if needed.

Default policy: keep destination default if present; otherwise prefer the surviving equivalent of the source default; otherwise leave no default. For a source entry needing review, preserve review state; never infer structured values.

Before moving rows, clear the source default so moving it cannot violate the destination partial index:
```sql
UPDATE user_addresses SET is_default=false,updated_at=now() WHERE user_id=$1 AND is_default;
```
For each duplicate source association:
```sql
DELETE FROM user_addresses WHERE user_id=$1 AND id=$2;
```
For each surviving source association:
```sql
UPDATE user_addresses SET user_id=$2,updated_at=now() WHERE user_id=$1 AND id=$3;
```
If destination had no default and the chosen surviving source-default equivalent exists:
```sql
UPDATE user_addresses SET is_default=true,updated_at=now() WHERE user_id=$1 AND id=$2;
```
Here the final statement's $1 is destinationID. All mutations must check expected affected rows; unexpected missing rows abort the outer transaction. Preserve source entries' created_at, names and structured columns. Do not delete shared address rows or change order delivery snapshots. Finish with no source-owned associations. SQL errors abort the entire account merge, including identities, orders and pairing proof consumption.

The helper validates positive, different IDs and verifies two owners exist with `SELECT id FROM users WHERE id=ANY($1) ORDER BY id` ($1 is the nonempty two-ID slice); caller-held locks remain in force. It returns an English internal error for misuse/missing owners without printing customer data. Do not reject the source's merged_into_user_id: the outer transaction has already marked it as merged before moving child rows. The public merge contract/errors remain unchanged.

## Tests to write
- `TestMergeEntriesUnique`: different entries retain IDs, names, structured fields and created_at under destination ownership.
- `TestMergeEntriesDuplicate`: same/normalized equivalent address → destination metadata and ID win; unit differences remain separate.
- `TestMergeEntriesDefaults`: destination wins; no destination default adopts source default/equivalent; neither default stays none; one default maximum.
- `TestMergeEntriesLegacy`: raw text/review flag preserved, no lossy parsing; existing destination duplicates remain visible.
- `TestMergeEntriesSourceDuplicates`: deterministic lowest-ID survivor and mapped default; no uniqueness violation.
- `TestMergeEntriesRollback`: forced error mid-move → all source/destination rows and outer merge changes restored.
- `TestMergeEntriesConcurrentWrite`: concurrent address mutation and merge serialize under ordered owner locks; no lost metadata or stranded source entries.
- `TestMergeAccountsAddressMetadata`: actual MergeAccounts/consented pairing integration retains metadata, orders' delivery snapshots and existing session revocation behavior.
- `TestMergeEntriesInvalid`: same/missing/nonpositive owners, canceled context/DB failure are handled without partial changes.

## API documentation
Update existing pairing/merge notes: destination metadata/default wins; unique addresses transfer; no new endpoints or wire schemas. Confirm OpenAPI descriptions remain accurate.

## Done when
Load `.env.development`; gofmt/build/vet/full serial tests pass; coverage of address/customer and any other changed internal packages is at least 80%. Run existing app customer-linking integration tests through the full suite. Tick 98 and commit `task 98: preserve saved addresses during account merge`.

## Out of scope
Changing merge consent/provider rules, automatic guest import, frontend or deployment. Backend 96–98 may only be released together after all checks pass.
