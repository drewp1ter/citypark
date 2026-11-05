package moneta

import (
	"crypto/md5"
	"encoding/hex"
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
		txID     string
		amount   string
		want     string
	}{
		{"live", false, "15", "790.00", "8c65fcd4afe3dc4cb915e04aaa273574"},
		{"test mode", true, "15", "790.00", "047418b1c4a31fbc826ff955c7bdf010"},
		{"small amount", false, "15", "0.05", "0bd20edc998ca5c8e7be3780211c47f5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New("100", "secret", tt.testMode).RequestSignature(tt.txID, tt.amount)
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
			got := tt.signer.VerifyNotification(Notification{MerchantID: tt.signer.id, Currency: Currency, TestMode: tt.signer.testMode, TransactionID: tt.transaction, OperationID: tt.operation, Amount: tt.amount, Signature: tt.signature})
			if got != tt.want {
				t.Errorf("VerifyCallback = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaymentURL(t *testing.T) {
	const base = "https://demo.moneta.ru/assistant.htm"
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
			got := New("100", "secret", tt.testMode).PaymentURL(base, "15", "790.00", tt.success, tt.fail)
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

func TestCheckProtocol(t *testing.T) {
	s := New("54600817", "QWERTY", false)
	n := Notification{Command: "CHECK", MerchantID: "54600817", TransactionID: "FF790ABCD", Amount: "120.25", Currency: "RUB", TestMode: "0", Signature: "ea2d49048bdf11857f1b50270aedbc8d"}
	if !s.VerifyNotification(n) {
		t.Fatal("published request rejected")
	}
	r := s.CheckResponse(n.TransactionID, "402", "120.25")
	if r.Signature != "5ebb58862cf8781b62bcc2cc8d66913e" {
		t.Fatalf("response signature %s", r.Signature)
	}
	for _, field := range []string{"merchant", "currency", "mode", "subscriber", "command", "amount", "empty transaction", "empty signature"} {
		bad := n
		switch field {
		case "merchant":
			bad.MerchantID = "other"
		case "currency":
			bad.Currency = "USD"
		case "mode":
			bad.TestMode = "1"
		case "subscriber":
			bad.SubscriberID = "user"
		case "command":
			bad.Command = "PAY"
		case "amount":
			bad.Amount = "120.2"
		case "empty transaction":
			bad.TransactionID = ""
		case "empty signature":
			bad.Signature = ""
		}
		if s.VerifyNotification(bad) {
			t.Errorf("accepted %s", field)
		}
	}
}

func TestOptionalCheckFields(t *testing.T) {
	s := New("100", "secret", false)
	for _, subscriber := range []string{"", "customer"} {
		n := Notification{Command: "CHECK", MerchantID: "100", TransactionID: "15", Currency: "RUB", TestMode: "0", SubscriberID: subscriber}
		n.Signature = s.sum("CHECK10015RUB" + subscriber + "0secret")
		if !s.VerifyNotification(n) {
			t.Fatal("optional fields rejected")
		}
	}
}

func TestPaymentURLExistingQuery(t *testing.T) {
	s := New("100", "secret", false)
	u, err := url.Parse(s.PaymentURL("https://demo.moneta.ru/assistant.htm?moneta.locale=ru#form", "15", "790.00", "", ""))
	if err != nil || u.Query().Get("moneta.locale") != "ru" || u.Query().Get("MNT_AMOUNT") != "790.00" || u.Fragment != "form" {
		t.Fatalf("url %v, %v", u, err)
	}
	if s.PaymentURL("://%", "15", "790.00", "", "") != "" {
		t.Fatal("invalid URL accepted")
	}
}

func TestMonetaPaymentURLUsesTransactionID(t *testing.T) {
	const txID = "3f2b8c1e-9d4a-4e7b-a1c6-5d0e8f7a2b94"
	s := New("100", "secret", false)
	u, err := url.Parse(s.PaymentURL("https://demo.moneta.ru/assistant.htm", txID, "790.00", "", ""))
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum([]byte("100" + txID + "790.00" + "RUB" + "0" + "secret"))
	q := u.Query()
	if q.Get("MNT_TRANSACTION_ID") != txID || q.Get("MNT_SIGNATURE") != hex.EncodeToString(sum[:]) {
		t.Errorf("MNT_TRANSACTION_ID = %q, MNT_SIGNATURE = %q; want %q and %x", q.Get("MNT_TRANSACTION_ID"), q.Get("MNT_SIGNATURE"), txID, sum)
	}
}
