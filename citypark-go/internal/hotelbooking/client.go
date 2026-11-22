package hotelbooking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxResponse = 64 << 10
const requestTimeout = 15 * time.Second

var ErrUnavailable = errors.New("booking service unavailable")
var ErrInvalidKey = errors.New("invalid booking key")

// ProtocolError contains only recognized status/code values, never upstream text.
type ProtocolError struct {
	StatusCode int
	Code       string
}

func (e *ProtocolError) Error() string { return "booking service: " + e.Code }

// AmbiguousCreateError means a reservation may have committed. Recover by the
// original key; never create a replacement key in response to this error.
type AmbiguousCreateError struct{ Cause error }

func (e *AmbiguousCreateError) Error() string { return "booking creation outcome unknown" }
func (e *AmbiguousCreateError) Unwrap() error { return e.Cause }

type Client struct {
	baseURL, user, password string
	http                    *http.Client
	configured              bool
}

// NewClient copies the HTTP client, preserving its transport but disabling all
// redirects and imposing a 15-second upper bound. It never changes the caller's client.
func NewClient(baseURL, user, password string, httpClient *http.Client) *Client {
	h := http.Client{}
	if httpClient != nil {
		h = *httpClient
	}
	if h.Timeout <= 0 || h.Timeout > requestTimeout {
		h.Timeout = requestTimeout
	}
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	u, err := url.Parse(baseURL)
	valid := err == nil && u.Hostname() != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == "" && strings.HasSuffix(u.Path, "/hs/booking/v1") && user != "" && !strings.ContainsAny(user, ":\r\n") && password != ""
	return &Client{baseURL: baseURL, user: user, password: password, http: &h, configured: valid}
}

func (c *Client) Availability(ctx context.Context, req AvailabilityRequest) (AvailabilityResponse, error) {
	var result AvailabilityResponse
	err := c.request(ctx, http.MethodPost, "/availability", req, &result, false)
	if err == nil && (result.RoomTypeID != req.RoomTypeID || result.CheckIn != req.CheckIn || result.CheckOut != req.CheckOut || result.Guests != req.Guests) {
		err = ErrUnavailable
	}
	return result, err
}

func (c *Client) Create(ctx context.Context, req CreateRequest) (Reservation, error) {
	if !validKey(req.IdempotencyKey) {
		return Reservation{}, ErrInvalidKey
	}
	var result Reservation
	err := c.request(ctx, http.MethodPost, "/reservations", req, &result, true)
	if err == nil && (!validReservation(result) || result.RoomTypeID != req.RoomTypeID || result.CheckIn != req.CheckIn || result.CheckOut != req.CheckOut || result.Guests != req.Guests || result.Price != req.Price) {
		err = &AmbiguousCreateError{ErrUnavailable}
	}
	return result, err
}

func (c *Client) Lookup(ctx context.Context, key string) (Reservation, error) {
	if !validKey(key) {
		return Reservation{}, ErrInvalidKey
	}
	var result Reservation
	err := c.request(ctx, http.MethodGet, "/reservations/"+key, nil, &result, false)
	if err == nil && !validReservation(result) {
		err = ErrUnavailable
	}
	return result, err
}

func validKey(key string) bool {
	return regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(key) && key != "00000000-0000-0000-0000-000000000000"
}

func validReservation(r Reservation) bool {
	in, e1 := time.Parse("2006-01-02", r.CheckIn)
	out, e2 := time.Parse("2006-01-02", r.CheckOut)
	capacity := map[string]int{"single": 1, "single-balcony": 1, "double-twin": 2, "double-euro": 2, "vip": 4}[r.RoomTypeID]
	p := r.Price
	if !validKey(r.ReservationID) || r.Status != "confirmed" || r.Guests <= 0 || r.Guests > capacity {
		return false
	}
	if e1 != nil || e2 != nil || in.Year() < 1 || out.Year() > 3999 || !out.After(in) {
		return false
	}
	// Unix seconds avoid time.Duration's roughly 290-year limit for long stays.
	nights := int((out.Unix() - in.Unix()) / 86400)
	if p.Currency != "RUB" || p.Nights != nights || p.NightlyPriceKopecks <= 0 || p.Nights <= 0 || int64(p.NightlyPriceKopecks) > math.MaxInt64/int64(p.Nights) {
		return false
	}
	return p.TotalKopecks == p.NightlyPriceKopecks.Mul(p.Nights)
}

func decode(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return ErrUnavailable
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return ErrUnavailable
	}
	// Null/missing properties must not silently become valid zero values.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return ErrUnavailable
	}
	for _, v := range fields {
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return ErrUnavailable
		}
	}
	expected, _ := json.Marshal(out)
	var required map[string]json.RawMessage
	_ = json.Unmarshal(expected, &required)
	for key := range required {
		if _, ok := fields[key]; !ok {
			return ErrUnavailable
		}
	}
	return nil
}

func (c *Client) request(ctx context.Context, method, path string, body, out any, create bool) error {
	if !c.configured {
		return ErrUnavailable
	}
	fail := func() error {
		cause := ErrUnavailable
		if ctx.Err() != nil {
			cause = ctx.Err()
		}
		if create {
			return &AmbiguousCreateError{cause}
		}
		return cause
	}
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fail()
	}
	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fail()
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(raw) > maxResponse {
		return fail()
	}
	media, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return fail()
	}
	// 1C's UTF-8 HTTP writer can prepend a BOM. Accept it only at the wire
	// boundary; keep stored JSON and all remaining response validation strict.
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if resp.StatusCode != http.StatusOK && !(create && resp.StatusCode == http.StatusCreated) {
		var wire struct {
			StatusCode int    `json:"statusCode"`
			Code       string `json:"code"`
			Message    string `json:"message"`
		}
		if decode(raw, &wire) != nil || wire.StatusCode != resp.StatusCode {
			return fail()
		}
		statuses := map[string]int{"INVALID_REQUEST": 400, "UNAUTHORIZED": 401, "FORBIDDEN": 403, "RESERVATION_NOT_FOUND": 404, "ROOM_UNAVAILABLE": 409, "IDEMPOTENCY_CONFLICT": 409, "BOOKING_UNAVAILABLE": 503}
		if statuses[wire.Code] != resp.StatusCode {
			return fail()
		}
		return &ProtocolError{wire.StatusCode, wire.Code}
	}
	if decode(raw, out) != nil {
		return fail()
	}
	return nil
}
