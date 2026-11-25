# Task 112: Receive iiko stop-list updates

## Goal

Refresh saved dish markers when iiko reports a change for the delivery terminal group.

## Depends on

Task 111 provides stop-list fetching, terminal group selection, and persisted product flags.

## Read first

`docs/tasks/CONVENTIONS.md`, `internal/iiko/stop_list.go`, `internal/iiko/sync.go`, and the [iiko webhook guide](https://api.iiko.help/docs/en/guides/webhooks/).

## Changes

- Add `POST /api/v1/webhooks/iiko`. Match `Authorization` to `IIKO_WEBHOOK_TOKEN`. An empty configured token disables the endpoint.
- Accept event batches and queue a refresh for matching `StopListUpdate` events. Check the configured organization and terminal group IDs.
- Acknowledge valid batches with HTTP 200 before making any iiko or database calls. Coalesce pending notifications and retry failed refreshes after 30 seconds.
- Use the existing sync lock and stop-list client. Update only product stop-list markers. Keep scheduled menu sync as a fallback.
- Document the endpoint in the API reference and OpenAPI and document registration in the deployment guide.

## Done when

Run the webhook receiver and worker tests, Go build and vet, and the PostgreSQL integration suite. Verify that adding and removing a dish from the delivery terminal group's stop list updates the saved marker.
