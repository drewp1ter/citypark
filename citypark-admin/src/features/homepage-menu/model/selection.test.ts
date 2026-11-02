import { describe, expect, it } from 'vitest'
import type { Product } from '../api'
import { categoryProducts, moveItem } from './selection'

describe('homepage selections', () => {
	it('moves items in both directions and preserves invalid moves', () => {
		const items = [1, 2, 3]
		expect(moveItem(items, 0, 1)).toEqual([2, 1, 3])
		expect(moveItem(items, 2, -1)).toEqual([1, 3, 2])
		for (const [index, direction] of [
			[0, -1],
			[2, 1],
			[-1, 1],
			[3, -1],
		] as const)
			expect(moveItem(items, index, direction)).toBe(items)
		expect(items).toEqual([1, 2, 3])
	})
	it('searches only within the category, ignoring case and surrounding spaces', () => {
		const products = [
			{ id: 1, categoryId: 1, name: 'Салат' },
			{ id: 2, categoryId: 2, name: 'Салат' },
			{ id: 3, categoryId: 1, name: 'Суп' },
		] as Product[]
		expect(categoryProducts(products, 1, ' САЛ ')).toEqual([products[0]])
		expect(categoryProducts(products, 1, '')).toEqual([products[0], products[2]])
		expect(categoryProducts(products, 5, '')).toEqual([])
	})
})
