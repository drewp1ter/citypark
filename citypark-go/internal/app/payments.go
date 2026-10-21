package app

import (
	"github.com/drewp1ter/citypark-go/internal/moneta"
	"github.com/drewp1ter/citypark-go/internal/order"
)

// MonetaPayments implements order.Payments with a moneta.Signer.
type MonetaPayments struct {
	signer *moneta.Signer
}

var _ order.Payments = (*MonetaPayments)(nil)

func NewMonetaPayments(signer *moneta.Signer) *MonetaPayments {
	return &MonetaPayments{signer: signer}
}

func (p *MonetaPayments) Form(r order.PaymentRequest) order.PaymentForm {
	amount := moneta.FormatAmount(r.Amount)
	return order.PaymentForm{Amount: amount, Signature: p.signer.RequestSignature(r.OrderID, amount)}
}

func (p *MonetaPayments) Verify(c order.PaymentCallback) bool {
	return p.signer.VerifyCallback(c.TransactionID, c.OperationID, c.Amount, c.Signature)
}
