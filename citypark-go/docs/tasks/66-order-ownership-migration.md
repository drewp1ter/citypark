# Task 66: Order ownership migration

## Goal
Add the database foundation for customer accounts without changing running HTTP
interfaces. Separate authenticated ownership from the phone-based checkout contact
and backfill historical contact snapshots.

## Depends on
65.

## Read first
`docs/tasks/CONVENTIONS.md`, `migrations/00001_init.sql`,
`internal/store/migrate.go`, `internal/store/migrate_test.go`.

## Files to create
- `migrations/00019_order_ownership.sql`
- `internal/store/order_ownership_test.go`
- `docs/tasks/CUSTOMER_ACCOUNTS.md`: release requirements and numbered follow-ups.

## Files to change
- `docs/tasks/PROGRESS.md`: register and complete this task after verification.

## Spec
- Keep `users` as the existing account/contact/staff table; new identities and
  sessions will reference its internal ID. Do not introduce a second account ID.
- Add nullable `orders.account_id bigint REFERENCES users(id) ON DELETE SET NULL`.
  Keep `customer_id` and `address_id` unchanged. Index `(account_id, id DESC)`
  where `account_id IS NOT NULL` for future paginated account history.
- Add nullable text columns `contact_name`, `contact_phone`, `delivery_city`,
  `delivery_address`. Nullable columns allow the current application to keep
  inserting orders during additive deployment; task 67 writes and reads them.
- Backfill all existing orders from the referenced user and address. Pickup or
  already deleted addresses have null delivery snapshots. Lost historical data
  cannot be recovered; copy only what is currently available.
- Backfill `account_id = customer_id` only for `role = 'user'` with a nonempty
  `password_hash` or `legacy_password_hash`. Admins, phone-only contacts and
  email-only contacts remain unclaimed. Never match different rows by phone,
  email or name. Set `updated_at = now()` for the backfilled rows.
- Snapshot values are stored text, independent of later edits/deletion of the
  source records. No HTTP response shape, authentication or authorization changes
  in this additive migration. New orders must not acquire ownership by default.
- Down removes just the new index and columns. Existing order data survives.
- Use goose's transactional SQL migration handling and embedded migration FS.

## Tests to write
- `TestOrderOwnershipMigration`: use `openTestDB`, goose provider and the existing
  configured PostgreSQL database. Migrate to 18, seed password, legacy-password,
  phone-only, email-only and admin contacts with delivery and pickup orders; migrate
  up. Assert exact snapshot values and ownership, stable legacy order IDs and
  statuses; confirm an address deleted before migration stays unknown.
- In the same test, edit the referenced user/address and delete the address;
  assert stored snapshots survive. Insert a new order with the old column set;
  assert ownership and snapshots default to NULL. Assert invalid owners fail the
  foreign key. Down to 18 must preserve old rows and remove new columns; up again
  must succeed. Run in an isolated schema and drop it in cleanup, including on
  failure, leaving the public schema untouched by this test.

## API documentation
No endpoint changes. API/OpenAPI updates belong to task 67, when the application
starts using the columns. No admin type regeneration is needed for this task.

## Done when
Run all commands in CONVENTIONS "Done means", with `internal/store` coverage at
least 80%, plus `go test -p 1 ./internal/store -run '^TestOrderOwnershipMigration$' -count=1`.
Load `.env.development` first; never start or replace the database.

## Out of scope
Runtime snapshot usage, account authorization and duplicate isolation (67),
sessions/email (68–72), Google (73–74), Telegram/pairing (75–78), frontend (35–40),
production migration/deployment and real payments.
