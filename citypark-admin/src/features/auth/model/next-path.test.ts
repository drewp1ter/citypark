import { describe, expect, it } from 'vitest'
import { safeNextPath } from './next-path'

describe('safeNextPath', () => {
	it.each(['/orders?page=2', '/delivery', '/content/news#top'])('keeps %s', (path) => {
		expect(safeNextPath(path)).toBe(path)
	})

	it.each([
		undefined,
		'',
		'//evil.com',
		'/\\evil.com',
		'https://evil.com',
		'orders',
		'/login',
		'/login?next=%2Forders',
		'/login#x',
	])('falls back to /orders for %s', (path) => {
		expect(safeNextPath(path)).toBe('/orders')
	})

	it('keeps paths that only start like /login', () => {
		expect(safeNextPath('/login-history')).toBe('/login-history')
	})
})
