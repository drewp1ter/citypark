# Task 11: Auth middleware

## Goal
A chi middleware that requires a valid Bearer token with a given role and exposes the signed-in user through the request context.

## Depends on
03, 10.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/auth/token.go`, `internal/httpx/httpx.go`.

## Files to create
- `internal/auth/middleware.go`
- `internal/auth/middleware_test.go`

## Spec

```go
package auth

// Identity is the signed-in user of a request.
type Identity struct {
    UserID int64
    Role   string
}

// NewContext returns ctx carrying id. Used by the middleware and by tests.
func NewContext(ctx context.Context, id Identity) context.Context

// FromContext returns the identity stored by RequireRole.
func FromContext(ctx context.Context) (Identity, bool)

// Allows reports whether a user with role `have` may use a route that needs role `need`.
//   need "user":  have "user" or "admin"
//   need "admin": have "admin"
func Allows(have, need string) bool

// RequireRole returns a middleware for routes that need the given role.
func (t *Tokens) RequireRole(role string) func(http.Handler) http.Handler
```

`RequireRole` behaviour, in order:
1. Read the `Authorization` header. It must be `Bearer <token>` (the word `Bearer`, one space, a non-empty token). Otherwise answer with `httpx.WriteError(w, r, httpx.Unauthorized())` and stop.
2. `t.Parse(token)`. On error answer `httpx.Unauthorized()` and stop.
3. If `!Allows(claims.Role, role)` answer `httpx.Forbidden()` and stop.
4. Call the next handler with `r.WithContext(NewContext(r.Context(), Identity{claims.UserID, claims.Role}))`.

Use an unexported struct type as the context key.

## How handlers use it (for later tasks)
```go
func (h *Handler) Routes(r chi.Router) {
    r.Group(func(r chi.Router) {
        r.Use(h.tokens.RequireRole(auth.RoleUser))
        r.Get("/user", httpx.Handle(h.me))
    })
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) error {
    id, ok := auth.FromContext(r.Context())
    if !ok {
        return httpx.Unauthorized()
    }
    ...
}
```

## Tests to write
Build a chi router with one route `GET /x` behind `RequireRole`, whose handler writes the identity as JSON. Use `clock.Fixed` for the `Tokens`.

- `TestAllows`: table for the four role combinations.
- `TestRequireRoleNoHeader`: 401 and body `{"statusCode":401,"message":"Требуется авторизация"}`.
- `TestRequireRoleBadScheme`: `Authorization: Token abc` → 401.
- `TestRequireRoleBadToken`: `Authorization: Bearer garbage` → 401.
- `TestRequireRoleUserOnUserRoute`: 200 and the identity has the right `UserID` and `Role`.
- `TestRequireRoleAdminOnUserRoute`: 200.
- `TestRequireRoleUserOnAdminRoute`: 403 and body `{"statusCode":403,"message":"Недостаточно прав"}`.
- `TestRequireRoleAdminOnAdminRoute`: 200.
- `TestFromContextEmpty`: `FromContext(context.Background())` returns `ok == false`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Loading the user from the database. The middleware trusts the token.
