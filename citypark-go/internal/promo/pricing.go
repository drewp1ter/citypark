package promo

import (
	"fmt"
	"github.com/drewp1ter/citypark-go/internal/money"
	"math"
	"math/bits"
	"slices"
	"time"
)

type Line struct {
	ProductID, CategoryID int64
	Amount                int
	UnitPrice             money.Kopecks
}
type Pricing struct {
	Subtotal, EligibleSubtotal, Discount, Total money.Kopecks
	LineDiscounts                               map[int64]money.Kopecks
}

// Calculate expects merged product lines and uses no mutable catalog or campaign data.
// Intermediate products use 128 bits; monetary outputs must fit signed 64 bits.
func Calculate(c Campaign, lines []Line, now time.Time) (Pricing, error) {
	p := Pricing{LineDiscounts: make(map[int64]money.Kopecks)}
	fail := func(message string) (Pricing, error) { return Pricing{}, &ValidationError{"promoCode", message} }
	if !c.Enabled || c.StartsAt != nil && now.Before(*c.StartsAt) || c.EndsAt != nil && !now.Before(*c.EndsAt) {
		return fail("unavailable")
	}
	if c.Percentage < 1 || c.Percentage > 99 {
		return Pricing{}, fmt.Errorf("invalid campaign percentage")
	}
	type allocation struct {
		id        int64
		gross     money.Kopecks
		remainder uint64
	}
	eligible := []allocation{}
	seen := map[int64]bool{}
	for _, l := range lines {
		if seen[l.ProductID] || l.Amount <= 0 || l.UnitPrice < 0 {
			return Pricing{}, fmt.Errorf("invalid pricing line")
		}
		seen[l.ProductID] = true
		if int64(l.Amount) > 0 && int64(l.UnitPrice) > math.MaxInt64/int64(l.Amount) {
			return Pricing{}, fmt.Errorf("pricing overflow")
		}
		gross := l.UnitPrice * money.Kopecks(l.Amount)
		if p.Subtotal > money.Kopecks(math.MaxInt64)-gross {
			return Pricing{}, fmt.Errorf("pricing overflow")
		}
		p.Subtotal += gross
		p.LineDiscounts[l.ProductID] = 0
		if c.Target == "all" || slices.Contains(c.ProductIDs, l.ProductID) || slices.Contains(c.CategoryIDs, l.CategoryID) {
			p.EligibleSubtotal += gross
			eligible = append(eligible, allocation{id: l.ProductID, gross: gross})
		}
	}
	if p.EligibleSubtotal <= 0 {
		return fail("no eligible items")
	}
	if c.MinimumEligibleSubtotal != nil && p.EligibleSubtotal < *c.MinimumEligibleSubtotal {
		return fail("minimum")
	}
	d, _ := mulDiv(uint64(p.EligibleSubtotal), uint64(c.Percentage), 100)
	p.Discount = money.Kopecks(d)
	if c.MaximumDiscount != nil && p.Discount > *c.MaximumDiscount {
		p.Discount = *c.MaximumDiscount
	}
	if p.Discount < 0 {
		return Pricing{}, fmt.Errorf("invalid campaign cap")
	}
	var assigned money.Kopecks
	for i := range eligible {
		a := &eligible[i]
		v, r := mulDiv(uint64(p.Discount), uint64(a.gross), uint64(p.EligibleSubtotal))
		a.remainder = r
		p.LineDiscounts[a.id] = money.Kopecks(v)
		assigned += money.Kopecks(v)
	}
	slices.SortFunc(eligible, func(a, b allocation) int {
		if a.remainder > b.remainder {
			return -1
		}
		if a.remainder < b.remainder {
			return 1
		}
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})
	for i := money.Kopecks(0); i < p.Discount-assigned; i++ {
		p.LineDiscounts[eligible[int(i)].id]++
	}
	p.Total = p.Subtotal - p.Discount
	return p, nil
}
func mulDiv(a, b, d uint64) (uint64, uint64) {
	hi, lo := bits.Mul64(a, b)
	return bits.Div64(hi, lo, d)
}
