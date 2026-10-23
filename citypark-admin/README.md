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
