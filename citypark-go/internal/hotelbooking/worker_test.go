package hotelbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/room"
	"github.com/drewp1ter/citypark-go/internal/settings"
)

func TestOutboxRecovery(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	m := &testMailer{}
	w := h.worker(m)
	// Two processes race to claim one row; only one owns the lease.
	claim := randomID()
	row, err := h.service.store.claimConfirmation(context.Background(), h.clock.Now(), claim, false)
	if err != nil || row.Key != req.IdempotencyKey || row.Attempts != 1 {
		t.Fatalf("claim %+v %v", row, err)
	}
	w.Tick(context.Background())
	if m.calls != 0 {
		t.Fatal("stole active lease")
	}
	h.clock.now = h.clock.now.Add(time.Minute)
	w.Tick(context.Background())
	if m.calls != 1 || m.to != "Ivan@example.com" {
		t.Fatalf("send %+v", m)
	}
	for _, want := range []string{"Номер бронирования: 00000001234", "Одноместный", "2026-12-01", "2026-12-03", "Гостей: 1", "8000 RUB", "Телефон гостиницы:"} {
		if !strings.Contains(m.body, want) {
			t.Fatalf("missing %q: %s", want, m.body)
		}
	}
	for _, want := range []string{"СИТИ ПАРК", "00000001234", "Одноместный", "2026-12-01", "2026-12-03", "8000 RUB", "Телефон гостиницы", "Предоплата не требуется."} {
		if !strings.Contains(m.html, want) {
			t.Fatalf("HTML confirmation missing %q", want)
		}
	}
	rec := h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil)
	if !strings.Contains(rec.Body.String(), `"confirmationEmailStatus":"sent"`) {
		t.Fatal(rec.Body.String())
	}
	// A late acknowledgement from the old lease cannot revert 'sent'.
	if err = h.service.store.confirmationResult(context.Background(), req.IdempotencyKey, claim, false, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	w.Tick(context.Background())
	if m.calls != 1 || h.client.calls != 1 {
		t.Fatal("duplicate delivery/booking")
	}
}
func TestMailerRetry(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	m := &testMailer{fail: true}
	w := h.worker(m)
	for attempt, delay := range []time.Duration{30 * time.Second, time.Minute, 5 * time.Minute, 30 * time.Minute, 30 * time.Minute} {
		w.Tick(context.Background())
		if m.calls != attempt+1 {
			t.Fatalf("calls %d", m.calls)
		}
		var next time.Time
		if err := h.db.Get(&next, `SELECT next_attempt_at FROM hotel_booking_email_outbox`); err != nil {
			t.Fatal(err)
		}
		if !next.Equal(h.clock.Now().Add(delay)) {
			t.Fatalf("delay %s", next.Sub(h.clock.Now()))
		}
		w.Tick(context.Background())
		if m.calls != attempt+1 {
			t.Fatal("ignored backoff")
		}
		h.clock.now = h.clock.now.Add(delay)
	}
	m.fail = false
	w.Tick(context.Background())
	if m.calls != 6 || h.client.calls != 1 || h.count("hotel_booking_email_outbox") != 1 {
		t.Fatal("email retry caused duplicate")
	}
	if strings.Contains(h.logs.String(), "private smtp") {
		t.Fatal("smtp error leaked")
	}
}
func TestRecoveryLeases(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	h.client.createError = errors.New("outage")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	h.clock.now = h.clock.now.Add(30 * time.Second)
	old, err := h.service.store.claimRecovery(context.Background(), h.clock.Now(), randomID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.service.store.claimRecovery(context.Background(), h.clock.Now(), randomID()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("claimed twice %v", err)
	}
	h.clock.now = h.clock.now.Add(time.Minute)
	fresh, err := h.service.store.claimRecovery(context.Background(), h.clock.Now(), randomID())
	if err != nil {
		t.Fatal(err)
	}
	if old.Claim == fresh.Claim {
		t.Fatal("reused lease token")
	}
	if err = h.service.store.reject(context.Background(), old, "ROOM_UNAVAILABLE"); err != nil {
		t.Fatal(err)
	}
	if err = h.service.store.release(context.Background(), old, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	h.client.createError = nil
	h.service.dispatch(context.Background(), fresh, true)
	// A stale worker cannot change a confirmed result or enqueue another email.
	if err = h.service.store.finish(context.Background(), old, h.client.saved[req.IdempotencyKey], true, h.clock.Now()); err != nil {
		t.Fatal(err)
	}
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
	if h.count("hotel_booking_email_outbox") != 1 {
		t.Fatal("duplicate outbox")
	}
}
func TestConcurrentOutbox(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	// Claim via two independent connections/process-equivalent stores.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewStore(h.db).claimConfirmation(context.Background(), h.clock.Now(), randomID(), false)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	claimed, empty := 0, 0
	for err := range results {
		switch {
		case err == nil:
			claimed++
		case errors.Is(err, sql.ErrNoRows):
			empty++
		default:
			t.Fatal(err)
		}
	}
	if claimed != 1 || empty != 1 {
		t.Fatalf("claims %d empty %d", claimed, empty)
	}
}
func TestWorkerRunCancellation(t *testing.T) {
	h := newBooking(t)
	w := h.worker(&testMailer{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.Run(ctx)
	done := make(chan struct{})
	ctx, cancel = context.WithCancel(context.Background())
	go func() { defer close(done); w.Run(ctx) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
func TestRecoveryOutageAndOriginalSnapshot(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	h.client.createError = errors.New("transport")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	h.clock.now = h.clock.now.Add(time.Minute)
	h.client.lookupError = errors.New("upstream")
	h.worker(&testMailer{}).Tick(context.Background())
	if h.client.calls != 1 {
		t.Fatal("resent without a trustworthy lookup")
	}
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 202, "")
	// Lookup success with a different snapshot cannot confirm our persisted attempt.
	h.client.lookupError = nil
	h.client.mu.Lock()
	saved := h.client.requests[0]
	h.client.saved[req.IdempotencyKey] = Reservation{randomID(), "00000001234", "confirmed", saved.RoomTypeID, saved.CheckIn, saved.CheckOut, saved.Guests, Price{"RUB", 2, 500000, 1000000}}
	h.client.mu.Unlock()
	h.clock.now = h.clock.now.Add(time.Minute)
	h.worker(&testMailer{}).Tick(context.Background())
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 202, "")
	if h.count("hotel_booking_email_outbox") != 0 {
		t.Fatal("emailed wrong snapshot")
	}
}

type contactFake struct {
	value settings.Settings
	err   error
}

func (c contactFake) Get(context.Context) (settings.Settings, time.Time, error) {
	return c.value, time.Time{}, c.err
}
func TestEmailPrerequisiteFailures(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	row, err := h.service.store.claimConfirmation(context.Background(), h.clock.Now(), randomID(), false)
	if err != nil {
		t.Fatal(err)
	}
	w := h.worker(&testMailer{})
	for _, bad := range []confirmationRow{{Canonical: "{"}, {Canonical: row.Canonical, Result: []byte("{")}} {
		if w.send(context.Background(), bad) == nil {
			t.Fatal("accepted corrupt data")
		}
	}
	w.contacts = contactFake{err: errors.New("db")}
	if w.send(context.Background(), row) == nil {
		t.Fatal("missing contact accepted")
	}
	w.contacts = contactFake{}
	h.service.rooms = roomFake{err: errors.New("missing")}
	if w.send(context.Background(), row) == nil {
		t.Fatal("missing room accepted")
	}
	h.service.rooms = room.NewStore(h.db)
	if w.send(context.Background(), row) == nil {
		t.Fatal("missing hotel phone accepted")
	}
}
func TestStorageFailures(t *testing.T) {
	h := newBooking(t)
	q := h.quote(stayRequest())
	req := createRequest(q)
	// A read failure before an attempt is persisted is a bounded 503.
	tx, err := h.db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	bad := NewStore(tx)
	original := h.service.store
	h.service.store = bad
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 503, "BOOKING_UNAVAILABLE")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 503, "BOOKING_UNAVAILABLE")
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 503, "BOOKING_UNAVAILABLE")
	w := NewWorker(h.service, &testMailer{}, contactFake{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	w.Tick(context.Background())
	h.service.store = original
	h.service.rooms = roomFake{err: errors.New("db")}
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 503, "BOOKING_UNAVAILABLE")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 503, "BOOKING_UNAVAILABLE")
	h.service.rooms = room.NewStore(h.db)
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	var a attemptRow
	a.State = "confirmed"
	a.Result = []byte("{")
	if _, err = response(a); err == nil {
		t.Fatal("accepted invalid result")
	}
	a.ContextHash = digest(h.token)
	a.Canonical = "{"
	if _, err = h.service.replay(context.Background(), a, a.ContextHash, &req); err == nil {
		t.Fatal("accepted invalid canonical")
	}
	// Corrupted persisted request is never resent with fabricated/default fields.
	a.Key = req.IdempotencyKey
	a.Claim.String = randomID()
	h.service.dispatch(context.Background(), a, true)
	raw, _ := json.Marshal(guestRequest())
	if exactFields(raw, new(Guest)) != nil {
		t.Fatal("valid shape rejected")
	}
}

func TestLostLocalConfirmationCommit(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	_, err := h.db.Exec(`CREATE FUNCTION test_hotel_outbox_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected outbox failure'; END $$;
 CREATE TRIGGER test_hotel_outbox_failure BEFORE INSERT ON hotel_booking_email_outbox FOR EACH ROW EXECUTE FUNCTION test_hotel_outbox_failure()`)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_, err := h.db.Exec(`DROP TRIGGER IF EXISTS test_hotel_outbox_failure ON hotel_booking_email_outbox; DROP FUNCTION IF EXISTS test_hotel_outbox_failure()`)
		if err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(cleanup)
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	if h.count("hotel_booking_email_outbox") != 0 || len(h.client.saved) != 1 {
		t.Fatal("failed transaction not rolled back")
	}
	cleanup()
	h.clock.now = h.clock.now.Add(time.Minute)
	h.worker(&testMailer{}).Tick(context.Background())
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
	if h.client.calls != 1 || h.count("hotel_booking_email_outbox") != 1 {
		t.Fatal("reconciliation duplicated booking or lost email")
	}
}

type testSMS struct {
	calls          int
	phone, message string
	fail           bool
}

func (s *testSMS) Send(_ context.Context, phone, message string) error {
	s.calls++
	s.phone, s.message = phone, message
	if s.fail {
		return errors.New("SMS provider unavailable")
	}
	return nil
}

func TestSMSConfirmationRetry(t *testing.T) {
	h := newBooking(t)
	h.service.EnableCancellation("https://hotel.example", strings.Repeat("x", 32))
	req := createRequest(h.quote(stayRequest()))
	req.Guest.Email = "  "
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 200, "")

	mail := &testMailer{}
	sms := &testSMS{fail: true}
	w := h.worker(mail)
	w.ConfigureSMS(sms)
	w.Tick(context.Background())
	rec := h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil)
	if !strings.Contains(rec.Body.String(), `"confirmationSmsStatus":"pending"`) {
		t.Fatal(rec.Body.String())
	}
	sms.fail = false
	h.clock.now = h.clock.now.Add(30 * time.Second)
	w.Tick(context.Background())
	w.Tick(context.Background())
	if sms.calls != 2 || mail.calls != 0 || h.client.calls != 1 ||
		!strings.HasPrefix(sms.phone, "+") ||
		!strings.Contains(sms.message, "00000001234") ||
		!strings.Contains(sms.message, "https://hotel.example/booking/cancel#") {
		t.Fatal("expected one booking and two SMS attempts with the booking number and cancellation link")
	}
	rec = h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil)
	if !strings.Contains(rec.Body.String(), `"confirmationSmsStatus":"sent"`) {
		t.Fatal(rec.Body.String())
	}
}
