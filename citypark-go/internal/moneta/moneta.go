// Package moneta computes and checks MONETA.RU (PayAnyWay) payment signatures. It makes no network calls.
package moneta

import (
	"crypto/md5" //nolint:gosec // MD5 is the signature algorithm Moneta defines.
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
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

// RequestSignature signs the payment form for an order. amount comes from FormatAmount.
func (s *Signer) RequestSignature(orderID int64, amount string) string {
	return s.sum(s.id + strconv.FormatInt(orderID, 10) + amount + Currency + s.testMode + s.secret)
}

// PaymentURL returns the MONETA.Assistant address for an order: base followed by the query
// MNT_ID, MNT_AMOUNT, MNT_TRANSACTION_ID, MNT_CURRENCY_CODE, MNT_TEST_MODE, MNT_SIGNATURE and,
// when not empty, MNT_SUCCESS_URL and MNT_FAIL_URL. amount comes from FormatAmount.
func (s *Signer) PaymentURL(base string, orderID int64, amount, successURL, failURL string) string {
	q := url.Values{}
	q.Set("MNT_ID", s.id)
	q.Set("MNT_AMOUNT", amount)
	q.Set("MNT_TRANSACTION_ID", strconv.FormatInt(orderID, 10))
	q.Set("MNT_CURRENCY_CODE", Currency)
	q.Set("MNT_TEST_MODE", s.testMode)
	q.Set("MNT_SIGNATURE", s.RequestSignature(orderID, amount))
	if successURL != "" {
		q.Set("MNT_SUCCESS_URL", successURL)
	}
	if failURL != "" {
		q.Set("MNT_FAIL_URL", failURL)
	}
	return base + "?" + q.Encode()
}

// VerifyCallback reports whether signature is the right one for a payment callback.
func (s *Signer) VerifyCallback(transactionID, operationID, amount, signature string) bool {
	if transactionID == "" || operationID == "" || amount == "" || signature == "" {
		return false
	}
	want := s.sum(s.id + transactionID + operationID + amount + Currency + s.testMode + s.secret)
	return subtle.ConstantTimeCompare([]byte(want), []byte(strings.ToLower(signature))) == 1
}

// sum returns the lowercase hex MD5 of data.
func (s *Signer) sum(data string) string {
	h := md5.Sum([]byte(data)) //nolint:gosec // see the import comment.
	return hex.EncodeToString(h[:])
}
