package money

import (
	"encoding/json"
	"testing"
)

func TestFromRubles(t *testing.T) {
	tests := []struct {
		in   float64
		want Kopecks
	}{
		{5.99, 599},
		{320, 32000},
		{0.1 + 0.2, 30},
		{0, 0},
	}
	for _, tt := range tests {
		if got := FromRubles(tt.in); got != tt.want {
			t.Errorf("FromRubles(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestRubles(t *testing.T) {
	if got := Kopecks(599).Rubles(); got != 5.99 {
		t.Fatalf("Rubles() = %v, want 5.99", got)
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		in   Kopecks
		want string
	}{
		{15000, "150"},
		{15050, "150.50"},
		{599, "5.99"},
		{0, "0"},
		{5, "0.05"},
		{-15050, "-150.50"},
		{-15000, "-150"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Kopecks(%d).String() = %q, want %q", int64(tt.in), got, tt.want)
		}
	}
}

func TestJSON(t *testing.T) {
	type payload struct{ P Kopecks }

	marshal := []struct {
		in   Kopecks
		want string
	}{
		{15000, `{"P":150}`},
		{599, `{"P":5.99}`},
		{1050, `{"P":10.5}`},
	}
	for _, tt := range marshal {
		b, err := json.Marshal(payload{P: tt.in})
		if err != nil {
			t.Fatalf("Marshal(%d): %v", int64(tt.in), err)
		}
		if string(b) != tt.want {
			t.Errorf("Marshal(%d) = %s, want %s", int64(tt.in), b, tt.want)
		}
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"P": 5.99}`), &p); err != nil {
		t.Fatalf("Unmarshal number: %v", err)
	}
	if p.P != 599 {
		t.Errorf("Unmarshal 5.99 = %d, want 599", int64(p.P))
	}

	for _, bad := range []string{`{"P": "5"}`, `{"P": null}`, `{"P": true}`, `{"P": {}}`} {
		var q payload
		if err := json.Unmarshal([]byte(bad), &q); err == nil {
			t.Errorf("Unmarshal(%s) succeeded with %d, want error", bad, int64(q.P))
		}
	}
}

func TestMul(t *testing.T) {
	if got := Kopecks(599).Mul(3); got != 1797 {
		t.Fatalf("Mul(3) = %d, want 1797", int64(got))
	}
}
