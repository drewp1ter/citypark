import type { Dish, Menu, MenuCategory } from '../model/types'

export type HomepageMenuConfiguration = {
	categories: { categoryId: number; productIds: number[] }[]
}
export type HomepageMenuGroup = { category: MenuCategory; dishes: readonly Dish[] }

/** Resolve saved IDs against the live visible menu. Never top up a nonempty manual selection. */
export function selectHomepageMenu(
	menu: Menu,
	configuration: HomepageMenuConfiguration,
	placeholderPhoto: string
): HomepageMenuGroup[] {
	const selections =
		configuration.categories.length > 0
			? configuration.categories
			: menu.categories.slice(0, 4).map((category) => ({ categoryId: Number(category.id), productIds: [] }))
	const groups: HomepageMenuGroup[] = []
	for (const selection of selections) {
		const category = menu.categories.find((item) => item.id === String(selection.categoryId))
		if (!category) continue
		const candidates = menu.dishes.filter((dish) => dish.categoryId === category.id)
		const manual = selection.productIds.flatMap((id) => {
			const dish = candidates.find((item) => item.id === String(id))
			return dish ? [dish] : []
		})
		const score = (dish: Dish) => (dish.available ? 2 : 0) + (dish.photo.src !== placeholderPhoto ? 1 : 0)
		const dishes =
			manual.length > 0 ? manual.slice(0, 3) : [...candidates].sort((a, b) => score(b) - score(a)).slice(0, 3)
		if (dishes.length > 0) groups.push({ category, dishes })
	}
	return groups
}
