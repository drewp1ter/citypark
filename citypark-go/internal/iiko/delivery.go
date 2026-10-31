package iiko

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	ServiceDeliveryByCourier = "DeliveryByCourier"
	ServiceDeliveryByClient  = "DeliveryByClient" // pickup
)

// DeliveryOrder is the "order" object of deliveries/create.
type DeliveryOrder struct {
	DiscountsInfo    *DeliveryDiscountsInfo `json:"discountsInfo,omitempty"`
	ID               string                 `json:"id"`                       // GUID, made by the caller
	ExternalNumber   string                 `json:"externalNumber,omitempty"` // our order id
	Phone            string                 `json:"phone"`
	OrderServiceType string                 `json:"orderServiceType"`
	DeliveryPoint    *DeliveryPoint         `json:"deliveryPoint,omitempty"` // nil for pickup
	Comment          string                 `json:"comment,omitempty"`
	Customer         DeliveryCustomer       `json:"customer"`
	Items            []DeliveryItem         `json:"items"`
	Payments         []DeliveryPayment      `json:"payments"`
}

type DeliveryCustomer struct {
	Type string `json:"type"` // "one-time"
	Name string `json:"name"`
}

type DeliveryPoint struct {
	Address DeliveryAddress `json:"address"`
	Comment string          `json:"comment,omitempty"`
}

type DeliveryAddress struct {
	Type   string         `json:"type"` // "legacy"
	Street DeliveryStreet `json:"street"`
	House  string         `json:"house"`
}

type DeliveryStreet struct {
	Name string `json:"name"`
	City string `json:"city"`
}

type DeliveryDiscountsInfo struct {
	Discounts []DeliveryRMSDiscount `json:"discounts"`
}
type DeliveryRMSDiscount struct {
	Type               string   `json:"type"`
	DiscountTypeID     string   `json:"discountTypeId"`
	Sum                float64  `json:"sum"`
	SelectivePositions []string `json:"selectivePositions"`
}

type DeliveryItem struct {
	PositionID string  `json:"positionId,omitempty"`
	Type       string  `json:"type"` // "Product"
	ProductID  string  `json:"productId"`
	Amount     float64 `json:"amount"`
	Price      float64 `json:"price"` // rubles per unit
}

type DeliveryPayment struct {
	PaymentTypeKind       string  `json:"paymentTypeKind"`
	Sum                   float64 `json:"sum"` // rubles
	PaymentTypeID         string  `json:"paymentTypeId"`
	IsProcessedExternally bool    `json:"isProcessedExternally"`
}

type DeliveryResult struct{ ID string } // iiko's order id

// DeliveryError: iiko answered creationStatus "Error" (or the command ended with "Error").
type DeliveryError struct{ Reason string }

func (e *DeliveryError) Error() string { return "iiko: delivery not created: " + e.Reason }

// ErrDeliveryPending: iiko still reported InProgress after the last status check. The order may yet appear in iikoFront.
var ErrDeliveryPending = errors.New("iiko: delivery creation not confirmed")

const unknownReason = "unknown error"

// deliveryRequestError preserves error matching and the staff-facing message while
// exposing request identifiers to structured logging. Never log the order payload.
type deliveryRequestError struct {
	err                              error
	endpoint, correlationID, orderID string
}

func (e *deliveryRequestError) Error() string { return e.err.Error() }
func (e *deliveryRequestError) Unwrap() error { return e.err }
func (e *deliveryRequestError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("message", e.err.Error()),
		slog.String("endpoint", e.endpoint),
		slog.String("correlation_id", e.correlationID),
		slog.String("iiko_order_id", e.orderID),
	)
}

// commandErrorReason only extracts a message; arbitrary exception data may contain
// customer details and must not be dumped into logs or Telegram notifications.
func commandErrorReason(reason string, exception json.RawMessage) string {
	if reason = strings.TrimSpace(reason); reason != "" {
		return reason
	}
	var message string
	if json.Unmarshal(exception, &message) == nil && strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	var detail struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(exception, &detail) == nil && strings.TrimSpace(detail.Message) != "" {
		return strings.TrimSpace(detail.Message)
	}
	return unknownReason
}

