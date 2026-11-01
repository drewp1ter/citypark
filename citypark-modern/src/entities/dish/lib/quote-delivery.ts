import { formatPrice } from '@/shared/lib'
import type { DeliveryInfo } from '../model/delivery'
import { townDeliveryPrice } from './town-delivery-price'

export type DeliveryMethod = 'delivery' | 'pickup'
/** "city" = Белореченск, "outskirts" = СОТ and ДНТ. */
export type DeliveryZone = 'city' | 'outskirts'

export type DeliveryQuote = {
	/** Rubles. `null` = the price is not fixed, the operator confirms it by phone. */
	amount: number | null
	/** What to show: "0 ₽", "200–350 ₽", "уточним". */
	label: string
}

export function quoteDelivery(
	input: { method: DeliveryMethod; zone: DeliveryZone; subtotal: number },
	delivery: DeliveryInfo
): DeliveryQuote {
	if (input.method === 'pickup') return { amount: 0, label: formatPrice(0) }
	if (input.zone === 'outskirts') return { amount: null, label: delivery.outskirtsRange }
	if (delivery.city) {
		const amount = townDeliveryPrice(delivery.city, input.subtotal)
		return { amount, label: formatPrice(amount) }
	}
	return { amount: null, label: 'уточним' }
}
