package hotelbooking

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drewp1ter/citypark-go/internal/logging"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/room"
	"github.com/drewp1ter/citypark-go/internal/settings"
	"github.com/drewp1ter/citypark-go/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

type bookingFake struct {
	mu                sync.Mutex
	available         bool
	availabilityError error
	createError       error
	lookupError       error
	commitOnError     bool
	calls             int
	lookups           int
	requests          []CreateRequest
	saved             map[string]Reservation
	block             chan struct{}
	entered           chan struct{}
}

func (f *bookingFake) Availability(_ context.Context, req AvailabilityRequest) (AvailabilityResponse, error) {
	return AvailabilityResponse{req.RoomTypeID, req.CheckIn, req.CheckOut, req.Guests, f.available}, f.availabilityError
}
func (f *bookingFake) Create(ctx context.Context, req CreateRequest) (Reservation, error) {
	f.mu.Lock()
	f.calls++
	f.requests = append(f.requests, req)
	block, entered := f.block, f.entered
	if entered != nil {
		close(entered)
		f.entered = nil
	}
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return Reservation{}, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.saved[req.IdempotencyKey]; ok {
		return r, nil
	}
	r := Reservation{"6e37e63e-b130-4ff6-a1b4-e0166f470922", "00000001234", "confirmed", req.RoomTypeID, req.CheckIn, req.CheckOut, req.Guests, req.Price}
	if f.createError == nil || f.commitOnError {
		f.saved[req.IdempotencyKey] = r
	}
	return r, f.createError
}
func (f *bookingFake) Lookup(_ context.Context, key string) (Reservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookups++
	if f.lookupError != nil {
		return Reservation{}, f.lookupError
	}
	r, ok := f.saved[key]
	if !ok {
		return Reservation{}, &ProtocolError{404, "RESERVATION_NOT_FOUND"}
	}
	return r, nil
}

type bookingHarness struct {
	t       *testing.T
	db      *sqlx.DB
	service *Service
	clock   *testClock
	client  *bookingFake
	handler http.Handler
	token   string
	logs    bytes.Buffer
}

