export type QuickRange = { label: string; days: number }

/** «Сегодня», «Вчера», «7 дней», «30 дней»: the range ends on `end`, or on yesterday for «Вчера». */
export const quickRanges = {
	today: { label: 'Сегодня', days: 1 },
	yesterday: { label: 'Вчера', days: -1 },
	week: { label: '7 дней', days: 7 },
	month: { label: '30 дней', days: 30 },
} satisfies Record<string, QuickRange>

const moscowDay = new Intl.DateTimeFormat('en-CA', {
	timeZone: 'Europe/Moscow',
	year: 'numeric',
	month: '2-digit',
	day: '2-digit',
})

/** The current local day in Moscow as "YYYY-MM-DD". */
export function moscowToday(now: Date = new Date()): string {
	return moscowDay.format(now)
}

/** "2026-10-22", -1 → "2026-10-21". Pure calendar arithmetic, no time zone involved. */
export function shiftDay(day: string, delta: number): string {
	const [year = 0, month = 1, date = 1] = day.split('-').map(Number)
	return new Date(Date.UTC(year, month - 1, date + delta)).toISOString().slice(0, 10)
}

/** The inclusive day range of a quick button, given today's day. */
export function rangeFor(range: QuickRange, today: string): { from: string; to: string } {
	if (range.days === -1) {
		const yesterday = shiftDay(today, -1)
		return { from: yesterday, to: yesterday }
	}
	return { from: shiftDay(today, 1 - range.days), to: today }
}
