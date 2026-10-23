export type DeliveryPriceForm = { name: string; value: string; price: string }
type FormErrors = Partial<Record<keyof DeliveryPriceForm, string>>

const MAX_TEXT = 255
const MAX_TEXT_MESSAGE = `Не больше ${MAX_TEXT} символов`
// Digits with an optional fraction of at most two digits; the comma is already replaced by a dot.
const PRICE_PATTERN = /^\d+(\.\d{1,2})?$/

function length(text: string): number {
	return Array.from(text).length
}

function normalizePrice(price: string): string {
	return price.trim().replace(',', '.')
}

/** Mirrors the backend rules for one delivery price; returns Russian messages by field (empty object when valid). */
export function validateDeliveryPrice(f: DeliveryPriceForm): FormErrors {
	const errors: FormErrors = {}
	const name = f.name.trim()
	if (name === '') errors.name = 'Укажите название города'
	else if (length(name) > MAX_TEXT) errors.name = MAX_TEXT_MESSAGE
	if (length(f.value.trim()) > MAX_TEXT) errors.value = MAX_TEXT_MESSAGE

	const price = normalizePrice(f.price)
	if (price === '') errors.price = 'Укажите цену'
	else if (!PRICE_PATTERN.test(price)) errors.price = 'Введите число не больше чем с двумя знаками после запятой'
	else if (Number(price) <= 0) errors.price = 'Цена должна быть больше 0'
	return errors
}

/** Trims the texts and parses the price: "150,5" → 150.5. Call it only after validation. */
export function toRequest(f: DeliveryPriceForm): { name: string; value: string; price: number } {
	return { name: f.name.trim(), value: f.value.trim(), price: Number(normalizePrice(f.price)) }
}

const SERVER_FIELD = /^(?:deliveryPrices\[\d+\]\.)?(name|value|price)$/

/** Maps the backend's `errors` keys (`deliveryPrices[0].name` or `name`) to form fields; unknown keys are dropped. */
export function mapServerFields(fields: Record<string, string>): FormErrors {
	const errors: FormErrors = {}
	for (const [key, message] of Object.entries(fields)) {
		const match = SERVER_FIELD.exec(key)
		if (match) errors[match[1] as keyof DeliveryPriceForm] = message
	}
	return errors
}
