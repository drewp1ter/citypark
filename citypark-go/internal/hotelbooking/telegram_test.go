package hotelbooking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/telegram"
)

func TestBookingTelegramDelivery(t *testing.T) {
	h := newBooking(t)
	ctx := context.Background()
	disabled := createRequest(h.quote(stayRequest()))
	checkCode(t, h.request("POST", "/reservations", h.token, disabled), 201, "")
	if h.count("hotel_booking_telegram_outbox") != 0 {
		t.Fatal("queued an alert while disabled")
	}
	h.service.store = NewStoreWithTelegram(h.db)
	rejected := createRequest(h.quote(stayRequest()))
	h.client.createError = &ProtocolError{409, "ROOM_UNAVAILABLE"}
	checkCode(t, h.request("POST", "/reservations", h.token, rejected), 409, "ROOM_UNAVAILABLE")
	req := createRequest(h.quote(stayRequest()))
	h.client.createError = errors.New("lost response")
	h.client.commitOnError = true
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	if h.count("hotel_booking_telegram_outbox") != 0 {
		t.Fatal("queued an unconfirmed alert")
	}
	m := &testMailer{}
	w := h.worker(m)
	h.clock.now = h.clock.now.Add(time.Minute)
	w.Tick(ctx)
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 200, "")
	if h.count("hotel_booking_telegram_outbox") != 1 || m.calls != 2 {
		t.Fatal("expected one alert and independent email delivery")
	}

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		const want = "Новое бронирование гостиницы\nБронирование: 00000001234\nТип номера: Одноместный с балконом\nЗаезд: 2026-12-01\nВыезд: 2026-12-03\nГостей: 1\nСтоимость: 8000 ₽"
		if r.URL.Path != "/botfake/sendMessage" || len(payload) != 2 || payload["chat_id"] != "-10042" || payload["text"] != want {
			t.Errorf("unexpected Telegram request %s %+v", r.URL.Path, payload)
		}
		if calls.Add(1) == 1 {
			resp.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		resp.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	w.ConfigureTelegram(telegram.NewClient(server.URL, "fake", nil), "-10042")
	w.tickTelegram(ctx)
	w.tickTelegram(ctx)
	if calls.Load() != 1 {
		t.Fatal("ignored retry backoff")
	}

	h.clock.now = h.clock.now.Add(30 * time.Second)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { w.tickTelegram(ctx) })
	}
	workers.Wait()
	w.tickTelegram(ctx)
	var sent bool
	if err := h.db.Get(&sent, "SELECT sent FROM hotel_booking_telegram_outbox"); err != nil || !sent || calls.Load() != 2 {
		t.Fatalf("sent=%v calls=%d err=%v", sent, calls.Load(), err)
	}
}

func TestBookingTelegramConfirmationRollsBack(t *testing.T) {
	h := newBooking(t)
	h.service.store = NewStoreWithTelegram(h.db)
	req := createRequest(h.quote(stayRequest()))
	_, err := h.db.Exec(`ALTER TABLE hotel_booking_telegram_outbox ADD CONSTRAINT test_no_telegram CHECK (false)`)
	if err != nil {
		t.Fatal(err)
	}
	removeConstraint := func() {
		if _, err := h.db.Exec(`ALTER TABLE hotel_booking_telegram_outbox DROP CONSTRAINT IF EXISTS test_no_telegram`); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(removeConstraint)
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	if h.count("hotel_booking_email_outbox") != 0 || h.count("hotel_booking_telegram_outbox") != 0 {
		t.Fatal("confirmation outboxes were partially committed")
	}
	removeConstraint()
	h.clock.now = h.clock.now.Add(time.Minute)
	h.worker(&testMailer{}).Tick(context.Background())
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
	if h.count("hotel_booking_telegram_outbox") != 1 || h.client.calls != 1 {
		t.Fatal("recovery lost the alert or recreated the booking")
	}
}

func TestBookingTelegramMessageFallback(t *testing.T) {
	h := newBooking(t)
	w := h.worker(&testMailer{})
	server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		const want = "Новое бронирование гостиницы\nБронирование: reservation-id\nТип номера: retired-room\nЗаезд: 2026-12-01\nВыезд: 2026-12-03\nГостей: 2\nСтоимость: 8000.50 ₽"
		if payload["text"] != want {
			t.Errorf("unexpected notification %q", payload["text"])
		}
		resp.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w.ConfigureTelegram(telegram.NewClient(server.URL, "fake", nil), "-10042")
	result, err := json.Marshal(Reservation{
		ReservationID: "reservation-id", RoomTypeID: "retired-room",
		CheckIn: "2026-12-01", CheckOut: "2026-12-03", Guests: 2,
		Price: Price{Currency: "RUB", Nights: 2, NightlyPriceKopecks: 400025, TotalKopecks: 800050},
	})
	if err != nil {
		t.Fatal(err)
	}
	row := telegramRow{Result: result}
	if err := w.sendTelegram(context.Background(), row); err != nil {
		t.Fatal(err)
	}
}
