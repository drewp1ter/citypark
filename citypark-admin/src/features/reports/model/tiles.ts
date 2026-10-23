import { shiftDay, type Period } from './periods'

export type Day = { date: string; sum: number }

export type Tiles = { today: number | null; yesterday: number | null; week: number | null; period: number }

function sumRange(days: readonly Day[], from: string, to: string): number {
	return days.filter((d) => d.date >= from && d.date <= to).reduce((acc, d) => acc + d.sum, 0)
}

/**
 * Totals for the dashboard tiles. A tile is null («—») when the period does not cover its days:
 * today, yesterday, and the last 7 days including today.
 */
export function dashboardTiles(days: readonly Day[], period: Period, today: string): Tiles {
	const covers = (from: string, to: string) => period.from <= from && period.to >= to
	const yesterday = shiftDay(today, -1)
	const weekFrom = shiftDay(today, -6)
	return {
		today: covers(today, today) ? sumRange(days, today, today) : null,
		yesterday: covers(yesterday, yesterday) ? sumRange(days, yesterday, yesterday) : null,
		week: covers(weekFrom, today) ? sumRange(days, weekFrom, today) : null,
		period: sumRange(days, period.from, period.to),
	}
}

/** "2026-10-22" → "22.10" for the chart axis. */
export function shortDate(day: string): string {
	const [, month, date] = day.split('-')
	return `${date}.${month}`
}
