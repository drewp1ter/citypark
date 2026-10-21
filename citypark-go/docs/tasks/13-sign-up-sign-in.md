# Task 13: Sign-up and sign-in

## Goal
`POST /auth/sign-up` and `POST /auth/sign-in`.

## Depends on
05, 10, 12.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/user/store.go`, `internal/auth/password.go`, `internal/auth/token.go`.
You also need the signatures of `httpx.DecodeJSON`, `httpx.JSON`, `httpx.NewError`, `validate.Struct`, `validate.NormalizePhone`; look them up in `internal/httpx/httpx.go` and `internal/validate/` when needed.

## Files to create
- `internal/user/auth_handler.go`
- `internal/user/auth_handler_test.go`

## Spec

```go
package user

type AuthHandler struct { /* store *Store, tokens *auth.Tokens */ }

func NewAuthHandler(s *Store, tokens *auth.Tokens) *AuthHandler

// Routes registers POST /auth/sign-up and POST /auth/sign-in.
func (h *AuthHandler) Routes(r chi.Router)
```

### `POST /auth/sign-up`
Request:
```go
type signUpRequest struct {
    Name     string `json:"name" validate:"required,max=80"`
    Email    string `json:"email" validate:"omitempty,email"`
    Phone    string `json:"phone" validate:"required,ruphone"`
    Password string `json:"password" validate:"required,min=8,max=64"`
}
```
Steps:
1. `httpx.DecodeJSON`, then `validate.Struct`.
2. Normalise: `phone, _ = validate.NormalizePhone(req.Phone)`; email = `strings.ToLower(strings.TrimSpace(req.Email))`; an empty email is stored as `nil`.
3. `auth.HashPassword`, then `store.Create` with role `auth.RoleUser`.
4. If the error is a unique violation (email or phone already used), ignore it. This is deliberate: the response must not reveal whether an account exists.
5. Answer `201` with body `{}`.

### `POST /auth/sign-in`
Request:
```go
type signInRequest struct {
    Email    string `json:"email" validate:"required"`
    Password string `json:"password" validate:"required"`
}
```
Steps:
1. Decode and validate.
2. `store.ByEmail(lowercased trimmed email)`. Not found → the 401 below.
3. Check the password:
   - `user.PasswordHash != ""`: `auth.CheckPassword`.
   - otherwise, `user.LegacyPasswordHash != ""`: `auth.CheckLegacyPassword`. If it matches, upgrade the account: `auth.HashPassword` + `store.SetPasswordHash`.
   - otherwise: no match.
4. No match → `httpx.NewError(401, "неверный логин или пароль.")`. Use the same error for an unknown email.
5. `tokens.Issue(user.ID, user.Role)`.
6. Answer `200`:
```json
{ "id": 1, "name": "Иван", "email": "ivan@example.com", "phone": "+79991234567", "role": "user", "token": "<jwt>" }
```
Build it by embedding `Response` in a struct with an extra `Token string` field tagged `json:"token"`.

## Tests to write
Each test: `db := testutil.NewDB(t)`, a `Tokens` with `clock.Fixed`, a chi router with `Routes`, requests through `testutil.DoJSON`.

- `TestSignUpCreatesUser`: 201, body `{}`; the user exists with role `user`, phone `+79991234567` (send it as `8 (999) 123-45-67`), a non-empty `password_hash` that is not the plain password.
- `TestSignUpWithoutEmail`: `email` omitted → 201 and the stored email is NULL.
- `TestSignUpValidation`: body `{"name":"","phone":"","password":"","email":"_"}` → 400, top-level message `Ошибка валидации, пожалуйста проверьте введенные вами данные`, and `errors` has the keys `name`, `email`, `phone`, `password`.
- `TestSignUpDuplicate`: the same request twice → both 201, one row in `users`.
- `TestSignInOK`: sign up, then sign in → 200; the body has `id`, `name`, `email`, `phone`, `role`, `token`, and `tokens.Parse(token)` gives the user's id.
- `TestSignInEmailCase`: sign up with `Ivan@Example.com`, sign in with `ivan@example.com` → 200.
- `TestSignInValidation`: body `{}` → 400 with `errors` keys `email` and `password`.
- `TestSignInWrongPassword` and `TestSignInUnknownEmail`: 401 with body `{"statusCode":401,"message":"неверный логин или пароль."}`.
- `TestSignInLegacyUpgrade`: insert a user with plain SQL: email `old@example.com`, `password_hash = ''`, `legacy_password_hash = auth.LegacyHash("password123")`. Sign in with `password123` → 200. Afterwards `legacy_password_hash` is empty and `auth.CheckPassword(password_hash, "password123")` is true. A second sign-in also gives 200.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
`GET /user`, `PUT /user`, admin bootstrap (task 14). Email or phone confirmation.
