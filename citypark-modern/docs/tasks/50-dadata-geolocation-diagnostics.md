# Task 50: Local DaData geolocation diagnostics

## Goal
Inspect DaData's response and explain automatic delivery-town selection failures in the local Next.js terminal.

## Spec
- Log only with `DADATA_DEBUG=1` and `NODE_ENV=development`; document usage in `.env.example` and `docs/PLAN.md`.
- Include the full provider response before validation, HTTP status, provider failures, locality selection, normalized delivery-town names, candidates and acceptance/rejection reasons.
- Cover empty results, unsupported region, no/duplicate matches, malformed provider responses and delivery-list failures.
- Never print credentials or authentication headers. Keep API responses and selection behavior unchanged.
- This explicitly requested development diagnostics exception supersedes task 46's default prohibition on raw provider-response logging.

## Verification
- Controlled provider/handler checks: successful and failed matching, provider errors, delivery-list failures, HTTP response preservation, credential redaction and silence unless explicitly enabled in development. No live writes or DaData requests required.

## Done when
```bash
npm run check
npm run build
```

Controlled diagnostics checks pass; record their results in `PROGRESS.md`.
