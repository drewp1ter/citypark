# Task 93: Sentry backend monitoring

## Goal
Report unexpected HTTP errors and panics to the approved citypark-go Sentry project, with request tracing and privacy controls.

## Depends on
92.

## Read first
- docs/tasks/CONVENTIONS.md
- internal/app/app.go
- internal/httpx/httpx.go
- cmd/server/main.go

## Files to create
- internal/monitoring/sentry.go and sentry_test.go: client construction, request middleware, privacy filters and tests.
- docs/SENTRY.md: configuration, verification and manual deployment.

## Files to change
- internal/config/config.go and config_test.go: optional Sentry configuration.
- internal/httpx/httpx.go and httpx_test.go: capture unexpected returned errors once.
- internal/app/app.go and app_test.go: pass optional monitoring middleware through construction.
- cmd/server/main.go: construct the client before I/O, inject middleware, flush on exit.
- go.mod/go.sum, .env.example, README.md, docs/tasks/PROGRESS.md.

## Spec
Explicitly allow github.com/getsentry/sentry-go and its http integration. Use a constructed client and a new request hub, never sentry.Init or new mutable globals. Keep app.New source-compatible with optional middleware. Panic middleware sits inside existing recovery and re-panics after capture. Unexpected returned errors are captured using the request hub, expected *httpx.Error failures are excluded. Preserve response bodies/statuses. Add SENTRY_DSN (empty disables), SENTRY_RELEASE and SENTRY_TRACES_SAMPLE_RATE (finite 0..1; default development 1, other environments 0.1). Environment is APP_ENV. Enable tracing, attach stacks and flush after graceful shutdown. Do not send request bodies, queries, credentials, cookies, customer identities, breadcrumbs or unreviewed span attributes. Keep method, normalized route, stack frames and trace IDs. Trace continuation uses standard Sentry HTTP middleware.

## Tests to write
- TestConfigSentry: defaults, overrides, invalid/non-finite rate.
- TestMonitoring: unexpected errors and panic each captured once; expected errors excluded; disabled client; invalid DSN; request isolation; filtering; flush.
Use fake Sentry transport and httptest, no outbound telemetry in automated tests.

## API documentation
No endpoints or response contracts change. No permanent test route.

## Done when
All CONVENTIONS checks pass; changed internal packages meet 80% coverage. Verification uses the real application's initialization and HTTP chain, temporary test triggers are removed. If a user-managed running backend must restart, document the pending verification without restarting it. Record unrelated suite failures honestly and do not skip them. Document manual deployment and production verification; never deploy without approval.

## Out of scope
Admin, background jobs, alerts, replay, custom database spans, production deployment and customer-data mutations.
