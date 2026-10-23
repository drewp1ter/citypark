import { describe, expect, it } from 'vitest'
import {
	addressPreview,
	displayToNumber,
	mapServerFields,
	toForm,
	toRequest,
	validateSettings,
	type SettingsData,
} from './form'

const phone = (display: string, number: string) => ({
	label: 'Подпись',
	shortLabel: 'Кратко',
	display,
	number,
	altDisplay: '',
	altNumber: '',
})

const settings: SettingsData = {
	phones: {
		restaurant: phone('8 918 311-97-91', '+79183119791'),
		hotel: phone('8 918 311-97-10', '+79183119710'),
		free: phone('8 800 100-24-41', '88001002441'),
		landline: phone('(86155) 2-25-38', '+78615522538'),
	},
	address: { postcode: '352630', city: 'Белореченск', street: 'ул. Гоголя, 61' },
	email: 'belcitypark@mail.ru',
	company: { name: 'ООО «Олимп»', inn: '2309117400', ogrn: '1092309002378' },
	social: [{ name: 'ВКонтакте', label: 'VK', url: 'https://vk.com/cityparkvip', display: 'vk.com/cityparkvip' }],
}

describe('displayToNumber', () => {
	it.each([
		['8 918 311-97-91', '+79183119791'],
		['8 800 100-24-41', '88001002441'],
		['(86155) 2-25-38', '+78615522538'],
		['+7 918 311 97 91', '+79183119791'],
		['918 311-97-91', '+79183119791'],
		['311-97', '31197'],
		['', ''],
	])('%s → %s', (display, number) => {
		expect(displayToNumber(display)).toBe(number)
	})
})

describe('toForm / toRequest', () => {
	it('round-trips valid settings', () => {
		expect(toRequest(toForm(settings))).toEqual(settings)
	})

	it('drops updatedAt and trims texts', () => {
		const form = toForm({ ...settings, updatedAt: 'x' } as SettingsData)
		expect('updatedAt' in form).toBe(false)
		form.email = '  a@b.ru '
		form.social[0]!.url = ' https://vk.com/x '
		const request = toRequest(form)
		expect(request.email).toBe('a@b.ru')
		expect(request.social[0]?.url).toBe('https://vk.com/x')
	})
})

describe('validateSettings', () => {
	it('accepts valid settings', () => {
		expect(validateSettings(toForm(settings))).toEqual({})
	})

	it('rejects a bad e-mail', () => {
		const form = toForm(settings)
		form.email = 'not-an-email'
		expect(validateSettings(form)).toEqual({ email: 'Введите адрес электронной почты' })
	})

	it('rejects a 5-digit postcode', () => {
		const form = toForm(settings)
		form.address.postcode = '35263'
		expect(validateSettings(form)['address.postcode']).toBe('Индекс — ровно 6 цифр')
	})

	it('rejects an http:// social link', () => {
		const form = toForm(settings)
		form.social[0]!.url = 'http://vk.com/x'
		expect(validateSettings(form)['social[0].url']).toBe('Ссылка должна начинаться с https://')
	})

	it('rejects a half-filled second number on the empty field', () => {
		const form = toForm(settings)
		form.phones.hotel.altDisplay = '8 (86155) 3-30-02'
		expect(validateSettings(form)).toEqual({ 'phones.hotel.altNumber': 'Заполните оба поля' })
		form.phones.hotel.altDisplay = ''
		form.phones.hotel.altNumber = '+78615533002'
		expect(validateSettings(form)).toEqual({ 'phones.hotel.altDisplay': 'Заполните оба поля' })
	})

	it('rejects a bad phone number, INN, OGRN and too many socials', () => {
		const form = toForm(settings)
		form.phones.free.number = '8-800'
		form.company.inn = '123'
		form.company.ogrn = '12345678901234567'
		form.social = Array.from({ length: 11 }, () => ({ name: 'a', label: 'b', url: 'https://c', display: 'd' }))
		const errors = validateSettings(form)
		expect(errors['phones.free.number']).toBeDefined()
		expect(errors['company.inn']).toBeDefined()
		expect(errors['company.ogrn']).toBeDefined()
		expect(errors.social).toBeDefined()
	})

	it('requires the texts and counts characters, not bytes', () => {
		const form = toForm(settings)
		form.address.city = '  '
		form.social[0]!.label = '😀'.repeat(10)
		const errors = validateSettings(form)
		expect(errors['address.city']).toBe('Укажите город')
		expect(errors['social[0].label']).toBeUndefined()
		form.social[0]!.label = '😀'.repeat(11)
		expect(validateSettings(form)['social[0].label']).toBe('Не больше 10 символов')
	})
})

describe('mapServerFields', () => {
	it('keeps keys that belong to a field and drops the rest', () => {
		expect(
			mapServerFields({
				email: 'плохой',
				'phones.hotel.number': 'номер: только цифры и «+»',
				'social[0].url': 'ссылка должна начинаться с https://',
				updatedAt: 'x',
				'phones.other.number': 'x',
			})
		).toEqual({
			email: 'плохой',
			'phones.hotel.number': 'номер: только цифры и «+»',
			'social[0].url': 'ссылка должна начинаться с https://',
		})
	})
})

describe('addressPreview', () => {
	it('builds the line the site shows', () => {
		expect(addressPreview(settings.address)).toBe('352630, г. Белореченск, ул. Гоголя, 61')
	})
})