func newBooking(t *testing.T) *bookingHarness {
	t.Helper()
	db := testutil.NewDB(t)
	_, err := db.Exec(`TRUNCATE hotel_booking_email_outbox,hotel_booking_attempts,hotel_booking_quotes,hotel_booking_rate_limits; UPDATE rooms SET price_kopecks=400000`)
	if err != nil {
		t.Fatal(err)
	}
	h := &bookingHarness{t: t, db: db, clock: &testClock{time.Date(2026, 11, 22, 20, 59, 0, 0, time.UTC)}, client: &bookingFake{available: true, saved: map[string]Reservation{}}, token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))}
	h.service = NewService(NewStore(db), room.NewStore(db), h.client, h.clock)
	h.route(true, nil)
	return h
}
func (h *bookingHarness) route(enabled bool, trusted []netip.Prefix) {
	r := chi.NewRouter()
	r.Use(logging.Requests(slog.New(slog.NewJSONHandler(&h.logs, nil))))
	r.Route("/api/v1", NewHandler(h.service, enabled, trusted).Routes)
	h.handler = r
}
func (h *bookingHarness) request(method, path, token string, body any) *httptest.ResponseRecorder {
	h.t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		h.t.Fatal(err)
	}
	if text, ok := body.(string); ok {
		raw = []byte(text)
	}
	req := httptest.NewRequest(method, "/api/v1/hotel"+path, bytes.NewReader(raw))
	req.Header.Set("X-Hotel-Booking-Context", token)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	if rec.Header().Get("Cache-Control") != "no-store" {
		h.t.Fatalf("cache header: %v", rec.Header())
	}
	return rec
}
func checkCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	if code != "" {
		var e bookingError
		if json.Unmarshal(rec.Body.Bytes(), &e) != nil || e.Code != code || e.Status != status || e.Message != failure(code).Message {
			t.Fatalf("error: %s", rec.Body.String())
		}
	}
}
func (h *bookingHarness) quote(req QuoteRequest) QuoteResponse {
	h.t.Helper()
	rec := h.request("POST", "/quotes", h.token, req)
	checkCode(h.t, rec, 200, "")
	var q QuoteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &q); err != nil {
		h.t.Fatal(err)
	}
	return q
}
func stayRequest() QuoteRequest { return QuoteRequest{"single-balcony", "2026-12-01", "2026-12-03", 1} }
func guestRequest() Guest {
	return Guest{" Иван Иванов ", "Ivan@EXAMPLE.COM", "8 (918) 000-00-00"}
}
func createRequest(q QuoteResponse) PublicCreateRequest {
	return PublicCreateRequest{randomID(), q.QuoteID, guestRequest()}
}
func (h *bookingHarness) count(table string) int {
	h.t.Helper()
	var n int
	// table names are test-owned constants, never caller input.
	if err := h.db.Get(&n, "SELECT count(*) FROM "+table); err != nil {
		h.t.Fatal(err)
	}
	return n
}
func TestQuoteCalendarBounds(t *testing.T) {
	h := newBooking(t)
	for _, tc := range []struct {
		in, out string
		valid   bool
	}{
		{"2026-11-22", "2026-11-23", false}, {"2026-11-21", "2026-11-23", false},
		{"2026-11-23", "2026-11-24", true}, {"2026-12-03", "2026-12-03", false}, {"2026-12-04", "2026-12-03", false},
		{"2027-02-29", "2027-03-01", false}, {"2028-02-29", "2028-03-01", true},
		{"2027-11-23", "2028-11-24", true}, {"3999-01-01", "3999-12-31", true}, {"3999-12-31", "4000-01-01", false},
		{"0000-01-01", "2026-12-03", false}, {"2026-1-01", "2026-12-03", false},
	} {
		req := stayRequest()
		req.CheckIn, req.CheckOut = tc.in, tc.out
		rec := h.request("POST", "/quotes", h.token, req)
		if tc.valid {
			checkCode(t, rec, 200, "")
		} else {
			checkCode(t, rec, 400, "INVALID_REQUEST")
		}
	}
	h.clock.now = h.clock.now.Add(time.Minute) // Moscow midnight, previously valid arrival is today.
	req := stayRequest()
	req.CheckIn = "2026-11-23"
	checkCode(t, h.request("POST", "/quotes", h.token, req), 400, "INVALID_REQUEST")
}
func TestQuoteCapacity(t *testing.T) {
	h := newBooking(t)
	for _, id := range []string{"single", "single-balcony", "double-twin", "double-euro", "vip", "missing"} {
		for _, guests := range []int{0, 1, 2, 3, 4, 5} {
			req := stayRequest()
			req.RoomTypeID, req.Guests = id, guests
			max := map[string]int{"single": 1, "single-balcony": 1, "double-twin": 2, "double-euro": 2, "vip": 4}[id]
			rec := h.request("POST", "/quotes", h.token, req)
			if guests > 0 && guests <= max {
				checkCode(t, rec, 200, "")
			} else {
				checkCode(t, rec, 400, "INVALID_REQUEST")
			}
		}
	}
}
func TestQuoteMoney(t *testing.T) {
	h := newBooking(t)
	for _, tc := range []struct {
		rate   int64
		status int
	}{{400001, 200}, {0, 503}, {-1, 503}, {math.MaxInt64, 503}} {
		h.service.rooms = roomFake{value: room.Room{Price: money.Kopecks(tc.rate)}}
		rec := h.request("POST", "/quotes", h.token, stayRequest())
		checkCode(t, rec, tc.status, "")
		if tc.status == 200 {
			var q QuoteResponse
			_ = json.Unmarshal(rec.Body.Bytes(), &q)
			if q.Price.Nights != 2 || q.Price.Total != 800002 || q.Price.PricePerNight != 400001 || q.ExpiresAt != h.clock.Now().Add(10*time.Minute) || !q.Available || !validKey(q.QuoteID) {
				t.Fatalf("quote: %+v", q)
			}
		}
	}
}
func TestCreateRateChange(t *testing.T) {
	h := newBooking(t)
	q := h.quote(stayRequest())
	req := createRequest(q)
	if _, err := h.db.Exec(`UPDATE rooms SET price_kopecks=450000`); err != nil {
		t.Fatal(err)
	}
	checkCode(t, h.request("POST", "/reservations", h.token, req), 409, "PRICE_CHANGED")
	if h.client.calls != 0 || h.count("hotel_booking_attempts") != 0 {
		t.Fatal("created on price change")
	}
	if _, err := h.db.Exec(`UPDATE rooms SET price_kopecks=400000`); err != nil {
		t.Fatal(err)
	}
	h.clock.now = h.clock.now.Add(10 * time.Minute)
	checkCode(t, h.request("POST", "/reservations", h.token, req), 409, "QUOTE_EXPIRED")
	// A quote crossing Moscow midnight is invalid even while unexpired.
	h.clock.now = time.Date(2026, 11, 22, 20, 59, 0, 0, time.UTC)
	tomorrow := stayRequest()
	tomorrow.CheckIn = "2026-11-23"
	q = h.quote(tomorrow)
	h.clock.now = h.clock.now.Add(time.Minute)
	checkCode(t, h.request("POST", "/reservations", h.token, createRequest(q)), 400, "INVALID_REQUEST")
	if h.client.calls != 0 {
		t.Fatal("called upstream")
	}
}
func TestCreateIdempotency(t *testing.T) {
	h := newBooking(t)
	q := h.quote(stayRequest())
	req := createRequest(q)
	first := h.request("POST", "/reservations", h.token, req)
	checkCode(t, first, 201, "")
	var result PublicReservationResponse
	_ = json.Unmarshal(first.Body.Bytes(), &result)
	if result.Status != "confirmed" || result.ReservationID == "" || result.ReservationNumber != "00000001234" || result.Price.Total != 800000 || result.ConfirmationEmailStatus != "pending" {
		t.Fatalf("result %+v", result)
	}
	sent := h.client.requests[0]
	if sent.Guest != (Guest{"Иван Иванов", "Ivan@example.com", "+79180000000"}) || sent.Price.TotalKopecks != 800000 {
		t.Fatalf("normalized %+v", sent)
	}
	h.clock.now = h.clock.now.Add(24 * time.Hour)
	if _, err := h.db.Exec(`UPDATE rooms SET price_kopecks=500000`); err != nil {
		t.Fatal(err)
	}
	replay := h.request("POST", "/reservations", h.token, req)
	checkCode(t, replay, 200, "")
	if first.Body.String() != replay.Body.String() {
		t.Fatal("replay changed")
	}
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
	req.Guest.Name = "Другой"
	checkCode(t, h.request("POST", "/reservations", h.token, req), 409, "IDEMPOTENCY_CONFLICT")
	req.Guest = guestRequest()
	req.QuoteID = randomID()
	checkCode(t, h.request("POST", "/reservations", h.token, req), 409, "IDEMPOTENCY_CONFLICT")
	if h.client.calls != 1 || h.count("hotel_booking_attempts") != 1 || h.count("hotel_booking_email_outbox") != 1 {
		t.Fatal("duplicate creation")
	}
	quoteJSON, _ := json.Marshal(q)
	t.Logf("quote example: %s", quoteJSON)
	t.Logf("confirmed example: %s", first.Body.String())
}
func TestConcurrentCreate(t *testing.T) {
	h := newBooking(t)
	q := h.quote(stayRequest())
	req := createRequest(q)
	h.client.block = make(chan struct{})
	h.client.entered = make(chan struct{})
	result := make(chan PublicReservationResponse, 1)
	errs := make(chan error, 1)
	entered := h.client.entered
	go func() { r, e := h.service.Create(context.Background(), h.token, req); result <- r; errs <- e }()
	<-entered
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := h.service.Create(context.Background(), h.token, req)
			if err != nil || r.Status != "pending" {
				t.Errorf("concurrent %+v %v", r, err)
			}
		}()
	}
	wg.Wait()
	close(h.client.block)
	if r := <-result; r.Status != "confirmed" {
		t.Fatalf("result %+v", r)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if h.client.calls != 1 || h.count("hotel_booking_email_outbox") != 1 {
		t.Fatal("duplicate dispatch")
	}
}
func TestPendingRecovery(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprint(committed), func(t *testing.T) {
			h := newBooking(t)
			req := createRequest(h.quote(stayRequest()))
			h.client.createError = &AmbiguousCreateError{errors.New("secret email@example.com")}
			h.client.commitOnError = committed
			pending := h.request("POST", "/reservations", h.token, req)
			checkCode(t, pending, 202, "")
			if pending.Body.String() != "{\"status\":\"pending\"}\n" {
				t.Fatal(pending.Body.String())
			}
			checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 202, "")
			replay := h.request("POST", "/reservations", h.token, req)
			checkCode(t, replay, 202, "")
			h.clock.now = h.clock.now.Add(48 * time.Hour)
			if _, err := h.db.Exec(`UPDATE rooms SET price_kopecks=999999`); err != nil {
				t.Fatal(err)
			}
			h.client.createError = nil
			w := h.worker(&testMailer{})
			w.Tick(context.Background())
			checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, h.token, nil), 200, "")
			wantCalls := 2
			if committed {
				wantCalls = 1
			}
			if h.client.calls != wantCalls || h.count("hotel_booking_attempts") != 1 || len(h.client.saved) != 1 || h.client.requests[len(h.client.requests)-1] != h.client.requests[0] {
				t.Fatal("recovery replaced request")
			}
		})
	}
}
func TestContextIsolation(t *testing.T) {
	h := newBooking(t)
	q := h.quote(stayRequest())
	req := createRequest(q)
	other := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32))
	checkCode(t, h.request("POST", "/reservations", other, req), 404, "RESERVATION_NOT_FOUND")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 201, "")
	checkCode(t, h.request("POST", "/reservations", other, req), 404, "RESERVATION_NOT_FOUND")
	checkCode(t, h.request("GET", "/reservations/"+req.IdempotencyKey, other, nil), 404, "RESERVATION_NOT_FOUND")
	checkCode(t, h.request("GET", "/reservations/"+randomID(), h.token, nil), 404, "RESERVATION_NOT_FOUND")
	for _, token := range []string{"", "bad", h.token + "=", base64.RawURLEncoding.EncodeToString([]byte("short"))} {
		checkCode(t, h.request("POST", "/quotes", token, stayRequest()), 400, "INVALID_REQUEST")
	}
	if h.client.calls != 1 {
		t.Fatal("context escaped")
	}
	var hashes []string
	if err := h.db.Select(&hashes, `SELECT context_hash FROM hotel_booking_quotes UNION SELECT context_hash FROM hotel_booking_attempts`); err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if hash != digest(h.token) {
			t.Fatal("unhashed context")
		}
	}
}
func TestGuestValidation(t *testing.T) {
	for _, tc := range []struct {
		field, value string
		valid        bool
	}{
		{"name", "", false}, {"name", strings.Repeat("я", 151), false}, {"name", strings.Repeat("я", 150), true}, {"name", "a\nb", false}, {"name", string([]byte{255}), false},
		{"email", "wrong", false}, {"email", "Person <a@example.com>", false}, {"email", "x\r@example.com", false}, {"email", strings.Repeat("a", 243) + "@example.com", false},
		{"phone", "", false}, {"phone", "+1234567", false}, {"phone", "+1234567890123456", false}, {"phone", "+123456789012345", true}, {"phone", "+7abc1234567", false}, {"phone", "+7\t9180000000", false}, {"phone", "++79180000000", false},
	} {
		t.Run(tc.field+fmt.Sprint(len(tc.value))+fmt.Sprint(tc.valid), func(t *testing.T) {
			g := guestRequest()
			switch tc.field {
			case "name":
				g.Name = tc.value
			case "email":
				g.Email = tc.value
			case "phone":
				g.Phone = tc.value
			}
			normalized, err := normalizeGuest(g)
			if tc.valid && err != nil || !tc.valid && err == nil {
				t.Fatalf("normalized %+v err %v", normalized, err)
			}
		})
	}
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	req.Guest.Name = ""
	checkCode(t, h.request("POST", "/reservations", h.token, req), 400, "INVALID_REQUEST")
	if h.client.calls != 0 {
		t.Fatal("invalid guest dispatched")
	}
}
func TestStrictBookingBodies(t *testing.T) {
	h := newBooking(t)
	for _, raw := range []string{
		"", "null", "[]", "{}", "{", `{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"extra":1}`,
		`{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":null}`,
		`{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1} {}`,
		`{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"Guests":1}`,
		`{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"guests":1}`,
		`{"roomTypeId":"single","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1.5}`,
		strings.Repeat(" ", 16<<10) + "{}",
	} {
		checkCode(t, h.request("POST", "/quotes", h.token, raw), 400, "INVALID_REQUEST")
	}
	q := h.quote(stayRequest())
	req := createRequest(q)
	raw, _ := json.Marshal(req)
	for _, bad := range []string{
		strings.Replace(string(raw), `"name":" Иван Иванов "`, `"name":" Иван Иванов ","Name":"secret"`, 1),
		strings.Replace(string(raw), `"name":" Иван Иванов "`, `"name":null`, 1),
		strings.Replace(string(raw), `"name":" Иван Иванов ",`, "", 1),
		`{"idempotencyKey":"bad","quoteId":"bad","guest":{"name":"x","email":"x@y.ru","phone":"+79180000000"}}`,
	} {
		checkCode(t, h.request("POST", "/reservations", h.token, bad), 400, "INVALID_REQUEST")
	}
	checkCode(t, h.request("GET", "/reservations/bad", h.token, nil), 400, "INVALID_REQUEST")
}
func TestRateLimits(t *testing.T) {
	h := newBooking(t)
	for i := 0; i < 30; i++ {
		h.quote(stayRequest())
	}
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 429, "RATE_LIMITED")
	h.clock.now = h.clock.now.Add(time.Minute)
	q := h.quote(stayRequest())
	var first PublicCreateRequest
	for i := 0; i < 6; i++ {
		req := createRequest(q)
		req.Guest.Email = fmt.Sprintf("person%d@example.com", i)
		rec := h.request("POST", "/reservations", h.token, req)
		if i < 5 {
			checkCode(t, rec, 201, "")
		} else {
			checkCode(t, rec, 429, "RATE_LIMITED")
		}
		if i == 0 {
			first = req
		}
	}
	// Replays are outside creation quotas, but bounded independently.
	for i := 0; i < 30; i++ {
		checkCode(t, h.request("POST", "/reservations", h.token, first), 200, "")
	}
	checkCode(t, h.request("GET", "/reservations/"+first.IdempotencyKey, h.token, nil), 429, "RATE_LIMITED")
	if h.count("hotel_booking_attempts") != 5 {
		t.Fatal("quota failure left attempt")
	}
	h.clock.now = h.clock.now.Add(time.Hour)
	for i := 0; i < 4; i++ {
		req := createRequest(h.quote(stayRequest()))
		rec := h.request("POST", "/reservations", h.token, req)
		if i < 3 {
			checkCode(t, rec, 201, "")
		} else {
			checkCode(t, rec, 429, "RATE_LIMITED")
		}
	}
	var hashes []string
	if err := h.db.Select(&hashes, `SELECT hash FROM hotel_booking_rate_limits`); err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if len(hash) != 64 {
			t.Fatal("unhashed quota")
		}
	}
}
func TestBookingUpstreamFailures(t *testing.T) {
	h := newBooking(t)
	h.client.available = false
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 409, "ROOM_UNAVAILABLE")
	h.client.available = true
	h.client.availabilityError = errors.New("secret")
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 503, "BOOKING_UNAVAILABLE")
	h.client.availabilityError = nil
	for i, code := range []string{"ROOM_UNAVAILABLE", "IDEMPOTENCY_CONFLICT", "INVALID_REQUEST", "BOOKING_UNAVAILABLE"} {
		q := h.quote(stayRequest())
		req := createRequest(q)
		req.Guest.Email = fmt.Sprintf("guest%d@example.com", i)
		h.client.createError = &ProtocolError{failure(code).Status, code}
		rec := h.request("POST", "/reservations", h.token, req)
		if code == "BOOKING_UNAVAILABLE" {
			checkCode(t, rec, 202, "")
		} else {
			checkCode(t, rec, failure(code).Status, code)
			checkCode(t, h.request("POST", "/reservations", h.token, req), failure(code).Status, code)
		}
	}
}
func TestTrustedBookingProxies(t *testing.T) {
	h := newBooking(t)
	handler := NewHandler(h.service, true, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	for _, tc := range []struct {
		peer, xff, want string
		bad             bool
	}{
		{"192.0.2.1:80", "1.2.3.4", "192.0.2.1", false},
		{"10.0.0.1:80", "198.51.100.1, 10.0.0.2", "198.51.100.1", false},
		{"10.0.0.1:80", "spoof, 198.51.100.1", "198.51.100.1", false},
		{"::ffff:192.0.2.1", "", "192.0.2.1", false},
		{"bad", "", "", true}, {"10.0.0.1:80", "garbage", "", true},
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tc.peer
		req.Header.Set("X-Forwarded-For", tc.xff)
		got, err := handler.clientIP(req)
		if got != tc.want || (err != nil) != tc.bad {
			t.Fatalf("ip %q %v", got, err)
		}
	}
	h.route(false, nil)
	checkCode(t, h.request("POST", "/quotes", h.token, stayRequest()), 503, "BOOKING_UNAVAILABLE")
	h.route(true, nil)
	req := httptest.NewRequest("POST", "/api/v1/hotel/quotes", nil)
	req.RemoteAddr = "bad"
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	checkCode(t, rec, 400, "INVALID_REQUEST")
}
func TestNoPIILogging(t *testing.T) {
	h := newBooking(t)
	req := createRequest(h.quote(stayRequest()))
	h.client.createError = errors.New("Ivan@example.com +79180000000 Иван Иванов PASSWORD")
	checkCode(t, h.request("POST", "/reservations", h.token, req), 202, "")
	checkCode(t, h.request("GET", "/reservations/Ivan@example.com", h.token, nil), 400, "INVALID_REQUEST")
	h.clock.now = h.clock.now.Add(time.Minute)
	h.client.lookupError = errors.New("PASSWORD")
	h.worker(&testMailer{}).Tick(context.Background())
	for _, value := range []string{"Ivan", "Иван", "7918", "PASSWORD", h.token, "192.0.2.1"} {
		if strings.Contains(h.logs.String(), value) {
			t.Fatalf("PII leaked: %s", h.logs.String())
		}
	}
	if !strings.Contains(h.logs.String(), "[private]") {
		t.Fatal("missing safe operational request logs")
	}
}

type testMailer struct {
	calls          int
	fail           bool
	to, body       string
	block, entered chan struct{}
}

func (m *testMailer) Send(ctx context.Context, to, subject, body string) error {
	m.calls++
	m.to, m.body = to, body
	if m.entered != nil {
		close(m.entered)
		m.entered = nil
	}
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.fail {
		return errors.New("private smtp detail")
	}
	return nil
}
func (h *bookingHarness) worker(m *testMailer) *Worker {
	return NewWorker(h.service, m, settings.NewStore(h.db), slog.New(slog.NewJSONHandler(&h.logs, nil)))
}

type roomFake struct {
	value room.Room
	err   error
}

func (r roomFake) Get(context.Context, string) (room.Room, error) { return r.value, r.err }
