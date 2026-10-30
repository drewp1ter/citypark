package customer_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
)

const telegramBot = "customer:secret"

var telegramNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func launchFixture(t *testing.T, id string, date time.Time, name string) string {
	t.Helper()
	person := fmt.Sprintf(`{"id":%s,"first_name":%q}`, id, name)
	return signLaunch(url.Values{"auth_date": {fmt.Sprint(date.Unix())}, "user": {person}, "query_id": {"q + &= привет"}, "signature": {"third-party-signature"}})
}
func signLaunch(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+v.Get(k))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(telegramBot))
	proof := hmac.New(sha256.New, secret.Sum(nil))
	proof.Write([]byte(strings.Join(lines, "\n")))
	v.Set("hash", hex.EncodeToString(proof.Sum(nil)))
	return v.Encode()
}

func TestTelegramInitData(t *testing.T) {
	raw := launchFixture(t, "4503599627370495", telegramNow, " Иван +")
	got, err := customer.ValidateTelegramInitData(raw, telegramBot, telegramNow)
	if err != nil || got.ID != (1<<52)-1 || got.Name != "Иван +" || len(got.ProofHash) != 32 {
		t.Fatalf("valid: %+v %v", got, err)
	}
	permuted := strings.Join(reverse(strings.Split(raw, "&")), "&")
	again, err := customer.ValidateTelegramInitData(permuted, telegramBot, telegramNow)
	if err != nil || string(got.ProofHash) != string(again.ProofHash) {
		t.Fatalf("canonical proof: %+v %v", again, err)
	}
	for _, tc := range []struct {
		name, raw, bot string
		now            time.Time
	}{
		{"wrong bot", raw, "staff:secret", telegramNow},
		{"tamper", strings.Replace(raw, "query_id=", "query_id=x", 1), telegramBot, telegramNow},
		{"escape", raw + "&broken=%ZZ", telegramBot, telegramNow},
		{"duplicate", raw + "&user=x", telegramBot, telegramNow},
		{"missing proof", "auth_date=1&user=x", telegramBot, telegramNow},
		{"client id", signLaunch(url.Values{"auth_date": {fmt.Sprint(telegramNow.Unix())}, "id": {"1"}}), telegramBot, telegramNow},
		{"old", raw, telegramBot, telegramNow.Add(5*time.Minute + time.Second)},
		{"future", raw, telegramBot, telegramNow.Add(-31 * time.Second)},
		{"zero id", launchFixture(t, "0", telegramNow, "x"), telegramBot, telegramNow},
		{"large id", launchFixture(t, "4503599627370496", telegramNow, "x"), telegramBot, telegramNow},
		{"string id", launchFixture(t, `"12"`, telegramNow, "x"), telegramBot, telegramNow},
		{"broken json", signLaunch(url.Values{"auth_date": {fmt.Sprint(telegramNow.Unix())}, "user": {"{"}}), telegramBot, telegramNow},
		{"oversize", strings.Repeat("x", (16<<10)+1), telegramBot, telegramNow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := customer.ValidateTelegramInitData(tc.raw, tc.bot, tc.now); err == nil {
				t.Fatal("accepted invalid launch")
			}
		})
	}
	if _, err := customer.ValidateTelegramInitData(raw, telegramBot, telegramNow.Add(5*time.Minute)); err != nil {
		t.Fatalf("five minute boundary: %v", err)
	}
	if _, err := customer.ValidateTelegramInitData(raw, telegramBot, telegramNow.Add(-30*time.Second)); err != nil {
		t.Fatalf("future boundary: %v", err)
	}
}
func reverse(v []string) []string {
	for i, j := 0, len(v)-1; i < j; i, j = i+1, j-1 {
		v[i], v[j] = v[j], v[i]
	}
	return v
}

