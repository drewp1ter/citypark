import { describe, expect, it } from 'vitest'
import { activePreset, parseReportSearch, periodDays, presetRange, resolvePeriod, shiftDay } from './periods'

const today = '2026-10-22'

describe('presetRange', () => {
	it('today and yesterday are one day', () => {
		expect(presetRange('today', today)).toEqual({ from: '2026-10-22', to: '2026-10-22' })
		expect(presetRange('yesterday', today)).toEqual({ from: '2026-10-21', to: '2026-10-21' })
	})

	it('week is the last 7 days including today', () => {
		expect(presetRange('week', today)).toEqual({ from: '2026-10-16', to: '2026-10-22' })
	})

	it('month runs from the 1st to today', () => {
		expect(presetRange('month', today)).toEqual({ from: '2026-10-01', to: '2026-10-22' })
	})

	it('prevMonth is the whole previous month', () => {
		expect(presetRange('prevMonth', today)).toEqual({ from: '2026-09-01', to: '2026-09-30' })
		expect(presetRange('prevMonth', '2026-03-05')).toEqual({ from: '2026-02-01', to: '2026-02-28' })
		expect(presetRange('prevMonth', '2026-01-10')).toEqual({ from: '2025-12-01', to: '2025-12-31' })
	})
})

describe('days', () => {
	it('shifts across month and year borders', () => {
		expect(shiftDay('2026-01-01', -1)).toBe('2025-12-31')
		expect(shiftDay('2026-02-28', 1)).toBe('2026-03-01')
	})

	it('counts both ends', () => {
		expect(periodDays({ from: '2026-10-22', to: '2026-10-22' })).toBe(1)
		expect(periodDays({ from: '2026-10-16', to: '2026-10-22' })).toBe(7)
	})
})

describe('activePreset', () => {
	it('finds the matching preset', () => {
		expect(activePreset({ from: '2026-10-16', to: '2026-10-22' }, today)).toBe('week')
	})

	it('is undefined for a custom range', () => {
		expect(activePreset({ from: '2026-10-05', to: '2026-10-07' }, today)).toBeUndefined()
	})
})

describe('parseReportSearch', () => {
	it('keeps a valid pair', () => {
		expect(parseReportSearch({ from: '2026-10-01', to: '2026-10-22' })).toEqual({
			from: '2026-10-01',
			to: '2026-10-22',
		})
	})

	it('drops a pair with a missing, invalid or reversed day', () => {
		expect(parseReportSearch({ from: '2026-10-01' })).toEqual({})
		expect(parseReportSearch({ from: '2026-02-30', to: '2026-03-01' })).toEqual({})
		expect(parseReportSearch({ from: '2026-10-22', to: '2026-10-01' })).toEqual({})
		expect(parseReportSearch({ from: 5, to: 'x' })).toEqual({})
	})

	it('drops a period longer than 366 days', () => {
		expect(parseReportSearch({ from: '2025-10-21', to: '2026-10-22' })).toEqual({})
		expect(parseReportSearch({ from: '2025-10-22', to: '2026-10-22' })).toEqual({
			from: '2025-10-22',
			to: '2026-10-22',
		})
	})
})

describe('resolvePeriod', () => {
	it('defaults to the current month', () => {
		expect(resolvePeriod({}, today)).toEqual({ from: '2026-10-01', to: '2026-10-22' })
	})

	it('uses the URL period when present', () => {
		expect(resolvePeriod({ from: '2026-09-01', to: '2026-09-05' }, today)).toEqual({
			from: '2026-09-01',
			to: '2026-09-05',
		})
	})
})
