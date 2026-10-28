package moneta

import (
	"net/url"
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

func TestPaymentURL(t *testing.T) {
	const base = "https://assistant.demo.moneta.ru/"
	tests := []struct {
		name          string
		testMode      bool
		success, fail string
		want          map[string]string
	}{
		{"live", false, "http://localhost:3001/checkout/done", "http://localhost:3001/checkout?payment=failed", map[string]string{
			"MNT_ID": "100", "MNT_AMOUNT": "790.00", "MNT_TRANSACTION_ID": "15", "MNT_CURRENCY_CODE": "RUB",
			"MNT_TEST_MODE": "0", "MNT_SIGNATURE": "8c65fcd4afe3dc4cb915e04aaa273574",
			"MNT_SUCCESS_URL": "http://localhost:3001/checkout/done", "MNT_FAIL_URL": "http://localhost:3001/checkout?payment=failed",
		}},
		{"test mode", true, "http://s", "http://f", map[string]string{
			"MNT_ID": "100", "MNT_AMOUNT": "790.00", "MNT_TRANSACTION_ID": "15", "MNT_CURRENCY_CODE": "RUB",
			"MNT_TEST_MODE": "1", "MNT_SIGNATURE": "047418b1c4a31fbc826ff955c7bdf010",
			"MNT_SUCCESS_URL": "http://s", "MNT_FAIL_URL": "http://f",
		}},
		{"no return pages", false, "", "", map[string]string{
			"MNT_ID": "100", "MNT_AMOUNT": "790.00", "MNT_TRANSACTION_ID": "15", "MNT_CURRENCY_CODE": "RUB",
			"MNT_TEST_MODE": "0", "MNT_SIGNATURE": "8c65fcd4afe3dc4cb915e04aaa273574",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New("100", "secret", tt.testMode).PaymentURL(base, 15, "790.00", tt.success, tt.fail)
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("PaymentURL %q is not a URL: %v", got, err)
			}
			if head := u.Scheme + "://" + u.Host + u.Path; head != base {
				t.Errorf("address = %q, want %q", head, base)
			}
			q := u.Query()
			if len(q) != len(tt.want) {
				t.Errorf("query has %d keys, want %d: %v", len(q), len(tt.want), q)
			}
			for k, v := range tt.want {
				if q.Get(k) != v {
					t.Errorf("%s = %q, want %q", k, q.Get(k), v)
				}
			}
		})
	}
}
