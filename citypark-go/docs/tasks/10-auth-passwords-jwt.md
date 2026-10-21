# Task 10: Passwords and JWT

## Goal
Password hashing (bcrypt, plus verification of the legacy hash format) and JWT issue/parse.

## Depends on
09.

## Read first
`docs/tasks/CONVENTIONS.md`, `internal/clock/clock.go`.

## Files to create
- `internal/auth/password.go`
- `internal/auth/token.go`
- `internal/auth/auth_test.go`

Run `go get golang.org/x/crypto/bcrypt github.com/golang-jwt/jwt/v5`.

## Spec

### `password.go`
```go
package auth

// HashPassword returns a bcrypt hash (bcrypt.DefaultCost).
func HashPassword(password string) (string, error)

// CheckPassword reports whether password matches the bcrypt hash. An empty hash never matches.
func CheckPassword(hash, password string) bool

// LegacyHash returns the hash format of the old system:
// hex(HMAC-SHA256(key = password, message = empty)).
func LegacyHash(password string) string

// CheckLegacyPassword compares LegacyHash(password) with hash in constant time.
// An empty hash never matches.
func CheckLegacyPassword(hash, password string) bool
```
`LegacyHash`: `mac := hmac.New(sha256.New, []byte(password)); return hex.EncodeToString(mac.Sum(nil))` (nothing is written to the MAC). Compare with `subtle.ConstantTimeCompare`.

Known value: `LegacyHash("password123")` is
`498c81ea3c7c52027981967388f9fbce0ae0e3c54deb8eb5b249968c07aaf1c9`.

### `token.go`
```go
const (
    RoleUser  = "user"
    RoleAdmin = "admin"
)

// Claims is the JWT payload.
type Claims struct {
    UserID int64  `json:"id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

type Tokens struct { /* secret []byte, ttl time.Duration, clock clock.Clock */ }

func NewTokens(secret string, ttl time.Duration, c clock.Clock) *Tokens

// Issue signs an HS256 token with IssuedAt = now and ExpiresAt = now + ttl.
func (t *Tokens) Issue(userID int64, role string) (string, error)

// Parse verifies the signature and expiry and returns the claims.
// Any failure returns ErrInvalidToken.
func (t *Tokens) Parse(token string) (Claims, error)

var ErrInvalidToken = errors.New("invalid token")
```
`Parse` must use:
```go
jwt.ParseWithClaims(token, &claims, keyFunc,
    jwt.WithValidMethods([]string{"HS256"}),
    jwt.WithTimeFunc(t.clock.Now),
    jwt.WithExpirationRequired())
```
Also return `ErrInvalidToken` when `claims.UserID <= 0` or the role is not `user` or `admin`.

## Tests to write
- `TestHashAndCheck`: a hashed password checks true; a wrong password false; empty hash false.
- `TestLegacyHash`: the known value above; `CheckLegacyPassword` true for it, false for a wrong password and for an empty hash.
- `TestIssueParse`: issue for `(7, "admin")` with a fixed clock, parse it back, `UserID == 7`, `Role == "admin"`.
- `TestParseExpired`: issue with a clock at T and ttl 1h; parse with a `Tokens` whose clock is T+2h (same secret) → `ErrInvalidToken`.
- `TestParseWrongSecret`: token from another secret → `ErrInvalidToken`.
- `TestParseGarbage`: `""`, `"abc"`, `"a.b.c"` → `ErrInvalidToken`.
- `TestParseWrongAlg`: a token signed with `jwt.SigningMethodNone` (`jwt.UnsafeAllowNoneSignatureType` as key) → `ErrInvalidToken`.
- `TestParseBadRole`: a token signed with the right secret but role `boss` → `ErrInvalidToken`.

## Done when
The commands in CONVENTIONS "Done means" pass.

## Out of scope
HTTP middleware (task 11) and user storage (task 12).
