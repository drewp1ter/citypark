package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
)

const wrongCredentialsBody = `{"statusCode":401,"message":"неверный логин или пароль."}`

type authEnv struct {
	db     *sqlx.DB
	tokens *auth.Tokens
	router http.Handler
}

func newAuthEnv(t *testing.T) authEnv {
	t.Helper()
	db := testutil.NewDB(t)
	tokens := auth.NewTokens("test-secret", time.Hour, clock.Fixed(time.Date(2024, 5, 24, 12, 0, 0, 0, time.UTC)))
	r := chi.NewRouter()
	user.NewAuthHandler(user.NewStore(db), tokens).Routes(r)
	return authEnv{db: db, tokens: tokens, router: r}
}

func (e authEnv) post(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.DoJSON(t, e.router, http.MethodPost, path, "", body)
}

func signUpBody() map[string]any {
	return map[string]any{
		"name":     "Иван",
		"email":    "ivan@example.com",
		"phone":    "8 (999) 123-45-67",
		"password": "secret-pass",
	}
}

func errorKeys(t *testing.T, rec *httptest.ResponseRecorder) (message string, keys map[string]bool) {
	t.Helper()
	var body struct {
		Message string            `json:"message"`
		Errors  map[string]string `json:"errors"`
	}
	testutil.Decode(t, rec, &body)
	keys = make(map[string]bool, len(body.Errors))
	for k := range body.Errors {
		keys[k] = true
	}
	return body.Message, keys
}

func TestSignUpCreatesUser(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.post(t, "/auth/sign-up", signUpBody())
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); got != "{}\n" {
		t.Errorf("body = %q, want {}", got)
	}

	u, err := user.NewStore(e.db).ByEmail(context.Background(), "ivan@example.com")
	if err != nil {
		t.Fatalf("ByEmail: %v", err)
	}
	if u.Role != auth.RoleUser || u.Phone != "+79991234567" || u.Name != "Иван" {
		t.Errorf("unexpected user: %+v", u)
	}
	if u.PasswordHash == "" || u.PasswordHash == "secret-pass" {
		t.Errorf("password_hash = %q, want a hash", u.PasswordHash)
	}
	if !auth.CheckPassword(u.PasswordHash, "secret-pass") {
		t.Error("stored hash does not match the password")
	}
}

func TestSignUpWithoutEmail(t *testing.T) {
	e := newAuthEnv(t)
	body := signUpBody()
	delete(body, "email")

	rec := e.post(t, "/auth/sign-up", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}

	u, err := user.NewStore(e.db).ByPhone(context.Background(), "+79991234567")
	if err != nil {
		t.Fatalf("ByPhone: %v", err)
	}
	if u.Email != nil {
		t.Errorf("email = %q, want NULL", *u.Email)
	}
}

