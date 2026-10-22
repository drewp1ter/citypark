# Task 17: Docker image and deployment notes

## Goal
`docker build` produces an nginx image that serves the built admin with SPA fallback and long-cached assets, and the
README explains how to deploy it on `admin.<domain>`.

## Depends on
16.

## Read first
`CLAUDE.md`, `docs/tasks/CONVENTIONS.md`, `vite.config.ts`, `.env.example`, `../citypark-go/Dockerfile` (style),
`../docs/adr/0001-admin-panel.md`.

## Background
The admin is static files on its own subdomain and calls the backend directly (the backend allows any origin and the
`Authorization` header). `VITE_API_URL` and `VITE_SITE_URL` are build-time values. The old admin at `/manage` is
switched off by the owner by hand; the main site's nginx can redirect `/manage` to the new subdomain.

## Files to create
- `Dockerfile` (multi-stage: `node:<current LTS>-alpine` build with `ARG VITE_API_URL`, `ARG VITE_SITE_URL`; then
  `nginx:<stable>-alpine`), `nginx.conf`, `.dockerignore`

## Files to change
- `README.md`: a «Deployment» section.

## Spec
- `nginx.conf`: `try_files $uri /index.html;`; `/assets/` with `Cache-Control: public, max-age=31536000, immutable`;
  `index.html` with `Cache-Control: no-cache`; security headers `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: same-origin`, `X-Frame-Options: DENY`, and a `Content-Security-Policy` that allows `self`, the API
  origin (`connect-src` and `img-src` for `/uploads`), the site origin in `img-src` (for `/images/...` previews), and
  `style-src 'self' 'unsafe-inline'` only if HeroUI needs inline styles (check in the built app). Pass the two origins
  as `envsubst` variables at container start (`API_ORIGIN`, `SITE_ORIGIN`), using the official nginx image's
  `/etc/nginx/templates/*.template` mechanism.
- `<meta name="robots" content="noindex">` in `index.html`.
- README «Deployment»: build command with the two args; run command with the two env vars; DNS/TLS note
  (`admin.<domain>` → this container behind the main reverse proxy); the backend must have `PUBLIC_BASE_URL` set to its
  public origin so uploaded images load; the redirect snippet for the main site's nginx:
  `location /manage { return 301 https://admin.<domain>/; }`.

## Tests to write
No new unit tests. By hand: `docker build --build-arg VITE_API_URL=http://localhost:3000/api/v1 -t citypark-admin .`
and `docker run -p 8080:80 -e API_ORIGIN=http://localhost:3000 -e SITE_ORIGIN=http://localhost:3001 citypark-admin`;
open `http://localhost:8080/delivery` directly (SPA fallback) → login page; sign in works; no CSP errors in the
console on the pages of tasks 04–16.

## Done when
CONVENTIONS "Done means" passes and the manual check above succeeds; write the image size under the task line in
`PROGRESS.md`.

## Out of scope
CI, the backend's or the site's deployment, TLS certificates.
