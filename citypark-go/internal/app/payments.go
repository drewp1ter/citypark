package app

import (
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/order"
)

// MonetaPayments implements order.Payments with a moneta.Signer.
type MonetaPayments struct {
	signer *moneta.Signer
	pages  PaymentPages
}

// PaymentPages are the Moneta addresses the customer is sent to and back from.
type PaymentPages struct{ Base, Success, Fail string }

var _ order.Payments = (*MonetaPayments)(nil)

func NewMonetaPayments(signer *moneta.Signer, pages PaymentPages) *MonetaPayments {
	return &MonetaPayments{signer: signer, pages: pages}
}

func (p *MonetaPayments) Form(r order.PaymentRequest) order.PaymentForm {
	amount := moneta.FormatAmount(r.Amount)
	return order.PaymentForm{
		Amount:    amount,
		Signature: p.signer.RequestSignature(r.TransactionID, amount),
		URL:       p.signer.PaymentURL(p.pages.Base, r.TransactionID, amount, p.pages.Success, p.pages.Fail),
	}
}

func (p *MonetaPayments) Verify(c order.PaymentCallback) bool {
	return p.signer.VerifyNotification(c)
}

func (p *MonetaPayments) CheckResponse(transactionID, code, amount string) moneta.CheckResponse {
	return p.signer.CheckResponse(transactionID, code, amount)
}
