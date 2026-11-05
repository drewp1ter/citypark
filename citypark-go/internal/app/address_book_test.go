package app_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestAppLegacyAddressBook(t *testing.T) {
	db := testutil.NewDB(t)
	a := newApp(t, db, newFakeTelegram(t), nil)
	uid := testutil.InsertUser(t, db, "Admin", "+79000000001", auth.RoleAdmin)
	tokens := auth.NewTokens("changeme", time.Hour, clock.Fixed(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)))
	token, err := tokens.Issue(uid, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, a.Handler, http.MethodPost, "/api/v1/addresses", token, map[string]string{"city": "Город", "address": "Дом"})
	if rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body)
	}
	var id int64
	if err := db.Get(&id, `SELECT address_id FROM user_addresses WHERE user_id=$1 AND is_default AND street IS NULL`, uid); err != nil {
		t.Fatal("production handler did not use service", err)
	}
	rec = testutil.DoJSON(t, a.Handler, http.MethodDelete, "/api/v1/addresses/"+strconv.FormatInt(id, 10), token, nil)
	if rec.Code != 204 || testutil.Count(t, db, `SELECT count(*) FROM user_addresses WHERE user_id=$1`, uid) != 0 {
		t.Fatal(rec.Code, rec.Body)
	}
	if testutil.Count(t, db, `SELECT count(*) FROM addresses WHERE id=$1`, id) != 1 {
		t.Fatal("shared address removed")
	}
}
