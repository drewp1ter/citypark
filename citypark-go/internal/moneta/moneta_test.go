package moneta

import (
	"testing"

	"github.com/drewp1ter/citypark-go/internal/money"
)

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		in   money.Kopecks
		want string
	}{
		{79000, "790.00"},
		{599, "5.99"},
		{5, "0.05"},
		{100050, "1000.50"},
		{0, "0.00"},
	}
	for _, tt := range tests {
		if got := FormatAmount(tt.in); got != tt.want {
			t.Errorf("FormatAmount(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRequestSignature(t *testing.T) {
	tests := []struct {
		name     string
		testMode bool
		orderID  int64
		amount   string
		want     string
	}{
		{"live", false, 15, "790.00", "8c65fcd4afe3dc4cb915e04aaa273574"},
		{"test mode", true, 15, "790.00", "047418b1c4a31fbc826ff955c7bdf010"},
		{"small amount", false, 15, "0.05", "0bd20edc998ca5c8e7be3780211c47f5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New("100", "secret", tt.testMode).RequestSignature(tt.orderID, tt.amount)
			if got != tt.want {
				t.Errorf("RequestSignature = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerifyCallback(t *testing.T) {
	const sig = "998b6268fe3a6c7f4eeb8ccec65a9566"
	tests := []struct {
		name        string
		signer      *Signer
		transaction string
		operation   string
		amount      string
		signature   string
		want        bool
	}{
		{"valid", New("100", "secret", false), "15", "555", "790.00", sig, true},
		{"upper case", New("100", "secret", false), "15", "555", "790.00", "998B6268FE3A6C7F4EEB8CCEC65A9566", true},
		{"test mode valid", New("100", "secret", true), "15", "555", "790.00", "700d371ec181494bb14aacaa23399dea", true},
		{"test mode with live signature", New("100", "secret", true), "15", "555", "790.00", sig, false},
		{"other amount", New("100", "secret", false), "15", "555", "790.01", sig, false},
		{"other operation id", New("100", "secret", false), "15", "556", "790.00", sig, false},
		{"other transaction id", New("100", "secret", false), "16", "555", "790.00", sig, false},
		{"other secret", New("100", "other", false), "15", "555", "790.00", sig, false},
		{"request signature", New("100", "secret", false), "15", "555", "790.00", "8c65fcd4afe3dc4cb915e04aaa273574", false},
		{"empty signature", New("100", "secret", false), "15", "555", "790.00", "", false},
		{"empty operation id", New("100", "secret", false), "15", "", "790.00", New("100", "secret", false).sum("100" + "15" + "" + "790.00" + Currency + "0" + "secret"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.signer.VerifyCallback(tt.transaction, tt.operation, tt.amount, tt.signature)
			if got != tt.want {
				t.Errorf("VerifyCallback = %v, want %v", got, tt.want)
			}
		})
	}
}
