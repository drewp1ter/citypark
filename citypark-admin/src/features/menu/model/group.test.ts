import { describe, expect, it } from 'vitest'
import { filterDishes, findGroup, groupMenu, HIDDEN_PARAM, type ApiCategory, type ApiProduct } from './group'

function category(id: number, name = `Категория ${id}`): ApiCategory {
	return {
		id,
		name,
		description: '',
		images: [],
		isAvailableForOrder: true,
		salesDays: '',
		startTime: '',
		endTime: '',
	}
}

function product(id: number, categoryId: number, name = `Блюдо ${id}`): ApiProduct {
	return {
		id,
		categoryId,
		name,
		description: '',
		measureUnit: '',
		weight: 0,
		proteins: 0,
		fats: 0,
		carbohydrates: 0,
		calories: 0,
		flags: 0,
		price: 100,
		images: [],
		isAvailableForOrder: true,
		salesDays: '',
		startTime: '',
		endTime: '',
	}
}

describe('groupMenu', () => {
	it('puts products into their categories, in category order', () => {
		const groups = groupMenu([category(2), category(1)], [product(10, 1), product(11, 2), product(12, 1)])
		expect(groups.map((g) => g.id)).toEqual([2, 1])
		expect(groups[0]?.dishes.map((d) => d.id)).toEqual([11])
		expect(groups[1]?.dishes.map((d) => d.id)).toEqual([10, 12])
	})

	it('puts products of an unknown category into the hidden group, last', () => {
		const groups = groupMenu([category(1)], [product(10, 1), product(11, 99)])
		expect(groups.map((g) => g.id)).toEqual([1, 'hidden'])
		expect(groups[1]).toMatchObject({ name: 'Скрытые в iiko', dishes: [{ id: 11 }] })
	})

	it('has no hidden group when there are none', () => {
		expect(groupMenu([category(1)], [product(10, 1)]).some((g) => g.id === 'hidden')).toBe(false)
	})

	it('keeps an empty category and returns [] for empty input', () => {
		expect(groupMenu([category(1)], [])[0]?.dishes).toEqual([])
		expect(groupMenu([], [])).toEqual([])
	})
})

describe('findGroup', () => {
	const groups = groupMenu([category(1)], [product(10, 99)])

	it('finds a category by id and the hidden group by HIDDEN_PARAM', () => {
		expect(findGroup(groups, 1)?.id).toBe(1)
		expect(findGroup(groups, HIDDEN_PARAM)?.id).toBe('hidden')
	})

	it('returns undefined for no param or an unknown id', () => {
		expect(findGroup(groups, undefined)).toBeUndefined()
		expect(findGroup(groups, 5)).toBeUndefined()
	})
})

describe('filterDishes', () => {
	const dishes = [product(1, 1, 'Шашлык из баранины'), product(2, 1, 'Салат')]

	it('matches part of the name ignoring case', () => {
		expect(filterDishes(dishes, ' ШАШЛЫК ').map((d) => d.id)).toEqual([1])
	})

	it('keeps everything for an empty text', () => {
		expect(filterDishes(dishes, undefined)).toBe(dishes)
		expect(filterDishes(dishes, '  ')).toBe(dishes)
	})
})