// Permanent reports whether err cannot be fixed by sending the same request again:
// a *DeliveryError, or a *StatusError with status 400.
func Permanent(err error) bool {
	var de *DeliveryError
	if errors.As(err, &de) {
		return true
	}
	var se *StatusError
	return errors.As(err, &se) && se.Status == http.StatusBadRequest
}

// CreateDelivery sends the order to deliveries/create on the given terminal group and waits until it is created.
func (c *Client) CreateDelivery(ctx context.Context, terminalGroupID string, o DeliveryOrder) (result DeliveryResult, err error) {
	const endpoint = "1/deliveries/create"
	stage, correlationID := endpoint, ""
	defer func() {
		if err != nil {
			err = &deliveryRequestError{err: err, endpoint: stage, correlationID: correlationID, orderID: o.ID}
		}
	}()
	body := map[string]any{
		"organizationId":      c.organizationID,
		"terminalGroupId":     terminalGroupID,
		"createOrderSettings": map[string]any{"transportToFrontTimeout": 180},
		"order":               o,
	}
	var resp struct {
		CorrelationID string `json:"correlationId"`
		OrderInfo     struct {
			ID             string `json:"id"`
			CreationStatus string `json:"creationStatus"`
			ErrorInfo      *struct {
				Code        string `json:"code"`
				Message     string `json:"message"`
				Description string `json:"description"`
			} `json:"errorInfo"`
		} `json:"orderInfo"`
	}
	err = c.withToken(ctx, func(token string) error {
		return c.post(ctx, endpoint, token, body, &resp)
	})
	if err != nil {
		return DeliveryResult{}, err
	}
	correlationID = resp.CorrelationID

	switch resp.OrderInfo.CreationStatus {
	case "Success":
		return DeliveryResult{ID: resp.OrderInfo.ID}, nil
	case "Error":
		reason := unknownReason
		if ei := resp.OrderInfo.ErrorInfo; ei != nil {
			for _, s := range []string{ei.Message, ei.Description, ei.Code} {
				if s != "" {
					reason = s
					break
				}
			}
		}
		return DeliveryResult{}, &DeliveryError{Reason: reason}
	case "InProgress":
		stage = "1/commands/status"
		return c.waitDelivery(ctx, resp.CorrelationID, resp.OrderInfo.ID)
	default:
		return DeliveryResult{}, fmt.Errorf("iiko: %s: unknown creationStatus %q", endpoint, resp.OrderInfo.CreationStatus)
	}
}

// waitDelivery polls commands/status until the command ends or the checks run out.
// Each check takes the token lock on its own, so a menu sync is not blocked while we wait.
func (c *Client) waitDelivery(ctx context.Context, correlationID, orderID string) (DeliveryResult, error) {
	const endpoint = "1/commands/status"
	body := map[string]string{"organizationId": c.organizationID, "correlationId": correlationID}

	for range c.pollTimes {
		select {
		case <-time.After(c.pollEvery):
		case <-ctx.Done():
			return DeliveryResult{}, ctx.Err()
		}

		var st struct {
			State       string          `json:"state"`
			ErrorReason string          `json:"errorReason"`
			Exception   json.RawMessage `json:"exception"`
		}
		err := c.withToken(ctx, func(token string) error {
			return c.post(ctx, endpoint, token, body, &st)
		})
		if err != nil {
			return DeliveryResult{}, err
		}

		switch st.State {
		case "Success":
			return DeliveryResult{ID: orderID}, nil
		case "Error":
			return DeliveryResult{}, &DeliveryError{Reason: commandErrorReason(st.ErrorReason, st.Exception)}
		case "InProgress":
		default:
			return DeliveryResult{}, fmt.Errorf("iiko: %s: unknown state %q", endpoint, st.State)
		}
	}
	return DeliveryResult{}, ErrDeliveryPending
}
