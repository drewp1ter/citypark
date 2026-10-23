/** Price placeholders the backend accepts (`../citypark-go/docs/API.md`, section Content). */
export const PRICE_PLACEHOLDERS = [
	{ name: 'minRoomPrice', label: 'Мин. цена номера' },
	{ name: 'lateCheckoutPerHour', label: 'Поздний выезд, за час' },
	{ name: 'smokingFine', label: 'Штраф за курение' },
	{ name: 'breakfastDelivery', label: 'Доставка завтрака' },
] as const

/** Puts `{{name}}` into `value` over the selection `start..end`; returns the new text and the caret position after it. */
export function insertPlaceholder(
	value: string,
	name: string,
	start: number,
	end: number
): { value: string; caret: number } {
	const token = `{{${name}}}`
	const from = Math.min(start, end)
	const to = Math.max(start, end)
	return { value: value.slice(0, from) + token + value.slice(to), caret: from + token.length }
}
