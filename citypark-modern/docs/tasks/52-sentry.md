# Task 52: Sentry website monitoring

## Goal
Capture browser and Next.js server errors and traces in citypark-modern, with readable production stacks.

## Depends on
51; backend task 93 for cross-service tracing verification.

## Read first
- docs/tasks/CONVENTIONS.md
- next.config.ts
- src/pages/menu/ui/menu-error.tsx
- src/shared/config/env.ts
Read installed Next.js instrumentation, error and configuration docs, and Sentry's current Next.js manual setup guide.

## Files to create
- Root instrumentation.ts, instrumentation-client.ts, sentry.server.config.ts, sentry.edge.config.ts: Next runtime entry points alongside root app.
- src/shared/monitoring/: public API and shared configuration/privacy logic.
- src/app/errors/: global Russian error fallback and public API; app/global-error.tsx re-exports it.
- scripts/qa/sentry.mjs: configuration/privacy assertions and opt-in verification instructions.
- docs/SENTRY.md: DSNs, source-map uploads, verification, manual deployment.

## Files to change
- package.json/package-lock.json: allow @sentry/nextjs.
- next.config.ts: wrap existing config, upload source maps with build-only auth token.
- src/pages/menu/ui/menu-error.tsx: capture boundary errors without changing the visible fallback.
- .env.example, README.md, docs/tasks/PROGRESS.md.

## Spec
Initialize browser, Node and Edge SDKs using the current documented API. Configure onRequestError and onRouterTransitionStart. NEXT_PUBLIC_SENTRY_DSN empty disables capture; NEXT_PUBLIC_SENTRY_ENVIRONMENT defaults to development in dev and production otherwise. NEXT_PUBLIC_SENTRY_RELEASE is the website commit release. NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE defaults to 1 in development, 0.1 otherwise, rejects invalid/non-finite/out-of-range values. Server uses SENTRY_DSN with NEXT_PUBLIC_SENTRY_DSN fallback. SENTRY_RELEASE overrides server release. Restrict server trace propagation to MENU_API_URL origin; browser uses same-origin relative requests only. Keep request/customer data, cookies, credentials, breadcrumb content, query strings and unreviewed span attributes out of telemetry. Configure source-map uploads to citypark-ym/citypark-modern with SENTRY_AUTH_TOKEN available only during builds. Remove uploaded source maps from deployable output. Preserve FSD rules and Russian fallback UI. No permanent error-trigger route. Manual deploy and real production stack validation remain explicit steps, not assumed passes.

## Tests to write
Use a lightweight Node harness (or installed TypeScript support) to assert sampling defaults and validation, DSN disabling, privacy filtering and propagation restrictions. Production browser verification triggers a genuine application error through temporary code, not a console throw or standalone SDK. Also verify a server error and cross-service trace after the user-managed backend is updated.

## API documentation
No API contract changes.

## Done when
npm run check, npm run build and monitoring QA pass. Local checks do not require a source-map auth token; production source-map upload is mandatory before release acceptance. Record credentials/runtime-dependent verification as pending with exact next commands. Temporary verification code is removed before commit.

## Out of scope
Admin, replay, feedback widgets, alert rules, production deployment and live orders/payments.
