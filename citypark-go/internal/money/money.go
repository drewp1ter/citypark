// Package money holds the Kopecks type used for every amount of money.
package money

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Kopecks is an amount of money in 1/100 of a ruble.
type Kopecks int64

// FromRubles converts rubles to kopecks, rounding to the nearest kopeck.
func FromRubles(r float64) Kopecks { return Kopecks(math.Round(r * 100)) }

// Rubles returns the amount in rubles.
func (k Kopecks) Rubles() float64 { return float64(k) / 100 }

// Mul multiplies the amount by n.
func (k Kopecks) Mul(n int) Kopecks { return k * Kopecks(n) }

// String formats rubles: "150" for whole rubles, "150.50" otherwise (always two decimals).
func (k Kopecks) String() string {
	sign := ""
	abs := int64(k)
	if abs < 0 {
		sign, abs = "-", -abs
	}
	if abs%100 == 0 {
		return fmt.Sprintf("%s%d", sign, abs/100)
	}
	return fmt.Sprintf("%s%d.%02d", sign, abs/100, abs%100)
}

// MarshalJSON writes rubles as a JSON number: 15000 -> 150, 599 -> 5.99, 1050 -> 10.5.
func (k Kopecks) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(k)/100, 'f', -1, 64)), nil
}

// UnmarshalJSON reads a JSON number of rubles. Anything that is not a number is an error.
func (k *Kopecks) UnmarshalJSON(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return fmt.Errorf("money: cannot unmarshal null into Kopecks")
	}
	var r float64
	if err := json.Unmarshal(b, &r); err != nil {
		return fmt.Errorf("money: %w", err)
	}
	*k = FromRubles(r)
	return nil
}
