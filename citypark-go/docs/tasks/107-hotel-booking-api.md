# Task 107: Expose hotel booking API

## Goal
Expose hotel booking API for one-room, instantly confirmed hotel reservations without prepayment. 1C is authoritative for inventory and tariffs.

## Depends on
backend 106; 1C task 01

## Read first
- `AGENTS.md`
- `docs/tasks/PROGRESS.md`
- `../citypark-hotel-managment/docs/tasks/01-booking-contract.md` (resolve from this project root).

## Files to create
- `internal/hotelbooking/handler.go`
- `internal/hotelbooking/handler_test.go`

## Spec
Expose public availability/quote and booking endpoints specified by the contract. Validate dates, capacity and guest fields; take prices only from 1C. Support stable idempotency and uncertain-result recovery without duplicate reservations. Add agreed abuse controls. Never report confirmed until 1C commits. Keep private responses uncached and redact personal data. Wire application dependencies and configuration. Update docs/API.md and internal/apidocs/openapi.yaml.

## Tests to write
Valid booking, invalid fields, occupied room, changed tariff, repeat submission, mismatched key payload, upstream outage, commit timeout/recovery and absence of PII in logs. Serial DB tests only if persistence is required by the contract.

## API documentation
Document any new endpoints in the project's API contract and OpenAPI where applicable. Freeze exact request/response shapes, Russian messages and retry semantics in the prerequisite contract before implementation; do not invent incompatible interfaces.

## Done when
Dependencies are complete; all specified scenarios pass. Follow project checks (Go: formatting, build, vet, serial tests and required coverage; website: npm run check, npm run build and booking browser QA; 1C: Designer checks and development-base runtime tests). Record blockers rather than claiming production validation. Documentation-only tasks require link and consistency checks, not runtime tests.

## Out of scope
Online payment, multi-room carts, guest self-service cancellation or changes, and automatic production deployment. Do not implement another task in this session.
