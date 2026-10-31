package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/catalog"
	"github.com/drewp1ter/citypark-go/internal/deliveryprice"
	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/promo"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

type QuoteInput struct {
	Items       []ItemInput `json:"items" validate:"required,min=1,dive"`
	Fulfillment string      `json:"fulfillment" validate:"required,oneof=delivery pickup"`
	City        string      `json:"city" validate:"required_if=Fulfillment delivery"`
	PromoCode   string      `json:"promoCode"`
}
type QuoteItem struct {
	ProductID    int64         `json:"productId"`
	Amount       int           `json:"amount"`
	OfferedPrice money.Kopecks `json:"offeredPrice"`
	Discount     money.Kopecks `json:"discount"`
}
type Quote struct {
	Subtotal         money.Kopecks `json:"subtotal"`
	EligibleSubtotal money.Kopecks `json:"eligibleSubtotal"`
	Discount         money.Kopecks `json:"discount"`
	Total            money.Kopecks `json:"total"`
	DeliveryPrice    money.Kopecks `json:"deliveryPrice"`
	PayableTotal     money.Kopecks `json:"payableTotal"`
	PromoCode        *string       `json:"promoCode"`
	Items            []QuoteItem   `json:"items"`
}

func quoteCodeError(message string) error {
	e := httpx.NewError(400, validationMessage)
	e.Fields = map[string]string{"promoCode": message}
	return e
}

// Quote reads a repeatable snapshot without creating any business records or payments.
func (s *Service) Quote(ctx context.Context, in QuoteInput) (Quote, error) {
	if err := validate.Struct(in); err != nil {
		return Quote{}, err
	}
	// Check duplicate quantity addition before using the shared merge helper.
	counts := map[int64]int{}
	for _, it := range in.Items {
		if counts[it.ProductID] > math.MaxInt-it.Amount {
			return Quote{}, httpx.BadRequest("Некорректное количество.")
		}
		counts[it.ProductID] += it.Amount
	}
	items := mergeItems(in.Items)
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return Quote{}, err
	}
	defer tx.Rollback()
	var q Quote
	q.Items = make([]QuoteItem, 0, len(items))
	var campaign *promo.Campaign
	code := strings.TrimSpace(in.PromoCode)
	if code != "" {
		// Validate format without invoking campaign configuration validation.
		code = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r - 32
			}
			return r
		}, code)
		valid := len(code) >= 3 && len(code) <= 32
		for _, c := range code {
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
				valid = false
			}
		}
		if !valid {
			return Quote{}, quoteCodeError("Промокод должен содержать от 3 до 32 латинских букв, цифр, дефисов или подчёркиваний.")
		}
		c, e := promo.NewStore(tx).ByCode(ctx, code)
		if errors.Is(e, store.ErrNotFound) {
			return Quote{}, quoteCodeError("Промокод недействителен.")
		}
		if e != nil {
			return Quote{}, e
		}
		campaign = &c
		q.PromoCode = &c.Code
	}
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.ProductID
	}
	products, err := catalog.NewStore(tx).ProductsByIDs(ctx, ids)
	if err != nil {
		return Quote{}, err
	}
	if len(products) != len(ids) {
		return Quote{}, httpx.BadRequest("Продукт не найден.")
	}
	byID := map[int64]catalog.Product{}
	for _, p := range products {
		byID[p.ID] = p
	}
	lines := make([]promo.Line, 0, len(items))
	now := s.clock.Now()
	for _, it := range items {
		p := byID[it.ProductID]
		if err := checkWindows(p, now); err != nil {
			return Quote{}, err
		}
		if p.Price < 0 || int64(p.Price) > math.MaxInt64/int64(it.Amount) {
			return Quote{}, fmt.Errorf("quote overflow")
		}
		gross := p.Price * money.Kopecks(it.Amount)
		if q.Subtotal > money.Kopecks(math.MaxInt64)-gross {
			return Quote{}, fmt.Errorf("quote overflow")
		}
		q.Subtotal += gross
		lines = append(lines, promo.Line{ProductID: p.ID, CategoryID: p.CategoryID, Amount: it.Amount, UnitPrice: p.Price})
		q.Items = append(q.Items, QuoteItem{ProductID: p.ID, Amount: it.Amount, OfferedPrice: p.Price})
	}
	q.Total = q.Subtotal
	if campaign != nil {
		p, e := promo.Calculate(*campaign, lines, now)
		if e != nil {
			var v *promo.ValidationError
			if errors.As(e, &v) {
				messages := map[string]string{"unavailable": "Промокод недействителен.", "no eligible items": "В корзине нет блюд для этого промокода.", "minimum": "Недостаточная сумма подходящих блюд для промокода."}
				return Quote{}, quoteCodeError(messages[v.Message])
			}
			return Quote{}, e
		}
		q.EligibleSubtotal = p.EligibleSubtotal
		q.Discount = p.Discount
		q.Total = p.Total
		for i := range q.Items {
			q.Items[i].Discount = p.LineDiscounts[q.Items[i].ProductID]
		}
	}
	if in.Fulfillment == "delivery" {
		dp, e := deliveryprice.NewStore(tx).ByName(ctx, in.City)
		if errors.Is(e, store.ErrNotFound) {
			return Quote{}, httpx.BadRequest("Указано не существующее имя населенного пункта.")
		}
		if e != nil {
			return Quote{}, e
		}
		q.DeliveryPrice = deliveryCost(dp, q.Total)
	}
	if q.DeliveryPrice < 0 || q.Total > money.Kopecks(math.MaxInt64)-q.DeliveryPrice {
		return Quote{}, fmt.Errorf("quote overflow")
	}
	q.PayableTotal = q.Total + q.DeliveryPrice
	if err = tx.Commit(); err != nil {
		return Quote{}, err
	}
	return q, nil
}
func (h *Handler) quote(w http.ResponseWriter, r *http.Request) error {
	var in QuoteInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	q, err := h.svc.Quote(r.Context(), in)
	if err != nil {
		return err
	}
	httpx.JSON(w, 200, q)
	return nil
}
