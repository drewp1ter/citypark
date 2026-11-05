# Sentry backend monitoring

Project: [citypark-go](https://citypark-ym.sentry.io/issues/?project=4512363062034512), EU region.

## Configuration

The DSN is configured in the ignored local `.env`; `.env.example` intentionally has no DSN. Use the project's Sentry **Settings → Client Keys (DSN)** to retrieve it for another host.

- `SENTRY_DSN`: empty disables reporting; otherwise the project's DSN.
- `SENTRY_RELEASE`: `citypark-go@<full git SHA>` of the deployed code.
- `SENTRY_TRACES_SAMPLE_RATE`: finite 0–1; default development 1, production/test 0.1. An incoming sampled trace can retain its upstream sampling decision.
- `APP_ENV`: existing development, production or test setting; used as Sentry environment.

Each request gets an independent hub. Sentry captures panics before the existing recovery writes its usual 500. The shared HTTP wrapper captures unexpected returned errors once. Expected `httpx.Error` responses are excluded. No API endpoint or response schema changes.

Free-form exception values are replaced with `Unexpected application error`: they can embed SQL parameters, customer information or credentials. Issue diagnosis uses exception types, stacks, normalized route grouping, trace IDs and releases; original text remains in existing local logs. Request data, user identity, breadcrumbs, custom contexts and span data are removed. Transaction names use chi route patterns rather than actual IDs. An anonymous IP placeholder prevents location inference at ingestion. No log forwarding, background-job instrumentation or custom DB spans are configured.

## MONETA CHECK failures

Every CHECK that is rejected (HTTP 400, 403, 500) or answered with result code 500 sends one warning event `MONETA check failed: <reason>`, grouped by reason. Reasons: `bad_request`, `bad_signature`, `internal_error`, `invalid_transaction_id`, `order_not_found`, `order_canceled`, `order_not_payable`, `retry_not_allowed`, `amount_mismatch`. The event has no order id, UUID or amount. Look in the server log for the line `moneta check` with the same time to see the transaction id, order id, order status and both amounts. Every CHECK writes that line (Info when it succeeds, Warn when it fails).

## Checks

```bash
set -a
source .env.development
set +a
gofmt -l .
go build ./...
go vet ./...
go test -p 1 ./...
go test -p 1 -cover ./internal/app ./internal/config ./internal/httpx ./internal/monitoring
```

Monitoring tests use a fake transport and never send to Sentry. Do not start the database; ask its owner if unavailable.

## Manual build and deployment

From this project directory, in the environment where you normally build:

```bash
export SENTRY_RELEASE="citypark-go@$(git rev-parse HEAD)"
go build -trimpath -o bin/server ./cmd/server
```

Set `SENTRY_DSN`, `SENTRY_RELEASE`, `APP_ENV=production`, and `SENTRY_TRACES_SAMPLE_RATE=0.1` in the deployed process environment. Keep all existing database/payment/etc. settings. Copy the binary using the normal manual process, then have the owner restart the existing service. This task does not start or restart it. For image-based manual deployments, pass the same variables to the container; do not bake secrets into the image. Go stacks retain function/file/line information with the existing stripped binary flags; JavaScript source-map uploads do not apply.

Before calling production monitoring verified, confirm the running process is the new build and see its sampled read-only requests in Sentry. Then use a temporary, local-only error trigger in the actual application HTTP chain, capture an unexpected returned error (panic capture is covered by the middleware tests), and remove the trigger before deploying. Never trigger orders, callbacks, payments or notifications to generate errors. Confirm Sentry issue title/message/URL, release, environment, readable app frames, one event per error, and request isolation. Cross-service tracing requires the updated Go process and website together; a local fake API is not a substitute.

## Verified during setup (2026-11-02)

The owner restarted the real backend. A temporary read-only route returned a deliberate error; Sentry confirmed [CITYPARK-GO-1](https://citypark-ym.sentry.io/issues/CITYPARK-GO-1), `*errors.errorString: Unexpected application error`, with original Go file/function/line frames and the development environment/release. Two controlled requests produced two events, not duplicated capture.

[Trace 8409f2e72c634806a9182b324130a770](https://citypark-ym.sentry.io/explore/traces/trace/8409f2e72c634806a9182b324130a770) contains the website request, its outgoing HTTP span, the Go handler, and both error events. No orders, payments or staff notifications were generated.

The temporary route is removed from source and the clean binary rebuilt. The owner restarted the final clean backend; its former path `/api/v1/__sentry_verify_20261102` returns 404, and health returns 200. No production deployment has been performed by the agent. Production rollout still needs the release/env settings above and owner approval.
