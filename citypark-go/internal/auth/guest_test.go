package auth

import (
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/golang-jwt/jwt/v5"
)

func TestGuestCheckout(t *testing.T) {
	tokens := newTokens("secret", t0)
	proof, err := tokens.IssueGuestCheckout()
	if err != nil {
		t.Fatal(err)
	}
	id, err := tokens.GuestCheckoutID(proof)
	if err != nil || len(id) != 64 {
		t.Fatalf("ID = %q, %v", id, err)
	}
	other, err := tokens.IssueGuestCheckout()
	if err != nil {
		t.Fatal(err)
	}
	otherID, err := tokens.GuestCheckoutID(other)
	if err != nil || id == otherID {
		t.Fatal("guest IDs must be random")
	}
	if _, err := tokens.Parse(proof); err != ErrInvalidToken {
		t.Fatalf("guest as user: %v", err)
	}
	user, err := tokens.Issue(1, RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	expired := NewTokens("secret", time.Hour, clock.Fixed(t0.Add(24*time.Hour)))
	for name, tt := range map[string]struct {
		tokens *Tokens
		proof  string
	}{
		"user": {tokens, user}, "bad signature": {newTokens("other", t0), proof},
		"expired": {expired, proof}, "invalid": {tokens, "garbage"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := tt.tokens.GuestCheckoutID(tt.proof); err != ErrInvalidToken {
				t.Fatalf("err = %v", err)
			}
		})
	}
	for _, subject := range []string{"not-hex", "abcd"} {
		bad, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject: subject, Audience: jwt.ClaimStrings{guestPurpose},
			ExpiresAt: jwt.NewNumericDate(t0.Add(time.Hour)),
		}).SignedString(tokens.secret)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tokens.GuestCheckoutID(bad); err != ErrInvalidToken {
			t.Fatalf("subject %q accepted", subject)
		}
	}
}
