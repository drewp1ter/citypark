import { describe, expect, it } from 'vitest'
import { dashboardTiles, shortDate } from './tiles'

const days = [
	{ date: '2026-10-15', sum: 1000 },
	{ date: '2026-10-16', sum: 100 },
	{ date: '2026-10-20', sum: 8000 },
	{ date: '2026-10-21', sum: 0 },
	{ date: '2026-10-22', sum: 500 },
]
const today = '2026-10-22'

describe('dashboardTiles', () => {
	it('fills every tile when the period covers the last week', () => {
		expect(dashboardTiles(days, { from: '2026-10-15', to: today }, today)).toEqual({
			today: 500,
			yesterday: 0,
			week: 8600,
			period: 9600,
		})
	})

	it('leaves the tiles the period does not cover empty', () => {
		expect(dashboardTiles(days, { from: '2026-10-20', to: '2026-10-21' }, today)).toEqual({
			today: null,
			yesterday: 0,
			week: null,
			period: 8000,
		})
	})

	it('needs the whole week for the 7-day tile', () => {
		expect(dashboardTiles(days, { from: '2026-10-17', to: today }, today).week).toBeNull()
		expect(dashboardTiles(days, { from: '2026-10-16', to: today }, today).week).toBe(8600)
	})
})

describe('shortDate', () => {
	it('formats dd.MM', () => {
		expect(shortDate('2026-10-22')).toBe('22.10')
	})
})
