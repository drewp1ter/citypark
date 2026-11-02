# Sentry website monitoring

Project: [citypark-modern](https://citypark-ym.sentry.io/issues/?project=4512363062100048), EU region. The Go API uses a separate project. The website DSN is in the ignored local `.env`; retrieve it for other hosts from Sentry **Settings → Projects → citypark-modern → Client Keys (DSN)**.

## Configuration

Public settings are embedded during `npm run build`. Rebuild when changing them; setting them only on `npm start` cannot change an existing browser bundle.

| Variable | Meaning |
| --- | --- |
| `NEXT_PUBLIC_SENTRY_DSN` | Website DSN; empty disables browser reporting. |
| `NEXT_PUBLIC_SENTRY_ENVIRONMENT` | Explicit environment, usually `development` or `production`. Defaults according to NODE_ENV. |
| `NEXT_PUBLIC_SENTRY_RELEASE` | `citypark-modern@<full commit SHA>`, available at build and runtime. |
| `NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE` | Finite 0–1; defaults to 1 in development and 0.1 otherwise. |
| `SENTRY_DSN` | Optional server DSN override; defaults to the public DSN. |
| `SENTRY_RELEASE` | Optional server release override; defaults to the public release. |
| `SENTRY_AUTH_TOKEN` | Build-only source-map upload token. Never use a NEXT_PUBLIC prefix or commit it. |

`MENU_API_URL` remains server-only. Trace headers go only to its exact origin from the Next server. Browser propagation is limited to relative same-origin requests. Keep the backend's existing endpoint and update its process to backend task 93 for connected traces. Browser exceptions, global/route boundaries and uncaught server request errors are captured. Expected form validation responses and swallowed fallback errors are not automatically exceptions; background-job/log instrumentation is separate work.

Privacy filters remove request bodies/headers/cookies, user identity, URL queries, breadcrumbs and unreviewed span attributes. An anonymous IP placeholder prevents ingestion from inferring a location. Exception messages become `Unexpected application error`; exception types, stack positions, release/environment and trace IDs remain. Span names are operation categories, so arbitrary URLs and customer identifiers do not leave the process. Browser/server logs keep their existing behavior. Session Replay and feedback widgets are not installed.

## Local verification

```bash
npm run check
node scripts/qa/sentry.mjs
npm run build
npm start -- --port 3111
```

Pure monitoring checks never initialize Sentry or send telemetry. Verify real ingestion with temporary code in this actual application: a button throwing an Error in a page and a read-only route handler throwing an Error. Click the button in Playwright or the browser; a console throw is not a valid test. Confirm both issues using Sentry MCP, including title, sanitized message, direct issue URL, release/environment and source frames. Remove both temporary routes afterward and rerun checks/build. Do not use checkout, payments or notifications as error generators.

## Manual production build and deployment

Load production settings in your normal build environment, including `MENU_API_URL`, then:

```bash
export NEXT_PUBLIC_SENTRY_ENVIRONMENT=production
export NEXT_PUBLIC_SENTRY_RELEASE="citypark-modern@$(git rev-parse HEAD)"
export NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE=0.1
# Supply NEXT_PUBLIC_SENTRY_DSN and SENTRY_AUTH_TOKEN through your secret/env manager.
npm ci
npm run check
node scripts/qa/sentry.mjs
npm run build
```

The wrapper uploads source maps to `citypark-ym/citypark-modern` when the token is supplied, then deletes uploaded maps from deployable output. A local build without the token is allowed and disables source-map generation/upload by the Sentry wrapper. Such a build is not accepted as a production monitoring release. Upload failures with a supplied token must fail the build, not be ignored. Do not expose `.next` or any remaining server source maps through a static web server.

Remove `SENTRY_AUTH_TOKEN` from the runtime environment after building; it is not required by `npm start`. Copy the normal build artifacts and have the owner restart the website using the existing manual process. Production deployment is a separate approval step.

After deployment, verify a safe temporary error from that exact production build has source file/function/line frames, the expected release and production environment, and no customer data. Confirm a read-only website→API request shares a trace with the updated Go backend. Do not claim production stack quality from a development-only event or a successful build without upload credentials.

## Verified during setup (2026-11-02)

The source-map upload completed successfully, and no public `.map` files remained in `.next/static`. Fresh errors from a production build, running locally with environment `verification`, resolve to original TypeScript including source context:

- [Browser verification](https://citypark-ym.sentry.io/issues/CITYPARK-MODERN-6): `SentryProductionBuildVerificationError: Unexpected application error`, `app/sentry-verification/page.tsx:5`.
- [Server verification](https://citypark-ym.sentry.io/issues/CITYPARK-MODERN-7): `SentryServerBuildVerificationError: Unexpected application error`, `app/api/sentry-verification/route.ts:2`.

These temporary routes are removed from the final source. This proves the production artifact/source-map configuration locally, not a production deployment. The existing menu QA flow also passed against the production build with the real configured backend. An earlier run overlapped the backend restart and timed out; rerunning with the backend running passed.

[Cross-service verification trace](https://citypark-ym.sentry.io/explore/traces/trace/8409f2e72c634806a9182b324130a770) confirmed the Next request, outgoing HTTP span and Go handler share one trace; both errors were attached. The backend owner ran the temporary error route, then restarted the final clean code. The former route now returns 404 and backend health returns 200.

## Address autofill diagnostics

Failed and partial checkout autofill attempts emit `AddressAutofillError`. Filter by `autofill_reason`; issues group by stage and reason. Permission denial, empty results, unsupported regions, missing/ambiguous towns and partial addresses use warning severity; technical failures use error severity.

The `autofill` context permits only stage, checkout surface (`web`/`mini`), elapsed milliseconds, HTTP status, browser error code, secure-context status, delivery-town/candidate counts and locality source (`city`/`settlement`). The final SDK filter validates these fields again. Coordinates, locality/address text, raw responses, credentials and arbitrary errors remain excluded. Successful autofill and intentional skips emit nothing. Recognized server failures are reported by the server; browser transport failures are reported by the browser. A browser timeout can coincide with a later server failure and produce two distinct stage events.

Verification commands (production server on port 3111, real configured backend):

```bash
node scripts/qa/sentry.mjs
node scripts/qa/autofill-provider.mjs
node --env-file=.env scripts/qa/flows/checkout-geolocation.mjs
node --env-file=.env scripts/qa/autofill-browser.mjs
```

Provider tests use controlled responses and never contact DaData. Browser tests intercept telemetry by default; `SEND_AUTOFILL_VERIFICATION=1` forwards the synthetic permission-denied event to the configured Sentry project. The geolocation flow also sends a malformed request through the real route to verify server validation reporting. No orders or payments are created.

Verified on 2026-11-02 from a local production build with uploaded source maps:
- [Browser permission denial: CITYPARK-MODERN-A](https://citypark-ym.sentry.io/issues/CITYPARK-MODERN-A): warning, `browser_code: 1`, `surface: web`, original hook and reporter TypeScript frames.
- [Server invalid input: CITYPARK-MODERN-9](https://citypark-ym.sentry.io/issues/CITYPARK-MODERN-9): error, validation stage, original handler/reporter TypeScript frames.

Both retain the anonymous IP placeholder and exclude customer location data. Production rollout still uses the owner-managed build/restart process above.
