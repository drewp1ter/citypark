package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/golang-jwt/jwt/v5"
)

var t0 = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

func newTokens(secret string, at time.Time) *Tokens {
	return NewTokens(secret, time.Hour, clock.Fixed(at))
}

func TestHashAndCheck(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "password123" {
		t.Fatal("hash equals the plain password")
	}
	tests := []struct {
		name, hash, password string
		want                 bool
	}{
		{"right password", hash, "password123", true},
		{"wrong password", hash, "password124", false},
		{"empty hash", "", "password123", false},
		{"not a bcrypt hash", "garbage", "password123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(tt.hash, tt.password); got != tt.want {
				t.Errorf("CheckPassword = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashPasswordTooLong(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 100)); err == nil {
		t.Fatal("err = nil, want an error for a password longer than 72 bytes")
	}
}

func TestLegacyHash(t *testing.T) {
	const known = "498c81ea3c7c52027981967388f9fbce0ae0e3c54deb8eb5b249968c07aaf1c9"
	if got := LegacyHash("password123"); got != known {
		t.Fatalf("LegacyHash = %q, want %q", got, known)
	}
	tests := []struct {
		name, hash, password string
		want                 bool
	}{
		{"right password", known, "password123", true},
		{"wrong password", known, "password124", false},
		{"empty hash", "", "password123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckLegacyPassword(tt.hash, tt.password); got != tt.want {
				t.Errorf("CheckLegacyPassword = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIssueParse(t *testing.T) {
	tk := newTokens("secret", t0)
	token, err := tk.Issue(7, RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	c, err := tk.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != 7 || c.Role != RoleAdmin {
		t.Errorf("claims = {%d %q}, want {7 %q}", c.UserID, c.Role, RoleAdmin)
	}
	if !c.IssuedAt.Time.Equal(t0) {
		t.Errorf("IssuedAt = %v, want %v", c.IssuedAt.Time, t0)
	}
	if want := t0.Add(time.Hour); !c.ExpiresAt.Time.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", c.ExpiresAt.Time, want)
	}
}

func TestParseExpired(t *testing.T) {
	token, err := newTokens("secret", t0).Issue(7, RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTokens("secret", t0.Add(30*time.Minute)).Parse(token); err != nil {
		t.Fatalf("parse before expiry: %v", err)
	}
	_, err = newTokens("secret", t0.Add(2*time.Hour)).Parse(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, err := newTokens("other", t0).Issue(7, RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTokens("secret", t0).Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseGarbage(t *testing.T) {
	tk := newTokens("secret", t0)
	for _, token := range []string{"", "abc", "a.b.c"} {
		if _, err := tk.Parse(token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("Parse(%q) err = %v, want ErrInvalidToken", token, err)
		}
	}
}

func validClaims() Claims {
	return Claims{
		UserID: 7,
		Role:   RoleUser,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(t0),
			ExpiresAt: jwt.NewNumericDate(t0.Add(time.Hour)),
		},
	}
}

func TestParseWrongAlg(t *testing.T) {
	tk := newTokens("secret", t0)
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims()).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, validClaims()).SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	for name, token := range map[string]string{"none": none, "HS512": hs512} {
		if _, err := tk.Parse(token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestParseBadClaims(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Claims)
	}{
		{"unknown role", func(c *Claims) { c.Role = "boss" }},
		{"empty role", func(c *Claims) { c.Role = "" }},
		{"zero user id", func(c *Claims) { c.UserID = 0 }},
		{"negative user id", func(c *Claims) { c.UserID = -3 }},
		{"no expiry", func(c *Claims) { c.ExpiresAt = nil }},
	}
	tk := newTokens("secret", t0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClaims()
			tt.modify(&c)
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte("secret"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tk.Parse(token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
		})
	}
}
