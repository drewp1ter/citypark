package google

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultDiscoveryURL = "https://accounts.google.com/.well-known/openid-configuration"

var errInvalidToken = errors.New("invalid Google ID token")
var errProvider = errors.New("Google provider unavailable")
var errProviderRejected = errors.New("Google provider rejected code")

type googleClaims struct {
	jwt.RegisteredClaims
	Nonce           string `json:"nonce"`
	AuthorizedParty string `json:"azp"`
	Email           string `json:"email"`
	EmailVerified   bool   `json:"email_verified"`
	Name            string `json:"name"`
}

type providerCache struct {
	mu                         sync.Mutex
	tokenURL, jwksURL          string
	discoveryExpiry, keyExpiry time.Time
	keys                       map[string]*rsa.PublicKey
}

func cacheExpiry(header http.Header, now time.Time) time.Time {
	for _, part := range strings.Split(header.Get("Cache-Control"), ",") {
		part = strings.TrimSpace(part)
		if part == "no-store" || part == "no-cache" {
			return now
		}
		if strings.HasPrefix(part, "max-age=") {
			seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
			if err == nil && seconds >= 0 {
				return now.Add(time.Duration(min(seconds, 3600)) * time.Second)
			}
		}
	}
	if expires, err := http.ParseTime(header.Get("Expires")); err == nil {
		if expires.Before(now) {
			return now
		}
		if expires.After(now.Add(time.Hour)) {
			return now.Add(time.Hour)
		}
		return expires
	}
	return now.Add(5 * time.Minute)
}
func (s *Service) getJSON(ctx context.Context, method, endpoint string, body io.Reader, contentType string, dest any) (http.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"))) || u.Host == "" {
		return nil, errProvider
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, errProvider
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, errProviderRejected
	}
	if resp.StatusCode != 200 {
		return nil, errProvider
	}
	limited := io.LimitReader(resp.Body, 1<<20)
	if err := json.NewDecoder(limited).Decode(dest); err != nil {
		return nil, errProvider
	}
	return resp.Header, nil
}
func (s *Service) endpoints(ctx context.Context) (string, string, error) {
	p := &s.provider
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tokenURL != "" && s.clock.Now().Before(p.discoveryExpiry) {
		return p.tokenURL, p.jwksURL, nil
	}
	var v struct {
		Issuer string `json:"issuer"`
		Token  string `json:"token_endpoint"`
		JWKS   string `json:"jwks_uri"`
	}
	hdr, err := s.getJSON(ctx, "GET", s.discoveryURL, nil, "", &v)
	if err != nil {
		return "", "", err
	}
	if v.Issuer != "https://accounts.google.com" && v.Issuer != "accounts.google.com" {
		return "", "", errProvider
	}
	if v.Token == "" || v.JWKS == "" {
		return "", "", errProvider
	}
	p.tokenURL, p.jwksURL, p.discoveryExpiry = v.Token, v.JWKS, cacheExpiry(hdr, s.clock.Now())
	p.keys = nil
	return p.tokenURL, p.jwksURL, nil
}
func (s *Service) key(ctx context.Context, kid string, force bool) (*rsa.PublicKey, error) {
	_, uri, err := s.endpoints(ctx)
	if err != nil {
		return nil, err
	}
	p := &s.provider
	p.mu.Lock()
	defer p.mu.Unlock()
	if !force && s.clock.Now().Before(p.keyExpiry) && p.keys != nil {
		if key := p.keys[kid]; key != nil {
			return key, nil
		}
		return nil, errInvalidToken
	}
	var v struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	hdr, err := s.getJSON(ctx, "GET", uri, nil, "", &v)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range v.Keys {
		if item.Kty != "RSA" || item.Kid == "" || (item.Use != "" && item.Use != "sig") || (item.Alg != "" && item.Alg != "RS256") {
			continue
		}
		n, e1 := base64.RawURLEncoding.DecodeString(item.N)
		e, e2 := base64.RawURLEncoding.DecodeString(item.E)
		if e1 != nil || e2 != nil || len(n) < 256 || len(e) == 0 || len(e) > 4 {
			continue
		}
		exponent := new(big.Int).SetBytes(e).Int64()
		if exponent < 3 || exponent > 1<<31-1 || exponent%2 == 0 {
			continue
		}
		keys[item.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}
	}
	p.keys, p.keyExpiry = keys, cacheExpiry(hdr, s.clock.Now())
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errInvalidToken
}
func (s *Service) validateIDToken(ctx context.Context, raw string, nonceHash []byte) (googleClaims, error) {
	var claims googleClaims
	parsed, _, err := new(jwt.Parser).ParseUnverified(raw, &claims)
	if err != nil || parsed.Method.Alg() != "RS256" {
		return googleClaims{}, errInvalidToken
	}
	kid, ok := parsed.Header["kid"].(string)
	if !ok || kid == "" {
		return googleClaims{}, errInvalidToken
	}
	key, err := s.key(ctx, kid, false)
	if errors.Is(err, errInvalidToken) {
		key, err = s.key(ctx, kid, true)
	}
	if err != nil {
		return googleClaims{}, err
	}
	parsed, err = jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}), jwt.WithLeeway(30*time.Second), jwt.WithTimeFunc(s.clock.Now), jwt.WithExpirationRequired(), jwt.WithIssuedAt()).ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) { return key, nil })
	if err != nil || !parsed.Valid {
		return googleClaims{}, errInvalidToken
	}
	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return googleClaims{}, errInvalidToken
	}
	audienceMatches := false
	for _, audience := range claims.Audience {
		if audience == s.clientID {
			audienceMatches = true
		}
	}
	if claims.Subject == "" || len(claims.Audience) == 0 || !audienceMatches {
		return googleClaims{}, errInvalidToken
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return googleClaims{}, errInvalidToken
	}
	if len(claims.Audience) > 1 && claims.AuthorizedParty != s.clientID {
		return googleClaims{}, errInvalidToken
	}
	if claims.AuthorizedParty != "" && claims.AuthorizedParty != s.clientID {
		return googleClaims{}, errInvalidToken
	}
	nonce, err := base64.RawURLEncoding.Strict().DecodeString(claims.Nonce)
	if err != nil || len(nonce) != 32 {
		return googleClaims{}, errInvalidToken
	}
	digest := sha256Sum(nonce)
	if !equalBytes(digest[:], nonceHash) {
		return googleClaims{}, errInvalidToken
	}
	if !claims.EmailVerified {
		claims.Email = ""
	}
	return claims, nil
}