func TestSignUpValidation(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.post(t, "/auth/sign-up", map[string]any{"name": "", "phone": "", "password": "", "email": "_"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	msg, keys := errorKeys(t, rec)
	if msg != "Ошибка валидации, пожалуйста проверьте введенные вами данные" {
		t.Errorf("message = %q", msg)
	}
	for _, k := range []string{"name", "email", "phone", "password"} {
		if !keys[k] {
			t.Errorf("errors has no key %q: %v", k, keys)
		}
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM users`); n != 0 {
		t.Errorf("users rows = %d, want 0", n)
	}
}

func TestSignUpDuplicate(t *testing.T) {
	e := newAuthEnv(t)

	for i := range 2 {
		rec := e.post(t, "/auth/sign-up", signUpBody())
		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, body %s", i+1, rec.Code, rec.Body)
		}
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM users`); n != 1 {
		t.Errorf("users rows = %d, want 1", n)
	}
}

func TestSignUpDuplicatePhoneOtherEmail(t *testing.T) {
	e := newAuthEnv(t)
	e.post(t, "/auth/sign-up", signUpBody())

	body := signUpBody()
	body["email"] = "other@example.com"
	rec := e.post(t, "/auth/sign-up", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if n := testutil.Count(t, e.db, `SELECT count(*) FROM users`); n != 1 {
		t.Errorf("users rows = %d, want 1", n)
	}
}

func TestSignUpMalformedJSON(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.post(t, "/auth/sign-up", "not an object")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
}

func TestSignInOK(t *testing.T) {
	e := newAuthEnv(t)
	e.post(t, "/auth/sign-up", signUpBody())

	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "ivan@example.com", "password": "secret-pass"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
		Role  string `json:"role"`
		Token string `json:"token"`
	}
	testutil.Decode(t, rec, &got)
	if got.ID == 0 || got.Name != "Иван" || got.Email != "ivan@example.com" ||
		got.Phone != "+79991234567" || got.Role != "user" || got.Token == "" {
		t.Fatalf("unexpected body: %+v", got)
	}
	claims, err := e.tokens.Parse(got.Token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.UserID != got.ID || claims.Role != auth.RoleUser {
		t.Errorf("claims = %+v, want id %d role user", claims, got.ID)
	}
}

func TestSignInEmailCase(t *testing.T) {
	e := newAuthEnv(t)
	body := signUpBody()
	body["email"] = "Ivan@Example.com"
	e.post(t, "/auth/sign-up", body)

	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "ivan@example.com", "password": "secret-pass"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
}

func TestSignInValidation(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.post(t, "/auth/sign-in", map[string]any{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	_, keys := errorKeys(t, rec)
	if len(keys) != 2 || !keys["email"] || !keys["password"] {
		t.Errorf("error keys = %v, want email and password", keys)
	}
}

func TestSignInWrongPassword(t *testing.T) {
	e := newAuthEnv(t)
	e.post(t, "/auth/sign-up", signUpBody())

	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "ivan@example.com", "password": "wrong-pass"})
	assertWrongCredentials(t, rec)
}

func TestSignInUnknownEmail(t *testing.T) {
	e := newAuthEnv(t)

	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "nobody@example.com", "password": "secret-pass"})
	assertWrongCredentials(t, rec)
}

func TestSignInNoPassword(t *testing.T) {
	e := newAuthEnv(t)
	// A customer created by an order has no email and no password; give it an email
	// but keep both hashes empty.
	_, err := e.db.Exec(`INSERT INTO users (name, email, phone, password_hash, legacy_password_hash)
		VALUES ('Гость', 'guest@example.com', '+79990000000', '', '')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "guest@example.com", "password": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty password: status = %d, body %s", rec.Code, rec.Body)
	}
	rec = e.post(t, "/auth/sign-in", map[string]any{"email": "guest@example.com", "password": "anything"})
	assertWrongCredentials(t, rec)
}

func TestSignInLegacyUpgrade(t *testing.T) {
	e := newAuthEnv(t)
	_, err := e.db.Exec(`INSERT INTO users (name, email, phone, password_hash, legacy_password_hash)
		VALUES ('Старый', 'old@example.com', '+79990000001', '', $1)`, auth.LegacyHash("password123"))
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// A wrong password must not upgrade the account.
	rec := e.post(t, "/auth/sign-in", map[string]any{"email": "old@example.com", "password": "nope-nope"})
	assertWrongCredentials(t, rec)
	var legacy string
	if err := e.db.Get(&legacy, `SELECT legacy_password_hash FROM users WHERE email = 'old@example.com'`); err != nil {
		t.Fatalf("select legacy hash: %v", err)
	}
	if legacy == "" {
		t.Fatal("legacy hash was cleared by a failed sign-in")
	}

	for i := range 2 {
		rec := e.post(t, "/auth/sign-in", map[string]any{"email": "old@example.com", "password": "password123"})
		if rec.Code != http.StatusOK {
			t.Fatalf("sign-in %d: status = %d, body %s", i+1, rec.Code, rec.Body)
		}
		if i > 0 {
			continue
		}
		var row struct {
			Hash   string `db:"password_hash"`
			Legacy string `db:"legacy_password_hash"`
		}
		if err := e.db.Get(&row, `SELECT password_hash, legacy_password_hash FROM users WHERE email = 'old@example.com'`); err != nil {
			t.Fatalf("select hashes: %v", err)
		}
		if row.Legacy != "" {
			t.Errorf("legacy_password_hash = %q, want empty", row.Legacy)
		}
		if !auth.CheckPassword(row.Hash, "password123") {
			t.Error("password_hash does not match the password after the upgrade")
		}
	}
}

func assertWrongCredentials(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got, want map[string]any
	testutil.Decode(t, rec, &got)
	if err := json.Unmarshal([]byte(wrongCredentialsBody), &want); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) || got["statusCode"] != want["statusCode"] || got["message"] != want["message"] {
		t.Errorf("body = %s, want %s", rec.Body, wrongCredentialsBody)
	}
}
