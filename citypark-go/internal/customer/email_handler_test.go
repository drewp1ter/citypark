package customer_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type movingClock struct{ now time.Time }

func (c *movingClock) Now() time.Time { return c.now }

type emailEnv struct {
	db       *sqlx.DB
	clock    *movingClock
	mail     *fakeMailer
	sessions *customer.Service
	router   http.Handler
}

func newEmailEnv(t *testing.T) emailEnv {
	t.Helper()
	db := testutil.NewDB(t)
	c := &movingClock{time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
	s := customer.NewService(db, c)
	m := &fakeMailer{}
	s.ConfigureEmail(m, "https://site.example", nil, nil)
	r := chi.NewRouter()
	customer.NewEmailHandler(db, s, customer.NewLimits(c), true).Routes(r)
	return emailEnv{db, c, m, s, r}
}
func (e emailEnv) post(t *testing.T, path, token string, body any, want int) string {
	t.Helper()
	rec := testutil.DoJSON(t, e.router, http.MethodPost, path, token, body)
	if rec.Code != want {
		t.Fatalf("%s = %d, want %d: %s", path, rec.Code, want, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("missing no-store")
	}
	return rec.Body.String()
}
func proofFromMail(t *testing.T, m *fakeMailer) string {
	t.Helper()
	parts := strings.Split(m.sent[len(m.sent)-1], "|")
	u, err := url.Parse(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("token")
}
func registerBody(email string) map[string]any {
	return map[string]any{"name": "Иван", "email": email, "password": "example-password"}
}
func TestVerifiedRegistration(t *testing.T) {
	e := newEmailEnv(t)
	body := e.post(t, "/auth/customer/register", "", registerBody(" Ivan@Example.com "), 202)
	if !strings.Contains(body, "Если адрес доступен") {
		t.Fatal(body)
	}
	if len(e.mail.sent) != 1 || !strings.Contains(e.mail.sent[0], "/account/verify?token=") {
		t.Fatal(e.mail.sent)
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "ivan@example.com", "password": "example-password"}, 401)
	e.post(t, "/auth/customer/register", "", registerBody("ivan@example.com"), 202)
	if len(e.mail.sent) != 1 {
		t.Fatal("duplicate sent proof")
	}
	proof := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": proof}, 200)
	signed := e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "ivan@example.com", "password": "example-password"}, 200)
	if !strings.Contains(signed, `"token":"cs_`) || !strings.Contains(signed, `"phone":""`) {
		t.Fatal(signed)
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "ivan@example.com", "password": "bad-password"}, 401)
	e.post(t, "/auth/customer/register", "", map[string]any{"name": "Иван", "email": "no-phone@example.com", "phone": "bad", "password": "example-password"}, 400)
}
func TestVerification(t *testing.T) {
	e := newEmailEnv(t)
	e.post(t, "/auth/customer/register", "", registerBody("ivan@example.com"), 202)
	first := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/resend", "", map[string]any{"email": "ivan@example.com"}, 202)
	second := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": first}, 400)
	e.clock.now = e.clock.now.Add(24 * time.Hour)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": second}, 400)
	e.clock.now = e.clock.now.Add(-time.Nanosecond)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": second}, 200)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": second}, 400)
}
func TestVerifiedEmailChange(t *testing.T) {
	e := newEmailEnv(t)
	e.post(t, "/auth/customer/register", "", registerBody("ivan@example.com"), 202)
	proof := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": proof}, 200)
	token, _, err := e.sessions.CreateSession(context.Background(), 1, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := e.sessions.CreateSession(context.Background(), 1, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, "/customer/email/change", token, map[string]any{"email": "new@example.com"}, 202)
	changeProof := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "ivan@example.com", "password": "example-password"}, 200)
	e.post(t, "/customer/email/confirm", other, map[string]any{"token": changeProof}, 200)
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "ivan@example.com", "password": "example-password"}, 401)
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "new@example.com", "password": "example-password"}, 200)
	if _, err := e.sessions.Authenticate(context.Background(), token); err == nil {
		t.Fatal("other session survived")
	}
	if _, err := e.sessions.Authenticate(context.Background(), other); err != nil {
		t.Fatal("confirming session revoked", err)
	}
	e.post(t, "/customer/email/change", "", map[string]any{"email": "x@example.com"}, 401)
	e.clock.now = e.clock.now.Add(6 * time.Minute)
	e.post(t, "/customer/email/change", other, map[string]any{"email": "x@example.com"}, 403)
}
func TestEmailCollisionWithContact(t *testing.T) {
	db := testutil.NewDB(t)
	c := &movingClock{time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
	s := customer.NewService(db, c)
	m := &fakeMailer{}
	s.ConfigureEmail(m, "https://site.example", nil, nil)
	r := chi.NewRouter()
	customer.NewEmailHandler(db, s, customer.NewLimits(c), true).Routes(r)
	email := "occupied@example.com"
	_, err := user.NewStore(db).Create(context.Background(), user.CreateParams{Name: "Contact", Email: &email, Role: auth.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	e := emailEnv{db, c, m, s, r}
	e.post(t, "/auth/customer/register", "", registerBody(email), 202)
	if len(m.sent) != 0 {
		t.Fatal("proof sent to occupied contact")
	}
}
func TestCustomerEmailErrorsAndLimits(t *testing.T) {
	e := newEmailEnv(t)
	e.post(t, "/auth/customer/verification/resend", "", map[string]any{"email": "nobody@example.com"}, 202)
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "nobody@example.com", "password": "bad-password"}, 401)
	e.post(t, "/auth/customer/register", "", map[string]any{"name": "", "email": "invalid", "password": "short"}, 400)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": "bad"}, 400)
	e.mail.err = context.DeadlineExceeded
	e.post(t, "/auth/customer/register", "", registerBody("failed@example.com"), 202)
	e.mail.err = nil
	e.post(t, "/auth/customer/verification/resend", "", map[string]any{"email": "failed@example.com"}, 202)
	for i := 0; i < 9; i++ {
		e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "nobody@example.com", "password": "bad-password"}, 401)
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": "nobody@example.com", "password": "bad-password"}, 429)
	for i := 0; i < 19; i++ {
		e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": "bad"}, 400)
	}
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": "bad"}, 429)
}
func TestEmailChangeConflictAndAccountMatch(t *testing.T) {
	e := newEmailEnv(t)
	e.post(t, "/auth/customer/register", "", registerBody("first@example.com"), 202)
	first := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": first}, 200)
	token, _, err := e.sessions.CreateSession(context.Background(), 1, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, "/customer/email/change", token, map[string]any{"email": "second@example.com"}, 202)
	changeProof := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/register", "", registerBody("second@example.com"), 202)
	second := proofFromMail(t, e.mail)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": second}, 200)
	e.post(t, "/customer/email/confirm", token, map[string]any{"token": changeProof}, 409)
	e.post(t, "/customer/email/confirm", token, map[string]any{"token": changeProof}, 409)
	e.post(t, "/customer/email/change", token, map[string]any{"email": "third@example.com"}, 202)
	third := proofFromMail(t, e.mail)
	secondSession, _, err := e.sessions.CreateSession(context.Background(), 2, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, "/customer/email/confirm", secondSession, map[string]any{"token": third}, 400)
	e.post(t, "/customer/email/confirm", token, map[string]any{"token": third}, 200)
}
func TestCustomerEmailLegacyAndDisabled(t *testing.T) {
	e := newEmailEnv(t)
	ctx := context.Background()
	email := "legacy@example.com"
	u, err := user.NewStore(e.db).Create(ctx, user.CreateParams{Name: "Legacy", Email: &email, Role: auth.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.db.Exec(`UPDATE users SET legacy_password_hash=$2,is_email_confirmed=true WHERE id=$1`, u.ID, auth.LegacyHash("legacy-password"))
	if err != nil {
		t.Fatal(err)
	}
	err = customer.NewStore(e.db).AddIdentity(ctx, customer.Identity{UserID: u.ID, Provider: "email", Subject: email})
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": email, "password": "wrong"}, 401)
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": email, "password": "legacy-password"}, 200)
	got, err := user.NewStore(e.db).ByID(ctx, u.ID)
	if err != nil || got.PasswordHash == "" || got.LegacyPasswordHash != "" {
		t.Fatalf("legacy upgrade: %+v %v", got, err)
	}
	e.post(t, "/auth/customer/verification/resend", "", map[string]any{"email": email}, 202)
	if len(e.mail.sent) != 0 {
		t.Fatal("confirmed account resent proof")
	}
	r := chi.NewRouter()
	customer.NewEmailHandler(e.db, e.sessions, customer.NewLimits(e.clock), false).Routes(r)
	rec := testutil.DoJSON(t, r, http.MethodPost, "/auth/customer/register", "", registerBody("x@example.com"))
	if rec.Code != 503 || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("disabled: %d %s", rec.Code, rec.Body.String())
	}
}
func TestGoogleOnlyEmailDoesNotBecomePassword(t *testing.T) {
	e := newEmailEnv(t)
	ctx := context.Background()
	email := "google@example.com"
	u, err := user.NewStore(e.db).Create(ctx, user.CreateParams{Name: "Google", Email: &email, Role: auth.RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if err = customer.NewStore(e.db).AddIdentity(ctx, customer.Identity{UserID: u.ID, Provider: "google", Subject: "google-subject"}); err != nil {
		t.Fatal(err)
	}
	token, _, err := e.sessions.CreateSession(ctx, u.ID, "google", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]any{"email": email, "password": "example-password"}, 401)
	e.post(t, "/customer/email/change", token, map[string]any{"email": "new@example.com"}, 403)
	if len(e.mail.sent) != 0 {
		t.Fatal("google-only account sent change proof")
	}
}
func TestCustomerEmailValidationAndThrottleHeaders(t *testing.T) {
	e := newEmailEnv(t)
	e.post(t, "/auth/customer/register", "", registerBody("test@example.com"), 202)
	for _, tc := range []struct {
		path string
		body any
	}{{"/auth/customer/verification/resend", map[string]any{"email": "bad"}}, {"/auth/customer/sign-in", map[string]any{"email": "bad", "password": "x"}}, {"/auth/customer/verification/confirm", map[string]any{}}, {"/auth/customer/register", map[string]any{"name": "", "email": "test@example.com", "password": "x"}}} {
		e.post(t, tc.path, "", tc.body, 400)
	}
	for i := 0; i < 20; i++ {
		e.post(t, "/auth/customer/verification/confirm", "", map[string]any{"token": "bad"}, 400)
	}
	rec := testutil.DoJSON(t, e.router, http.MethodPost, "/auth/customer/verification/confirm", "", map[string]any{"token": "bad"})
	if rec.Code != 429 || rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("throttle: %d %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}
