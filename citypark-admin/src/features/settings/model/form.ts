import type { components } from '../../../shared/api/schema'

export type SettingsData = components['schemas']['Settings']
type PhoneData = components['schemas']['SettingsPhone']
type SocialData = components['schemas']['SettingsSocial']

export type PhoneName = keyof SettingsData['phones']
export type PhoneField = keyof PhoneData

/** The phones in the order the form shows them, with their fixed titles. */
export const PHONES: readonly { name: PhoneName; title: string }[] = [
	{ name: 'restaurant', title: 'Ресторан и доставка' },
	{ name: 'hotel', title: 'Отель' },
	{ name: 'free', title: 'Бесплатно по России' },
	{ name: 'landline', title: 'Городской' },
]

export const MAX_SOCIAL = 10

// Every value is a string, so the form has the same shape as the request.
export type SettingsForm = SettingsData

/** Error messages by the backend's dotted keys: `email`, `phones.hotel.number`, `social[0].url`. */
export type SettingsErrors = Record<string, string>

const MAX_PHONE: Record<PhoneField, number> = {
	label: 60,
	shortLabel: 30,
	display: 40,
	number: 20,
	altDisplay: 40,
	altNumber: 20,
}
const NUMBER_PATTERN = /^\+?\d{10,15}$/
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const POSTCODE_PATTERN = /^\d{6}$/
const INN_PATTERN = /^\d{10,12}$/
const OGRN_PATTERN = /^\d{13,15}$/

function length(text: string): number {
	return Array.from(text).length
}

export function toForm(settings: SettingsData): SettingsForm {
	const phone = (p: PhoneData): PhoneData => ({
		label: p.label,
		shortLabel: p.shortLabel,
		display: p.display,
		number: p.number,
		altDisplay: p.altDisplay,
		altNumber: p.altNumber,
	})
	return {
		phones: {
			restaurant: phone(settings.phones.restaurant),
			hotel: phone(settings.phones.hotel),
			free: phone(settings.phones.free),
			landline: phone(settings.phones.landline),
		},
		address: { ...settings.address },
		email: settings.email,
		company: { ...settings.company },
		social: settings.social.map((s) => ({ name: s.name, label: s.label, url: s.url, display: s.display })),
	}
}

/** Trims every text. Call it only after validation. */
export function toRequest(f: SettingsForm): SettingsData {
	const phone = (p: PhoneData): PhoneData => ({
		label: p.label.trim(),
		shortLabel: p.shortLabel.trim(),
		display: p.display.trim(),
		number: p.number.trim(),
		altDisplay: p.altDisplay.trim(),
		altNumber: p.altNumber.trim(),
	})
	return {
		phones: {
			restaurant: phone(f.phones.restaurant),
			hotel: phone(f.phones.hotel),
			free: phone(f.phones.free),
			landline: phone(f.phones.landline),
		},
		address: {
			postcode: f.address.postcode.trim(),
			city: f.address.city.trim(),
			street: f.address.street.trim(),
		},
		email: f.email.trim(),
		company: { name: f.company.name.trim(), inn: f.company.inn.trim(), ogrn: f.company.ogrn.trim() },
		social: f.social.map((s): SocialData => ({
			name: s.name.trim(),
			label: s.label.trim(),
			url: s.url.trim(),
			display: s.display.trim(),
		})),
	}
}

