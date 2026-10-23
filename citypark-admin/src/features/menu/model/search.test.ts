import { describe, expect, it } from 'vitest'
import { parseMenuSearch } from './search'

describe('parseMenuSearch', () => {
	it('keeps valid values', () => {
		expect(parseMenuSearch({ category: 3, q: 'салат' })).toEqual({ category: 3, q: 'салат' })
		expect(parseMenuSearch({ category: '0' })).toEqual({ category: 0 })
		expect(parseMenuSearch({ q: 123 })).toEqual({ q: '123' })
	})

	it('drops invalid values', () => {
		expect(parseMenuSearch({ category: -1, q: '  ' })).toEqual({})
		expect(parseMenuSearch({ category: 1.5 })).toEqual({})
		expect(parseMenuSearch({ category: 'abc', q: 'x'.repeat(101) })).toEqual({})
	})
})
