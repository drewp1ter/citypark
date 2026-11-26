import type { Dish, DishVariant } from '../model/types'

/** Structurally the same as a cart line; the dish entity does not import the cart entity. */
type LineRef = { dishId: string; variantId: string; quantity: number }

export type PricedLine = { dish: Dish; variant: DishVariant; quantity: number; total: number }
export type UnavailableLine = LineRef & { name?: string }

export type PricedCart = {
	items: PricedLine[]
	unavailable: UnavailableLine[]
	/** Number of portions ("4 блюда"). */
	count: number
	/** Sum of the lines, rubles. */
	subtotal: number
}

/**
 * Joins cart lines with the current menu. The cart stores only ids and quantities, so prices always come
 * from the menu and are never taken from the browser. Keep unavailable lines so they can be removed.
 */
export function priceLines(
	lines: readonly LineRef[],
	dishes: readonly Dish[],
	rejected: readonly Pick<LineRef, 'dishId' | 'variantId'>[] = []
): PricedCart {
	const byId = new Map(dishes.map((dish) => [dish.id, dish]))
	const items: PricedLine[] = []
	const unavailable: UnavailableLine[] = []

	for (const line of lines) {
		const dish = byId.get(line.dishId)
		const variant = dish?.variants.find((v) => v.id === line.variantId)
		if (
			!dish?.available ||
			dish.orderBlocked ||
			!variant ||
			variant.price === null ||
			rejected.some((key) => key.dishId === line.dishId && key.variantId === line.variantId)
		) {
			const name = dish && variant?.label && dish.variants.length > 1 ? `${dish.name} · ${variant.label}` : dish?.name
			unavailable.push({ ...line, name })
			continue
		}
		items.push({ dish, variant, quantity: line.quantity, total: variant.price * line.quantity })
	}

	return {
		items,
		unavailable,
		count: items.reduce((sum, item) => sum + item.quantity, 0),
		subtotal: items.reduce((sum, item) => sum + item.total, 0),
	}
}