func telegramRouter(db *customer.TelegramHandler) http.Handler {
	r := chi.NewRouter()
	db.Routes(r)
	return r
}
func TestTelegramSessions(t *testing.T) {
	db := testutil.NewDB(t)
	svc := customer.NewService(db, clock.Fixed(telegramNow))
	h := telegramRouter(customer.NewTelegramHandler(db, svc, customer.NewLimits(clock.Fixed(telegramNow)), telegramBot, true))
	raw := launchFixture(t, "123", telegramNow, "Тест")
	rec := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": raw})
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("sign-in: %d %s", rec.Code, rec.Body.String())
	}
	var answer struct {
		User struct {
			ID                       int64 `json:"id"`
			Name, Email, Phone, Role string
		}
		Token     string
		ExpiresAt time.Time
	}
	testutil.Decode(t, rec, &answer)
	if answer.User.Name != "Тест" || answer.User.Email != "" || answer.User.Phone != "" || answer.User.Role != "user" || !strings.HasPrefix(answer.Token, "cs_") || answer.ExpiresAt != telegramNow.Add(30*24*time.Hour) {
		t.Fatalf("session: %+v", answer)
	}
	if s, err := svc.Authenticate(context.Background(), answer.Token); err != nil || s.Method != "telegram" || s.Surface != "mini" {
		t.Fatalf("stored session: %+v %v", s, err)
	}
	existing := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": launchFixture(t, "123", telegramNow.Add(time.Second), "Changed")})
	var same struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	testutil.Decode(t, existing, &same)
	if existing.Code != 200 || same.User.ID != answer.User.ID {
		t.Fatalf("existing identity: %d %+v", existing.Code, same)
	}
	other := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": launchFixture(t, "124", telegramNow, "Тест")})
	var second struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	testutil.Decode(t, other, &second)
	if second.User.ID == answer.User.ID {
		t.Fatal("matching names merged accounts")
	}
	for _, token := range []string{"", "staff"} {
		whenDisabled := telegramRouter(customer.NewTelegramHandler(db, svc, customer.NewLimits(clock.Fixed(telegramNow)), token, true))
		if rec := testutil.DoJSON(t, whenDisabled, "POST", "/auth/customer/telegram", "", map[string]string{"initData": raw}); token == "" && (rec.Code != 503 || rec.Header().Get("Cache-Control") != "private, no-store") {
			t.Fatalf("disabled: %d %s", rec.Code, rec.Body.String())
		}
	}
	bad := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": "not-valid"})
	if bad.Code != 401 || bad.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("invalid: %d %s", bad.Code, bad.Body.String())
	}
	oversize := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": strings.Repeat("x", (16<<10)+1)})
	if oversize.Code != 400 {
		t.Fatalf("oversize: %d %s", oversize.Code, oversize.Body.String())
	}
	for i := 0; i < 27; i++ {
		testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": "bad"})
	}
	limited := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": "bad"})
	if limited.Code != 429 || limited.Header().Get("Retry-After") == "" {
		t.Fatalf("limit: %d %s", limited.Code, limited.Body.String())
	}
}
func TestTelegramReplay(t *testing.T) {
	db := testutil.NewDB(t)
	h := telegramRouter(customer.NewTelegramHandler(db, customer.NewService(db, clock.Fixed(telegramNow)), customer.NewLimits(clock.Fixed(telegramNow)), telegramBot, true))
	raw := launchFixture(t, "123", telegramNow, "Иван")
	first := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": raw})
	if first.Code != 200 {
		t.Fatalf("first: %d %s", first.Code, first.Body.String())
	}
	second := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": strings.Join(reverse(strings.Split(raw, "&")), "&")})
	if second.Code != 401 {
		t.Fatalf("replay: %d %s", second.Code, second.Body.String())
	}
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": launchFixture(t, "124", telegramNow, "Иван")})
			codes <- rec.Code
		}()
	}
	wg.Wait()
	close(codes)
	winners := 0
	for code := range codes {
		if code == 200 {
			winners++
		} else if code != 401 {
			t.Fatalf("concurrent code %d", code)
		}
	}
	if winners != 1 {
		t.Fatalf("winners = %d", winners)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM customer_sessions WHERE method='telegram'`); err != nil || count != 2 {
		t.Fatalf("sessions: %d %v", count, err)
	}
}
func TestTelegramIdentityRace(t *testing.T) {
	db := testutil.NewDB(t)
	h := telegramRouter(customer.NewTelegramHandler(db, customer.NewService(db, clock.Fixed(telegramNow)), customer.NewLimits(clock.Fixed(telegramNow)), telegramBot, true))
	var wg sync.WaitGroup
	ids := make(chan int64, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := testutil.DoJSON(t, h, "POST", "/auth/customer/telegram", "", map[string]string{"initData": launchFixture(t, "777", telegramNow, fmt.Sprint(i))})
			if rec.Code != 200 {
				t.Errorf("race: %d %s", rec.Code, rec.Body.String())
				return
			}
			var v struct {
				User struct {
					ID int64 `json:"id"`
				} `json:"user"`
			}
			if json.Unmarshal(rec.Body.Bytes(), &v) != nil {
				t.Error("bad JSON")
				return
			}
			ids <- v.User.ID
		}(i)
	}
	wg.Wait()
	close(ids)
	var owner int64
	for id := range ids {
		if owner == 0 {
			owner = id
		} else if id != owner {
			t.Fatal("two owners")
		}
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM users`); err != nil || count != 1 {
		t.Fatalf("orphan users: %d %v", count, err)
	}
}
