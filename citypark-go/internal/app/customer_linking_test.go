package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestCustomerLinkingIntegration(t *testing.T) {
	db := testutil.NewDB(t)
	app := newApp(t, db, newFakeTelegram(t), customerTestEnv())
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC))
	sessions := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	destination := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	other := testutil.InsertUser(t, db, "Other", "+79990000003", "user")
	if _, err := db.Exec(`INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','tg-101'),($2,'email','site@example.com')`, source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET email='site@example.com',is_email_confirmed=true WHERE id=$1`, destination); err != nil {
		t.Fatal(err)
	}
	address := testutil.InsertAddress(t, db, source, "Белореченск", "Гоголя, 61")
	var sourceOrder, destinationOrder, guestOrder int64
	for _, item := range []struct {
		id    *int64
		owner int64
		hash  string
	}{{&sourceOrder, source, "source"}, {&destinationOrder, destination, "destination"}, {&guestOrder, 0, "guest"}} {
		var owner any = item.owner
		if item.owner == 0 {
			owner = nil
		}
		if err := db.Get(&*item.id, `INSERT INTO orders(customer_id,account_id,contact_name,contact_phone,items_hash) VALUES($1,$2,'Guest','+79990000004',$3) RETURNING id`, source, owner, item.hash); err != nil {
			t.Fatal(err)
		}
	}
	mini, _, err := sessions.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	web, _, err := sessions.CreateSession(ctx, destination, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	otherToken, _, err := sessions.CreateSession(ctx, other, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, token string, body any, want int) map[string]any {
		t.Helper()
		rec := testutil.DoJSON(t, app.Handler, method, "/api/v1"+path, token, body)
		if rec.Code != want || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("%s %s: %d %s (cache %q), want %d", method, path, rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"), want)
		}
		var result map[string]any
		if rec.Body.Len() > 0 && json.Unmarshal(rec.Body.Bytes(), &result) != nil {
			t.Fatal(rec.Body.String())
		}
		return result
	}
	if got := call("GET", "/customer/methods", web, nil, 200); fmt.Sprint(got["methods"]) != "[map[provider:email verified:true]]" {
		t.Fatal(got)
	}
	call("GET", fmt.Sprintf("/orders/%d", sourceOrder), web, nil, 404)
	call("GET", fmt.Sprintf("/orders/%d", sourceOrder), otherToken, nil, 404)
	created := call("POST", "/customer/pairings", mini, map[string]any{}, 201)
	id, code := created["id"].(string), created["code"].(string)
	call("POST", "/customer/pairings/redeem", web, map[string]string{"code": code}, 200)
	call("POST", "/customer/pairings/"+id+"/confirm", mini, map[string]any{}, 200)
	completed := call("POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}, 200)
	if completed["status"] != "completed" {
		t.Fatal(completed)
	}
	if _, err := sessions.Authenticate(ctx, mini); err == nil {
		t.Fatal("source session survived merge")
	}
	if got := call("GET", "/customer/methods", web, nil, 200); fmt.Sprint(got["methods"]) != "[map[provider:email verified:true] map[provider:telegram verified:true]]" {
		t.Fatal(got)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1 AND id=ANY($2)`, destination, []int64{sourceOrder, destinationOrder}) != 2 {
		t.Fatal("owned orders did not transfer")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE id=$1 AND account_id IS NULL`, guestOrder) != 1 {
		t.Fatal("guest order was claimed")
	}
	if testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=$1 AND address_id=$2`, destination, address) != 1 {
		t.Fatal("address did not transfer")
	}
	call("GET", fmt.Sprintf("/orders/%d", sourceOrder), otherToken, nil, 404)
	call("GET", fmt.Sprintf("/orders/%d", guestOrder), web, nil, 404)
	call("POST", "/customer/pairings/redeem", web, map[string]string{"code": code}, 400)
}

