import type { PhotoValue } from '../../../shared/ui/image-field'
import { textLength } from '../../../shared/lib/rich-text'

export type NewsForm = {
	title: string
	shortTitle: string
	excerpt: string
	date: string // "YYYY-MM-DD"
	slug: string
	photo: PhotoValue
	body: string // HTML
}

type TextField = 'title' | 'shortTitle' | 'excerpt' | 'date' | 'slug' | 'body'
export type NewsErrors = Partial<Record<TextField, string>> & {
	photo?: Partial<Record<keyof PhotoValue, string>>
}

/** The fields of a news item as the backend returns and accepts them. */
export type NewsData = {
	title: string
	shortTitle: string
	excerpt: string
	date: string
	slug: string
	photo: PhotoValue
	body: string
}

export const MAX_TITLE = 200
export const MAX_SHORT_TITLE = 100
export const MAX_EXCERPT = 300
export const MAX_BODY = 50_000
const MAX_ALT = 200
const SLUG_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/
const PERCENT_POSITION = /^\d{1,3}(\.\d+)?% \d{1,3}(\.\d+)?%$/

function length(text: string): number {
	return Array.from(text).length
}

/** A new item: today's date, everything else empty. `today` is "YYYY-MM-DD". */
export function emptyNews(today: string): NewsForm {
	return {
		title: '',
		shortTitle: '',
		excerpt: '',
		date: today,
		slug: '',
		photo: { url: '', alt: '', position: '' },
		body: '',
	}
}

export function toForm(item: NewsData): NewsForm {
	return {
		title: item.title,
		shortTitle: item.shortTitle,
		excerpt: item.excerpt,
		date: item.date,
		slug: item.slug,
		photo: {
			url: item.photo.url,
			alt: item.photo.alt,
			position: PERCENT_POSITION.test(item.photo.position) ? item.photo.position : '',
		},
		body: item.body,
	}
}

/** Trims the texts. An empty slug stays `''` (the backend makes it from the title). Call it after validation. */
export function toRequest(f: NewsForm): NewsData {
	return {
		title: f.title.trim(),
		shortTitle: f.shortTitle.trim(),
		excerpt: f.excerpt.trim(),
		date: f.date,
		slug: f.slug.trim(),
		photo: { url: f.photo.url, alt: f.photo.alt.trim(), position: f.photo.position },
		body: f.body,
	}
}

/** Mirrors the backend rules for one news item; returns Russian messages by field (empty object when valid). */
export function validateNews(f: NewsForm): NewsErrors {
	const errors: NewsErrors = {}
	const title = f.title.trim()
	if (title === '') errors.title = 'Укажите заголовок'
	else if (length(title) > MAX_TITLE) errors.title = `Не больше ${MAX_TITLE} символов`
	if (length(f.shortTitle.trim()) > MAX_SHORT_TITLE) errors.shortTitle = `Не больше ${MAX_SHORT_TITLE} символов`
	if (length(f.excerpt.trim()) > MAX_EXCERPT) errors.excerpt = `Не больше ${MAX_EXCERPT} символов`

	const slug = f.slug.trim()
	if (slug !== '') {
		if (!SLUG_PATTERN.test(slug)) errors.slug = 'Только латиница, цифры и дефисы'
		else if (slug.length > 100) errors.slug = 'Не больше 100 символов'
	}

	if (f.date === '') errors.date = 'Укажите дату'

	if (textLength(f.body) > MAX_BODY) errors.body = `Не больше ${MAX_BODY} символов`

	if (f.photo.url !== '' && length(f.photo.alt.trim()) > MAX_ALT) {
		errors.photo = { alt: `Не больше ${MAX_ALT} символов` }
	}
	return errors
}

const TEXT_FIELDS: readonly string[] = ['title', 'shortTitle', 'excerpt', 'date', 'slug', 'body']
const PHOTO_FIELD = /^photo\.(url|alt|position)$/

/** Maps the backend's `errors` keys (`title`, `photo.url`, ...) to form fields; unknown keys are dropped. */
export function mapServerFields(fields: Record<string, string>): NewsErrors {
	const errors: NewsErrors = {}
	for (const [key, message] of Object.entries(fields)) {
		const photo = PHOTO_FIELD.exec(key)
		if (photo) {
			errors.photo = { ...errors.photo, [photo[1] as keyof PhotoValue]: message }
		} else if (TEXT_FIELDS.includes(key)) {
			errors[key as TextField] = message
		}
	}
	return errors
}
