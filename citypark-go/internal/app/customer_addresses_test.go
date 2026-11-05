package app_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/address"
	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/customer"
	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestAppCustomerAddresses(t *testing.T) {
	db := testutil.NewDB(t)
	a := newApp(t, db, newFakeTelegram(t), customerTestEnv())
	uid := testutil.InsertUser(t, db, "User", "+79990000001", "user")
	sessions := customer.NewService(db, clock.Fixed(time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)))
	token, _, err := sessions.CreateSession(context.Background(), uid, "password", "web")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`SELECT setval(pg_get_serial_sequence('user_addresses','id'),100,false)`); err != nil {
		t.Fatal(err)
	}
	rec := testutil.DoJSON(t, a.Handler, "POST", "/api/v1/customer/addresses", token, address.Fields{City: "Город", Street: "Дом"})
	if rec.Code != 201 || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(rec.Code, rec.Body)
	}
	var out struct {
		Entry   address.Entry `json:"entry"`
		Created bool          `json:"created"`
	}
	testutil.Decode(t, rec, &out)
	if !out.Created || out.Entry.ID != 100 {
		t.Fatal(rec.Body)
	}
	rec = testutil.DoJSON(t, a.Handler, http.MethodGet, "/api/v1/addresses", token, nil)
	if rec.Code != 200 {
		t.Fatal(rec.Code, rec.Body)
	}
	var legacy []address.Address
	testutil.Decode(t, rec, &legacy)
	if len(legacy) != 1 || legacy[0].ID == out.Entry.ID {
		t.Fatal(rec.Body)
	}
	entryPath := "/api/v1/customer/addresses/" + strconv.FormatInt(out.Entry.ID, 10)
	for _, path := range []string{entryPath, entryPath + "/default"} {
		var body any
		if path == entryPath {
			body = address.Fields{Name: "Работа", City: "Город", Street: "Дом 2"}
		}
		rec = testutil.DoJSON(t, a.Handler, "PUT", path, token, body)
		if rec.Code != 200 {
			t.Fatal(rec.Code, rec.Body)
		}
	}
	rec = testutil.DoJSON(t, a.Handler, "DELETE", "/api/v1/customer/addresses/"+strconv.FormatInt(legacy[0].ID, 10), token, nil)
	if rec.Code != 404 {
		t.Fatal(rec.Code, rec.Body)
	}
	rec = testutil.DoJSON(t, a.Handler, "DELETE", entryPath, token, nil)
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatal(rec.Code, rec.Body)
	}
	rec = testutil.DoJSON(t, a.Handler, "GET", "/api/v1/customer/addresses", token, nil)
	if rec.Code != 200 || rec.Body.String() != "{\"addresses\":[]}\n" {
		t.Fatal(rec.Code, rec.Body)
	}
	rec = testutil.DoJSON(t, a.Handler, "POST", "/api/v1/addresses", token, map[string]string{"city": "Город", "address": "Старый"})
	if rec.Code != 201 {
		t.Fatal(rec.Code, rec.Body)
	}
	var old address.Address
	testutil.Decode(t, rec, &old)
	rec = testutil.DoJSON(t, a.Handler, "DELETE", "/api/v1/addresses/"+strconv.FormatInt(old.ID, 10), token, nil)
	if rec.Code != 204 {
		t.Fatal(rec.Code, rec.Body)
	}
}
