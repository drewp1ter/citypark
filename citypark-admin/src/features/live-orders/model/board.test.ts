import { describe, expect, it } from 'vitest'
import { applyEvent, groupOrders, type LiveOrder } from './board'

// The OpenAPI enum for status lacks `pending`, which the backend sends (API.md), so the status is a plain string here.
function order(id: number, patch: Omit<Partial<LiveOrder>, 'status'> & { status?: string } = {}): LiveOrder {
	return {
		id,
		createdAt: '2026-10-28T09:00:00Z',
		updatedAt: '2026-10-28T09:00:00Z',
		status: 'new',
		comment: '',
		deliveryTime: '',
		paymentType: 'cash',
		total: 500,
		deliveryPrice: 0,
		customer: { id: 1, name: 'Иван', phone: '+79990000000' },
		address: null,
		items: [],
		...patch,
	} as LiveOrder
}

const today = '2026-10-28'

describe('applyEvent', () => {
	it('inserts a new order', () => {
		const result = applyEvent([order(1)], { type: 'order.created', order: order(2) }, today)
		expect(result.map((o) => o.id)).toEqual([1, 2])
	})

	it('replaces an order with a newer updatedAt, and with the same one', () => {
		const paid = order(1, { status: 'paid', updatedAt: '2026-10-28T09:05:00Z' })
		const result = applyEvent([order(1)], { type: 'order.paid', order: paid }, today)
		expect(result).toEqual([paid])
		const again = order(1, { status: 'confirmed', updatedAt: '2026-10-28T09:05:00Z' })
		expect(applyEvent(result, { type: 'order.confirmed', order: again }, today)).toEqual([again])
	})

	it('ignores an older event', () => {
		const current = order(1, { status: 'paid', updatedAt: '2026-10-28T09:05:00Z' })
		const stale = order(1, { status: 'pending', updatedAt: '2026-10-28T09:01:00Z' })
		const orders = [current]
		expect(applyEvent(orders, { type: 'order.pending', order: stale }, today)).toBe(orders)
	})

	it('ignores an order of another Moscow day', () => {
		const orders = [order(1)]
		const yesterday = order(2, { createdAt: '2026-10-27T20:59:00Z' })
		expect(applyEvent(orders, { type: 'order.created', order: yesterday }, today)).toBe(orders)
		// 21:00 UTC is already the next day in Moscow.
		const midnight = order(3, { createdAt: '2026-10-27T21:00:00Z' })
		expect(applyEvent(orders, { type: 'order.created', order: midnight }, today)).toHaveLength(2)
	})

	it('leaves the list unchanged on hello', () => {
		const orders = [order(1)]
		expect(applyEvent(orders, { type: 'hello' }, today)).toBe(orders)
	})
})

describe('groupOrders', () => {
	it('puts each status in its column and drops canceled', () => {
		const board = groupOrders([
			order(1, { status: 'pending' }),
			order(2, { status: 'new' }),
			order(3, { status: 'paid' }),
			order(4, { status: 'confirmed' }),
			order(5, { status: 'canceled' }),
		])
		expect(board.pending.map((o) => o.id)).toEqual([1])
		expect(board.fresh.map((o) => o.id).sort()).toEqual([2, 3])
		expect(board.confirmed.map((o) => o.id)).toEqual([4])
	})

	it('sorts every column newest first', () => {
		const board = groupOrders([
			order(1, { createdAt: '2026-10-28T08:00:00Z' }),
			order(2, { createdAt: '2026-10-28T10:00:00Z', status: 'paid' }),
			order(3, { createdAt: '2026-10-28T09:00:00Z' }),
		])
		expect(board.fresh.map((o) => o.id)).toEqual([2, 3, 1])
	})
})
