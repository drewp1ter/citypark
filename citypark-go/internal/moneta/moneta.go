// Package moneta computes and checks MONETA.RU (PayAnyWay) payment signatures. It makes no network calls.
package moneta

import (
	"crypto/md5" //nolint:gosec // MD5 is the signature algorithm Moneta defines.
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/drewp1ter/citypark-go/internal/money"
)

// Currency is the only currency the shop accepts.
const Currency = "RUB"

// Signer signs payment forms and checks payment callbacks with one shop id and secret.
type Signer struct {
	id       string
	secret   string
	testMode string
}

// New: id is the shop id (MNT_ID), secret is the integrity code from the Moneta cabinet.
// testMode is stored as "1" or "0".
func New(id, secret string, testMode bool) *Signer {
	mode := "0"
	if testMode {
		mode = "1"
	}
	return &Signer{id: id, secret: secret, testMode: mode}
}

// FormatAmount returns the amount in rubles with exactly two decimals: 79000 -> "790.00".
func FormatAmount(k money.Kopecks) string {
	return fmt.Sprintf("%d.%02d", k/100, k%100)
}

// RequestSignature signs the payment form for an order. transactionID is the order's payment UUID;
// amount comes from FormatAmount.
func (s *Signer) RequestSignature(transactionID, amount string) string {
	return s.sum(s.id + transactionID + amount + Currency + s.testMode + s.secret)
}

// PaymentURL returns the MONETA.Assistant address for an order: base followed by the query
// MNT_ID, MNT_AMOUNT, MNT_TRANSACTION_ID, MNT_CURRENCY_CODE, MNT_TEST_MODE, MNT_SIGNATURE and,
// when not empty, MNT_SUCCESS_URL and MNT_FAIL_URL. amount comes from FormatAmount.
func (s *Signer) PaymentURL(base, transactionID, amount, successURL, failURL string) string {
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("MNT_ID", s.id)
	q.Set("MNT_AMOUNT", amount)
	q.Set("MNT_TRANSACTION_ID", transactionID)
	q.Set("MNT_CURRENCY_CODE", Currency)
	q.Set("MNT_TEST_MODE", s.testMode)
	q.Set("MNT_SIGNATURE", s.RequestSignature(transactionID, amount))
	if successURL != "" {
		q.Set("MNT_SUCCESS_URL", successURL)
	}
	if failURL != "" {
		q.Set("MNT_FAIL_URL", failURL)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Notification contains the signed fields of a Pay or Check request.
type Notification struct {
	MerchantID, TransactionID, OperationID, Amount, Currency, SubscriberID, TestMode, Command, Signature string
}

// ValidAmount accepts the protocol's canonical nonnegative two-decimal amounts.
func ValidAmount(amount string) bool {
	return regexp.MustCompile(`^[0-9]+\.[0-9]{2}$`).MatchString(amount)
}

// VerifyNotification verifies received fields before enforcing the shop configuration.
func (s *Signer) VerifyNotification(n Notification) bool {
	if n.TransactionID == "" || n.Signature == "" {
		return false
	}
	if n.Command != "" && n.Command != "CHECK" {
		return false
	}
	if n.Command == "" && (n.OperationID == "" || !ValidAmount(n.Amount)) {
		return false
	}
	if n.Amount != "" && !ValidAmount(n.Amount) {
		return false
	}
	want := s.sum(n.Command + n.MerchantID + n.TransactionID + n.OperationID + n.Amount + n.Currency + n.SubscriberID + n.TestMode + s.secret)
	return subtle.ConstantTimeCompare([]byte(want), []byte(strings.ToLower(n.Signature))) == 1 &&
		n.MerchantID == s.id && n.Currency == Currency && n.TestMode == s.testMode
}

// CheckResponse is the signed XML response required by MONETA.
type CheckResponse struct {
	XMLName       struct{} `xml:"MNT_RESPONSE"`
	MerchantID    string   `xml:"MNT_ID"`
	TransactionID string   `xml:"MNT_TRANSACTION_ID"`
	ResultCode    string   `xml:"MNT_RESULT_CODE"`
	Amount        string   `xml:"MNT_AMOUNT,omitempty"`
	Signature     string   `xml:"MNT_SIGNATURE"`
}

func (s *Signer) CheckResponse(transactionID, code, amount string) CheckResponse {
	return CheckResponse{MerchantID: s.id, TransactionID: transactionID, ResultCode: code, Amount: amount,
		Signature: s.sum(code + s.id + transactionID + s.secret)}
}

// sum returns the lowercase hex MD5 of data.
func (s *Signer) sum(data string) string {
	h := md5.Sum([]byte(data)) //nolint:gosec // see the import comment.
	return hex.EncodeToString(h[:])
}
