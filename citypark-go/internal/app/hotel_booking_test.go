package app_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/drewp1ter/citypark-go/internal/testutil"
)

func TestHotelBookingWiring(t *testing.T) {
	db := testutil.NewDB(t)
	for _, enabled := range []string{"false", "true"} {
		a := newApp(t, db, newFakeTelegram(t), map[string]string{
			"HOTEL_BOOKING_ENABLED": enabled,
			"ONEC_BOOKING_URL":      "http://127.0.0.1:1/hs/booking/v1",
			"ONEC_BOOKING_USER":     "booking",
			"ONEC_BOOKING_PASSWORD": "private",
			"SMTP_HOST":             "smtp.example",
			"SMTP_USERNAME":         "hotel",
			"SMTP_PASSWORD":         "private",
			"SMTP_FROM":             "hotel@example.com",
		})
		if (a.BookingWorker != nil) != (enabled == "true") {
			t.Fatal("worker enabled state")
		}
		for _, path := range []string{"/api/v1/hotel/quotes", "/api/v1/hotel/reservations"} {
			rec := httptest.NewRecorder()
			a.Handler.ServeHTTP(rec, httptest.NewRequest("POST", path, bytes.NewBufferString("{}")))
			want := 400
			if enabled == "false" {
				want = 503
			}
			if rec.Code != want || rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
			}
		}
	}
}
