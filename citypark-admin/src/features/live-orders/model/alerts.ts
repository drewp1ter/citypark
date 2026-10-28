import { formatRub } from '../../../shared/lib/format'
import type { LiveOrder } from './board'
import type { LiveOrderEvent } from './socket'

/** Only a paid order alerts (toast, tab counter, notification); other events just move cards. */
export function isPaidAlert(e: LiveOrderEvent): boolean {
	return e.type === 'order.paid'
}

export function titleWithCount(base: string, count: number): string {
	return count === 0 ? base : `(${count}) ${base}`
}

export function alertText(o: LiveOrder): string {
	return `Оплачен заказ №${o.id} — ${formatRub(o.total + o.deliveryPrice)}`
}