/** Mirrors the backend rules for the settings; returns Russian messages by dotted key (empty object when valid). */
export function validateSettings(form: SettingsForm): SettingsErrors {
	const errors: SettingsErrors = {}

	// Checks one required text; the first problem wins.
	function text(
		key: string,
		value: string,
		max: number,
		requiredMessage: string,
		pattern?: RegExp,
		badMessage?: string
	) {
		const v = value.trim()
		if (v === '') errors[key] = requiredMessage
		else if (length(v) > max) errors[key] = `Не больше ${max} символов`
		else if (pattern !== undefined && !pattern.test(v)) errors[key] = badMessage ?? 'Неверный формат'
	}

	for (const { name } of PHONES) {
		const p = form.phones[name]
		const key = (field: PhoneField) => `phones.${name}.${field}`
		text(key('label'), p.label, MAX_PHONE.label, 'Укажите подпись')
		text(key('shortLabel'), p.shortLabel, MAX_PHONE.shortLabel, 'Укажите короткую подпись')
		text(key('display'), p.display, MAX_PHONE.display, 'Укажите, как показывать номер')
		text(
			key('number'),
			p.number,
			MAX_PHONE.number,
			'Укажите номер для звонка',
			NUMBER_PATTERN,
			'Номер: «+» и от 10 до 15 цифр'
		)

		const altDisplay = p.altDisplay.trim()
		const altNumber = p.altNumber.trim()
		if (altDisplay !== '' || altNumber !== '') {
			if (altDisplay === '') errors[key('altDisplay')] = 'Заполните оба поля'
			else if (length(altDisplay) > MAX_PHONE.altDisplay)
				errors[key('altDisplay')] = `Не больше ${MAX_PHONE.altDisplay} символов`
			if (altNumber === '') errors[key('altNumber')] = 'Заполните оба поля'
			else if (length(altNumber) > MAX_PHONE.altNumber)
				errors[key('altNumber')] = `Не больше ${MAX_PHONE.altNumber} символов`
			else if (!NUMBER_PATTERN.test(altNumber)) errors[key('altNumber')] = 'Номер: «+» и от 10 до 15 цифр'
		}
	}

	text('address.postcode', form.address.postcode, 6, 'Укажите индекс', POSTCODE_PATTERN, 'Индекс — ровно 6 цифр')
	text('address.city', form.address.city, 60, 'Укажите город')
	text('address.street', form.address.street, 100, 'Укажите улицу и дом')
	text('email', form.email, 100, 'Укажите e-mail', EMAIL_PATTERN, 'Введите адрес электронной почты')
	text('company.name', form.company.name, 100, 'Укажите название')
	text('company.inn', form.company.inn, 12, 'Укажите ИНН', INN_PATTERN, 'ИНН — от 10 до 12 цифр')
	text('company.ogrn', form.company.ogrn, 15, 'Укажите ОГРН', OGRN_PATTERN, 'ОГРН — от 13 до 15 цифр')

	if (form.social.length > MAX_SOCIAL) errors.social = `Не больше ${MAX_SOCIAL} соцсетей`
	form.social.forEach((s, i) => {
		text(`social[${i}].name`, s.name, 40, 'Укажите название')
		text(`social[${i}].label`, s.label, 10, 'Укажите короткую подпись')
		text(`social[${i}].url`, s.url, 200, 'Укажите ссылку')
		if (errors[`social[${i}].url`] === undefined && !s.url.trim().startsWith('https://')) {
			errors[`social[${i}].url`] = 'Ссылка должна начинаться с https://'
		}
		text(`social[${i}].display`, s.display, 60, 'Укажите, как показывать ссылку')
	})
	return errors
}

const SERVER_KEY =
	/^(phones\.(restaurant|hotel|free|landline)\.(label|shortLabel|display|number|altDisplay|altNumber)|address\.(postcode|city|street)|email|company\.(name|inn|ogrn)|social|social\[\d+\]\.(name|label|url|display))$/

/** Keeps the backend's `errors` entries that belong to a form field (the keys are already the form's keys). */
export function mapServerFields(fields: Record<string, string>): SettingsErrors {
	const errors: SettingsErrors = {}
	for (const [key, message] of Object.entries(fields)) {
		if (SERVER_KEY.test(key)) errors[key] = message
	}
	return errors
}

/** "8 918 311-97-91" → "+79183119791"; "8 800 100-24-41" → "88001002441"; "(86155) 2-25-38" → "+78615522538". */
export function displayToNumber(display: string): string {
	const digits = display.replace(/\D/g, '')
	if (digits.startsWith('8800')) return digits
	if (digits.length === 11 && digits.startsWith('8')) return `+7${digits.slice(1)}`
	// Already written with the country code: "+7 918 311 97 91".
	if (digits.length === 11 && digits.startsWith('7')) return `+${digits}`
	if (digits.length === 10) return `+7${digits}`
	return digits
}

/** "352630, г. Белореченск, ул. Гоголя, 61" — the line the site builds from the address. */
export function addressPreview(address: SettingsForm['address']): string {
	return `${address.postcode.trim()}, г. ${address.city.trim()}, ${address.street.trim()}`
}
