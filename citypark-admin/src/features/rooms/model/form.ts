import type { PhotoValue } from '../../../shared/ui/image-field'
import { textLength } from '../../../shared/lib/rich-text'

export type RoomForm = {
	name: string
	hotelName: string
	numbers: string
	area: string
	areaDetailed: string
	capacity: string
	hotelNote: string
	summary: string
	price: string
	description: string // HTML
	photo: PhotoValue
	gallery: GalleryItem[]
}

export type GalleryItem = { url: string; alt: string }
export type GalleryItemErrors = { url?: string; alt?: string }

type TextField = 'name' | 'hotelName' | 'numbers' | 'area' | 'areaDetailed' | 'capacity' | 'hotelNote' | 'summary'
export type RoomErrors = Partial<Record<TextField | 'price' | 'description', string>> & {
	photo?: Partial<Record<keyof PhotoValue, string>>
	gallery?: string
	/** Errors of single gallery photos, by index. */
	galleryItems?: Record<number, GalleryItemErrors>
}

/** The fields of a room that the admin can edit, as the backend returns and accepts them. */
export type RoomData = {
	name: string
	hotelName: string
	numbers: string
	area: string
	areaDetailed: string
	capacity: string
	description: string
	pricePerNight: number
	photo: PhotoValue
	gallery: GalleryItem[]
	hotelNote: string
	summary: string
}

const MAX_TEXT = 100
const MAX_ALT = 200
const MAX_SUMMARY = 200
export const MAX_GALLERY = 12
const MAX_DESCRIPTION = 1000
const MAX_PRICE = 1_000_000
const MAX_TEXT_MESSAGE = `Не больше ${MAX_TEXT} символов`
// Digits with an optional fraction of at most two digits; spaces are removed and the comma replaced first.
const PRICE_PATTERN = /^\d+(\.\d{1,2})?$/

function length(text: string): number {
	return Array.from(text).length
}

/** "4 500" → "4500", "4500,5" → "4500.5" (any kind of space is dropped). */
function normalizePrice(price: string): string {
	return price.replace(/\s/g, '').replace(',', '.')
}

// The seed data has CSS keywords like "right center"; the backend accepts only "" or "X% Y%" on save.
const PERCENT_POSITION = /^\d{1,3}(\.\d+)?% \d{1,3}(\.\d+)?%$/

export function toForm(room: RoomData): RoomForm {
	return {
		name: room.name,
		hotelName: room.hotelName,
		numbers: room.numbers,
		area: room.area,
		areaDetailed: room.areaDetailed,
		capacity: room.capacity,
		hotelNote: room.hotelNote,
		summary: room.summary,
		price: String(room.pricePerNight),
		description: room.description,
		photo: {
			url: room.photo.url,
			alt: room.photo.alt,
			position: PERCENT_POSITION.test(room.photo.position) ? room.photo.position : '',
		},
		gallery: room.gallery.map(({ url, alt }) => ({ url, alt })),
	}
}

/** Trims the texts and parses the price. Call it only after validation. */
export function toRequest(f: RoomForm): RoomData {
	return {
		name: f.name.trim(),
		hotelName: f.hotelName.trim(),
		numbers: f.numbers.trim(),
		area: f.area.trim(),
		areaDetailed: f.areaDetailed.trim(),
		capacity: f.capacity.trim(),
		description: f.description,
		pricePerNight: Number(normalizePrice(f.price)),
		photo: { url: f.photo.url, alt: f.photo.alt.trim(), position: f.photo.position },
		gallery: f.gallery.map(({ url, alt }) => ({ url, alt: alt.trim() })),
		hotelNote: f.hotelNote.trim(),
		summary: f.summary.trim(),
	}
}

/** Mirrors the backend rules for one room; returns Russian messages by field (empty object when valid). */
export function validateRoom(f: RoomForm): RoomErrors {
	const errors: RoomErrors = {}
	const required: [TextField, string][] = [
		['name', 'Укажите название'],
		['numbers', 'Укажите номера комнат'],
		['area', 'Укажите площадь'],
	]
	for (const [field, message] of required) {
		if (f[field].trim() === '') errors[field] = message
	}
	const optional: TextField[] = ['hotelName', 'areaDetailed', 'capacity', 'hotelNote']
	for (const field of [...required.map(([name]) => name), ...optional]) {
		if (errors[field] === undefined && length(f[field].trim()) > MAX_TEXT) errors[field] = MAX_TEXT_MESSAGE
	}

	if (length(f.summary.trim()) > MAX_SUMMARY) errors.summary = `Не больше ${MAX_SUMMARY} символов`

	const price = normalizePrice(f.price)
	if (price === '') errors.price = 'Укажите цену'
	else if (!PRICE_PATTERN.test(price)) errors.price = 'Введите число не больше чем с двумя знаками после запятой'
	else if (Number(price) <= 0) errors.price = 'Цена должна быть больше 0'
	else if (Number(price) > MAX_PRICE) errors.price = 'Цена не больше 1 000 000 ₽'

	if (textLength(f.description) > MAX_DESCRIPTION) errors.description = `Не больше ${MAX_DESCRIPTION} символов`

	const photo: NonNullable<RoomErrors['photo']> = {}
	if (f.photo.url === '') photo.url = 'Загрузите фото'
	else if (f.photo.alt.trim() === '') photo.alt = 'Опишите, что на фото'
	else if (length(f.photo.alt.trim()) > MAX_ALT) photo.alt = `Не больше ${MAX_ALT} символов`
	if (Object.keys(photo).length > 0) errors.photo = photo

	if (f.gallery.length > MAX_GALLERY) errors.gallery = `Не больше ${MAX_GALLERY} фото`
	const items: Record<number, GalleryItemErrors> = {}
	f.gallery.forEach((item, index) => {
		if (item.url === '') items[index] = { url: 'Загрузите фото' }
		else if (length(item.alt.trim()) > MAX_ALT) items[index] = { alt: `Не больше ${MAX_ALT} символов` }
	})
	if (Object.keys(items).length > 0) errors.galleryItems = items
	return errors
}

const TEXT_FIELDS: readonly string[] = [
	'name',
	'hotelName',
	'numbers',
	'area',
	'areaDetailed',
	'capacity',
	'hotelNote',
	'summary',
	'description',
]
const PHOTO_FIELD = /^photo\.(url|alt|position)$/
const GALLERY_ITEM_FIELD = /^gallery\.(\d+)\.(url|alt)$/

/** Maps the backend's `errors` keys (`pricePerNight`, `photo.url`, ...) to form fields; unknown keys are dropped. */
export function mapServerFields(fields: Record<string, string>): RoomErrors {
	const errors: RoomErrors = {}
	for (const [key, message] of Object.entries(fields)) {
		const photo = PHOTO_FIELD.exec(key)
		if (photo) {
			errors.photo = { ...errors.photo, [photo[1] as keyof PhotoValue]: message }
		} else if (key === 'gallery') {
			errors.gallery = message
		} else if (GALLERY_ITEM_FIELD.test(key)) {
			const [, index, field] = GALLERY_ITEM_FIELD.exec(key) ?? []
			const at = Number(index)
			errors.galleryItems = {
				...errors.galleryItems,
				[at]: { ...errors.galleryItems?.[at], [field as 'url' | 'alt']: message },
			}
		} else if (key === 'pricePerNight') {
			errors.price = message
		} else if (TEXT_FIELDS.includes(key)) {
			errors[key as TextField | 'description'] = message
		}
	}
	return errors
}
