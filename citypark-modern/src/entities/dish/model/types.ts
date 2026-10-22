import type { Photo } from '@/shared/lib'

/**
 * One size / portion of a dish. Every dish has at least one variant;
 * the size switch ("500 г / 1 кг") is shown when there are two or more.
 */
export type DishVariant = {
	id: string
	/** Label on the size switch: "500 г". */
	label: string
	/** "200 г", "250/50/20 г". */
	weight: string
	/** Rubles. `null` = price not known yet (placeholder): the variant is shown but cannot be ordered. */
	price: number | null
}

export type Dish = {
	id: string
	/** Id of the POS item when a POS is connected. */
	externalId?: string
	name: string
	categoryId: string
	/** "Свекла, чеснок, грецкий орех" */
	composition?: string
	photo: Photo
	variants: readonly [DishVariant, ...DishVariant[]]
	/** False = stop-list. */
	available: boolean
}

export type MenuCategory = {
	id: string
	/** Name on the tab. */
	name: string
	/** Heading above the dishes ("Постное меню"); the name is used when there is none. */
	title?: string
}

export type Menu = { categories: readonly MenuCategory[]; dishes: readonly Dish[] }
