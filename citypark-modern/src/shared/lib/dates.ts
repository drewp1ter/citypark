const pad = (n: number) => String(n).padStart(2, '0')

/** A Date as "2026-10-16" in the local time of the machine that runs this code. */
export function toIsoDate(date: Date): string {
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** "2026-10-16" + 1 -> "2026-10-17". Returns "" for an empty or invalid input. */
export function addDays(iso: string, days: number): string {
	const [year, month, day] = iso.split('-').map(Number)
	if (!year || !month || !day) return ''
	return toIsoDate(new Date(year, month - 1, day + days))
}

/** Whole days from `from` to `to` ("2026-10-16", "2026-10-18" -> 2). NaN for invalid input. */
export function diffDays(from: string, to: string): number {
	const parse = (iso: string) => {
		const [year, month, day] = iso.split('-').map(Number)
		return Date.UTC(year, month - 1, day)
	}
	return Math.round((parse(to) - parse(from)) / 86_400_000)
}

/** Today in Белореченск (Moscow time), "2026-10-16". Servers may run in another time zone. */
export function todayInMoscow(): string {
	return new Date().toLocaleDateString('en-CA', { timeZone: 'Europe/Moscow' })
}
