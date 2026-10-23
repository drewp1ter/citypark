import type { components } from '../../../shared/api/schema'

export type ApiCategory = components['schemas']['Category']
export type ApiProduct = components['schemas']['Product']

export type MenuGroup = { id: number | 'hidden'; name: string; available: boolean; dishes: ApiProduct[] }

export const HIDDEN_NAME = 'Скрытые в iiko'
/** The value of the `category` search param that selects the 'hidden' group. */
export const HIDDEN_PARAM = 0

/** Categories in API order, then one 'hidden' group for products whose category is not in the list (if any). */
export function groupMenu(categories: ApiCategory[], products: ApiProduct[]): MenuGroup[] {
	const groups = new Map<number, MenuGroup>()
	for (const c of categories) {
		groups.set(c.id, { id: c.id, name: c.name, available: c.isAvailableForOrder, dishes: [] })
	}
	const hidden: ApiProduct[] = []
	for (const p of products) {
		const group = groups.get(p.categoryId)
		if (group === undefined) hidden.push(p)
		else group.dishes.push(p)
	}
	const result = [...groups.values()]
	if (hidden.length > 0) result.push({ id: 'hidden', name: HIDDEN_NAME, available: true, dishes: hidden })
	return result
}

/** The group a `category` search param points to; undefined means "all". */
export function findGroup(groups: MenuGroup[], param: number | undefined): MenuGroup | undefined {
	if (param === undefined) return undefined
	return groups.find((g) => (g.id === 'hidden' ? param === HIDDEN_PARAM : g.id === param))
}

/** Dishes whose name contains the text, case does not matter; an empty text keeps all. */
export function filterDishes(dishes: ApiProduct[], text: string | undefined): ApiProduct[] {
	const needle = text?.trim().toLowerCase() ?? ''
	if (needle === '') return dishes
	return dishes.filter((d) => d.name.toLowerCase().includes(needle))
}
