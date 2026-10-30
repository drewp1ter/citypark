import { describe, expect, it } from 'vitest'
import { alertText, isPaidAlert, titleWithCount } from './alerts'
import type { LiveOrder } from './board'

describe('isPaidAlert', () => {
	it('is true only for order.paid', () => {
		expect(isPaidAlert({ type: 'order.paid' })).toBe(true)
		for (const type of ['hello', 'order.pending'] as const) {
			expect(isPaidAlert({ type })).toBe(false)
		}
	})
})

describe('titleWithCount', () => {
	it('adds the counter only when it is above zero', () => {
		expect(titleWithCount('Заказы онлайн', 0)).toBe('Заказы онлайн')
		expect(titleWithCount('Заказы онлайн', 3)).toBe('(3) Заказы онлайн')
	})
})

describe('alertText', () => {
	it('shows the number and the sum with delivery, with non-breaking spaces', () => {
		const order = { id: 123, total: 1300, deliveryPrice: 150 } as LiveOrder
		expect(alertText(order)).toBe('Оплачен заказ №123 — 1 450 ₽')
	})
})
