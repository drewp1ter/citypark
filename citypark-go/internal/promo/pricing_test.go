package promo

import (
	"github.com/drewp1ter/citypark-go/internal/money"
	"math"
	"testing"
	"time"
)

func TestCalculate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pct      int
		cap, min *money.Kopecks
		target   string
		want     money.Kopecks
		bad      bool
	}{
		{"one", 1, nil, nil, "all", 5, false}, {"ninety nine", 99, nil, nil, "all", 495, false},
		{"cap", 99, kptr(7), nil, "all", 7, false}, {"minimum", 10, nil, kptr(501), "all", 50, false},
		{"below", 10, nil, kptr(502), "all", 0, true}, {"union", 10, nil, nil, "selected", 30, false},
		{"no match", 10, nil, nil, "none", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Campaign{Input: Input{Enabled: true, Percentage: tc.pct, Target: tc.target, ProductIDs: []int64{1}, CategoryIDs: []int64{10}, MaximumDiscount: tc.cap, MinimumEligibleSubtotal: tc.min}}
			if tc.target == "none" {
				c.ProductIDs = nil
				c.CategoryIDs = nil
			}
			p, e := Calculate(c, []Line{{1, 10, 3, 100}, {2, 20, 1, 201}}, time.Time{})
			if (e != nil) != tc.bad {
				t.Fatal(e)
			}
			if !tc.bad && (p.Subtotal != 501 || p.Discount != tc.want || p.Total != 501-tc.want) {
				t.Fatal(p)
			}
		})
	}
}
func kptr(v money.Kopecks) *money.Kopecks { return &v }
func TestAllocate(t *testing.T) {
	c := Campaign{Input: Input{Enabled: true, Percentage: 50, Target: "all"}}
	p, e := Calculate(c, []Line{{2, 1, 1, 1}, {1, 1, 1, 1}, {3, 1, 1, 1}}, time.Time{})
	if e != nil || p.Discount != 1 || p.LineDiscounts[1] != 1 || p.LineDiscounts[2] != 0 {
		t.Fatalf("%+v %v", p, e)
	}
	p, e = Calculate(c, []Line{{1, 1, 3, 101}, {2, 1, 2, 100}}, time.Time{})
	var sum money.Kopecks
	for _, d := range p.LineDiscounts {
		sum += d
	}
	if e != nil || sum != p.Discount || p.Discount != 251 {
		t.Fatalf("%+v %v", p, e)
	}
	for _, lines := range [][]Line{{{1, 1, 2, math.MaxInt64}}, {{1, 1, 1, math.MaxInt64}, {2, 1, 1, 1}}, {{1, 1, 0, 1}}, {{1, 1, 1, -1}}, {{1, 1, 1, 1}, {1, 1, 1, 1}}} {
		if _, e := Calculate(c, lines, time.Time{}); e == nil {
			t.Fatal("accepted invalid/overflow input")
		}
	}
	p, e = Calculate(c, []Line{{1, 1, 1, math.MaxInt64}}, time.Time{})
	if e != nil || p.Discount != math.MaxInt64/2 {
		t.Fatalf("wide intermediate: %v %v", p, e)
	}
}
func TestCampaignWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := now.Add(time.Hour)
	c := Campaign{Input: Input{Enabled: true, Percentage: 10, Target: "all", StartsAt: &now, EndsAt: &end}}
	for _, tc := range []struct {
		at  time.Time
		bad bool
	}{{now, false}, {now.Add(-time.Nanosecond), true}, {end, true}} {
		_, e := Calculate(c, []Line{{1, 1, 1, 100}}, tc.at)
		if (e != nil) != tc.bad {
			t.Fatal(tc, e)
		}
	}
	c.StartsAt = nil
	c.EndsAt = nil
	if _, e := Calculate(c, []Line{{1, 1, 1, 100}}, now); e != nil {
		t.Fatal(e)
	}
	c.Enabled = false
	if _, e := Calculate(c, nil, now); e == nil {
		t.Fatal("disabled")
	}
	c.Enabled = true
	c.Percentage = 0
	if _, e := Calculate(c, nil, now); e == nil {
		t.Fatal("invalid percent")
	}
	c.Percentage = 10
	c.MaximumDiscount = kptr(-1)
	if _, e := Calculate(c, []Line{{1, 1, 1, 100}}, now); e == nil {
		t.Fatal("negative cap")
	}
}
