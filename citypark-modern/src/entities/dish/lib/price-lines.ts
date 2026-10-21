import type { Dish, DishVariant } from "../model/types";

/** Structurally the same as a cart line; the dish entity does not import the cart entity. */
type LineRef = { dishId: string; variantId: string; quantity: number };

export type PricedLine = { dish: Dish; variant: DishVariant; quantity: number; total: number };

export type PricedCart = {
  items: PricedLine[];
  /** Number of portions ("4 блюда"). */
  count: number;
  /** Sum of the lines, rubles. */
  subtotal: number;
};

/**
 * Joins cart lines with the current menu. The cart stores only ids and quantities, so prices always come
 * from the menu and are never taken from the browser. Lines whose dish or size no longer exists, is on
 * the stop-list or has no price yet are skipped.
 */
export function priceLines(lines: readonly LineRef[], dishes: readonly Dish[]): PricedCart {
  const byId = new Map(dishes.map((dish) => [dish.id, dish]));
  const items: PricedLine[] = [];

  for (const line of lines) {
    const dish = byId.get(line.dishId);
    const variant = dish?.variants.find((v) => v.id === line.variantId);
    if (!dish || !dish.available || !variant || variant.price === null) continue;
    items.push({ dish, variant, quantity: line.quantity, total: variant.price * line.quantity });
  }

  return {
    items,
    count: items.reduce((sum, item) => sum + item.quantity, 0),
    subtotal: items.reduce((sum, item) => sum + item.total, 0),
  };
}
