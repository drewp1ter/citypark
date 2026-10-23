import { describe, expect, it } from 'vitest'
import { moscowToday, quickRanges, rangeFor, shiftDay } from './quick-ranges'

describe('quick ranges', () => {
	const today = '2026-10-22'
	const { today: now, yesterday, week, month } = quickRanges

	it('today and yesterday are one day', () => {
		expect(rangeFor(now, today)).toEqual({ from: '2026-10-22', to: '2026-10-22' })
		expect(rangeFor(yesterday, today)).toEqual({ from: '2026-10-21', to: '2026-10-21' })
	})

	it('7 and 30 days include today', () => {
		expect(rangeFor(week, today)).toEqual({ from: '2026-10-16', to: '2026-10-22' })
		expect(rangeFor(month, today)).toEqual({ from: '2026-09-23', to: '2026-10-22' })
	})

	it('shifts across month and year borders', () => {
		expect(shiftDay('2026-01-01', -1)).toBe('2025-12-31')
		expect(shiftDay('2026-03-01', -1)).toBe('2026-02-28')
	})

	it('takes the Moscow day, not the UTC one', () => {
		expect(moscowToday(new Date('2026-10-21T22:30:00Z'))).toBe('2026-10-22')
	})
})
