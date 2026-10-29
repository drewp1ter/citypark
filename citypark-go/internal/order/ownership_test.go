package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/drewp1ter/citypark-go/internal/user"
)

func TestCheckoutContactSafety(t *testing.T) {
	for _, role := range []string{auth.RoleUser, auth.RoleAdmin} {
		t.Run(role, func(t *testing.T) {
			e := newSvcEnv(t, friday)
			contact := testutil.InsertUser(t, e.db, "Account name", "+79991112233", role)
			if _, err := e.db.Exec(`UPDATE users SET email = 'private@example.com', password_hash = 'bcrypt-secret', legacy_password_hash = 'legacy-secret' WHERE id = $1`, contact); err != nil {
				t.Fatal(err)
			}
			before, err := user.NewStore(e.db).ByID(context.Background(), contact)
			if err != nil {
				t.Fatal(err)
			}
			in := e.input(true)
			first, err := e.svc.Create(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			after, err := user.NewStore(e.db).ByID(context.Background(), contact)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("checkout changed profile: before=%+v after=%+v err=%v", before, after, err)
			}
			o := e.order(t, first.OrderID)
			if o.AccountID != nil || o.Customer.Name != "Пётр" || o.Customer.Phone != "+79991112233" {
				t.Fatalf("order: %+v", o)
			}
			if n := e.count(t, `SELECT count(*) FROM user_addresses`); n != 0 {
				t.Fatalf("checkout linked %d addresses", n)
			}
			if n := e.count(t, `SELECT count(*) FROM orders WHERE id=$1 AND contact_name='Пётр' AND contact_phone='+79991112233' AND delivery_city='Сочи' AND delivery_address='Ленина, 1'`, o.ID); n != 1 {
				t.Fatal("missing snapshots")
			}
			raw, err := json.Marshal(after)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "PasswordHash") {
				t.Fatalf("credential leak: %s", raw)
			}
			if _, err := e.db.Exec(`UPDATE users SET name='Changed', phone='+79990000999', updated_at=now() WHERE id=$1`, contact); err != nil {
				t.Fatal(err)
			}
			if _, err := e.db.Exec(`UPDATE addresses SET city='Changed city', address='Changed address', updated_at=now() WHERE id=$1`, o.Address.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := e.db.Exec(`DELETE FROM addresses WHERE id=$1`, o.Address.ID); err != nil {
				t.Fatal(err)
			}
			saved := e.order(t, o.ID)
			if saved.Customer != o.Customer || saved.Address == nil || saved.Address.ID != 0 || saved.Address.City != "Сочи" || saved.Address.Address != "Ленина, 1" {
				t.Fatalf("snapshot lost: %+v", saved)
			}
			retry, err := e.svc.Create(context.Background(), in)
			if err != nil || retry.OrderID != first.OrderID {
				t.Fatalf("snapshot retry: %+v, %v", retry, err)
			}
			matches, count, err := NewStore(e.db).List(context.Background(), ListFilter{Query: "Пётр"}, 20, 0)
			if err != nil || count != 1 || matches[0].Customer != o.Customer {
				t.Fatalf("snapshot search: %+v, %d, %v", matches, count, err)
			}
			matches, count, err = NewStore(e.db).List(context.Background(), ListFilter{Query: "79991112233"}, 20, 0)
			if err != nil || count != 1 || matches[0].ID != o.ID {
				t.Fatalf("snapshot phone search: %d, %v", count, err)
			}
			e.payments.valid = true
			if err := e.svc.PaymentConfirmed(context.Background(), PaymentCallback{TransactionID: fmt.Sprint(o.ID), OperationID: "paid", Amount: "749.00"}); err != nil {
				t.Fatal(err)
			}
			if len(e.notifier.pending) != 1 || len(e.notifier.paid) != 1 {
				t.Fatal("unexpected notification count")
			}
			for _, notice := range []Order{e.notifier.pending[0], e.notifier.paid[0]} {
				if notice.Customer != o.Customer || notice.Address.City != "Сочи" || notice.Address.Address != "Ленина, 1" {
					t.Fatalf("notification lost snapshot: %+v", notice)
				}
			}
		})
	}
}

