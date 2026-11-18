# Task 109: Preserve shared dish images

## Goal
Allow multiple dishes to use the same iiko image URL without losing images in the admin panel or public website.

## Depends on
108. Explicit owner-requested image-sync fix; independent of hotel tasks 106–107.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/iiko/sync_extra.go`, `migrations/00001_init.sql`.

## Spec
- Add migration 39 replacing global product-image URL uniqueness with uniqueness of `(product_id, url)`.
- Preserve existing image rows and IDs. Rollback restores global URL uniqueness and refuses shared URLs without discarding data.
- Sync each product-image association independently, preserving IDs across repeated syncs.
- Remove stale iiko associations even when another product still uses that URL; preserve manual images and existing category-image cleanup behavior.
- Verify shared images reach catalog reads used by both clients. No HTTP contract or frontend changes are required.
- Cover repeat sync, shared-image removal, URL changes, empty menus, manual-image preservation and migration up/down behavior.

## Done when
- `gofmt -l .` prints nothing.
- `go build ./...` and `go vet ./...` pass.
- With `.env.development` loaded, `go test -p 1 ./...` passes.
- `go test -p 1 -cover ./internal/iiko/...` passes with at least 80% coverage.

## Release
Deploy the migration and backend together, then run the normal menu sync to restore missing associations. No production sync or deployment is part of this task.
