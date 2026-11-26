package app

import "context"

// DeliveryOrderForTest exposes deliveryOrder to the external tests.
var DeliveryOrderForTest = deliveryOrder

// ProcessIikoOrderForTest runs a single iteration of the worker.
func (n *IikoOrders) ProcessIikoOrderForTest(ctx context.Context) (bool, error) {
	return n.processOrder(ctx)
}
