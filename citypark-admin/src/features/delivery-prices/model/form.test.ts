import { describe, expect, it } from 'vitest'
import { mapServerFields, toRequest, validateDeliveryPrice } from './form'

const valid = { name: 'Сочи', value: '', price: '150' }

describe('validateDeliveryPrice', () => {
	it('accepts a valid form', () => {
		expect(validateDeliveryPrice(valid)).toEqual({})
		expect(validateDeliveryPrice({ ...valid, price: '150,5' })).toEqual({})
		expect(validateDeliveryPrice({ ...valid, price: '99.99' })).toEqual({})
	})

	it('requires a name', () => {
		expect(validateDeliveryPrice({ ...valid, name: '   ' }).name).toBe('Укажите название города')
	})

	it('limits name and description to 255 characters', () => {
		expect(validateDeliveryPrice({ ...valid, name: 'а'.repeat(255) }).name).toBeUndefined()
		expect(validateDeliveryPrice({ ...valid, name: 'а'.repeat(256) }).name).toBeDefined()
		expect(validateDeliveryPrice({ ...valid, value: 'а'.repeat(256) }).value).toBeDefined()
	})

	it.each(['', '0', '-1', 'abc', '1.555', '1 000'])('rejects the price %j', (price) => {
		expect(validateDeliveryPrice({ ...valid, price }).price).toBeDefined()
	})
})

describe('toRequest', () => {
	it('trims the texts and reads a comma as the decimal separator', () => {
		expect(toRequest({ name: '  Сочи ', value: ' Россия ', price: ' 150,5 ' })).toEqual({
			name: 'Сочи',
			value: 'Россия',
			price: 150.5,
		})
	})
})

describe('mapServerFields', () => {
	it('strips the array prefix', () => {
		expect(mapServerFields({ 'deliveryPrices[0].price': 'bad', name: 'taken' })).toEqual({
			price: 'bad',
			name: 'taken',
		})
	})

	it('ignores unknown keys', () => {
		expect(mapServerFields({ deliveryPrices: 'empty' })).toEqual({})
	})
})
