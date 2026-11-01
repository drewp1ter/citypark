import { describe, expect, it } from 'vitest'
import { mapServerFields, toForm, toRequest, validateDeliverySettings } from './form'
const base = { cooking: '~30 мин', outskirtsLabel: 'СОТ и ДНТ', outskirtsMin: 200, outskirtsMax: 350 }
describe('delivery settings form', () => {
	it('roundtrips and trims', () => {
		expect(toRequest(toForm(base))).toEqual(base)
		expect(toRequest({ ...toForm(base), cooking: ' x ', outskirtsMin: ' 0 ' })).toEqual({
			...base,
			cooking: 'x',
			outskirtsMin: 0,
		})
	})
	it('accepts zero equal and maximum bounds', () => {
		for (const n of [0, 200, 100000])
			expect(validateDeliverySettings({ ...toForm(base), outskirtsMin: String(n), outskirtsMax: String(n) })).toEqual(
				{}
			)
	})
	it('counts Unicode characters', () => {
		for (const [key, max] of [
			['cooking', 100],
			['outskirtsLabel', 60],
		] as const) {
			expect(validateDeliverySettings({ ...toForm(base), [key]: '😀'.repeat(max) })).toEqual({})
			expect(validateDeliverySettings({ ...toForm(base), [key]: '😀'.repeat(max + 1) })[key]).toBe(
				`значение должно быть не больше ${max}`
			)
		}
	})
	it('requires every field', () => {
		for (const key of Object.keys(base)) {
			for (const value of ['', ' \t'])
				expect(validateDeliverySettings({ ...toForm(base), [key]: value })).toHaveProperty(
					key,
					'поле не должно быть пустым'
				)
		}
	})
	it('rejects syntax and oversized numbers', () => {
		for (const key of ['outskirtsMin', 'outskirtsMax'] as const) {
			for (const value of ['-1', '+1', '1.5', '1e2', 'Infinity'])
				expect(validateDeliverySettings({ ...toForm(base), [key]: value })[key]).toBe(
					'Введите целое число от 0 до 100000'
				)
			for (const value of ['100001', '9'.repeat(400)])
				expect(validateDeliverySettings({ ...toForm(base), [key]: value })[key]).toBe(
					'значение должно быть не больше 100000'
				)
		}
	})
	it('rejects reversed range', () => {
		expect(validateDeliverySettings({ ...toForm(base), outskirtsMax: '199' }).outskirtsMax).toBe(
			'Максимальная цена должна быть не меньше минимальной'
		)
	})
	it('maps known fields only', () => {
		expect(mapServerFields(undefined)).toEqual({})
		expect(mapServerFields({ cooking: 'bad', unknown: 'bad', outskirtsMin: 'required' })).toEqual({
			cooking: 'bad',
			outskirtsMin: 'required',
		})
	})
})
