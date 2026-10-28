import { describe, expect, it } from 'vitest'
import { ORDER_STATUSES, statusColor, statusLabel } from './order-labels'

describe('labels', () => {
	it('names every status in Russian', () => {
		expect(ORDER_STATUSES.map(statusLabel)).toEqual(['Новый', 'Ожидает оплаты', 'Оплачен', 'Подтверждён', 'Отменён'])
	})

	it('shows an unknown value as it is', () => {
		expect(statusLabel('done')).toBe('done')
		expect(statusColor('done')).toBe('default')
	})

	it('colours statuses', () => {
		expect(statusColor('paid')).toBe('success')
		expect(statusColor('canceled')).toBe('danger')
		expect(statusColor('pending')).toBe('warning')
	})
})
