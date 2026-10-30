package customer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestTelegramSessionFailureRollsBack(t *testing.T) {
	db := testutil.NewDB(t)
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte("bot:secret"))
	check := `auth_date=1793361600` + "\n" + `user={"id":888,"first_name":"Rollback"}`
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(check))
	fields := url.Values{"auth_date": {"1793361600"}, "user": {`{"id":888,"first_name":"Rollback"}`}, "hash": {hex.EncodeToString(mac.Sum(nil))}}
	date := time.Unix(1793361600, 0)
	service := NewService(db, clock.Fixed(date))
	service.random = strings.NewReader("")
	router := chi.NewRouter()
	NewTelegramHandler(db, service, NewLimits(clock.Fixed(date)), "bot:secret", true).Routes(router)
	request := func() *httptest.ResponseRecorder {
		return testutil.DoJSON(t, router, http.MethodPost, "/auth/customer/telegram", "", map[string]string{"initData": fields.Encode()})
	}
	first := request()
	if first.Code != 500 {
		t.Fatalf("expected token generation failure: %d %s", first.Code, first.Body.String())
	}
	var users, proofs, sessions int
	for _, tc := range []struct {
		table string
		count *int
	}{{"users", &users}, {"customer_telegram_proofs", &proofs}, {"customer_sessions", &sessions}} {
		if err := db.Get(tc.count, "SELECT count(*) FROM "+tc.table); err != nil {
			t.Fatal(err)
		}
	}
	if users != 0 || proofs != 0 || sessions != 0 {
		t.Fatalf("partial transaction: users=%d proofs=%d sessions=%d", users, proofs, sessions)
	}
	retryRouter := chi.NewRouter()
	NewTelegramHandler(db, NewService(db, clock.Fixed(date)), NewLimits(clock.Fixed(date)), "bot:secret", true).Routes(retryRouter)
	retry := testutil.DoJSON(t, retryRouter, http.MethodPost, "/auth/customer/telegram", "", map[string]string{"initData": fields.Encode()})
	if retry.Code != 200 {
		t.Fatalf("proof was consumed by failed transaction: %d %s", retry.Code, retry.Body.String())
	}
}
