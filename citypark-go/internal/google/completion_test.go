package google

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

var completionTime = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type googleFixture struct {
	db                          *sqlx.DB
	svc                         *Service
	router                      *chi.Mux
	server                      *httptest.Server
	key                         *rsa.PrivateKey
	kid                         string
	nonce, subject, email, name string
	calls                       int
	tokenStatus                 int
	delayPath                   string
}

func fixture(t *testing.T) *googleFixture {
	t.Helper()
	f := &googleFixture{db: testutil.NewDB(t), subject: "google-123", email: "person@example.com", name: "Google User", kid: "one"}
	var err error
	f.key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == f.delayPath {
			time.Sleep(25 * time.Millisecond)
		}
		w.Header().Set("Cache-Control", "max-age=60")
		switch r.URL.Path {
		case "/discovery":
			json.NewEncoder(w).Encode(map[string]any{"issuer": "https://accounts.google.com", "token_endpoint": "http://127.0.0.1:" + r.Host[strings.LastIndex(r.Host, ":")+1:] + "/token", "jwks_uri": "http://127.0.0.1:" + r.Host[strings.LastIndex(r.Host, ":")+1:] + "/jwks"})
		case "/jwks":
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": f.kid, "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes())}}})
		case "/token":
			f.calls++
			if f.tokenStatus != 0 {
				w.WriteHeader(f.tokenStatus)
				return
			}
			_ = r.ParseForm()
			if r.Form.Get("client_secret") != "secret" || r.Form.Get("redirect_uri") != "https://site.example/account/google/callback" || r.Form.Get("code_verifier") == "" {
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"id_token": f.signed(t, nil)})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	c := clock.Fixed(completionTime)
	f.svc = NewService(f.db, c, "client-id", "https://site.example/account/google/callback", "")
	f.svc.ConfigureCompletion("secret", f.server.URL+"/discovery", nil)
	f.router = chi.NewRouter()
	NewHandler(f.svc, customer.NewService(f.db, c), customer.NewLimits(c), true).Routes(f.router)
	return f
}
func (f *googleFixture) signed(t *testing.T, change func(map[string]any)) string {
	t.Helper()
	v := map[string]any{"iss": "https://accounts.google.com", "aud": "client-id", "sub": f.subject, "exp": completionTime.Add(time.Hour).Unix(), "iat": completionTime.Unix(), "nonce": f.nonce, "email": f.email, "email_verified": true, "name": f.name}
	if change != nil {
		change(v)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(v))
	token.Header["kid"] = f.kid
	raw, err := token.SignedString(f.key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func (f *googleFixture) start(t *testing.T, intent, auth string) (string, string) {
	t.Helper()
	rec := testutil.DoJSON(t, f.router, "POST", "/auth/customer/google/start", auth, map[string]string{"intent": intent})
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var v StartResult
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(v.AuthorizationURL)
	f.nonce = u.Query().Get("nonce")
	return u.Query().Get("state"), v.FlowToken
}
func (f *googleFixture) complete(t *testing.T, state, flow, auth string, body map[string]string, want int) map[string]any {
	t.Helper()
	body["state"] = state
	body["flowToken"] = flow
	rec := testutil.DoJSON(t, f.router, "POST", "/auth/customer/google/complete", auth, body)
	if rec.Code != want {
		t.Fatalf("status %d want %d: %s", rec.Code, want, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("cache")
	}
	var v map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	return v
}
func TestGoogleCompletion(t *testing.T) {
	f := fixture(t)
	state, flow := f.start(t, "login", "")
	v := f.complete(t, state, flow, "", map[string]string{"code": "code"}, 200)
	u := v["user"].(map[string]any)
	if u["email"] != "" || u["phone"] != "" || v["token"] == "" {
		t.Fatal(v)
	}
	var count int
	_ = f.db.Get(&count, `SELECT count(*) FROM customer_identities WHERE provider='google' AND subject=$1`, f.subject)
	if count != 1 {
		t.Fatal(count)
	}
	state, flow = f.start(t, "login", "")
	v = f.complete(t, state, flow, "", map[string]string{"code": "code"}, 200)
	if v["user"].(map[string]any)["id"] != u["id"] {
		t.Fatal("new account on repeat login")
	}
	token, _, err := customer.NewService(f.db, clock.Fixed(completionTime)).CreateSession(context.Background(), int64(u["id"].(float64)), "google", "web")
	if err != nil {
		t.Fatal(err)
	}
	state, flow = f.start(t, "link", token)
	v = f.complete(t, state, flow, token, map[string]string{"code": "code"}, 200)
	if v["linked"] != true {
		t.Fatal(v)
	}
	other := testutil.InsertUser(t, f.db, "Other", "+79990000001", "user")
	otherToken, _, _ := customer.NewService(f.db, clock.Fixed(completionTime)).CreateSession(context.Background(), other, "password", "web")
	state, flow = f.start(t, "link", otherToken)
	f.complete(t, state, flow, otherToken, map[string]string{"code": "code"}, 409)
}
func TestGoogleFlowProofs(t *testing.T) {
	f := fixture(t)
	state, flow := f.start(t, "login", "")
	f.complete(t, state, "wrong", "", map[string]string{"code": "code"}, 400)
	f.complete(t, state, flow, "", map[string]string{"error": "access_denied"}, 400)
	f.complete(t, state, flow, "", map[string]string{"code": "code"}, 400)
	if f.calls != 0 {
		t.Fatal(f.calls)
	}
	state, flow = f.start(t, "login", "")
	f.complete(t, state, flow, "", map[string]string{"code": "code"}, 200)
	f.complete(t, state, flow, "", map[string]string{"code": "code"}, 400)
	if f.calls != 1 {
		t.Fatal(f.calls)
	}
}

func TestGoogleProviderFailure(t *testing.T) {
	f := fixture(t)
	f.tokenStatus = 400
	state, flow := f.start(t, "login", "")
	f.complete(t, state, flow, "", map[string]string{"code": "rejected"}, 400)
	f.complete(t, state, flow, "", map[string]string{"code": "rejected"}, 400)
	if f.calls != 1 {
		t.Fatal(f.calls)
	}
	f.tokenStatus = 503
	state, flow = f.start(t, "login", "")
	f.complete(t, state, flow, "", map[string]string{"code": "unavailable"}, 503)
	f.tokenStatus = 0
	state, flow = f.start(t, "login", "")
	f.complete(t, state, flow, "", map[string]string{"code": ""}, 400)
	f.complete(t, state, flow, "", map[string]string{"code": "code", "error": "access_denied"}, 400)
	for i := 0; i < 15; i++ {
		f.complete(t, "bad", "bad", "", map[string]string{"code": "code"}, 400)
	}
	f.complete(t, "bad", "bad", "", map[string]string{"code": "code"}, 429)
}
func TestGoogleLinkSessionAtCommit(t *testing.T) {
	f := fixture(t)
	id := testutil.InsertUser(t, f.db, "Owner", "+79990000009", "user")
	sessions := customer.NewService(f.db, clock.Fixed(completionTime))
	token, _, err := sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	state, flow := f.start(t, "link", token)
	_ = sessions.Revoke(context.Background(), token)
	f.complete(t, state, flow, "", map[string]string{"code": "code"}, 401)
	token, _, _ = sessions.CreateSession(context.Background(), id, "password", "web")
	state, flow = f.start(t, "link", token)
	f.complete(t, state, flow, token, map[string]string{"code": "code"}, 200)
	state, flow = f.start(t, "link", token)
	f.complete(t, state, flow, token, map[string]string{"code": "code"}, 200)
}

func TestGoogleConcurrentClaimAndExpiry(t *testing.T) {
	f := fixture(t)
	state, flow := f.start(t, "login", "")
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := f.svc.consume(context.Background(), state, flow); results <- err }()
	}
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatal(first, second)
	}
	state, flow = f.start(t, "login", "")
	stateHash, _ := proofHash(state)
	if _, err := f.db.Exec(`UPDATE customer_oauth_attempts SET expires_at=$2 WHERE state_hash=$1`, stateHash, completionTime.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	f.complete(t, state, flow, "", map[string]string{"code": "code"}, 400)
}

func TestGoogleProviderTimeout(t *testing.T) {
	for _, path := range []string{"/token", "/jwks"} {
		t.Run(path, func(t *testing.T) {
			f := fixture(t)
			state, flow := f.start(t, "login", "")
			if _, _, err := f.svc.endpoints(context.Background()); err != nil {
				t.Fatal(err)
			}
			f.delayPath = path
			f.svc.httpClient.Timeout = 5 * time.Millisecond
			f.complete(t, state, flow, "", map[string]string{"code": "code"}, 503)
		})
	}
}
