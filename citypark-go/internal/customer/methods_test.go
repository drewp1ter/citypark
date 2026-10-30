package customer_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
	"github.com/go-chi/chi/v5"
)

func TestMethods(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	tokens := auth.NewTokens("secret", time.Hour, c)
	sessions := customer.NewService(db, c)
	store := customer.NewStore(db)
	id := testutil.InsertUser(t, db, "Иван", "+79990000001", "user")
	admin := testutil.InsertUser(t, db, "Admin", "+79990000002", "admin")
	token, _, err := sessions.CreateSession(ctx, id, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	staff, err := tokens.Issue(admin, "admin")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	adapter := &customer.Adapter{Sessions: sessions, Tokens: tokens, Enabled: true}
	customer.NewHandler(user.NewStore(db), store, sessions, adapter).Routes(r)
	check := func(token, want string, status int) {
		t.Helper()
		rec := testutil.DoJSON(t, r, "GET", "/customer/methods", token, nil)
		if rec.Code != status || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("status %d body %s cache %q; want %d %s", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"), status, want)
		}
	}
	check(token, `{"methods":[]}`, 200)
	if _, err := db.Exec(`INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','tg-secret'),($1,'google','google-secret'),($1,'email','private@example.com')`, id); err != nil {
		t.Fatal(err)
	}
	check(token, `{"methods":[{"provider":"email","verified":false},{"provider":"google","verified":true},{"provider":"telegram","verified":true}]}`, 200)
	if _, err := db.Exec(`UPDATE users SET is_email_confirmed=true WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	check(token, `{"methods":[{"provider":"email","verified":true},{"provider":"google","verified":true},{"provider":"telegram","verified":true}]}`, 200)
	for _, provider := range []string{"email", "google", "telegram"} {
		if _, err := db.Exec(`DELETE FROM customer_identities WHERE user_id=$1 AND provider<>$2`, id, provider); err != nil {
			t.Fatal(err)
		}
		check(token, fmt.Sprintf(`{"methods":[{"provider":%q,"verified":true}]}`, provider), 200)
		if _, err := db.Exec(`INSERT INTO customer_identities(user_id,provider,subject) SELECT $1,provider,subject FROM (VALUES ('email','private@example.com'),('google','google-secret'),('telegram','tg-secret')) AS identities(provider,subject) WHERE provider<>$2 ON CONFLICT DO NOTHING`, id, provider); err != nil {
			t.Fatal(err)
		}
	}
	check("", `{"statusCode":401,"message":"Требуется авторизация"}`, 401)
	check("invalid", `{"statusCode":401,"message":"Требуется авторизация"}`, 401)
	check(staff, `{"statusCode":403,"message":"Недостаточно прав"}`, 403)
	adapter.Enabled = false
	check(token, `{"statusCode":503,"message":"Вход временно недоступен"}`, 503)
}
