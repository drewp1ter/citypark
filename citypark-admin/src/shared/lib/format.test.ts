import { describe, expect, it } from 'vitest'
import { formatDate, formatDateTime, formatRub } from './format'

describe('formatRub', () => {
	it('groups thousands with non-breaking spaces', () => {
		expect(formatRub(4000)).toBe('4 000 ₽')
	})
	it('formats zero and small integers', () => {
		expect(formatRub(0)).toBe('0 ₽')
		expect(formatRub(150)).toBe('150 ₽')
	})
	it('shows kopecks for fractions', () => {
		expect(formatRub(5.5)).toBe('5,50 ₽')
	})
})

describe('formatDateTime', () => {
	it('converts UTC to Moscow time', () => {
		expect(formatDateTime('2026-10-22T09:05:00Z')).toBe('22.10.2026, 12:05')
	})
	it('rolls over the date', () => {
		expect(formatDateTime('2026-12-31T22:30:00Z')).toBe('01.01.2027, 01:30')
	})
})

describe('formatDate', () => {
	it('formats an ISO date', () => {
		expect(formatDate('2026-10-22')).toBe('22.10.2026')
	})
})
