# citypark-admin

Admin panel for «Сити Парк» (Russian UI): delivery prices, orders, menu sync, site content, hotel reports.
A static single-page app that talks to `../citypark-go` with a Bearer JWT. Rules are in `CLAUDE.md`.

```bash
npm run dev          # http://localhost:5173, backend expected at VITE_API_URL (default http://localhost:3000/api/v1)
npm run check        # typecheck + eslint + prettier --check + vitest run; must be clean
npm run api:gen      # regenerate src/shared/api/schema.d.ts from ../citypark-go/internal/apidocs/openapi.yaml
npm run build        # static files in dist/
npm run e2e          # Playwright flows; needs the backend running at VITE_API_URL
```

## Deployment

The admin is static files served by nginx on its own subdomain (`admin.<domain>`). It calls the backend directly with a
Bearer JWT. `VITE_API_URL` and `VITE_SITE_URL` are build-time values; `API_ORIGIN` and `SITE_ORIGIN` are run-time values
for the Content-Security-Policy header (nginx fills them in with `envsubst` at container start).

```bash
docker build \
  --build-arg VITE_API_URL=https://api.<domain>/api/v1 \
  --build-arg VITE_SITE_URL=https://<domain> \
  -t citypark-admin .

docker run -d -p 8080:80 \
  -e API_ORIGIN=https://api.<domain> \
  -e SITE_ORIGIN=https://<domain> \
  citypark-admin
```

`API_ORIGIN` is the scheme and host of `VITE_API_URL` without the path; `SITE_ORIGIN` is `VITE_SITE_URL`. They must match,
or the browser blocks requests and images.

- **DNS and TLS:** point `admin.<domain>` at the main reverse proxy, which terminates TLS and proxies to this container
  (port 80).
- **Uploaded images:** the backend must have `PUBLIC_BASE_URL` set to its public origin, otherwise `/uploads` images do
  not load in the admin.
- **Old admin:** switch it off by hand, then redirect it in the main site's nginx:

```nginx
location /manage { return 301 https://admin.<domain>/; }
```
