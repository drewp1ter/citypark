package google

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGoogleIDTokenValidation(t *testing.T) {
	f := fixture(t)
	f.nonce = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	hash, _ := proofHash(f.nonce)
	cases := []struct {
		name   string
		change func(map[string]any)
	}{
		{"valid", nil},
		{"wrong issuer", func(v map[string]any) { v["iss"] = "https://evil.example" }},
		{"missing issuer", func(v map[string]any) { delete(v, "iss") }},
		{"wrong audience", func(v map[string]any) { v["aud"] = "other" }},
		{"multiple audience missing azp", func(v map[string]any) { v["aud"] = []string{"client-id", "other"} }},
		{"multiple audience wrong azp", func(v map[string]any) { v["aud"] = []string{"client-id", "other"}; v["azp"] = "other" }},
		{"wrong nonce", func(v map[string]any) { v["nonce"] = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB" }},
		{"missing subject", func(v map[string]any) { delete(v, "sub") }},
		{"expired", func(v map[string]any) { v["exp"] = completionTime.Add(-time.Minute).Unix() }},
		{"future", func(v map[string]any) { v["iat"] = completionTime.Add(time.Minute).Unix() }},
		{"missing issued", func(v map[string]any) { delete(v, "iat") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := f.signed(t, tc.change)
			_, err := f.svc.validateIDToken(context.Background(), raw, hash)
			if (err == nil) != (tc.name == "valid") {
				t.Fatalf("error %v", err)
			}
		})
	}
	raw := f.signed(t, nil)
	replacement := byte('x')
	if raw[len(raw)-8] == replacement {
		replacement = 'y'
	}
	raw = raw[:len(raw)-8] + string(replacement) + raw[len(raw)-7:]
	if _, err := f.svc.validateIDToken(context.Background(), raw, hash); err == nil {
		t.Fatal("accepted tampering")
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "google-123"})
	unsigned.Header["kid"] = "one"
	none, _ := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := f.svc.validateIDToken(context.Background(), none, hash); err == nil {
		t.Fatal("accepted none")
	}
	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "google-123"})
	hs.Header["kid"] = "one"
	hmac, _ := hs.SignedString([]byte("secret"))
	if _, err := f.svc.validateIDToken(context.Background(), hmac, hash); err == nil {
		t.Fatal("accepted HS256")
	}
	unknown := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": "https://accounts.google.com", "aud": "client-id", "sub": "google-123", "exp": completionTime.Add(time.Hour).Unix(), "iat": completionTime.Unix(), "nonce": f.nonce})
	unknown.Header["kid"] = "unknown"
	unknownRaw, _ := unknown.SignedString(f.key)
	if _, err := f.svc.validateIDToken(context.Background(), unknownRaw, hash); !errors.Is(err, errInvalidToken) {
		t.Fatal(err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	bad := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": "https://accounts.google.com", "aud": "client-id", "sub": "google-123", "exp": completionTime.Add(time.Hour).Unix(), "iat": completionTime.Unix(), "nonce": f.nonce})
	bad.Header["kid"] = "one"
	badRaw, _ := bad.SignedString(other)
	if _, err := f.svc.validateIDToken(context.Background(), badRaw, hash); err == nil {
		t.Fatal("accepted different key")
	}
	unverified := f.signed(t, func(v map[string]any) { v["email_verified"] = false })
	claims, err := f.svc.validateIDToken(context.Background(), unverified, hash)
	if err != nil || claims.Email != "" {
		t.Fatal(claims, err)
	}
	f.key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f.kid = "two"
	rotated := f.signed(t, nil)
	if _, err := f.svc.validateIDToken(context.Background(), rotated, hash); err != nil {
		t.Fatalf("rotated key: %v", err)
	}
}

func TestGoogleCacheExpiry(t *testing.T) {
	now := completionTime
	for _, tc := range []struct {
		header string
		want   time.Duration
	}{{"max-age=120", 2 * time.Minute}, {"max-age=999999999999", time.Hour}, {"no-store", 0}, {"no-cache", 0}, {"max-age=0", 0}} {
		h := make(http.Header)
		h.Set("Cache-Control", tc.header)
		if got := cacheExpiry(h, now).Sub(now); got != tc.want {
			t.Fatalf("%s: %v", tc.header, got)
		}
	}
	h := make(http.Header)
	h.Set("Expires", now.Add(10*time.Minute).Format(http.TimeFormat))
	if got := cacheExpiry(h, now); !got.Equal(now.Add(10 * time.Minute)) {
		t.Fatal(got)
	}
}
