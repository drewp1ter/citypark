export const NBSP = '\u00A0'

/** Replaces regular spaces with non-breaking ones (phones, short labels). */
export function nb(text: string): string {
	return text.replaceAll(' ', NBSP)
}

/** 4000 -> "4 000 ₽" with non-breaking spaces. */
export function formatPrice(rubles: number): string {
	const grouped = String(Math.round(rubles)).replace(/\B(?=(\d{3})+(?!\d))/g, NBSP)
	return `${grouped}${NBSP}₽`
}

/**
 * Typography for copy taken from the design: keeps "4 000 ₽", "23 м²", "6 ч", "№ 301" together on one line.
 * Only touches a space that follows a digit (or "№"), so normal sentences still wrap.
 */
export function typo(text: string): string {
	return text
		.replace(/(\d) (?=\d{3}(?!\d))/g, `$1${NBSP}`)
		.replace(/(\d) (?=(?:₽|м²|кг|мин|г|ч)(?![\p{L}\d]))/gu, `$1${NBSP}`)
		.replace(/№ (?=\d)/g, `№${NBSP}`)
}

/** "2026-09-07" -> "07.09.2026" (the date format used in the design). */
export function formatDate(iso: string): string {
	const [year, month, day] = iso.split('-')
	return `${day}.${month}.${year}`
}
