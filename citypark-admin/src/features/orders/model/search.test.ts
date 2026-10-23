import { describe, expect, it } from 'vitest'
import { hasFilters, parseOrderSearch, toApiQuery } from './search'

describe('parseOrderSearch', () => {
	it('keeps valid values', () => {
		expect(
			parseOrderSearch({
				q: ' иван ',
				from: '2026-10-01',
				to: '2026-10-21',
				status: 'paid',
				paymentType: 'online',
				page: 3,
			})
		).toEqual({ q: 'иван', from: '2026-10-01', to: '2026-10-21', status: 'paid', paymentType: 'online', page: 3 })
	})

	it('drops an invalid page', () => {
		expect(parseOrderSearch({ page: '0' })).toEqual({})
		expect(parseOrderSearch({ page: -1 })).toEqual({})
		expect(parseOrderSearch({ page: 1.5 })).toEqual({})
		expect(parseOrderSearch({ page: 'abc' })).toEqual({})
		expect(parseOrderSearch({ page: '2' })).toEqual({ page: 2 })
	})

	it('drops a date that is not YYYY-MM-DD', () => {
		expect(parseOrderSearch({ from: '21.10.2026' })).toEqual({})
		expect(parseOrderSearch({ to: '2026-02-30' })).toEqual({})
	})

	it('drops a "to" before "from"', () => {
		expect(parseOrderSearch({ from: '2026-10-21', to: '2026-10-01' })).toEqual({ from: '2026-10-21' })
	})

	it('drops an unknown status or payment type', () => {
		expect(parseOrderSearch({ status: 'done' })).toEqual({})
		expect(parseOrderSearch({ paymentType: 'card' })).toEqual({})
	})

	it('trims q, accepts a number, drops empty and too long', () => {
		expect(parseOrderSearch({ q: 123 })).toEqual({ q: '123' })
		expect(parseOrderSearch({ q: '   ' })).toEqual({})
		expect(parseOrderSearch({ q: 'a'.repeat(101) })).toEqual({})
		expect(parseOrderSearch({ q: 'a'.repeat(100) }).q).toHaveLength(100)
	})
})

describe('toApiQuery', () => {
	it('omits empty values and adds the page size', () => {
		expect(toApiQuery({ q: '', from: undefined }, 20)).toEqual({ page: 1, pageSize: 20 })
	})

	it('passes the filters', () => {
		expect(toApiQuery({ q: 'иван', status: 'paid', page: 2 }, 20)).toEqual({
			q: 'иван',
			status: 'paid',
			page: 2,
			pageSize: 20,
		})
	})
})

describe('hasFilters', () => {
	it('ignores the page', () => {
		expect(hasFilters({ page: 2 })).toBe(false)
		expect(hasFilters({ status: 'new' })).toBe(true)
	})
})
