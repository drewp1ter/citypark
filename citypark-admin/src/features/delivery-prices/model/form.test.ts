import { describe, expect, it } from 'vitest'
import { mapServerFields, toRequest, validateDeliveryPrice } from './form'

const valid = { name: 'Сочи', value: '', price: '150', freeFrom: '' }

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

	it.each(['', '0'])('accepts the free delivery threshold %j', (freeFrom) => {
		expect(validateDeliveryPrice({ ...valid, freeFrom }).freeFrom).toBeUndefined()
	})

	it.each(['-1', 'abc', '1.555'])('rejects the free delivery threshold %j', (freeFrom) => {
		expect(validateDeliveryPrice({ ...valid, freeFrom }).freeFrom).toBeDefined()
	})

	it.each(['', '0', '-1', 'abc', '1.555', '1 000'])('rejects the price %j', (price) => {
		expect(validateDeliveryPrice({ ...valid, price }).price).toBeDefined()
	})
})

describe('toRequest', () => {
	it('trims the texts and reads a comma as the decimal separator', () => {
		expect(toRequest({ name: '  Сочи ', value: ' Россия ', price: ' 150,5 ', freeFrom: '' })).toEqual({
			regionCode: 'RU-KDA',
			name: 'Сочи',
			value: 'Россия',
			price: 150.5,
			freeFrom: 0,
		})
	})

	it('reads the threshold with a comma', () => {
		expect(toRequest({ ...valid, freeFrom: '500,5' }).freeFrom).toBe(500.5)
	})
})

describe('mapServerFields', () => {
	it('strips the array prefix', () => {
		expect(mapServerFields({ 'deliveryPrices[0].price': 'bad', name: 'taken' })).toEqual({
			price: 'bad',
			name: 'taken',
		})
	})

	it('maps freeFrom', () => {
		expect(mapServerFields({ 'deliveryPrices[0].freeFrom': 'x' })).toEqual({ freeFrom: 'x' })
		expect(mapServerFields({ freeFrom: 'x' })).toEqual({ freeFrom: 'x' })
	})

	it('ignores unknown keys', () => {
		expect(mapServerFields({ deliveryPrices: 'empty' })).toEqual({})
	})
})

it('sends and maps the selected region', () => {
	expect(toRequest({ ...valid, regionCode: 'RU-AD' }).regionCode).toBe('RU-AD')
	expect(mapServerFields({ 'deliveryPrices[0].regionCode': 'bad' })).toEqual({ regionCode: 'bad' })
	expect(validateDeliveryPrice({ ...valid, regionCode: 'RU-AD' })).toEqual({})
})
