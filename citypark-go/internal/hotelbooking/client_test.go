package hotelbooking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testKey = "eaf9d7d3-2b21-43a9-8e27-e718b65d2588"
const reservationID = "6e37e63e-b130-4ff6-a1b4-e0166f470922"

func testRequest() CreateRequest {
	return CreateRequest{testKey, "single-balcony", "2026-12-01", "2026-12-03", 1, Guest{"Иван Иванов", "ivan@example.com", "+79180000000"}, Price{"RUB", 2, 400000, 800000}}
}
func testReservation() Reservation {
	r := testRequest()
	return Reservation{reservationID, "confirmed", r.RoomTypeID, r.CheckIn, r.CheckOut, r.Guests, r.Price}
}
func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL+"/hs/booking/v1", "booking", "private", server.Client())
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func TestClientAvailability(t *testing.T) {
	req := AvailabilityRequest{"single-balcony", "2026-12-01", "2026-12-03", 1}
	for _, available := range []bool{true, false} {
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			var got AvailabilityRequest
			if r.Method != "POST" || r.URL.Path != "/hs/booking/v1/availability" || json.NewDecoder(r.Body).Decode(&got) != nil || got != req {
				t.Error("incorrect availability request")
			}
			writeJSON(w, 200, AvailabilityResponse{req.RoomTypeID, req.CheckIn, req.CheckOut, req.Guests, available})
		})
		got, err := client.Availability(context.Background(), req)
		if err != nil || got.Available != available {
			t.Fatalf("availability: %+v %v", got, err)
		}
	}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, AvailabilityResponse{"vip", req.CheckIn, req.CheckOut, 1, true})
	})
	if _, err := client.Availability(context.Background(), req); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
