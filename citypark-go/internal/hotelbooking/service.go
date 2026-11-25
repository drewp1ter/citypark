package hotelbooking

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/drewp1ter/citypark-go/internal/clock"
	"github.com/drewp1ter/citypark-go/internal/room"
)

type BookingClient interface {
	Availability(context.Context, AvailabilityRequest) (AvailabilityResponse, error)
	Create(context.Context, CreateRequest) (Reservation, error)
	Lookup(context.Context, string) (Reservation, error)
}
type RoomReader interface {
	Get(context.Context, string) (room.Room, error)
}
type Service struct {
	store        *Store
	rooms        RoomReader
	client       BookingClient
	clock        clock.Clock
	cancellation cancellationConfig
}

func NewService(s *Store, rooms RoomReader, client BookingClient, c clock.Clock) *Service {
	return &Service{store: s, rooms: rooms, client: client, clock: c}
}
func randomID() string {
	var b [16]byte
	// The operating system random source is mandatory; failure cannot produce a reusable key.
	if _, err := rand.Read(b[:]); err != nil {
		panic("booking random source unavailable")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func digest(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
func contextHash(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 || base64.RawURLEncoding.EncodeToString(raw) != token {
		return "", failure("INVALID_REQUEST")
	}
	return digest(token), nil
}

type ipKey struct{}

func requestIP(ctx context.Context) string { ip, _ := ctx.Value(ipKey{}).(string); return digest(ip) }
func (s *Service) stay(req QuoteRequest) (int, error) {
	capacity := map[string]int{"single": 1, "single-balcony": 1, "double-twin": 2, "double-euro": 2, "vip": 4}[req.RoomTypeID]
	in, e1 := time.Parse("2006-01-02", req.CheckIn)
	out, e2 := time.Parse("2006-01-02", req.CheckOut)
	loc, _ := time.LoadLocation("Europe/Moscow")
	today := s.clock.Now().In(loc).Format("2006-01-02")
	if capacity == 0 || req.Guests < 1 || req.Guests > capacity || e1 != nil || e2 != nil || in.Year() < 1 || out.Year() > 3999 || req.CheckIn < today || !out.After(in) {
		return 0, failure("INVALID_REQUEST")
	}
	return int((out.Unix() - in.Unix()) / 86400), nil
}
func normalizeGuest(g Guest) (Guest, error) {
	for _, value := range []string{g.Name, g.Email, g.Phone} {
		if !utf8.ValidString(value) {
			return Guest{}, failure("INVALID_REQUEST")
		}
		for _, c := range value {
			if unicode.IsControl(c) {
				return Guest{}, failure("INVALID_REQUEST")
			}
		}
	}
	g.Name = strings.TrimSpace(g.Name)
	g.Email = strings.TrimSpace(g.Email)
	g.Phone = strings.TrimSpace(g.Phone)
	if n := utf8.RuneCountInString(g.Name); n < 1 || n > 150 {
		return Guest{}, failure("INVALID_REQUEST")
	}
	if g.Email != "" {
		address, err := mail.ParseAddress(g.Email)
		if err != nil || address.Address != g.Email || len(g.Email) > 254 {
			return Guest{}, failure("INVALID_REQUEST")
		}
		at := strings.LastIndexByte(g.Email, '@')
		g.Email = g.Email[:at+1] + strings.ToLower(g.Email[at+1:])
	}
	var phone strings.Builder
	for _, c := range g.Phone {
		if c == ' ' || c == '(' || c == ')' || c == '-' {
			continue
		}
		phone.WriteRune(c)
	}
	p := phone.String()
	if len(p) == 11 && p[0] == '8' {
		p = "7" + p[1:]
	}
	if !strings.HasPrefix(p, "+") {
		p = "+" + p
	}
	if len(p) < 9 || len(p) > 16 {
		return Guest{}, failure("INVALID_REQUEST")
	}
	for _, c := range p[1:] {
		if c < '0' || c > '9' {
			return Guest{}, failure("INVALID_REQUEST")
		}
	}
	g.Phone = p
	return g, nil
}
func safeError(err error) error {
	var e *bookingError
	if errors.As(err, &e) {
		return e
	}
	return failure("BOOKING_UNAVAILABLE")
}
func (s *Service) Quote(ctx context.Context, token string, req QuoteRequest) (QuoteResponse, error) {
	hash, err := contextHash(token)
	if err != nil {
		return QuoteResponse{}, err
	}
	nights, err := s.stay(req)
	if err != nil {
		return QuoteResponse{}, err
	}
	now := s.clock.Now()
	if err = s.store.limit(ctx, "quote", requestIP(ctx), now, time.Minute, 30); err != nil {
		return QuoteResponse{}, safeError(err)
	}
	room, err := s.rooms.Get(ctx, req.RoomTypeID)
	if err != nil || room.Price <= 0 || int64(room.Price) > math.MaxInt64/int64(nights) {
		return QuoteResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	available, err := s.client.Availability(ctx, req)
	if err != nil {
		return QuoteResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	if !available.Available {
		return QuoteResponse{}, failure("ROOM_UNAVAILABLE")
	}
	q := QuoteResponse{randomID(), now.Add(10 * time.Minute), req.RoomTypeID, req.CheckIn, req.CheckOut, req.Guests, true, PublicPrice{"RUB", nights, room.Price, room.Price.Mul(nights)}}
	if err = s.store.saveQuote(ctx, q, hash, now); err != nil {
		return QuoteResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	return q, nil
}
func (s *Service) replay(ctx context.Context, a attemptRow, hash string, req *PublicCreateRequest) (PublicReservationResponse, error) {
	if err := s.store.limit(ctx, "recovery", requestIP(ctx), s.clock.Now(), time.Minute, 30); err != nil {
		return PublicReservationResponse{}, safeError(err)
	}
	if a.ContextHash != hash {
		return PublicReservationResponse{}, failure("RESERVATION_NOT_FOUND")
	}
	if req != nil {
		var original CreateRequest
		if json.Unmarshal([]byte(a.Canonical), &original) != nil {
			return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
		}
		if req.QuoteID != a.QuoteID || req.Guest != original.Guest {
			return PublicReservationResponse{}, failure("IDEMPOTENCY_CONFLICT")
		}
	}
	return response(a)
}
func response(a attemptRow) (PublicReservationResponse, error) {
	if a.State == "pending" {
		return PublicReservationResponse{Status: "pending"}, nil
	}
	if a.State == "rejected" {
		return PublicReservationResponse{}, failure(a.Rejection)
	}
	var r Reservation
	if json.Unmarshal(a.Result, &r) != nil || !validReservation(r) {
		return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	p := publicPrice(r.Price)
	return PublicReservationResponse{ReservationID: r.ReservationID, ReservationNumber: r.ReservationNumber, Status: r.Status, RoomTypeID: r.RoomTypeID, CheckIn: r.CheckIn, CheckOut: r.CheckOut, Guests: r.Guests, Price: &p, ConfirmationEmailStatus: a.EmailState, ConfirmationSMSStatus: a.SMSState}, nil
}
func (s *Service) Create(ctx context.Context, token string, req PublicCreateRequest) (PublicReservationResponse, error) {
	hash, err := contextHash(token)
	if err != nil {
		return PublicReservationResponse{}, err
	}
	if !validKey(req.IdempotencyKey) || !validKey(req.QuoteID) {
		return PublicReservationResponse{}, failure("INVALID_REQUEST")
	}
	req.Guest, err = normalizeGuest(req.Guest)
	if err != nil {
		return PublicReservationResponse{}, err
	}
	a, err := s.store.attempt(ctx, req.IdempotencyKey)
	if err == nil {
		return s.replay(ctx, a, hash, &req)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	q, err := s.store.quote(ctx, req.QuoteID, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicReservationResponse{}, failure("RESERVATION_NOT_FOUND")
	}
	if err != nil {
		return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	now := s.clock.Now()
	if !now.Before(q.ExpiresAt) {
		return PublicReservationResponse{}, failure("QUOTE_EXPIRED")
	}
	saved := q.request(req.IdempotencyKey, req.Guest)
	if _, err = s.stay(QuoteRequest{saved.RoomTypeID, saved.CheckIn, saved.CheckOut, saved.Guests}); err != nil {
		return PublicReservationResponse{}, err
	}
	current, err := s.rooms.Get(ctx, q.RoomTypeID)
	if err != nil {
		return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	if current.Price != q.Nightly {
		return PublicReservationResponse{}, failure("PRICE_CHANGED")
	}
	raw, _ := json.Marshal(saved)
	a, fresh, err := s.store.begin(ctx, req, string(raw), hash, requestIP(ctx), digest(req.Guest.Email+"\x00"+req.Guest.Phone), randomID(), now)
	if err != nil {
		return PublicReservationResponse{}, safeError(err)
	}
	if !fresh {
		return s.replay(ctx, a, hash, &req)
	}
	// Once persisted, every uncertain outcome stays pending, even if the caller
	// disconnects or the local confirmation transaction loses its acknowledgement.
	s.dispatch(ctx, a, false)
	a, err = s.store.attempt(ctx, a.Key)
	if err != nil {
		return PublicReservationResponse{Status: "pending"}, nil
	}
	result, err := response(a)
	result.created = result.Status == "confirmed"
	return result, err
}
func (s *Service) Lookup(ctx context.Context, token, key string) (PublicReservationResponse, error) {
	hash, err := contextHash(token)
	if err != nil {
		return PublicReservationResponse{}, err
	}
	if !validKey(key) {
		return PublicReservationResponse{}, failure("INVALID_REQUEST")
	}
	// Count unknown/wrong-context probes as recovery attempts too.
	if err = s.store.limit(ctx, "recovery", requestIP(ctx), s.clock.Now(), time.Minute, 30); err != nil {
		return PublicReservationResponse{}, safeError(err)
	}
	a, err := s.store.attempt(ctx, key)
	if errors.Is(err, sql.ErrNoRows) || err == nil && a.ContextHash != hash {
		return PublicReservationResponse{}, failure("RESERVATION_NOT_FOUND")
	}
	if err != nil {
		return PublicReservationResponse{}, failure("BOOKING_UNAVAILABLE")
	}
	return response(a)
}

// dispatch never replaces the persisted request, even after quote expiry or a
// tariff/date change. Lookup 404 is not proof that the previous POST failed.
func (s *Service) dispatch(ctx context.Context, a attemptRow, recovery bool) {
	var req CreateRequest
	if json.Unmarshal([]byte(a.Canonical), &req) != nil {
		_ = s.store.release(ctx, a, s.clock.Now())
		return
	}
	var r Reservation
	var err error
	if recovery {
		r, err = s.client.Lookup(ctx, a.Key)
		var p *ProtocolError
		if errors.As(err, &p) && p.Code == "RESERVATION_NOT_FOUND" {
			r, err = s.client.Create(ctx, req)
		}
	} else {
		r, err = s.client.Create(ctx, req)
	}
	if err == nil {
		if !validReservation(r) || r.RoomTypeID != req.RoomTypeID || r.CheckIn != req.CheckIn || r.CheckOut != req.CheckOut || r.Guests != req.Guests || r.Price != req.Price {
			_ = s.store.release(ctx, a, s.clock.Now())
			return
		}
		_ = s.store.finish(ctx, a, r, req.Guest.Email != "", s.clock.Now())
		return
	}
	var protocol *ProtocolError
	if errors.As(err, &protocol) && (protocol.Code == "ROOM_UNAVAILABLE" || protocol.Code == "IDEMPOTENCY_CONFLICT" || protocol.Code == "INVALID_REQUEST") {
		_ = s.store.reject(ctx, a, protocol.Code)
		return
	}
	_ = s.store.release(ctx, a, s.clock.Now())
}
