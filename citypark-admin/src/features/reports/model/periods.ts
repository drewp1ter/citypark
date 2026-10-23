export type Period = { from: string; to: string }

export type Preset = 'today' | 'yesterday' | 'week' | 'month' | 'prevMonth'

export const PRESETS: readonly { id: Preset; label: string }[] = [
	{ id: 'today', label: 'Сегодня' },
	{ id: 'yesterday', label: 'Вчера' },
	{ id: 'week', label: '7 дней' },
	{ id: 'month', label: 'Этот месяц' },
	{ id: 'prevMonth', label: 'Прошлый месяц' },
]

/** The backend accepts at most this many days per request. */
export const MAX_PERIOD_DAYS = 366

const dayPattern = /^(\d{4})-(\d{2})-(\d{2})$/

function toUtc(day: string): number {
	const [year = 0, month = 1, date = 1] = day.split('-').map(Number)
	return Date.UTC(year, month - 1, date)
}

/** "2026-10-22", -1 → "2026-10-21". Pure calendar arithmetic, no time zone involved. */
export function shiftDay(day: string, delta: number): string {
	return new Date(toUtc(day) + delta * 86_400_000).toISOString().slice(0, 10)
}

/** Number of days from `from` to `to`, both inclusive. */
export function periodDays(period: Period): number {
	return Math.round((toUtc(period.to) - toUtc(period.from)) / 86_400_000) + 1
}

/** The inclusive day range of a preset, given today's day (`YYYY-MM-DD`). */
export function presetRange(preset: Preset, today: string): Period {
	switch (preset) {
		case 'today':
			return { from: today, to: today }
		case 'yesterday': {
			const yesterday = shiftDay(today, -1)
			return { from: yesterday, to: yesterday }
		}
		case 'week':
			return { from: shiftDay(today, -6), to: today }
		case 'month':
			return { from: `${today.slice(0, 8)}01`, to: today }
		case 'prevMonth': {
			const to = shiftDay(`${today.slice(0, 8)}01`, -1)
			return { from: `${to.slice(0, 8)}01`, to }
		}
	}
}

/** The preset that gives exactly `period`, if any, to highlight its chip. */
export function activePreset(period: Period, today: string): Preset | undefined {
	return PRESETS.find((p) => {
		const range = presetRange(p.id, today)
		return range.from === period.from && range.to === period.to
	})?.id
}

export type ReportSearch = { from?: string; to?: string }

function parseDay(value: unknown): string | undefined {
	if (typeof value !== 'string') return undefined
	const match = dayPattern.exec(value)
	if (match === null) return undefined
	const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])]
	const date = new Date(Date.UTC(year, month - 1, day))
	const real = date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day
	return real ? value : undefined
}

/** Reads `from` and `to` from the URL; the pair is kept only when both are real days in a valid order and length. */
export function parseReportSearch(raw: Record<string, unknown>): ReportSearch {
	const from = parseDay(raw.from)
	const to = parseDay(raw.to)
	if (from === undefined || to === undefined) return {}
	if (to < from || periodDays({ from, to }) > MAX_PERIOD_DAYS) return {}
	return { from, to }
}

/** The period from the URL, or the current month up to today when the URL has none. */
export function resolvePeriod(search: ReportSearch, today: string): Period {
	if (search.from !== undefined && search.to !== undefined) return { from: search.from, to: search.to }
	return presetRange('month', today)
}
