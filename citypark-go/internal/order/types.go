package order

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
)

const (
	StatusNew       = "new"
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusCanceled  = "canceled"
	StatusConfirmed = "confirmed"
)

type Customer struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Address struct {
	ID      int64  `json:"id"`
	City    string `json:"city"`
	Address string `json:"address"`
}

type Item struct {
	Discount      money.Kopecks `db:"discount_kopecks" json:"discount"`
	PromoEligible bool          `db:"promo_eligible" json:"-"`
	OrderID       int64         `db:"order_id" json:"-"`
	ProductID     int64         `db:"product_id" json:"productId"`
	Name          string        `db:"name" json:"name"`
	CategoryName  string        `db:"category_name" json:"categoryName"`
	Amount        int           `db:"amount" json:"amount"`
	OfferedPrice  money.Kopecks `db:"offered_price_kopecks" json:"offeredPrice"`
	IikoID        string        `db:"iiko_id" json:"-"`
}

type Order struct {
	Subtotal        money.Kopecks `json:"subtotal"`
	Discount        money.Kopecks `json:"discount"`
	PromoCode       *string       `json:"promoCode"`
	PromoCodeID     *int64        `json:"-"`
	PromoPercentage *int          `json:"-"`
	ID              int64         `json:"id"`
	AccountID       *int64        `json:"-"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	Status          string        `json:"status"`
	Comment         string        `json:"comment"`
	DeliveryTime    string        `json:"deliveryTime"` // "HH:MM"; "" = as soon as possible
	Total           money.Kopecks `json:"total"`        // discounted food, without delivery
	DeliveryPrice   money.Kopecks `json:"deliveryPrice"`
	Customer        Customer      `json:"customer"`
	Address         *Address      `json:"address"` // nil = pickup, JSON null
	Items           []Item        `json:"items"`   // never nil
	ItemsHash       string        `json:"-"`
	PaymentID       string        `json:"-"`
	IikoOrderID     string        `json:"-"`
}

type ItemInput struct {
	ProductID int64 `json:"productId" validate:"gt=0"`
	Amount    int   `json:"amount" validate:"gt=0"`
}

// ItemsHash returns a stable hash of the items: sort by ProductID, join as "id:amount;",
// then hex(sha256). The order of the input does not matter.
func ItemsHash(items []ItemInput) string {
	sorted := slices.Clone(items)
	slices.SortFunc(sorted, func(a, b ItemInput) int {
		switch {
		case a.ProductID < b.ProductID:
			return -1
		case a.ProductID > b.ProductID:
			return 1
		}
		return 0
	})
	var sb strings.Builder
	for _, it := range sorted {
		sb.WriteString(strconv.FormatInt(it.ProductID, 10))
		sb.WriteByte(':')
		sb.WriteString(strconv.Itoa(it.Amount))
		sb.WriteByte(';')
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}
