package customer_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
)

const forgotPath = "/auth/customer/password/forgot"
const resetPath = "/auth/customer/password/reset"

func verifiedResetAccount(t *testing.T, e emailEnv, email string) int64 {
	t.Helper()
	e.post(t, "/auth/customer/register", "", registerBody(email), 202)
	e.post(t, "/auth/customer/verification/confirm", "", map[string]string{"token": proofFromMail(t, e.mail)}, 200)
	id, err := customer.NewStore(e.db).VerifiedEmailOwner(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestForgotPasswordPrivacy(t *testing.T) {
	e := newEmailEnv(t)
	verifiedResetAccount(t, e, "known@example.com")
	e.post(t, "/auth/customer/register", "", registerBody("pending@example.com"), 202)
	adminEmail := "admin@example.com"
	admin, err := user.NewStore(e.db).Create(context.Background(), user.CreateParams{Name: "Admin", Email: &adminEmail, Role: auth.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if err := customer.NewStore(e.db).AddIdentity(context.Background(), customer.Identity{UserID: admin.ID, Provider: "email", Subject: adminEmail}); err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{"google", "telegram"} {
		email := provider + "@example.com"
		u, err := user.NewStore(e.db).Create(context.Background(), user.CreateParams{Name: provider, Email: &email, Role: auth.RoleUser})
		if err != nil {
			t.Fatal(err)
		}
		if err := customer.NewStore(e.db).AddIdentity(context.Background(), customer.Identity{UserID: u.ID, Provider: provider, Subject: provider + "-subject"}); err != nil {
			t.Fatal(err)
		}
	}
	start := len(e.mail.sent)
	var response string
	for _, email := range []string{"known@example.com", "unknown@example.com", "pending@example.com", adminEmail, "google@example.com", "telegram@example.com"} {
		body := e.post(t, forgotPath, "", map[string]string{"email": email}, 202)
		if response == "" {
			response = body
		} else if body != response {
			t.Fatalf("different response for %s: %s", email, body)
		}
	}
	if len(e.mail.sent) != start+1 || !strings.Contains(e.mail.sent[start], "/account/reset?token=") {
		t.Fatalf("unexpected mail: %v", e.mail.sent[start:])
	}
	for i := 0; i < 4; i++ {
		if got := e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202); got != response {
			t.Fatal(got)
		}
	}
	if len(e.mail.sent) != start+2 {
		t.Fatal("send budget not shared with registration")
	}
	e.clock.now = e.clock.now.Add(time.Hour)
	e.mail.err = errors.New("SMTP secret")
	if got := e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202); got != response {
		t.Fatal(got)
	}
	var active int
	if err := e.db.Get(&active, `SELECT count(*) FROM customer_email_tokens WHERE purpose='reset' AND consumed_at IS NULL`); err != nil || active != 0 {
		t.Fatalf("failed SMTP left %d active proofs: %v", active, err)
	}
	e.post(t, forgotPath, "", map[string]string{"email": "bad"}, 400)
}

func TestResetPassword(t *testing.T) {
	e := newEmailEnv(t)
	id := verifiedResetAccount(t, e, "known@example.com")
	if _, err := e.db.Exec(`UPDATE users SET legacy_password_hash=$2 WHERE id=$1`, id, auth.LegacyHash("old-legacy")); err != nil {
		t.Fatal(err)
	}
	a, _, err := e.sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := e.sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202)
	proof := proofFromMail(t, e.mail)
	if body := e.post(t, resetPath, "", map[string]string{"token": proof, "password": "new-example-password"}, 204); body != "" {
		t.Fatal(body)
	}
	u, err := user.NewStore(e.db).ByID(context.Background(), id)
	if err != nil || !auth.CheckPassword(u.PasswordHash, "new-example-password") || u.LegacyPasswordHash != "" {
		t.Fatalf("password state: %+v %v", u, err)
	}
	for _, token := range []string{a, b} {
		if _, err := e.sessions.Authenticate(context.Background(), token); !errors.Is(err, customer.ErrInvalidSession) {
			t.Fatalf("session survived: %v", err)
		}
	}
	e.post(t, "/auth/customer/sign-in", "", map[string]string{"email": "known@example.com", "password": "example-password"}, 401)
	e.post(t, "/auth/customer/sign-in", "", map[string]string{"email": "known@example.com", "password": "new-example-password"}, 200)
	e.post(t, resetPath, "", map[string]string{"token": proof, "password": "another-password"}, 400)
}

func TestResetProofFailures(t *testing.T) {
	e := newEmailEnv(t)
	id := verifiedResetAccount(t, e, "known@example.com")
	for _, body := range []any{map[string]string{}, map[string]string{"token": "bad", "password": "short"}} {
		e.post(t, resetPath, "", body, 400)
	}
	e.post(t, resetPath, "", map[string]string{"token": "forged", "password": "new-example-password"}, 400)
	verify, err := e.sessions.IssueEmailToken(context.Background(), id, "verify", "known@example.com")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, resetPath, "", map[string]string{"token": verify, "password": "new-example-password"}, 400)
	e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202)
	expired := proofFromMail(t, e.mail)
	e.clock.now = e.clock.now.Add(time.Hour)
	e.post(t, resetPath, "", map[string]string{"token": expired, "password": "new-example-password"}, 400)
	e.clock.now = e.clock.now.Add(time.Nanosecond)
	e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202)
	changed := proofFromMail(t, e.mail)
	if _, err := e.db.Exec(`UPDATE users SET email='changed@example.com',updated_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	e.post(t, resetPath, "", map[string]string{"token": changed, "password": "new-example-password"}, 400)
	if _, err := e.db.Exec(`UPDATE users SET email='known@example.com',updated_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202)
	proof := proofFromMail(t, e.mail)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- e.sessions.ResetPassword(context.Background(), proof, "new-example-password")
		}()
	}
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, customer.ErrInvalidEmailToken) {
			invalid++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatalf("success=%d invalid=%d", success, invalid)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(e.post(t, resetPath, "", map[string]string{"token": proof, "password": "another-password"}, 400)), &body); err != nil || body["message"] != "Ссылка недействительна или устарела" {
		t.Fatalf("invalid proof response: %v %v", body, err)
	}
	for i := 0; i < 17; i++ {
		e.post(t, resetPath, "", map[string]string{"token": "bad", "password": "new-example-password"}, 400)
	}
	rec := testutil.DoJSON(t, e.router, http.MethodPost, resetPath, "", map[string]string{"token": "bad", "password": "new-example-password"})
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("throttle: %d %v", rec.Code, rec.Header())
	}
}