func TestStaffAuthRegression(t *testing.T) {
	db := testutil.NewDB(t)
	app := newApp(t, db, newFakeTelegram(t), customerTestEnv())
	admin := testutil.InsertUser(t, db, "Admin", "+79990000001", "admin")
	passwordHash, err := auth.HashPassword("admin-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET email='admin@example.com',password_hash=$2 WHERE id=$1`, admin, passwordHash); err != nil {
		t.Fatal(err)
	}
	login := testutil.DoJSON(t, app.Handler, "POST", "/api/v1/auth/sign-in", "", map[string]string{"email": "admin@example.com", "password": "admin-password"})
	if login.Code != 200 {
		t.Fatalf("staff login: %d %s", login.Code, login.Body.String())
	}
	var signed map[string]any
	if err := json.Unmarshal(login.Body.Bytes(), &signed); err != nil {
		t.Fatal(err)
	}
	if signed["id"] != float64(admin) || signed["role"] != "admin" || signed["email"] != "admin@example.com" || signed["name"] != "Admin" || signed["phone"] != "+79990000001" {
		t.Fatal(signed)
	}
	for _, path := range []string{"/api/v1/user", "/api/v1/orders"} {
		rec := testutil.DoJSON(t, app.Handler, "GET", path, signed["token"].(string), nil)
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("staff %s: %d %s", path, rec.Code, rec.Body.String())
		}
		if path == "/api/v1/user" && strings.TrimSpace(rec.Body.String()) != fmt.Sprintf(`{"id":%d,"name":"Admin","email":"admin@example.com","phone":"+79990000001","role":"admin"}`, admin) {
			t.Fatalf("staff profile shape: %s", rec.Body.String())
		}
		if path == "/api/v1/orders" && strings.TrimSpace(rec.Body.String()) != `{"orders":[],"count":0}` {
			t.Fatalf("staff order shape: %s", rec.Body.String())
		}
	}
	tokens := auth.NewTokens("changeme", time.Hour, clock.Fixed(time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)))
	token, err := tokens.Issue(admin, "admin")
	if err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, app.Handler, "GET", "/api/v1/customer/methods", token, nil)
	if rec.Code != 403 || !strings.Contains(rec.Body.String(), "Недостаточно прав") {
		t.Fatalf("methods: %d %s", rec.Code, rec.Body.String())
	}
}

func TestConcurrentLinkingIntegration(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC))
	s := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	dest := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	hash, err := auth.HashPassword("original-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET email='mini@example.com',password_hash=$2,is_email_confirmed=true WHERE id=$1`, source, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'email','mini@example.com'),($1,'telegram','tg-101'),($2,'google','google-101')`, source, dest); err != nil {
		t.Fatal(err)
	}
	miniToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	webToken, _, err := s.CreateSession(ctx, dest, "google", "web")
	if err != nil {
		t.Fatal(err)
	}
	mini, err := s.Authenticate(ctx, miniToken)
	if err != nil {
		t.Fatal(err)
	}
	web, err := s.Authenticate(ctx, webToken)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := s.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemPairing(ctx, pair.Code, web); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmPairing(ctx, pair.ID, mini); err != nil {
		t.Fatal(err)
	}
	proof := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	proofHash, err := customer.HashEmailToken(proof)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO customer_email_tokens(token_hash,user_id,purpose,email,expires_at) VALUES($1,$2,'reset','mini@example.com',$3)`, proofHash, source, c.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		action string
		err    error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	go func() { <-start; _, err := s.ConfirmPairing(ctx, pair.ID, web); results <- outcome{"pair", err} }()
	go func() { <-start; results <- outcome{"reset", s.ResetPassword(ctx, proof, "changed-password")} }()
	close(start)
	a, b := <-results, <-results
	if a.err != nil && b.err != nil {
		t.Fatalf("both operations failed: %v, %v", a, b)
	}
	var merged bool
	if err := db.Get(&merged, `SELECT merged_into_user_id IS NOT NULL FROM users WHERE id=$1`, source); err != nil {
		t.Fatal(err)
	}
	if merged {
		if testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, dest) != 3 {
			t.Fatal("incomplete merge")
		}
		if _, err := s.Authenticate(ctx, miniToken); err == nil {
			t.Fatal("merged source session survived")
		}
	} else {
		if testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, source) != 2 || testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, dest) != 1 {
			t.Fatal("partial merge")
		}
	}
	if err := s.ResetPassword(ctx, proof, "replay-password"); !errors.Is(err, customer.ErrInvalidEmailToken) {
		t.Fatalf("replayed proof: %v", err)
	}
	if _, err := s.RedeemPairing(ctx, pair.Code, web); !errors.Is(err, customer.ErrPairingInvalid) {
		t.Fatalf("replayed pairing code: %v", err)
	}
}

func customerTestEnv() map[string]string {
	return map[string]string{
		"CUSTOMER_AUTH_ENABLED": "true",
		"SMTP_HOST":             "smtp.example",
		"SMTP_USERNAME":         "user",
		"SMTP_PASSWORD":         "secret",
		"SMTP_FROM":             "mail@example.com",
	}
}
