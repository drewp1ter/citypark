# Task 94: Production content snapshot

## Goal
Export the current public content and assets, with a separate explicit production restore command. Existing demo seeding is unchanged.

## Depends on
93

## Read first
- docs/tasks/CONVENTIONS.md
- cmd/seed/main.go
- internal/store/store.go

## Spec
Implement `cmd/content-seed` export/check/restore with a versioned JSON bundle, column compatibility checks, deterministic ordering, repeatable-read source export, asset checksums, portable paths, and explicit --replace. Restore exactly the nine datasets listed in docs/CONTENT_SEED.md in a transaction, preserve identities, transactionally restart sequences, preserve excluded datasets and unrelated files. Do not modify the source DB. Validate assets and reject conflicting target files. Include a reviewed source snapshot and uploads, Docker binary, and operational documentation. Use the existing configured test DB only.

## Tests
Roundtrip equality, repeat restore, target-only removal, excluded address preservation, identity sequence advancement, failed SQL restore rollback, missing/corrupt assets, unsafe paths, and conflicting uploads.

## Done when
- gofmt -l . prints nothing
- go build ./...
- go vet ./...
- Load .env.development and run go test -p 1 ./...
- go test -p 1 -cover ./cmd/content-seed