func TestClientCreate(t *testing.T) {
	for _, status := range []int{200, 201} {
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			var got CreateRequest
			if r.Method != "POST" || r.URL.Path != "/hs/booking/v1/reservations" || json.NewDecoder(r.Body).Decode(&got) != nil || got != testRequest() {
				t.Error("incorrect create request or kopeck serialization")
			}
			writeJSON(w, status, testReservation())
		})
		got, err := client.Create(context.Background(), testRequest())
		if err != nil || got != testReservation() {
			t.Fatalf("create: %+v %v", got, err)
		}
	}
}
func TestClientLookup(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/hs/booking/v1/reservations/"+testKey {
			t.Error("incorrect lookup")
		}
		writeJSON(w, 200, testReservation())
	})
	got, err := client.Lookup(context.Background(), testKey)
	if err != nil || got != testReservation() {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	for _, key := range []string{"", "../private", strings.ToUpper(testKey), "00000000-0000-0000-0000-000000000000"} {
		if _, err := client.Lookup(context.Background(), key); !errors.Is(err, ErrInvalidKey) {
			t.Fatal("invalid lookup key accepted")
		}
		req := testRequest()
		req.IdempotencyKey = key
		if _, err := client.Create(context.Background(), req); !errors.Is(err, ErrInvalidKey) {
			t.Fatal("invalid create key accepted")
		}
	}
}
func TestClientProtocolErrors(t *testing.T) {
	for code, status := range map[string]int{"INVALID_REQUEST": 400, "UNAUTHORIZED": 401, "FORBIDDEN": 403, "RESERVATION_NOT_FOUND": 404, "ROOM_UNAVAILABLE": 409, "IDEMPOTENCY_CONFLICT": 409, "BOOKING_UNAVAILABLE": 503} {
		t.Run(code, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, status, map[string]any{"statusCode": status, "code": code, "message": "private upstream details"})
			})
			_, err := client.Lookup(context.Background(), testKey)
			var protocol *ProtocolError
			if !errors.As(err, &protocol) || protocol.StatusCode != status || protocol.Code != code || strings.Contains(err.Error(), "private") {
				t.Fatalf("protocol error: %v", err)
			}
			_, err = client.Create(context.Background(), testRequest())
			if !errors.As(err, &protocol) || protocol.StatusCode != status || protocol.Code != code {
				t.Fatalf("recognized create rejection lost: %v", err)
			}
		})
	}
	valid, _ := json.Marshal(testReservation())
	for _, tc := range []struct {
		status        int
		content, body string
	}{
		{500, "application/json", `{"private":"secret"}`},
		{200, "text/html", "private credentials"},
		{200, "application/json", "{"},
		{200, "application/json", "null"},
		{200, "application/json", string(valid) + ` {}`},
		{200, "application/json", strings.Replace(string(valid), `"confirmed"`, `"pending"`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"guests":1`, `"guests":null`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"guests":1,`, "", 1)},
		{200, "application/json", strings.Replace(string(valid), `"currency":"RUB"`, `"currency":"USD"`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"nights":2`, `"nights":1`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"totalKopecks":800000`, `"totalKopecks":8000.5`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"totalKopecks":800000`, `"totalKopecks":null`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"currency":"RUB"`, `"currency":"RUB","private":1`, 1)},
		{200, "application/json", strings.Replace(string(valid), `"status":"confirmed"`, `"private":1,"status":"confirmed"`, 1)},
		{409, "application/json", `{"statusCode":400,"code":"ROOM_UNAVAILABLE","message":"private"}`},
		{409, "application/json", `{"statusCode":409,"code":"UNKNOWN","message":"private"}`},
		{401, "text/plain", "unauthorized private"},
	} {
		t.Run(fmt.Sprint(tc.status, tc.body), func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.content)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			if _, err := client.Lookup(context.Background(), testKey); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("invalid lookup response accepted: %v", err)
			}
			_, err := client.Create(context.Background(), testRequest())
			var ambiguous *AmbiguousCreateError
			if !errors.As(err, &ambiguous) || strings.Contains(err.Error(), "private") {
				t.Fatalf("unrecognized create outcome must remain uncertain: %v", err)
			}
		})
	}
	// A structurally valid but mismatched creation result is uncertain too.
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		res := testReservation()
		res.RoomTypeID = "single"
		writeJSON(w, 201, res)
	})
	_, err := client.Create(context.Background(), testRequest())
	var ambiguous *AmbiguousCreateError
	if !errors.As(err, &ambiguous) {
		t.Fatal(err)
	}
}
func TestClientResponseLimit(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat(" ", maxResponse+1)))
	})
	if _, err := client.Lookup(context.Background(), testKey); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	_, err := client.Create(context.Background(), testRequest())
	var ambiguous *AmbiguousCreateError
	if !errors.As(err, &ambiguous) {
		t.Fatal(err)
	}
}
func TestClientTimeoutCancellation(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		if cancelled {
			cancel()
		}
		_, err := client.Lookup(ctx, testKey)
		if err == nil || !errors.Is(err, ctx.Err()) {
			t.Fatalf("context not honored: %v", err)
		}
		_, err = client.Create(ctx, testRequest())
		var ambiguous *AmbiguousCreateError
		if !errors.As(err, &ambiguous) || !errors.Is(err, ctx.Err()) {
			t.Fatal(err)
		}
		cancel()
	}
	client.http.Timeout = 10 * time.Millisecond
	if _, err := client.Lookup(context.Background(), testKey); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	for _, timeout := range []time.Duration{0, time.Minute, time.Second} {
		original := &http.Client{Timeout: timeout}
		copied := NewClient("https://hotel/hs/booking/v1", "user", "password", original)
		if copied.http.Timeout <= 0 || copied.http.Timeout > 15*time.Second || original.Timeout != timeout {
			t.Fatal("timeout bound or client copy broken")
		}
	}
}
func TestClientCredentialIsolation(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer target.Close()
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "booking" || password != "private" || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Error("incorrect private authentication or headers")
		}
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	if _, err := client.Create(context.Background(), testRequest()); err == nil || forwarded {
		t.Fatal("redirect followed")
	}
	for _, base := range []string{"", "%", "https://user:private@hotel/hs/booking/v1", "https://hotel/other", "https://hotel/hs/booking/v1?x=private"} {
		disabled := NewClient(base, "booking", "private", nil)
		if _, err := disabled.Lookup(context.Background(), testKey); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
	if _, err := NewClient("https://hotel/hs/booking/v1", "", "", nil).Create(context.Background(), testRequest()); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
func TestClientAmbiguousCreateRecovery(t *testing.T) {
	var creates atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			creates.Add(1)
			var req CreateRequest
			if json.NewDecoder(r.Body).Decode(&req) != nil || req.IdempotencyKey != testKey {
				t.Error("changed recovery key")
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		writeJSON(w, 200, testReservation())
	})
	_, err := client.Create(context.Background(), testRequest())
	var ambiguous *AmbiguousCreateError
	if !errors.As(err, &ambiguous) || creates.Load() != 1 {
		t.Fatalf("ambiguous create retried: %v, %d", err, creates.Load())
	}
	got, err := client.Lookup(context.Background(), testKey)
	if err != nil || got != testReservation() || creates.Load() != 1 {
		t.Fatalf("same-key recovery failed: %v", err)
	}
}

func TestReservationValidation(t *testing.T) {
	for _, mutate := range []func(*Reservation){
		func(r *Reservation) { r.ReservationID = "bad" },
		func(r *Reservation) { r.Status = "cancelled" },
		func(r *Reservation) { r.Guests = 0 },
		func(r *Reservation) { r.Guests = 2 },
		func(r *Reservation) { r.RoomTypeID = "unknown" },
		func(r *Reservation) { r.CheckIn = "invalid" },
		func(r *Reservation) { r.CheckOut = "invalid" },
		func(r *Reservation) { r.CheckIn = "0000-12-01" },
		func(r *Reservation) { r.CheckOut = "4000-12-03" },
		func(r *Reservation) { r.CheckOut = r.CheckIn },
		func(r *Reservation) { r.Price.Currency = "USD" },
		func(r *Reservation) { r.Price.Nights = 0 },
		func(r *Reservation) { r.Price.NightlyPriceKopecks = 0 },
		func(r *Reservation) { r.Price.NightlyPriceKopecks = 1 << 62 },
		func(r *Reservation) { r.Price.TotalKopecks++ },
	} {
		r := testReservation()
		mutate(&r)
		if validReservation(r) {
			t.Fatalf("invalid reservation accepted: %+v", r)
		}
	}
	r := testReservation()
	r.CheckIn = "2027-01-01"
	r.CheckOut = "3999-12-31"
	in, _ := time.Parse("2006-01-02", r.CheckIn)
	out, _ := time.Parse("2006-01-02", r.CheckOut)
	r.Price.Nights = int((out.Unix() - in.Unix()) / 86400)
	r.Price.TotalKopecks = r.Price.NightlyPriceKopecks.Mul(r.Price.Nights)
	if !validReservation(r) {
		t.Fatal("representable long stay rejected")
	}
}

