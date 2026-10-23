const rubFormat = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
const rubFormatCents = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const dateTimeFormat = new Intl.DateTimeFormat('ru-RU', {
	timeZone: 'Europe/Moscow',
	day: '2-digit',
	month: '2-digit',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit',
	hour12: false,
})

/** 4000 → "4 000 ₽", 5.5 → "5,50 ₽" (non-breaking spaces, ru-RU). */
export function formatRub(value: number): string {
	const number = Number.isInteger(value) ? rubFormat.format(value) : rubFormatCents.format(value)
	return `${number}\u00a0₽`
}

/** "2026-10-22T09:05:00Z" → "22.10.2026, 12:05" in Europe/Moscow. */
export function formatDateTime(iso: string): string {
	const parts = Object.fromEntries(dateTimeFormat.formatToParts(new Date(iso)).map((p) => [p.type, p.value]))
	return `${parts.day}.${parts.month}.${parts.year}, ${parts.hour}:${parts.minute}`
}

/** "2026-10-22" → "22.10.2026". */
export function formatDate(isoDate: string): string {
	const [year, month, day] = isoDate.split('-')
	return `${day}.${month}.${year}`
}