func TestResetRollback(t *testing.T) {
	e := newEmailEnv(t)
	id := verifiedResetAccount(t, e, "known@example.com")
	session, _, err := e.sessions.CreateSession(context.Background(), id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	e.post(t, forgotPath, "", map[string]string{"email": "known@example.com"}, 202)
	proof := proofFromMail(t, e.mail)
	if _, err := e.db.Exec(`CREATE FUNCTION reject_reset_revoke() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'revoke failed'; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`CREATE TRIGGER reject_reset_revoke BEFORE UPDATE ON customer_sessions FOR EACH ROW EXECUTE FUNCTION reject_reset_revoke()`); err != nil {
		t.Fatal(err)
	}
	if err := e.sessions.ResetPassword(context.Background(), proof, "new-example-password"); err == nil {
		t.Fatal("expected DB failure")
	}
	if _, err := e.db.Exec(`DROP TRIGGER reject_reset_revoke ON customer_sessions`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`DROP FUNCTION reject_reset_revoke()`); err != nil {
		t.Fatal(err)
	}
	u, err := user.NewStore(e.db).ByID(context.Background(), id)
	if err != nil || !auth.CheckPassword(u.PasswordHash, "example-password") {
		t.Fatalf("password changed: %v", err)
	}
	if _, err := e.sessions.Authenticate(context.Background(), session); err != nil {
		t.Fatalf("session revoked: %v", err)
	}
	if err := e.sessions.ResetPassword(context.Background(), proof, "new-example-password"); err != nil {
		t.Fatalf("proof consumed despite rollback: %v", err)
	}
}
