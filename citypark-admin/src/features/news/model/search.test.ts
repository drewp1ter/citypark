import { describe, expect, it } from 'vitest'
import { parsePage } from './search'

describe('parsePage', () => {
	it.each([
		[3, 3],
		['2', 2],
		[1, 1],
	])('reads %j as %j', (value, expected) => {
		expect(parsePage(value)).toBe(expected)
	})

	it.each([0, -1, 1.5, 'abc', '', ' ', null, undefined, {}])('drops %j', (value) => {
		expect(parsePage(value)).toBeUndefined()
	})
})
