package google_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/google"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

var startTime = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

func setup(t *testing.T) (*sqlx.DB, *google.Service, *customer.Service, *customer.Limits, *chi.Mux) {
	t.Helper()
	db := testutil.NewDB(t)
	c := clock.Fixed(startTime)
	sessions := customer.NewService(db, c)
	limits := customer.NewLimits(c)
	svc := google.NewService(db, c, "client-id", "https://site.example/account/google/callback", "")
	r := chi.NewRouter()
	google.NewHandler(svc, sessions, limits, true).Routes(r)
	return db, svc, sessions, limits, r
}

func post(t *testing.T, r http.Handler, token string, body any, want int) (google.StartResult, string) {
	t.Helper()
	rec := testutil.DoJSON(t, r, "POST", "/auth/customer/google/start", token, body)
	if rec.Code != want {
		t.Fatalf("status %d, want %d: %s", rec.Code, want, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("missing no-store")
	}
	var result google.StartResult
	if want == 200 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
	return result, rec.Body.String()
}

func TestGoogleStart(t *testing.T) {
	db, _, _, _, r := setup(t)
	a, _ := post(t, r, "", map[string]any{"intent": "login"}, 200)
	b, _ := post(t, r, "", map[string]any{"intent": "login"}, 200)
	u, err := url.Parse(a.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme != "https" || u.Host != "accounts.google.com" || u.Path != "/o/oauth2/v2/auth" || q.Get("scope") != "openid email profile" || q.Get("redirect_uri") != "https://site.example/account/google/callback" || q.Get("response_type") != "code" || q.Get("client_id") != "client-id" || q.Get("code_challenge_method") != "S256" {
		t.Fatal(u)
	}
	if q.Get("state") == "" || q.Get("nonce") == "" || a.FlowToken == "" || a.ExpiresAt != startTime.Add(10*time.Minute) {
		t.Fatal(a)
	}
	bu, _ := url.Parse(b.AuthorizationURL)
	if q.Get("state") == bu.Query().Get("state") || q.Get("nonce") == bu.Query().Get("nonce") || a.FlowToken == b.FlowToken {
		t.Fatal("reused random proof")
	}
	decode := func(raw string) []byte {
		t.Helper()
		v, e := base64.RawURLEncoding.DecodeString(raw)
		if e != nil || len(v) != 32 {
			t.Fatalf("bad proof %q: %v", raw, e)
		}
		return v
	}
	state, nonce, flow := decode(q.Get("state")), decode(q.Get("nonce")), decode(a.FlowToken)
	var saved struct {
		StateHash []byte    `db:"state_hash"`
		NonceHash []byte    `db:"nonce_hash"`
		FlowHash  []byte    `db:"flow_hash"`
		Verifier  string    `db:"pkce_verifier"`
		Intent    string    `db:"intent"`
		Expires   time.Time `db:"expires_at"`
	}
	if err := db.Get(&saved, `SELECT state_hash,nonce_hash,flow_hash,pkce_verifier,intent,expires_at FROM customer_oauth_attempts WHERE state_hash=$1`, sha256Bytes(state)); err != nil {
		t.Fatal(err)
	}
	if string(saved.StateHash) != string(sha256Bytes(state)) || string(saved.NonceHash) != string(sha256Bytes(nonce)) || string(saved.FlowHash) != string(sha256Bytes(flow)) || saved.Intent != "login" || !saved.Expires.Equal(a.ExpiresAt) {
		t.Fatal(saved)
	}
	if len(saved.Verifier) < 43 || len(saved.Verifier) > 128 || strings.Contains(a.AuthorizationURL, saved.Verifier) {
		t.Fatal("verifier exposure")
	}
	if q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(sha256Bytes([]byte(saved.Verifier))) {
		t.Fatal("incorrect PKCE challenge")
	}
}

func sha256Bytes(v []byte) []byte { h := sha256.Sum256(v); return h[:] }

func TestGoogleStartLink(t *testing.T) {
	db, _, sessions, _, r := setup(t)
	id := testutil.InsertUser(t, db, "Ivan", "+79990000001", "user")
	other := testutil.InsertUser(t, db, "Other", "+79990000002", "user")
	token, _, err := sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	post(t, r, "", map[string]any{"intent": "link"}, 401)
	post(t, r, "bad", map[string]any{"intent": "link"}, 401)
	post(t, r, "admin", map[string]any{"intent": "link"}, 401)
	result, _ := post(t, r, token, map[string]any{"intent": "link", "userId": other, "callback": "https://evil.example"}, 200)
	u, _ := url.Parse(result.AuthorizationURL)
	if u.Query().Get("redirect_uri") != "https://site.example/account/google/callback" {
		t.Fatal(u)
	}
	var saved struct {
		UserID      int64  `db:"user_id"`
		SessionHash []byte `db:"session_hash"`
	}
	if err := db.Get(&saved, `SELECT user_id,session_hash FROM customer_oauth_attempts WHERE intent='link'`); err != nil {
		t.Fatal(err)
	}
	hash, _ := customer.HashToken(token)
	if saved.UserID != id || string(saved.SessionHash) != string(hash) {
		t.Fatal(saved)
	}
	mini := &customer.Session{UserID: id, TokenHash: hash, Surface: "mini", AuthenticatedAt: startTime}
	svc := google.NewService(db, clock.Fixed(startTime), "client-id", "https://site.example/account/google/callback", "")
	if _, err := svc.Start(context.Background(), "link", mini); err == nil || err.Error() != "Откройте сайт для подключения Google" {
		t.Fatal(err)
	}
	stale := &customer.Session{UserID: id, TokenHash: hash, Surface: "web", AuthenticatedAt: startTime.Add(-6 * time.Minute)}
	if _, err := svc.Start(context.Background(), "link", stale); err == nil || err.Error() != "Подтвердите вход повторно" {
		t.Fatal(err)
	}
}

func TestGoogleStartLimits(t *testing.T) {
	_, _, _, _, r := setup(t)
	post(t, r, "", map[string]any{"intent": "other"}, 400)
	for range 20 {
		post(t, r, "", map[string]any{"intent": "login"}, 200)
	}
	post(t, r, "", map[string]any{"intent": "login"}, 429)
	db := testutil.NewDB(t)
	c := clock.Fixed(startTime)
	disabled := chi.NewRouter()
	google.NewHandler(google.NewService(db, c, "", "", ""), customer.NewService(db, c), customer.NewLimits(c), false).Routes(disabled)
	_, body := post(t, disabled, "", map[string]any{"intent": "login"}, 503)
	if !strings.Contains(body, "Вход через Google недоступен") {
		t.Fatal(body)
	}
}
