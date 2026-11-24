package hotelbooking

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/httpx"
)

type cancellationConfig struct {
	origin string
	secret string
}

func (s *Service) EnableCancellation(origin, secret string) {
	if len(secret) >= 32 {
		s.cancellation = cancellationConfig{strings.TrimRight(origin, "/"), secret}
	}
}

// Derive the same booking-scoped capability on each email retry without storing
// a bearer credential in the database.
func (c cancellationConfig) token(key string) string {
	mac := hmac.New(sha256.New, []byte(c.secret))
	mac.Write([]byte("hotel-cancellation:v1:" + key))
	return key + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (c cancellationConfig) link(key string) string {
	if c.secret == "" {
		return ""
	}
	return c.origin + "/booking/cancel#" + c.token(key)
}

type cancellationStatus struct {
	Status string `json:"status"`
}

type cancellationClient interface {
	cancellation(context.Context, string, string) (cancellationStatus, error)
}

func (c *Client) cancellation(ctx context.Context, method, key string) (cancellationStatus, error) {
	var result cancellationStatus
	err := c.request(ctx, method, "/reservations/"+key+"/cancellation", nil, &result, false)
	if err == nil {
		switch result.Status {
		case "eligible", "closed", "canceled":
		default:
			err = ErrUnavailable
		}
	}
	return result, err
}

func (h *Handler) Cancellation(w http.ResponseWriter, r *http.Request) error {
	cfg := h.service.cancellation
	token := r.Header.Get("X-Hotel-Cancellation-Token")
	key, _, _ := strings.Cut(token, ".")
	if cfg.secret == "" || !validKey(key) || !hmac.Equal([]byte(token), []byte(cfg.token(key))) {
		return writeFailure(w, failure("RESERVATION_NOT_FOUND"))
	}
	row, err := h.service.store.attempt(r.Context(), key)
	if errors.Is(err, sql.ErrNoRows) {
		return writeFailure(w, failure("RESERVATION_NOT_FOUND"))
	}
	if err != nil {
		return writeFailure(w, err)
	}
	var reservation Reservation
	if row.State != "confirmed" || json.Unmarshal(row.Result, &reservation) != nil || !validReservation(reservation) {
		return writeFailure(w, failure("RESERVATION_NOT_FOUND"))
	}
	client, ok := h.service.client.(cancellationClient)
	if !ok {
		return writeFailure(w, ErrUnavailable)
	}
	status, err := client.cancellation(r.Context(), r.Method, key)
	if err != nil {
		if r.Method == http.MethodPost {
			httpx.JSON(w, http.StatusAccepted, cancellationStatus{Status: "pending"})
			return nil
		}
		return writeFailure(w, err)
	}
	httpx.JSON(w, http.StatusOK, struct {
		cancellationStatus
		ReservationNumber string `json:"reservationNumber"`
		CheckIn           string `json:"checkIn"`
		CheckOut          string `json:"checkOut"`
	}{status, reservation.ReservationNumber, reservation.CheckIn, reservation.CheckOut})
	return nil
}
