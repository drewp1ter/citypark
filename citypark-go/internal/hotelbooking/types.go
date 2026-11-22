// Package hotelbooking implements the private 1C booking protocol.
package hotelbooking

import (
	"encoding/json"

	"github.com/drewp1ter/citypark-go/internal/money"
)

type AvailabilityRequest struct {
	RoomTypeID string `json:"roomTypeId"`
	CheckIn    string `json:"checkIn"`
	CheckOut   string `json:"checkOut"`
	Guests     int    `json:"guests"`
}

type AvailabilityResponse struct {
	RoomTypeID string `json:"roomTypeId"`
	CheckIn    string `json:"checkIn"`
	CheckOut   string `json:"checkOut"`
	Guests     int    `json:"guests"`
	Available  bool   `json:"available"`
}

type Guest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Price uses integer kopecks on this private wire, unlike public ruble JSON.
type Price struct {
	Currency            string        `json:"currency"`
	Nights              int           `json:"nights"`
	NightlyPriceKopecks money.Kopecks `json:"-"`
	TotalKopecks        money.Kopecks `json:"-"`
}

type wirePrice struct {
	Currency string `json:"currency"`
	Nights   int    `json:"nights"`
	Nightly  int64  `json:"nightlyPriceKopecks"`
	Total    int64  `json:"totalKopecks"`
}

func (p Price) MarshalJSON() ([]byte, error) {
	return json.Marshal(wirePrice{p.Currency, p.Nights, int64(p.NightlyPriceKopecks), int64(p.TotalKopecks)})
}

func (p *Price) UnmarshalJSON(data []byte) error {
	var w wirePrice
	if err := decode(data, &w); err != nil {
		return err
	}
	*p = Price{w.Currency, w.Nights, money.Kopecks(w.Nightly), money.Kopecks(w.Total)}
	return nil
}

type CreateRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
	RoomTypeID     string `json:"roomTypeId"`
	CheckIn        string `json:"checkIn"`
	CheckOut       string `json:"checkOut"`
	Guests         int    `json:"guests"`
	Guest          Guest  `json:"guest"`
	Price          Price  `json:"price"`
}

type Reservation struct {
	ReservationID string `json:"reservationId"`
	Status        string `json:"status"`
	RoomTypeID    string `json:"roomTypeId"`
	CheckIn       string `json:"checkIn"`
	CheckOut      string `json:"checkOut"`
	Guests        int    `json:"guests"`
	Price         Price  `json:"price"`
}
