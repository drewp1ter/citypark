# Task 111: Sync iiko dish stop-list status

## Goal

Save the delivery terminal group's stop-list status alongside the menu so staff can see which dishes iiko lists.

## Depends on

Task 108 (dish visibility) and the existing iiko menu sync.

## Read first

`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync.go`, `internal/iiko/client.go`, `docs/API.md`.

## Files to create

- `internal/iiko/stop_list.go` and `internal/iiko/stop_list_test.go`
- `migrations/00044_product_stop_list.sql`

## Files to change

- Wire stop-list fetching into the existing sync and pass `IIKO_TERMINAL_GROUP_ID` from the app configuration.
- Expose the stored flag through the catalog product view and OpenAPI schema. Update the catalog response contract test.

## Spec

- Fetch `POST /api/1/stop_lists` with the configured organization, asking for size information. Restrict the request to the delivery terminal group when configured. Otherwise require a single group for the organization.
- Match entries by iiko product ID, including size-specific entries. A valid empty list clears the flags. An upstream failure or an indeterminate terminal group aborts the sync before writes.
- Store `products.is_on_stop_list` as a non-null boolean with a default of false. Update the flag in the menu transaction. Expose it as `Product.isOnStopList` in the catalog API.

## Tests to write

- `TestFetchStopList` covers scoped and fallback group selection, size entries, duplicates, an empty list, malformed or missing groups, and upstream failure.
- `TestSyncStopListSnapshot` covers applying and clearing the marker, preserving it on an upstream error, and retaining a manual block.

## API documentation

Update the product-listing examples and the sync notes in `docs/API.md`. The existing sync endpoints retain their response and error contracts.

## Done when

Run the backend conventions checks using the configured PostgreSQL test database. The stop-list client test also runs without PostgreSQL.

## Out of scope

Customer availability and order enforcement. Admin markers are tracked in admin task 38.
