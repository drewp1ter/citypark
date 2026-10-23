import type { components } from '../../../shared/api/schema'

export type HotelPricesData = components['schemas']['HotelPrices']
type RequestData = components['schemas']['HotelPricesInput']

export type FeeName = 'breakfastDelivery' | 'lateCheckoutPerHour' | 'smokingFine'

/** The fees in the order the form shows them, with their labels. */
export const FEES: readonly { name: FeeName; label: string }[] = [
	{ name: 'breakfastDelivery', label: 'Доставка завтрака в номер, ₽' },
	{ name: 'lateCheckoutPerHour', label: 'Поздний выезд, ₽ в час' },
	{ name: 'smokingFine', label: 'Штраф за курение, ₽' },
]

export const MAX_BREAKFAST = 30
export const MAX_PRICE = 100_000
const MAX_NAME = 80
const MAX_SHORT_NAME = 40

// Prices are text while editing, so "250,5" and a half-typed number survive; `toRequest` parses them.
export type BreakfastItemForm = { name: string; shortName: string; price: string; desktopOnly: boolean }
export type HotelPricesForm = Record<FeeName, string> & { breakfast: BreakfastItemForm[] }

/** Error messages by the backend's keys: `smokingFine`, `breakfast`, `breakfast[2].price`. */
export type HotelPricesErrors = Record<string, string>

const PRICE_PATTERN = /^\d+(?:[.,]\d+)?$/

function length(text: string): number {
	return Array.from(text).length
}

function priceToText(value: number): string {
	return String(value).replace('.', ',')
}

export const EMPTY_ITEM: BreakfastItemForm = { name: '', shortName: '', price: '', desktopOnly: false }

export function toForm(data: HotelPricesData): HotelPricesForm {
	return {
		breakfastDelivery: priceToText(data.breakfastDelivery),
		lateCheckoutPerHour: priceToText(data.lateCheckoutPerHour),
		smokingFine: priceToText(data.smokingFine),
		breakfast: data.breakfast.map((item) => ({
			name: item.name,
			shortName: item.shortName,
			price: priceToText(item.price),
			desktopOnly: item.desktopOnly,
		})),
	}
}

/** "250,5" → 250.5; null when the text is not a plain non-negative number. */
export function parsePrice(text: string): number | null {
	const value = text.trim().replace(/\s/g, '')
	if (!PRICE_PATTERN.test(value)) return null
	return Number(value.replace(',', '.'))
}

/** Checks one price text; returns a Russian message or undefined. */
function priceError(text: string): string | undefined {
	const value = text.trim().replace(/\s/g, '')
	if (value === '') return 'Укажите цену'
	if (value.startsWith('-')) return 'Цена не может быть меньше 0'
	if (!PRICE_PATTERN.test(value)) return 'Введите число, например 250 или 250,50'
	if (/[.,]\d{3,}$/.test(value)) return 'Не более 2 знаков после запятой'
	if (Number(value.replace(',', '.')) > MAX_PRICE) return 'Не больше 100 000'
	return undefined
}

/** Parses the prices and trims the texts. Call it only after validation. */
export function toRequest(form: HotelPricesForm): RequestData {
	const price = (text: string) => parsePrice(text) ?? 0
	return {
		breakfastDelivery: price(form.breakfastDelivery),
		lateCheckoutPerHour: price(form.lateCheckoutPerHour),
		smokingFine: price(form.smokingFine),
		breakfast: form.breakfast.map((item) => ({
			name: item.name.trim(),
			shortName: item.shortName.trim(),
			price: price(item.price),
			desktopOnly: item.desktopOnly,
		})),
	}
}

/** Mirrors the backend rules; returns Russian messages by key (empty object when valid). */
export function validateHotelPrices(form: HotelPricesForm): HotelPricesErrors {
	const errors: HotelPricesErrors = {}
	for (const { name } of FEES) {
		const message = priceError(form[name])
		if (message !== undefined) errors[name] = message
	}
	if (form.breakfast.length > MAX_BREAKFAST) errors.breakfast = `Не больше ${MAX_BREAKFAST} блюд`
	form.breakfast.forEach((item, i) => {
		const name = item.name.trim()
		if (name === '') errors[`breakfast[${i}].name`] = 'Укажите название'
		else if (length(name) > MAX_NAME) errors[`breakfast[${i}].name`] = `Не больше ${MAX_NAME} символов`
		if (length(item.shortName.trim()) > MAX_SHORT_NAME) {
			errors[`breakfast[${i}].shortName`] = `Не больше ${MAX_SHORT_NAME} символов`
		}
		const message = priceError(item.price)
		if (message !== undefined) errors[`breakfast[${i}].price`] = message
	})
	return errors
}

const SERVER_KEY =
	/^(breakfastDelivery|lateCheckoutPerHour|smokingFine|breakfast|breakfast\[\d+\]\.(name|shortName|price))$/

/** Keeps the backend's `errors` entries that belong to a form field (the keys are already the form's keys). */
export function mapServerFields(fields: Record<string, string>): HotelPricesErrors {
	const errors: HotelPricesErrors = {}
	for (const [key, message] of Object.entries(fields)) {
		if (SERVER_KEY.test(key)) errors[key] = message
	}
	return errors
}

/** Moves the item at `index` by one place; returns the same list when it cannot move. */
export function moveItem<T>(list: T[], index: number, by: -1 | 1): T[] {
	const a = list[index]
	const b = list[index + by]
	if (a === undefined || b === undefined) return list
	return list.map((row, i) => (i === index ? b : i === index + by ? a : row))
}
