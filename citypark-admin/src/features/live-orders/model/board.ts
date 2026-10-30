import type { components } from '../../../shared/api/schema'
import { moscowToday } from '../../../shared/lib/today'
import type { LiveOrderEvent } from './socket'

export type LiveOrder = components['schemas']['Order']
export type Board = { pending: LiveOrder[]; fresh: LiveOrder[] }

/** Applies one event to the list of today's orders. `today` is the Moscow day as "YYYY-MM-DD". */
export function applyEvent(orders: LiveOrder[], e: LiveOrderEvent, today: string): LiveOrder[] {
	const incoming = e.order
	if (e.type === 'hello' || incoming === undefined) return orders
	if (moscowToday(new Date(incoming.createdAt)) !== today) return orders
	const known = orders.find((o) => o.id === incoming.id)
	if (known === undefined) return [...orders, incoming]
	if (Date.parse(incoming.updatedAt) < Date.parse(known.updatedAt)) return orders
	return orders.map((o) => (o.id === incoming.id ? incoming : o))
}

function newestFirst(a: LiveOrder, b: LiveOrder): number {
	return Date.parse(b.createdAt) - Date.parse(a.createdAt)
}

/** Splits orders into the two columns; canceled and historical confirmed orders are left out. */
export function groupOrders(orders: LiveOrder[]): Board {
	const pick = (...statuses: string[]) => orders.filter((o) => statuses.includes(o.status)).sort(newestFirst)
	return { pending: pick('pending'), fresh: pick('new', 'paid') }
}