func TestDuplicateIsolation(t *testing.T) {
	e := newSvcEnv(t, friday)
	a := testutil.InsertUser(t, e.db, "A", "+79990000001", auth.RoleUser)
	b := testutil.InsertUser(t, e.db, "B", "+79990000002", auth.RoleUser)
	firstIDs := map[string]int64{}
	for _, tt := range []struct {
		name    string
		account int64
		guest   string
	}{
		{"guest-a", 0, "guest-a"}, {"guest-b", 0, "guest-b"},
		{"account-a", a, "guest-a"}, {"account-b", b, "guest-a"},
		{"unscoped", 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.account != 0 {
				ctx = auth.NewContext(ctx, auth.Identity{UserID: tt.account, Role: auth.RoleUser})
			}
			in := e.input(false)
			in.GuestCheckoutID = tt.guest
			first, err := e.svc.Create(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			for other, id := range firstIDs {
				if id == first.OrderID {
					t.Fatalf("reused %s order", other)
				}
			}
			firstIDs[tt.name] = first.OrderID
			second, err := e.svc.Create(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if (second.OrderID == first.OrderID) != (tt.account != 0 || tt.guest != "") {
				t.Fatalf("retry IDs: %d, %d", first.OrderID, second.OrderID)
			}
			o := e.order(t, first.OrderID)
			if tt.account != 0 && (o.AccountID == nil || *o.AccountID != tt.account) {
				t.Fatal("missing owner")
			}
			if tt.account == 0 && o.AccountID != nil {
				t.Fatal("guest claimed order")
			}
			in.Name = "Changed contact"
			changed, err := e.svc.Create(ctx, in)
			if err != nil || changed.OrderID == first.OrderID {
				t.Fatalf("changed contact reused order: %+v %v", changed, err)
			}
		})
	}
}

func TestOrderOwnership(t *testing.T) {
	e := newReadEnv(t)
	ctx := auth.NewContext(context.Background(), auth.Identity{UserID: e.userID, Role: auth.RoleUser})
	in := e.input(false) // checkout phone is different from the signed-in account
	owned, err := e.svc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := e.svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	o := e.order(t, owned.OrderID)
	if o.Customer.ID == e.userID {
		t.Fatal("test requires independent contact")
	}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	contactToken := issue(t, tokens, o.Customer.ID, auth.RoleUser)
	for _, id := range []int64{owned.OrderID, guest.OrderID} {
		path := fmt.Sprintf("/orders/%d", id)
		wantBody(t, e, "GET", path, contactToken, 404, `{"statusCode":404,"message":"Не найдено"}`)
		wantBody(t, e, "GET", path, e.otherToken, 404, `{"statusCode":404,"message":"Не найдено"}`)
	}
	got := e.getList(t, "/orders/my?pageSize=1", e.userToken)
	if got.Count != 3 || len(got.Orders) != 1 || int64(got.Orders[0]["id"].(float64)) != owned.OrderID {
		t.Fatalf("owner history: %+v", got)
	}
	got = e.getList(t, "/orders/my", contactToken)
	if got.Count != 0 || len(got.Orders) != 0 {
		t.Fatalf("contact received history: %+v", got)
	}
	rec := testutil.DoJSON(t, e.router, "GET", fmt.Sprintf("/orders/%d", owned.OrderID), e.userToken, nil)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "accountId") || strings.Contains(rec.Body.String(), "guest") {
		t.Fatal("internal identity in response")
	}
}

