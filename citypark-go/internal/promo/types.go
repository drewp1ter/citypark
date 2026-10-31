// Package promo persists promo campaigns and their catalog targets.
package promo

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/drewp1ter/citypark-go/internal/money"
)

type Input struct {
	Code                    string         `json:"code" db:"code"`
	Percentage              int            `json:"percentage" db:"percentage"`
	Enabled                 bool           `json:"enabled" db:"enabled"`
	Target                  string         `json:"target" db:"target"`
	ProductIDs              []int64        `json:"productIds" db:"-"`
	CategoryIDs             []int64        `json:"categoryIds" db:"-"`
	MinimumEligibleSubtotal *money.Kopecks `json:"minimumEligibleSubtotal" db:"minimum_eligible_subtotal_kopecks"`
	MaximumDiscount         *money.Kopecks `json:"maximumDiscount" db:"maximum_discount_kopecks"`
	StartsAt                *time.Time     `json:"startsAt" db:"starts_at"`
	EndsAt                  *time.Time     `json:"endsAt" db:"ends_at"`
}

type Campaign struct {
	ID int64 `json:"id" db:"id"`
	Input
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

// ValidationError identifies a field without coupling persistence to HTTP.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid promo %s: %s", e.Field, e.Message)
}

func normalizeCode(code string) string {
	b := []byte(strings.TrimSpace(code))
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
func normalize(in Input) (Input, error) {
	in.Code = normalizeCode(in.Code)
	bad := func(field, message string) (Input, error) { return Input{}, &ValidationError{field, message} }
	if len(in.Code) < 3 || len(in.Code) > 32 {
		return bad("code", "invalid format")
	}
	for _, c := range in.Code {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return bad("code", "invalid format")
		}
	}
	if in.Percentage < 1 || in.Percentage > 99 {
		return bad("percentage", "must be between 1 and 99")
	}
	if in.MinimumEligibleSubtotal != nil && *in.MinimumEligibleSubtotal <= 0 {
		return bad("minimumEligibleSubtotal", "must be positive")
	}
	if in.MaximumDiscount != nil && *in.MaximumDiscount <= 0 {
		return bad("maximumDiscount", "must be positive")
	}
	if in.StartsAt != nil && in.EndsAt != nil && !in.EndsAt.After(*in.StartsAt) {
		return bad("endsAt", "must follow startsAt")
	}
	in.ProductIDs = slices.Compact(slices.Sorted(slices.Values(in.ProductIDs)))
	in.CategoryIDs = slices.Compact(slices.Sorted(slices.Values(in.CategoryIDs)))
	if in.ProductIDs == nil {
		in.ProductIDs = []int64{}
	}
	if in.CategoryIDs == nil {
		in.CategoryIDs = []int64{}
	}
	n := len(in.ProductIDs) + len(in.CategoryIDs)
	if (in.Target != "all" && in.Target != "selected") || (in.Target == "all" && n != 0) || (in.Target == "selected" && n == 0) {
		return bad("target", "invalid selection")
	}
	return in, nil
}
