import { categories, dishes } from "../model/menu.data";
import type { Dish, Menu } from "../model/types";

// Reads go through these functions only. Today they return static data;
// later they call the POS / CMS (prices, stop-list) and nothing else has to change.

export async function getMenu(): Promise<Menu> {
  return { categories, dishes };
}

export async function getDishesByIds(ids: readonly string[]): Promise<readonly Dish[]> {
  return dishes.filter((dish) => ids.includes(dish.id));
}
