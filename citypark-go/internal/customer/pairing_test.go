package customer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func pairingSetup(t *testing.T) (*customer.Service, http.Handler, string, string, int64, int64) {
	t.Helper()
	db := testutil.NewDB(t)
	c := clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	s := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	dest := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101')`, source)
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,contact_name,contact_phone,items_hash) VALUES($1,$1,'Mini','snap','a')`, source)
	mini, _, err := s.CreateSession(context.Background(), source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	web, _, err := s.CreateSession(context.Background(), dest, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	customer.NewPairingHandler(s, customer.NewLimits(c), true).Routes(r)
	return s, r, mini, web, source, dest
}

func pairingCall(t *testing.T, r http.Handler, method, path, token string, body any, want int) map[string]any {
	t.Helper()
	rec := testutil.DoJSON(t, r, method, path, token, body)
	if rec.Code != want {
		t.Fatalf("%s %s: %d %s, want %d", method, path, rec.Code, rec.Body.String(), want)
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache: %q", rec.Header().Get("Cache-Control"))
	}
	var v map[string]any
	if rec.Body.Len() > 0 && json.Unmarshal(rec.Body.Bytes(), &v) != nil {
		t.Fatal(rec.Body.String())
	}
	return v
}

func TestPairingLifecycle(t *testing.T) {
	s, r, mini, web, source, dest := pairingSetup(t)
	created := pairingCall(t, r, "POST", "/customer/pairings", mini, map[string]any{}, 201)
	id, code := created["id"].(string), created["code"].(string)
	if len(code) != 22 || id == code {
		t.Fatalf("weak code/id: %v", created)
	}
	pending := pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": code}, 200)
	if pending["id"] != id || pending["otherName"] != "Mini" || pending["sourceConfirmed"] != false || pending["destinationConfirmed"] != false {
		t.Fatalf("redeem: %v", pending)
	}
	for _, side := range []struct{ token string }{{mini}, {web}} {
		v := pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", side.token, map[string]any{}, 200)
		if side.token == mini {
			if v["status"] != "pending" {
				t.Fatalf("early merge: %v", v)
			}
			if _, err := s.Authenticate(context.Background(), mini); err != nil {
				t.Fatal(err)
			}
		} else if v["status"] != "completed" {
			t.Fatalf("completion: %v", v)
		}
	}
	if _, err := s.Authenticate(context.Background(), mini); err == nil {
		t.Fatal("source session survived")
	}
	if v := pairingCall(t, r, "GET", "/customer/pairings/"+id, web, nil, 200); v["status"] != "completed" {
		t.Fatal(v)
	}
	pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}, 200)
	pairingCall(t, r, "DELETE", "/customer/pairings/"+id, web, nil, 204)
	_ = source
	_ = dest
}

func TestPairingProofs(t *testing.T) {
	s, r, mini, web, source, _ := pairingSetup(t)
	created := pairingCall(t, r, "POST", "/customer/pairings", mini, map[string]any{}, 201)
	id, code := created["id"].(string), created["code"].(string)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", mini, map[string]any{"code": code}, 403)
	pairingCall(t, r, "GET", "/customer/pairings/"+id, web, nil, 404)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": "bad"}, 400)
	sameAccountWeb, _, err := s.CreateSession(context.Background(), source, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	pairingCall(t, r, "POST", "/customer/pairings/redeem", sameAccountWeb, map[string]any{"code": code}, 409)
	otherMini, _, err := s.CreateSession(context.Background(), source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	pairingCall(t, r, "GET", "/customer/pairings/"+id, otherMini, nil, 404)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": code}, 200)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": code}, 400)
	pairingCall(t, r, "DELETE", "/customer/pairings/"+id, mini, nil, 204)
	pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}, 400)
}

func TestPairingReplacementAndLimits(t *testing.T) {
	_, r, mini, web, _, _ := pairingSetup(t)
	first := pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	second := pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	if first["code"] == second["code"] || first["id"] == second["id"] {
		t.Fatal("random values repeated")
	}
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": first["code"]}, 400)
	for i := 0; i < 3; i++ {
		pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	}
	pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 429)
	pairingCall(t, r, "GET", "/customer/pairings/"+first["id"].(string), mini, nil, 200)
}

func TestPairingConcurrency(t *testing.T) {
	_, r, mini, web, _, _ := pairingSetup(t)
	created := pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	code := created["code"].(string)
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			rec := testutil.DoJSON(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": code})
			results <- rec.Code
		}()
	}
	a, b := <-results, <-results
	if !((a == 200 && b == 400) || (a == 400 && b == 200)) {
		t.Fatalf("redeem race: %d, %d", a, b)
	}
	id := created["id"].(string)
	results = make(chan int, 2)
	for _, token := range []string{mini, web} {
		token := token
		go func() {
			rec := testutil.DoJSON(t, r, "POST", "/customer/pairings/"+id+"/confirm", token, map[string]any{})
			results <- rec.Code
		}()
	}
	a, b = <-results, <-results
	if a != 200 || b != 200 {
		t.Fatalf("confirm race: %d, %d", a, b)
	}
	if v := pairingCall(t, r, "GET", "/customer/pairings/"+id, web, nil, 200); v["status"] != "completed" {
		t.Fatal(v)
	}
}

func TestPairingMergeConflict(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	s := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	dest := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101'),($2,'telegram','202')`, source, dest)
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,contact_name,contact_phone,items_hash) VALUES($1,$1,'Mini','snap','a')`, source)
	miniToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	webToken, _, err := s.CreateSession(ctx, dest, "password", "web")
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
	created, err := s.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemPairing(ctx, created.Code, web); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmPairing(ctx, created.ID, mini); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmPairing(ctx, created.ID, web); err != customer.ErrMergeConflict {
		t.Fatalf("conflict: %v", err)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1`, source) != 1 || testutil.Count(t, db, `SELECT count(*) FROM customer_identities WHERE user_id=$1`, source) != 1 || testutil.Count(t, db, `SELECT count(*) FROM customer_pairings WHERE id=$1 AND completed_at IS NULL AND destination_confirmed_at IS NULL`, created.ID) != 1 {
		t.Fatal("conflict partly committed")
	}
	if _, err := s.Authenticate(ctx, miniToken); err != nil {
		t.Fatal("source session revoked on failed merge")
	}
}

