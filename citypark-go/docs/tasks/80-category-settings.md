# Task 80: Category descriptions and ordering times

## Goal
Sync category descriptions from iiko and let admins set category ordering hours that survive sync.

## Depends on
79.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/catalog/handler.go`, `internal/iiko/sync.go`.

## Files to create
- `internal/catalog/category_hours.go`
- `internal/catalog/category_hours_test.go`

## Files to change
- `internal/iiko/types.go`, `internal/iiko/sync.go`, `internal/iiko/sync_test.go`: descriptions and preserved schedules.
- `internal/app/app.go`: register admin route.
- `internal/apidocs/openapi.yaml`, `docs/API.md`: contract.

## Spec
Trim and sync category description, including clearing on empty input. Preserve existing category sale windows on conflict; new categories are unrestricted.
Admin PUT `/categories/{id}/ordering-hours` accepts `startTime` and `endTime` strings. Both empty remove time restrictions. Otherwise parse H:MM or HH:MM; start must be earlier than end, end can be 24:00. A required `days` string is empty for every day or a seven-character 0/1 mask, Monday first; at least one weekday must be selected. Return the updated public category view. Authentication is required (401), admin role required (403), invalid input is 400 with field errors, missing/deleted category is 404. Availability uses the existing category window checks.

## Tests to write
Category description trimming/replacement/clearing and schedule preservation on repeated sync. Endpoint persistence, clearing, validation, missing category, authentication and role checks.

## API documentation
Document PUT `/categories/{id}/ordering-hours`; update sync notes.

## Done when
All conventions checks pass; catalog and iiko coverage at least 80%.

## Out of scope
Overnight windows and product schedule editing.
