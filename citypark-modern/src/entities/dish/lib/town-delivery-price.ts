import type { DeliveryTown } from '../model/types'

/** Delivery price for a town; same rule as the backend's `deliveryCost` (citypark-go/internal/order/service.go). */
export function townDeliveryPrice(town: DeliveryTown, subtotal: number): number {
	return town.freeFrom > 0 && subtotal >= town.freeFrom ? 0 : town.price
}
