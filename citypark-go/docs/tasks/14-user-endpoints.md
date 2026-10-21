# Task 14: Current user endpoints and admin bootstrap

## Goal
`GET /user`, `PUT /user`, and a function that creates the admin account at startup when it is missing.

## Depends on
11, 13.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/user/store.go`, `internal/auth/middleware.go`, `internal/user/auth_handler.go`.

## Files to create
- `internal/user/handler.go`
- `internal/user/admin.go`
- `internal/user/handler_test.go`

## Spec

### `handler.go`
```go
type Handler struct { /* store *Store, tokens *auth.Tokens */ }

func NewHandler(s *Store, tokens *auth.Tokens) *Handler

// Routes registers GET /user and PUT /user, both behind tokens.RequireRole(auth.RoleUser).
func (h *Handler) Routes(r chi.Router)
```

`GET /user`
- Identity from `auth.FromContext`.
- `store.ByID`. `store.ErrNotFound` → `httpx.Unauthorized()` (the token belongs to a deleted user).
- 200 with `user.Response()`.

`PUT /user`
```go
type updateRequest struct {
    Name  string `json:"name" validate:"required,max=80"`
    Email string `json:"email" validate:"omitempty,email"`
    Phone string `json:"phone" validate:"omitempty,ruphone"`
}
```
- Decode and validate.
- Load the current user (`ErrNotFound` → `httpx.Unauthorized()`).
- Email: lowercased and trimmed; empty → `nil` (removes the email).
- Phone: empty → keep the current phone; otherwise the normalised value.
- `store.Update`. A unique violation → `httpx.NewError(409, "email или телефон уже используется")`.
- 200 with the updated `Response()`.

### `admin.go`
```go
type AdminParams struct{ Email, Password, Phone string }

// EnsureAdmin creates the admin account when no user has this email or this phone.
// It returns true when it created the account.
func EnsureAdmin(ctx context.Context, s *Store, p AdminParams) (created bool, err error)
```
- Normalise the phone with `validate.NormalizePhone`; if it is invalid return an error `invalid admin phone`.
- Email is lowercased and trimmed.
- If `ByEmail` or `ByPhone` finds a user: return `false, nil` and change nothing (do not touch an existing account, even if it is not an admin).
- Otherwise `Create` with name `admin`, role `auth.RoleAdmin` and a bcrypt hash of the password.

## Tests to write
`db := testutil.NewDB(t)`; create users with `testutil.InsertUser`; issue tokens with `tokens.Issue`.

- `TestMe`: 200 and body `{"id":1,"name":"user","email":"","phone":"+79990000001","role":"user"}`.
- `TestMeNoToken`: 401 with body `{"statusCode":401,"message":"Требуется авторизация"}`.
- `TestMeBadToken`: `Bearer bad` → 401.
- `TestMeDeletedUser`: a token for id 999 → 401.
- `TestUpdate`: `{"name":"Новое имя","email":"New@Example.com","phone":"8 999 000 00 02"}` → 200, body has the new name, `new@example.com`, `+79990000002`; the database row matches.
- `TestUpdateKeepsPhone`: a body without `phone` keeps the old phone.
- `TestUpdateRemovesEmail`: user with an email, body with `"email":""` → the stored email is NULL.
- `TestUpdateValidation`: `{"name":""}` → 400 with `errors.name`.
- `TestUpdateConflict`: two users; the first sets the second's phone → 409.
- `TestEnsureAdminCreates`: empty database → `created == true`; the user has role `admin`, and `auth.CheckPassword` accepts the password.
- `TestEnsureAdminExists`: called twice → second call returns false and the table has one row.
- `TestEnsureAdminBadPhone`: phone `123` → error.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
Calling `EnsureAdmin` from `main.go` (task 31). Password change.
