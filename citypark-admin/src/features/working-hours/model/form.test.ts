import { describe, expect, it } from 'vitest'
import { mapServerFields, normalizeTime, toForm, toRequest, validateWorkingHours } from './form'

describe('normalizeTime', () => {
	it.each([
		['9:00', '09:00'],
		[' 10:30 ', '10:30'],
		['09.30', '09:30'],
		['23:59', '23:59'],
		['00:00', '00:00'],
	])('%j → %j', (input, expected) => {
		expect(normalizeTime(input)).toBe(expected)
	})

	it.each(['24:00', '9:60', '900', 'abc', ''])('%j → null', (input) => {
		expect(normalizeTime(input)).toBeNull()
	})
})

describe('validateWorkingHours', () => {
	const format = 'Время в формате ЧЧ:ММ, например 09:00'
	const later = 'Время окончания должно быть позже начала'

	it('requires both times', () => {
		expect(validateWorkingHours({ start: '', end: '22:00' })).toEqual({ start: 'Укажите время' })
		expect(validateWorkingHours({ start: '09:00', end: ' ' })).toEqual({ end: 'Укажите время' })
	})

	it('rejects a bad format', () => {
		expect(validateWorkingHours({ start: '25:00', end: '22:00' })).toEqual({ start: format })
		expect(validateWorkingHours({ start: '09:00', end: 'abc' })).toEqual({ end: format })
	})

	it('puts the order error on the end', () => {
		expect(validateWorkingHours({ start: '12:00', end: '12:00' })).toEqual({ end: later })
		expect(validateWorkingHours({ start: '13:00', end: '12:00' })).toEqual({ end: later })
		expect(validateWorkingHours({ start: '10:00', end: '9:00' })).toEqual({ end: later })
	})

	it('accepts a valid range', () => {
		expect(validateWorkingHours({ start: '9:00', end: '22:00' })).toEqual({})
	})
})

describe('toRequest and toForm', () => {
	it('normalises the times', () => {
		expect(toRequest({ start: '9:00', end: '22:00' })).toEqual({ start: '09:00', end: '22:00' })
	})

	it('copies the times from the server value', () => {
		expect(toForm({ start: '09:00', end: '22:00' })).toEqual({ start: '09:00', end: '22:00' })
	})
})

describe('mapServerFields', () => {
	it('picks start and end and ignores the rest', () => {
		expect(mapServerFields({ start: 'a', end: 'b', other: 'c' })).toEqual({ start: 'a', end: 'b' })
		expect(mapServerFields(undefined)).toEqual({})
	})
})