func TestClientResponseBoundaryAndTruncation(t *testing.T) {
	raw, _ := json.Marshal(testReservation())
	for _, truncated := range []bool{false, true} {
		client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if truncated {
				w.Header().Set("Content-Length", "10000")
				_, _ = w.Write(raw)
				return
			}
			_, _ = w.Write(append(raw, []byte(strings.Repeat(" ", maxResponse-len(raw)))...))
		})
		got, err := client.Lookup(context.Background(), testKey)
		if truncated {
			if !errors.Is(err, ErrUnavailable) {
				t.Fatal(err)
			}
		} else if err != nil || got != testReservation() {
			t.Fatal("exact response limit rejected")
		}
	}
}

func TestClientUTF8BOMResponses(t *testing.T) {
	stay := AvailabilityRequest{"single-balcony", "2026-12-01", "2026-12-03", 1}
	for _, prefix := range []string{"", "\xef\xbb\xbf"} {
		for _, available := range []bool{true, false} {
			t.Run(fmt.Sprintf("availability/%x/%t", prefix, available), func(t *testing.T) {
				want := AvailabilityResponse{stay.RoomTypeID, stay.CheckIn, stay.CheckOut, stay.Guests, available}
				client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					fmt.Fprint(w, prefix)
					_ = json.NewEncoder(w).Encode(want)
				})
				got, err := client.Availability(context.Background(), stay)
				if err != nil || got != want {
					t.Fatalf("availability: %+v %v", got, err)
				}
			})
		}
		for _, operation := range []string{"create", "replay", "lookup"} {
			t.Run(fmt.Sprintf("%s/%x", operation, prefix), func(t *testing.T) {
				client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					if operation == "create" {
						w.WriteHeader(http.StatusCreated)
					}
					fmt.Fprint(w, prefix)
					_ = json.NewEncoder(w).Encode(testReservation())
				})
				var got Reservation
				var err error
				if operation == "lookup" {
					got, err = client.Lookup(context.Background(), testKey)
				} else {
					got, err = client.Create(context.Background(), testRequest())
				}
				if err != nil || got != testReservation() {
					t.Fatalf("reservation: %+v %v", got, err)
				}
			})
		}
		for _, operation := range []string{"availability", "create", "lookup"} {
			t.Run(fmt.Sprintf("error/%s/%x", operation, prefix), func(t *testing.T) {
				client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprint(w, prefix+`{"statusCode":403,"code":"FORBIDDEN","message":"Доступ запрещён"}`)
				})
				var err error
				switch operation {
				case "availability":
					_, err = client.Availability(context.Background(), stay)
				case "create":
					_, err = client.Create(context.Background(), testRequest())
				case "lookup":
					_, err = client.Lookup(context.Background(), testKey)
				}
				var protocol *ProtocolError
				var ambiguous *AmbiguousCreateError
				if !errors.As(err, &protocol) || protocol.StatusCode != http.StatusForbidden || protocol.Code != "FORBIDDEN" || errors.As(err, &ambiguous) {
					t.Fatalf("expected definitive forbidden response, got %T %v", err, err)
				}
			})
		}
	}
}

func TestClientUTF8BOMStrictValidation(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	const valid = `{"roomTypeId":"single-balcony","checkIn":"2026-12-01","checkOut":"2026-12-03","guests":1,"available":true}`
	for name, body := range map[string]string{
		"double BOM":     bom + bom + valid,
		"misplaced BOM":  " " + bom + valid,
		"UTF16 BOM":      "\xff\xfe" + valid,
		"empty document": bom,
		"trailing JSON":  bom + valid + `{}`,
		"unknown field":  bom + strings.TrimSuffix(valid, "}") + `,"internal":"secret"}`,
		"missing field":  bom + strings.Replace(valid, `,"available":true`, "", 1),
		"null field":     bom + strings.Replace(valid, `"available":true`, `"available":null`, 1),
		"wrong type":     bom + strings.Replace(valid, `"available":true`, `"available":"true"`, 1),
		"wrong stay":     bom + strings.Replace(valid, "single-balcony", "vip", 1),
	} {
		t.Run(name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				fmt.Fprint(w, body)
			})
			got, err := client.Availability(context.Background(), AvailabilityRequest{"single-balcony", "2026-12-01", "2026-12-03", 1})
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("expected unavailable, got %+v %v", got, err)
			}
		})
	}
}
