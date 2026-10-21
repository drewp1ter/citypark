// Package auth hashes passwords and issues and parses JWTs.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash (bcrypt.DefaultCost).
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether password matches the bcrypt hash. An empty hash never matches.
func CheckPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// LegacyHash returns the hash format of the old system:
// hex(HMAC-SHA256(key = password, message = empty)).
func LegacyHash(password string) string {
	mac := hmac.New(sha256.New, []byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckLegacyPassword compares LegacyHash(password) with hash in constant time.
// An empty hash never matches.
func CheckLegacyPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(LegacyHash(password))) == 1
}
