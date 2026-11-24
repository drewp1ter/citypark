package hotelbooking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"unicode/utf8"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/validate"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	enabled bool
	trusted []netip.Prefix
}

func NewHandler(service *Service, enabled bool, trusted []netip.Prefix) *Handler {
	return &Handler{service, enabled, append([]netip.Prefix(nil), trusted...)}
}
func (h *Handler) Routes(r chi.Router) {
	r.Route("/hotel", func(r chi.Router) {
		r.Use(h.private)
		r.Get("/cancellation", httpx.Handle(h.Cancellation))
		r.Post("/cancellation", httpx.Handle(h.Cancellation))
		r.Post("/quotes", httpx.Handle(h.Quote))
		r.Post("/reservations", httpx.Handle(h.Create))
		r.Get("/reservations/{idempotencyKey}", httpx.Handle(h.Lookup))
	})
}
func (h *Handler) private(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !h.enabled {
			httpx.JSON(w, 503, failure("BOOKING_UNAVAILABLE"))
			return
		}
		ip, err := h.clientIP(r)
		if err != nil {
			httpx.JSON(w, 400, failure("INVALID_REQUEST"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ipKey{}, ip)))
	})
}
func (h *Handler) clientIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "", err
	}
	ip = ip.Unmap()
	trusted := func(ip netip.Addr) bool {
		for _, prefix := range h.trusted {
			if prefix.Contains(ip) {
				return true
			}
		}
		return false
	}
	if trusted(ip) && r.Header.Get("X-Forwarded-For") != "" {
		hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		for i := len(hops) - 1; i >= 0 && trusted(ip); i-- {
			ip, err = netip.ParseAddr(strings.TrimSpace(hops[i]))
			if err != nil {
				return "", err
			}
			ip = ip.Unmap()
		}
	}
	return ip.String(), nil
}
func readBody(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	raw, err := io.ReadAll(r.Body)
	// decode adds strict field/null/trailing-document checks to the shared decoder.
	// Use the common request decoder, then require the exact frozen wire fields.
	if err != nil {
		return failure("INVALID_REQUEST")
	}
	r.Body = io.NopCloser(strings.NewReader(string(raw)))
	if !utf8.Valid(raw) || httpx.DecodeJSON(w, r, out) != nil || decode(raw, out) != nil || exactFields(raw, out) != nil || validate.Struct(out) != nil {
		return failure("INVALID_REQUEST")
	}
	return nil
}
func writeFailure(w http.ResponseWriter, err error) error {
	var e *bookingError
	if !errors.As(err, &e) {
		e = failure("BOOKING_UNAVAILABLE")
	}
	httpx.JSON(w, e.Status, e)
	return nil
}
func (h *Handler) Quote(w http.ResponseWriter, r *http.Request) error {
	var req QuoteRequest
	if err := readBody(w, r, &req); err != nil {
		return writeFailure(w, err)
	}
	result, err := h.service.Quote(r.Context(), r.Header.Get("X-Hotel-Booking-Context"), req)
	if err != nil {
		return writeFailure(w, err)
	}
	httpx.JSON(w, 200, result)
	return nil
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	var req PublicCreateRequest
	if err := readBody(w, r, &req); err != nil {
		return writeFailure(w, err)
	}
	result, err := h.service.Create(r.Context(), r.Header.Get("X-Hotel-Booking-Context"), req)
	if err != nil {
		return writeFailure(w, err)
	}
	status := 200
	if result.Status == "pending" {
		status = 202
	} else if result.created {
		status = 201
	}
	httpx.JSON(w, status, result)
	return nil
}
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) error {
	result, err := h.service.Lookup(r.Context(), r.Header.Get("X-Hotel-Booking-Context"), chi.URLParam(r, "idempotencyKey"))
	if err != nil {
		return writeFailure(w, err)
	}
	status := 200
	if result.Status == "pending" {
		status = 202
	}
	httpx.JSON(w, status, result)
	return nil
}

// Verify case-sensitive field names, duplicates and nulls recursively. The
// standard decoder alone accepts alternate field casing and repeated keys.
func exactFields(raw []byte, out any) error {
	expected, _ := json.Marshal(out)
	var shape any
	_ = json.Unmarshal(expected, &shape)
	d := json.NewDecoder(bytes.NewReader(raw))
	return exactValue(d, shape)
}
func exactValue(d *json.Decoder, shape any) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	fields, isObject := shape.(map[string]any)
	if !isObject {
		if token == nil {
			return ErrUnavailable
		}
		return nil
	}
	if token != json.Delim('{') {
		return ErrUnavailable
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return err
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return ErrUnavailable
		}
		value, ok := fields[name]
		if !ok {
			return ErrUnavailable
		}
		seen[name] = true
		if err = exactValue(d, value); err != nil {
			return err
		}
	}
	_, err = d.Token()
	if err != nil {
		return err
	}
	if len(seen) != len(fields) {
		return ErrUnavailable
	}
	return nil
}
