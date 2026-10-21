package auth

import (
	"errors"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/golang-jwt/jwt/v5"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// ErrInvalidToken is returned by Parse for every kind of bad token.
var ErrInvalidToken = errors.New("invalid token")

// Claims is the JWT payload.
type Claims struct {
	UserID int64  `json:"id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Tokens issues and parses HS256 tokens.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	clock  clock.Clock
}

func NewTokens(secret string, ttl time.Duration, c clock.Clock) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl, clock: c}
}

// Issue signs an HS256 token with IssuedAt = now and ExpiresAt = now + ttl.
func (t *Tokens) Issue(userID int64, role string) (string, error) {
	now := t.clock.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

// Parse verifies the signature and expiry and returns the claims.
// Any failure returns ErrInvalidToken.
func (t *Tokens) Parse(token string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithTimeFunc(t.clock.Now),
		jwt.WithExpirationRequired())
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.UserID <= 0 || (claims.Role != RoleUser && claims.Role != RoleAdmin) {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