func TestPairingStaleAndWrongSession(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := &movingClock{time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
	s := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	dest := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101')`, source)
	miniToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	mini, err := s.Authenticate(ctx, miniToken)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	webToken, _, err := s.CreateSession(ctx, dest, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	web, err := s.Authenticate(ctx, webToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetPairing(ctx, created.ID, web); err != customer.ErrPairingNotFound {
		t.Fatalf("unrelated session: %v", err)
	}
	c.now = c.now.Add(5*time.Minute + time.Second)
	if _, err := s.RedeemPairing(ctx, created.Code, web); err != customer.ErrPairingInvalid {
		t.Fatalf("expired code: %v", err)
	}
	if _, err := s.GetPairing(ctx, created.ID, mini); err != customer.ErrPairingInvalid {
		t.Fatalf("expired pairing: %v", err)
	}
	freshToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := s.Authenticate(ctx, freshToken)
	if err != nil {
		t.Fatal(err)
	}
	freshPair, err := s.CreatePairing(ctx, fresh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemPairing(ctx, freshPair.Code, web); err != customer.ErrPairingStale {
		t.Fatalf("stale session: %v", err)
	}
}

func TestPairingCodePrivacyAndBudgets(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	s := customer.NewService(db, c)
	id := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101')`, id)
	token, _, err := s.CreateSession(ctx, id, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	mini, err := s.Authenticate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	var hash []byte
	if err := db.GetContext(ctx, &hash, `SELECT code_hash FROM customer_pairings WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if len(hash) != 32 || string(hash) == created.Code {
		t.Fatal("code stored in plaintext")
	}
	limits := customer.NewLimits(c)
	for i := 0; i < 5; i++ {
		if ok, _ := limits.PairCreate(id); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := limits.PairCreate(id); ok || wait != 3600 {
		t.Fatalf("create limit: %v %d", ok, wait)
	}
	for i := 0; i < 20; i++ {
		if ok, _ := limits.PairConfirm(id); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := limits.PairConfirm(id); ok || wait != 900 {
		t.Fatalf("confirm limit: %v %d", ok, wait)
	}
	req := httptest.NewRequest("POST", "/customer/pairings/redeem", nil)
	req.RemoteAddr = "192.0.2.1:5000"
	for i := 0; i < 5; i++ {
		if ok, _ := limits.PairRedeem(id, req); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := limits.PairRedeem(id, req); ok || wait != 900 {
		t.Fatalf("redeem user limit: %v %d", ok, wait)
	}
	other := id + 1
	for i := 0; i < 4; i++ {
		if ok, _ := limits.PairRedeem(other, req); !ok {
			t.Fatal(i)
		}
	}
	if ok, wait := limits.PairRedeem(other+1, req); ok || wait != 900 {
		t.Fatalf("redeem IP limit: %v %d", ok, wait)
	}
}

func TestPairingDestinationFirst(t *testing.T) {
	s, r, mini, web, _, _ := pairingSetup(t)
	created := pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	id := created["id"].(string)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": created["code"]}, 200)
	if v := pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}, 200); v["status"] != "pending" || v["destinationConfirmed"] != true || v["sourceConfirmed"] != false {
		t.Fatal(v)
	}
	if _, err := s.Authenticate(context.Background(), mini); err != nil {
		t.Fatal("merged before mini confirmation")
	}
	if v := pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", mini, map[string]any{}, 200); v["status"] != "completed" {
		t.Fatal(v)
	}
}

func TestPairingAdminRejected(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()
	c := clock.Fixed(time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	s := customer.NewService(db, c)
	source := testutil.InsertUser(t, db, "Mini", "+79990000001", "user")
	dest := testutil.InsertUser(t, db, "Website", "+79990000002", "user")
	mustExec(t, db, `INSERT INTO customer_identities(user_id,provider,subject) VALUES($1,'telegram','101')`, source)
	mustExec(t, db, `INSERT INTO orders(customer_id,account_id,contact_name,contact_phone,items_hash) VALUES($1,$1,'Mini','snap','a')`, source)
	miniToken, _, err := s.CreateSession(ctx, source, "telegram", "mini")
	if err != nil {
		t.Fatal(err)
	}
	webToken, _, err := s.CreateSession(ctx, dest, "password", "web")
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
	created, err := s.CreatePairing(ctx, mini)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemPairing(ctx, created.Code, web); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmPairing(ctx, created.ID, mini); err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, `UPDATE users SET role='admin' WHERE id=$1`, dest)
	if _, err := s.ConfirmPairing(ctx, created.ID, web); err != customer.ErrMergeForbidden {
		t.Fatalf("admin: %v", err)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM orders WHERE account_id=$1`, source) != 1 || testutil.Count(t, db, `SELECT count(*) FROM customer_pairings WHERE id=$1 AND completed_at IS NULL AND destination_confirmed_at IS NULL`, created.ID) != 1 {
		t.Fatal("admin conflict partly committed")
	}
}

func TestPairingCancelAgainstConfirm(t *testing.T) {
	_, r, mini, web, _, _ := pairingSetup(t)
	created := pairingCall(t, r, "POST", "/customer/pairings", mini, nil, 201)
	id := created["id"].(string)
	pairingCall(t, r, "POST", "/customer/pairings/redeem", web, map[string]any{"code": created["code"]}, 200)
	pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", mini, map[string]any{}, 200)
	results := make(chan int, 2)
	go func() {
		results <- testutil.DoJSON(t, r, "POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}).Code
	}()
	go func() { results <- testutil.DoJSON(t, r, "DELETE", "/customer/pairings/"+id, mini, nil).Code }()
	a, b := <-results, <-results
	if !((a == 200 || a == 204 || a == 400 || a == 401) && (b == 200 || b == 204 || b == 400 || b == 401)) {
		t.Fatalf("race statuses: %d, %d", a, b)
	}
	v := pairingCall(t, r, "GET", "/customer/pairings/"+id, web, nil, 200)
	if v["status"] != "completed" && v["status"] != "canceled" {
		t.Fatal(v)
	}
	if v["status"] == "canceled" {
		pairingCall(t, r, "POST", "/customer/pairings/"+id+"/confirm", web, map[string]any{}, 400)
	}
}
