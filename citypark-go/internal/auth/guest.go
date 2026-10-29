package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const guestPurpose = "guest-checkout"

// IssueGuestCheckout creates a purpose-specific proof, never a customer credential.
func (t *Tokens) IssueGuestCheckout() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	now := t.clock.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   hex.EncodeToString(random[:]),
		Audience:  jwt.ClaimStrings{guestPurpose},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
	}).SignedString(t.secret)
}

// GuestCheckoutID verifies a guest proof and returns its random identifier.
func (t *Tokens) GuestCheckoutID(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(t.clock.Now),
		jwt.WithExpirationRequired(), jwt.WithAudience(guestPurpose))
	if err != nil {
		return "", ErrInvalidToken
	}
	random, err := hex.DecodeString(claims.Subject)
	if err != nil || len(random) != 32 {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
