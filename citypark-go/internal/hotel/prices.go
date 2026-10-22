package hotel

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/drewp1ter/citypark-go/internal/httpx"
	"github.com/drewp1ter/citypark-go/internal/money"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

// BreakfastItem is one line of the breakfast menu.
type BreakfastItem struct {
	Name        string        `db:"name" json:"name"`
	ShortName   string        `db:"short_name" json:"shortName"`
	Price       money.Kopecks `db:"price_kopecks" json:"price"`
	DesktopOnly bool          `db:"desktop_only" json:"desktopOnly"`
}

// Fees are the three hotel fees.
type Fees struct {
	BreakfastDelivery   money.Kopecks `db:"breakfast_delivery_kopecks" json:"breakfastDelivery"`
	LateCheckoutPerHour money.Kopecks `db:"late_checkout_hour_kopecks" json:"lateCheckoutPerHour"`
	SmokingFine         money.Kopecks `db:"smoking_fine_kopecks" json:"smokingFine"`
}

// Prices is the stored state: the fees and the breakfast menu in order.
type Prices struct {
	Fees
	Breakfast []BreakfastItem `json:"breakfast"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// GetPrices returns the fees and the breakfast items ordered by sort.
func (s *Store) GetPrices(ctx context.Context) (Prices, error) {
	var row struct {
		Fees
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := s.db.GetContext(ctx, &row, `SELECT breakfast_delivery_kopecks, late_checkout_hour_kopecks,
		smoking_fine_kopecks, updated_at FROM hotel_prices WHERE id = 1`)
	if err != nil {
		return Prices{}, store.NotFound(err)
	}
	items := make([]BreakfastItem, 0)
	err = s.db.SelectContext(ctx, &items,
		`SELECT name, short_name, price_kopecks, desktop_only FROM breakfast_items ORDER BY sort, id`)
	if err != nil {
		return Prices{}, err
	}
	return Prices{Fees: row.Fees, Breakfast: items, UpdatedAt: row.UpdatedAt}, nil
}

// ReplacePrices stores the fees and replaces the whole breakfast list in one transaction, then returns the
// stored state. On a *sqlx.DB it opens the transaction; on a *sqlx.Tx it joins the caller's.
func (s *Store) ReplacePrices(ctx context.Context, fees Fees, items []BreakfastItem) (Prices, error) {
	switch db := s.db.(type) {
	case *sqlx.DB:
		err := store.InTx(ctx, db, func(tx *sqlx.Tx) error {
			return NewStore(tx).replacePrices(ctx, fees, items)
		})
		if err != nil {
			return Prices{}, err
		}
	case *sqlx.Tx:
		if err := s.replacePrices(ctx, fees, items); err != nil {
			return Prices{}, err
		}
	default:
		return Prices{}, fmt.Errorf("hotel: replace prices needs *sqlx.DB or *sqlx.Tx, got %T", s.db)
	}
	return s.GetPrices(ctx)
}

func (s *Store) replacePrices(ctx context.Context, fees Fees, items []BreakfastItem) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO hotel_prices (id, breakfast_delivery_kopecks, late_checkout_hour_kopecks, smoking_fine_kopecks)
		 VALUES (1, $1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET breakfast_delivery_kopecks = EXCLUDED.breakfast_delivery_kopecks,
		     late_checkout_hour_kopecks = EXCLUDED.late_checkout_hour_kopecks,
		     smoking_fine_kopecks = EXCLUDED.smoking_fine_kopecks, updated_at = now()`,
		int64(fees.BreakfastDelivery), int64(fees.LateCheckoutPerHour), int64(fees.SmokingFine))
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM breakfast_items`); err != nil {
		return err
	}
	for i, it := range items {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO breakfast_items (sort, name, short_name, price_kopecks, desktop_only)
			 VALUES ($1, $2, $3, $4, $5)`,
			i+1, it.Name, it.ShortName, int64(it.Price), it.DesktopOnly)
		if err != nil {
			return err
		}
	}
	return nil
}

// Prices are floats so that a third decimal can be detected (money.Kopecks would round it away while decoding);
// the fee fields are pointers so that 0 is valid and a missing field is an error.
type pricesRequest struct {
	BreakfastDelivery   *float64             `json:"breakfastDelivery" validate:"required,min=0,max=100000"`
	LateCheckoutPerHour *float64             `json:"lateCheckoutPerHour" validate:"required,min=0,max=100000"`
	SmokingFine         *float64             `json:"smokingFine" validate:"required,min=0,max=100000"`
	Breakfast           []breakfastItemInput `json:"breakfast" validate:"max=30,dive"`
}

type breakfastItemInput struct {
	Name        string   `json:"name" validate:"required,max=80"`
	ShortName   string   `json:"shortName" validate:"max=40"`
	Price       *float64 `json:"price" validate:"required,min=0,max=100000"`
	DesktopOnly bool     `json:"desktopOnly"`
}

func (h *Handler) getPrices(w http.ResponseWriter, r *http.Request) error {
	p, err := h.store.GetPrices(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		return httpx.NotFound()
	}
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, p)
	return nil
}

func (h *Handler) putPrices(w http.ResponseWriter, r *http.Request) error {
	var req pricesRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	for i := range req.Breakfast {
		it := &req.Breakfast[i]
		it.Name = strings.TrimSpace(it.Name)
		it.ShortName = strings.TrimSpace(it.ShortName)
	}
	fees, items, err := checkPrices(req)
	if err != nil {
		return err
	}
	p, err := h.store.ReplacePrices(r.Context(), fees, items)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, p)
	return nil
}

// checkPrices validates req and converts it to the values to store. All invalid fields are reported together.
func checkPrices(req pricesRequest) (Fees, []BreakfastItem, error) {
	fields := map[string]string{}
	var top *httpx.Error
	if err := validate.Struct(req); err != nil {
		if !errors.As(err, &top) {
			return Fees{}, nil, err
		}
		maps.Copy(fields, top.Fields)
	}

	checkPrecision := func(key string, v *float64) {
		if _, bad := fields[key]; !bad && v != nil && !hasKopecksPrecision(*v) {
			fields[key] = "не более 2 знаков после запятой"
		}
	}
	checkPrecision("breakfastDelivery", req.BreakfastDelivery)
	checkPrecision("lateCheckoutPerHour", req.LateCheckoutPerHour)
	checkPrecision("smokingFine", req.SmokingFine)
	for i, it := range req.Breakfast {
		checkPrecision(fmt.Sprintf("breakfast[%d].price", i), it.Price)
	}

	if len(fields) > 0 {
		if top == nil {
			top = httpx.NewError(http.StatusBadRequest, "Ошибка валидации, пожалуйста проверьте введенные вами данные")
		}
		top.Fields = fields
		return Fees{}, nil, top
	}

	items := make([]BreakfastItem, 0, len(req.Breakfast))
	for _, it := range req.Breakfast {
		items = append(items, BreakfastItem{
			Name: it.Name, ShortName: it.ShortName, Price: money.FromRubles(*it.Price), DesktopOnly: it.DesktopOnly,
		})
	}
	fees := Fees{
		BreakfastDelivery:   money.FromRubles(*req.BreakfastDelivery),
		LateCheckoutPerHour: money.FromRubles(*req.LateCheckoutPerHour),
		SmokingFine:         money.FromRubles(*req.SmokingFine),
	}
	return fees, items, nil
}

// hasKopecksPrecision reports whether rubles has at most two decimals.
func hasKopecksPrecision(rubles float64) bool {
	return math.Abs(rubles*100-float64(money.FromRubles(rubles))) < 1e-6
}
