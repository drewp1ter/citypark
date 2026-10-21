# Task 34: OpenAPI document

## Goal
Describe the API in an OpenAPI 3.0 file, serve it, and check with a test that it lists every route of the router.

## Depends on
31.

## Read first
`docs/tasks/CONVENTIONS.md`, `docs/API.md` (the documented endpoints, written task by task — this is the source for the document; `docs/design/api.md` is the original design and may be out of date), `internal/app/app.go`.

## Files to create
- `internal/apidocs/openapi.yaml`
- `internal/apidocs/apidocs.go`
- `internal/apidocs/apidocs_test.go`

## Files to change
- `internal/app/app.go`: mount the docs routes.

No new dependencies. The test parses the YAML with a few lines of string handling (see below), not with a YAML library.

## Spec

### `openapi.yaml`
- `openapi: 3.0.3`, `info.title: Citypark API`, `info.version: 1.0.0`, `servers: [{url: /api/v1}]`.
- `components.securitySchemes.bearerAuth`: `type: http`, `scheme: bearer`, `bearerFormat: JWT`.
- `components.schemas`: `Error`, `User`, `SignInResponse`, `Address`, `AddressInput`, `DeliveryPrice`, `DeliveryPriceInput`, `Image`, `Category`, `Product`, `OrderItemInput`, `CreateOrder`, `CreateOrderResult`, `Order`, `OrderList`, `SyncResult` — fields exactly as in `docs/API.md`.
- `paths`: every endpoint section in `docs/API.md`. For each operation: `summary`, `tags` (one of `auth`, `user`, `addresses`, `delivery-prices`, `catalog`, `orders`, `integrations`), request body schema when there is one, the success response, and the error responses that apply (`400`, `401`, `403`, `404`, `409`, `502`, `503`) referencing `Error`.
- Routes that need a token have `security: [{bearerAuth: []}]`. Say in the `description` whether the role is `user` or `admin`.
- `POST /iiko/update-products` documents the header parameter `X-Update-Token`.
- Write path keys at exactly two spaces of indentation and method keys at four, like this, because the test relies on it:
```yaml
paths:
  /orders/{id}:
    get:
      summary: Get one order
```

### `apidocs.go`
```go
package apidocs

//go:embed openapi.yaml
var spec []byte

// Routes registers GET /openapi.yaml and GET /docs.
func Routes(r chi.Router)
```
- `GET /openapi.yaml` → the file with `Content-Type: application/yaml`.
- `GET /docs` → a small HTML page that loads Swagger UI from a CDN and points it at `openapi.yaml` (relative URL):
```html
<!doctype html>
<html><head><meta charset="utf-8"><title>Citypark API</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({ url: "openapi.yaml", dom_id: "#ui" })</script>
</body></html>
```

Mount it in `app.go` inside the `/api/v1` route group: `apidocs.Routes(r)`.

## Tests to write
`apidocs_test.go`:
- `TestAPIMarkdownCoversAllRoutes`: read `../../docs/API.md`, take only the part below the line `<!-- Endpoint sections start below this line. -->` (the part above holds a format example), collect `METHOD path` from the lines that start with `#### ` (for example `#### POST /addresses`), and compare with the router's routes collected as in `TestSpecCoversAllRoutes`. The two sets must be equal. If an endpoint is missing from `docs/API.md`, add its section as part of this task.
- `TestSpecServed`: `GET /openapi.yaml` → 200, content type `application/yaml`, body starts with `openapi: 3.0`.
- `TestDocsPage`: `GET /docs` → 200, content type starts with `text/html`, body contains `openapi.yaml`.
- `TestSpecCoversAllRoutes`:
  1. Build the real router with `app.New(cfg, testutil.NewDB(t), logger, clock)` and a config from `config.Load` on an empty map.
  2. Collect `METHOD path` pairs with `chi.Walk`, removing the `/api/v1` prefix and skipping `/openapi.yaml`, `/docs` and `OPTIONS`.
  3. Collect `METHOD path` pairs from the YAML: inside the `paths:` block, a line with exactly two leading spaces that starts with `/` and ends with `:` is a path; a following line with exactly four leading spaces whose key is `get`, `post`, `put`, `patch` or `delete` is a method of that path.
  4. The two sets must be equal. On failure print which pairs are missing on each side.

  To avoid an import cycle (`app` imports `apidocs`), put this test in package `apidocs_test`.

## Done when
The commands in CONVENTIONS "Done means" pass, and `http://localhost:3000/api/v1/docs` shows the Swagger UI when the server runs.

## Out of scope
Generating code from the document. Request examples beyond what `api.md` has.
