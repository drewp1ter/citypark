import { describe, expect, it } from 'vitest'
import {
	mapServerFields,
	moveItem,
	parsePrice,
	toForm,
	toRequest,
	validateHotelPrices,
	type HotelPricesData,
	type HotelPricesForm,
} from './form'

const data: HotelPricesData = {
	breakfastDelivery: 100,
	lateCheckoutPerHour: 500,
	smokingFine: 2000.5,
	breakfast: [
		{ name: 'Омлет', shortName: 'Ом', price: 250, desktopOnly: false },
		{ name: 'Сырники', shortName: '', price: 300, desktopOnly: true },
	],
	updatedAt: '2026-10-22T21:30:03Z',
}

const valid = (): HotelPricesForm => toForm(data)

describe('parsePrice', () => {
	it('accepts a comma or a dot', () => {
		expect(parsePrice('250,5')).toBe(250.5)
		expect(parsePrice(' 250.50 ')).toBe(250.5)
		expect(parsePrice('0')).toBe(0)
	})
	it('rejects text, negatives and empty', () => {
		expect(parsePrice('')).toBeNull()
		expect(parsePrice('abc')).toBeNull()
		expect(parsePrice('-5')).toBeNull()
		expect(parsePrice('1,2,3')).toBeNull()
	})
})

describe('toForm / toRequest', () => {
	it('round-trips the data', () => {
		const { breakfastDelivery, lateCheckoutPerHour, smokingFine, breakfast } = data
		expect(toRequest(toForm(data))).toEqual({ breakfastDelivery, lateCheckoutPerHour, smokingFine, breakfast })
	})
	it('keeps item order, trims texts and parses 250,5', () => {
		const form = valid()
		form.breakfast = [
			{ name: ' Б ', shortName: ' ', price: '250,5', desktopOnly: true },
			{ name: 'А', shortName: 'а', price: '1', desktopOnly: false },
		]
		expect(toRequest(form).breakfast).toEqual([
			{ name: 'Б', shortName: '', price: 250.5, desktopOnly: true },
			{ name: 'А', shortName: 'а', price: 1, desktopOnly: false },
		])
	})
})

describe('validateHotelPrices', () => {
	it('accepts valid data and zero prices', () => {
		expect(validateHotelPrices(valid())).toEqual({})
		const form = { ...valid(), smokingFine: '0' }
		expect(validateHotelPrices(form)).toEqual({})
	})
	it('checks the fees', () => {
		const form = { ...valid(), breakfastDelivery: '', lateCheckoutPerHour: '100001', smokingFine: '1,234' }
		expect(Object.keys(validateHotelPrices(form)).sort()).toEqual([
			'breakfastDelivery',
			'lateCheckoutPerHour',
			'smokingFine',
		])
		expect(validateHotelPrices({ ...valid(), smokingFine: '-1' }).smokingFine).toBe('Цена не может быть меньше 0')
		expect(validateHotelPrices({ ...valid(), smokingFine: 'x' }).smokingFine).toMatch(/число/)
		expect(validateHotelPrices({ ...valid(), smokingFine: '100000' })).toEqual({})
	})
	it('checks the items by index', () => {
		const form = valid()
		form.breakfast = [
			...form.breakfast,
			{ name: ' ', shortName: 'я'.repeat(41), price: '', desktopOnly: false },
			{ name: 'я'.repeat(81), shortName: '', price: '5', desktopOnly: false },
		]
		const errors = validateHotelPrices(form)
		expect(errors['breakfast[2].name']).toBe('Укажите название')
		expect(errors['breakfast[2].shortName']).toBe('Не больше 40 символов')
		expect(errors['breakfast[2].price']).toBe('Укажите цену')
		expect(errors['breakfast[3].name']).toBe('Не больше 80 символов')
		expect(errors['breakfast[0].name']).toBeUndefined()
	})
	it('allows 30 items and rejects 31', () => {
		const item = { name: 'Блюдо', shortName: '', price: '1', desktopOnly: false }
		expect(validateHotelPrices({ ...valid(), breakfast: Array(30).fill(item) })).toEqual({})
		expect(validateHotelPrices({ ...valid(), breakfast: Array(31).fill(item) }).breakfast).toBe('Не больше 30 блюд')
	})
})

describe('mapServerFields', () => {
	it('keeps only form keys', () => {
		expect(
			mapServerFields({ 'breakfast[2].price': 'a', smokingFine: 'b', updatedAt: 'c', 'breakfast[x].name': 'd' })
		).toEqual({
			'breakfast[2].price': 'a',
			smokingFine: 'b',
		})
	})
})

describe('moveItem', () => {
	it('swaps neighbours and ignores the edges', () => {
		expect(moveItem([1, 2, 3], 2, -1)).toEqual([1, 3, 2])
		expect(moveItem([1, 2, 3], 0, -1)).toEqual([1, 2, 3])
		expect(moveItem([1, 2, 3], 2, 1)).toEqual([1, 2, 3])
	})
})
