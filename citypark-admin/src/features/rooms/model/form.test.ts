import { describe, expect, it } from 'vitest'
import { mapServerFields, toForm, toRequest, validateRoom, type RoomData } from './form'

const room: RoomData = {
	name: 'Одноместный',
	hotelName: '',
	numbers: '№ 204, 304',
	area: '19 м²',
	areaDetailed: '',
	capacity: 'до 2 гостей',
	description: '<p>Уютный номер</p>',
	pricePerNight: 4000,
	photo: { url: '/images/room-single.jpg', alt: 'Одноместный номер', position: '50% 30%' },
	hotelNote: '№ 204, 304 · двуспальная',
}

describe('toRequest(toForm(room))', () => {
	it('gives back the editable fields', () => {
		expect(toRequest(toForm(room))).toEqual(room)
	})

	it('drops a focal point that the backend would reject', () => {
		const form = toForm({ ...room, photo: { ...room.photo, position: 'right center' } })
		expect(form.photo.position).toBe('')
	})

	it('trims the texts', () => {
		const form = { ...toForm(room), name: '  Новый  ', numbers: ' № 1 ' }
		expect(toRequest(form)).toMatchObject({ name: 'Новый', numbers: '№ 1' })
	})

	it.each([
		['4 500', 4500],
		['4 500', 4500],
		['4500,5', 4500.5],
		['4500.25', 4500.25],
	])('parses the price %s', (price, expected) => {
		expect(toRequest({ ...toForm(room), price }).pricePerNight).toBe(expected)
		expect(validateRoom({ ...toForm(room), price }).price).toBeUndefined()
	})
})

describe('validateRoom', () => {
	it('accepts a valid room', () => {
		expect(validateRoom(toForm(room))).toEqual({})
	})

	it('requires name, numbers and area', () => {
		const errors = validateRoom({ ...toForm(room), name: ' ', numbers: '', area: '' })
		expect(Object.keys(errors).sort()).toEqual(['area', 'name', 'numbers'])
	})

	it('limits plain texts to 100 characters', () => {
		expect(validateRoom({ ...toForm(room), hotelName: 'а'.repeat(101) }).hotelName).toBeDefined()
		expect(validateRoom({ ...toForm(room), hotelName: 'а'.repeat(100) }).hotelName).toBeUndefined()
	})

	it.each(['', '0', '-5', 'abc', '10.123', '1000001'])('rejects the price "%s"', (price) => {
		expect(validateRoom({ ...toForm(room), price }).price).toBeDefined()
	})

	it('accepts the price limit', () => {
		expect(validateRoom({ ...toForm(room), price: '1000000' }).price).toBeUndefined()
	})

	it('limits the description to 1000 visible characters', () => {
		const long = `<p><strong>${'а'.repeat(1001)}</strong></p>`
		expect(validateRoom({ ...toForm(room), description: long }).description).toBeDefined()
	})

	it('requires a photo and its alt text', () => {
		const noPhoto = validateRoom({ ...toForm(room), photo: { url: '', alt: '', position: '' } })
		expect(noPhoto.photo?.url).toBeDefined()
		const noAlt = validateRoom({ ...toForm(room), photo: { ...room.photo, alt: '  ' } })
		expect(noAlt.photo?.alt).toBeDefined()
	})
})

describe('mapServerFields', () => {
	it('maps the backend keys to form fields', () => {
		expect(
			mapServerFields({
				name: 'обязательное поле',
				pricePerNight: 'должна быть больше 0',
				description: 'не более 1000 символов',
				'photo.url': 'недопустимый адрес фото',
				'photo.position': 'формат «50% 30%»',
			})
		).toEqual({
			name: 'обязательное поле',
			price: 'должна быть больше 0',
			description: 'не более 1000 символов',
			photo: { url: 'недопустимый адрес фото', position: 'формат «50% 30%»' },
		})
	})

	it('drops unknown keys', () => {
		expect(mapServerFields({ other: 'x' })).toEqual({})
	})
})
