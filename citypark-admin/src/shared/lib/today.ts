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
