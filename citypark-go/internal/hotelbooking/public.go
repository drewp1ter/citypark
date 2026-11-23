package hotelbooking

import (
	"fmt"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
)

type QuoteRequest = AvailabilityRequest

type PublicPrice struct {
	Currency      string        `json:"currency"`
	Nights        int           `json:"nights"`
	PricePerNight money.Kopecks `json:"pricePerNight"`
	Total         money.Kopecks `json:"total"`
}

type QuoteResponse struct {
	QuoteID    string      `json:"quoteId" validate:"required"`
	ExpiresAt  time.Time   `json:"expiresAt"`
	RoomTypeID string      `json:"roomTypeId"`
	CheckIn    string      `json:"checkIn"`
	CheckOut   string      `json:"checkOut"`
	Guests     int         `json:"guests"`
	Available  bool        `json:"available"`
	Price      PublicPrice `json:"price"`
}

type PublicCreateRequest struct {
	IdempotencyKey string `json:"idempotencyKey" validate:"required"`
	QuoteID        string `json:"quoteId" validate:"required"`
	Guest          Guest  `json:"guest"`
}

type PublicReservationResponse struct {
	ReservationID           string       `json:"reservationId,omitempty"`
	ReservationNumber       string       `json:"reservationNumber,omitempty"`
	Status                  string       `json:"status"`
	RoomTypeID              string       `json:"roomTypeId,omitempty"`
	CheckIn                 string       `json:"checkIn,omitempty"`
	CheckOut                string       `json:"checkOut,omitempty"`
	Guests                  int          `json:"guests,omitempty"`
	Price                   *PublicPrice `json:"price,omitempty"`
	ConfirmationEmailStatus string       `json:"confirmationEmailStatus,omitempty"`
	created                 bool
}

type bookingError struct {
	Status  int    `json:"statusCode"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *bookingError) Error() string { return fmt.Sprintf("hotel booking: %s", e.Code) }
func failure(code string) *bookingError {
	status, message := 503, "Бронирование временно недоступно"
	switch code {
	case "INVALID_REQUEST":
		status, message = 400, "Проверьте данные бронирования"
	case "RESERVATION_NOT_FOUND":
		status, message = 404, "Бронирование не найдено"
	case "ROOM_UNAVAILABLE":
		status, message = 409, "На выбранные даты нет свободного номера"
	case "PRICE_CHANGED":
		status, message = 409, "Стоимость проживания изменилась. Подтвердите новую стоимость"
	case "IDEMPOTENCY_CONFLICT":
		status, message = 409, "Ключ бронирования уже использован с другими данными"
	case "QUOTE_EXPIRED":
		status, message = 409, "Расчёт стоимости устарел. Обновите стоимость"
	case "RATE_LIMITED":
		status, message = 429, "Слишком много запросов. Попробуйте позже"
	default:
		code = "BOOKING_UNAVAILABLE"
	}
	return &bookingError{status, code, message}
}
func publicPrice(p Price) PublicPrice {
	return PublicPrice{p.Currency, p.Nights, p.NightlyPriceKopecks, p.TotalKopecks}
}