func TestCheckoutAuthentication(t *testing.T) {
	e := newReadEnv(t)
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	for _, tt := range []struct {
		name, token string
		status      int
		owner       bool
	}{
		{"bad", "bad", 401, false},
		{"deleted", issue(t, tokens, 999999, auth.RoleUser), 401, false},
		{"role mismatch", issue(t, tokens, e.userID, auth.RoleAdmin), 401, false},
		{"user", e.userToken, 201, true}, {"admin", e.adminToken, 201, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := e.count(t, `SELECT count(*) FROM orders`)
			rec := testutil.DoJSON(t, e.router, "POST", "/orders", tt.token, orderBody(e.svcEnv, true))
			if rec.Code != tt.status {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("missing no-store")
			}
			if tt.status == 401 {
				if strings.TrimSpace(rec.Body.String()) != unauthorizedBody {
					t.Fatal(rec.Body.String())
				}
				if e.count(t, `SELECT count(*) FROM orders`) != before {
					t.Fatal("unauthorized write")
				}
				return
			}
			var res CreateResult
			testutil.Decode(t, rec, &res)
			o := e.order(t, res.OrderID)
			if tt.owner && (o.AccountID == nil || *o.AccountID != e.userID) {
				t.Fatal("missing owner")
			}
			if !tt.owner && o.AccountID != nil {
				t.Fatal("admin checkout acquired ownership")
			}
		})
	}
	for _, path := range []string{"/orders/my", "/orders/999999", "/orders"} {
		rec := testutil.DoJSON(t, e.router, "GET", path, "", nil)
		if rec.Code != 401 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("%s: status=%d, cache=%q", path, rec.Code, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestGuestCheckoutCookie(t *testing.T) {
	e := newReadEnv(t)
	post := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		body, err := json.Marshal(orderBody(e.svcEnv, true))
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/orders", strings.NewReader(string(body)))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code != 201 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("checkout: %d %s", rec.Code, rec.Body.String())
		}
		return rec
	}
	first := post(nil)
	cookies := first.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %+v", cookies)
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/api/v1/orders" || cookie.MaxAge != 86400 {
		t.Fatalf("cookie: %+v", cookie)
	}
	var original CreateResult
	testutil.Decode(t, first, &original)
	second := post(cookie)
	var retry CreateResult
	testutil.Decode(t, second, &retry)
	if original != retry || len(second.Result().Cookies()) != 0 {
		t.Fatalf("retry: %+v vs %+v", original, retry)
	}
	tokens := auth.NewTokens("secret", time.Hour, clock.Fixed(friday))
	guestID, err := tokens.GuestCheckoutID(cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(guestID))
	if n := e.count(t, `SELECT count(*) FROM orders WHERE id=$1 AND account_id IS NULL AND guest_checkout_hash=$2`, original.OrderID, hex.EncodeToString(hash[:])); n != 1 {
		t.Fatal("guest hash not stored")
	}
	expired, err := auth.NewTokens("secret", time.Hour, clock.Fixed(friday.Add(-25*time.Hour))).IssueGuestCheckout()
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]*http.Cookie{
		"missing": nil, "forged": {Name: cookie.Name, Value: "forged"}, "expired": {Name: cookie.Name, Value: expired},
	} {
		t.Run(name, func(t *testing.T) {
			rec := post(c)
			var got CreateResult
			testutil.Decode(t, rec, &got)
			if got.OrderID == original.OrderID || len(rec.Result().Cookies()) != 1 {
				t.Fatal("invalid cookie reused order")
			}
		})
	}
	req := httptest.NewRequest("GET", fmt.Sprintf("/orders/%d", original.OrderID), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatal("guest cookie grants read access")
	}
	rec = testutil.DoJSON(t, e.router, "GET", fmt.Sprintf("/orders/%d", original.OrderID), cookie.Value, nil)
	if rec.Code != 401 {
		t.Fatal("guest proof authenticates as user")
	}
}

func TestLegacyOrderSnapshots(t *testing.T) {
	db := testutil.NewDB(t)
	sd := seed(t, db)
	var id int64
	if err := db.Get(&id, `INSERT INTO orders (customer_id,address_id,items_hash)
  VALUES ($1,$2,'legacy') RETURNING id`, sd.userID, sd.addressID); err != nil {
		t.Fatal(err)
	}
	o, err := NewStore(db).ByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if o.AccountID != nil || o.Customer.Name != "Иван" || o.Customer.Phone != "+79990000000" ||
		o.Address == nil || o.Address.ID != sd.addressID || o.Address.City != "Москва" || o.Address.Address != "Тверская, 1" {
		t.Fatalf("legacy fallback: %+v", o)
	}
	orders, count, err := NewStore(db).ListByAccount(context.Background(), sd.userID, 20, 0)
	if err != nil || count != 0 || len(orders) != 0 {
		t.Fatalf("legacy contact claimed history: %d, %v", count, err)
	}
}
