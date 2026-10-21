import { cache } from "react";
import type { Dish, Menu } from "../model/types";
import { fetchCategories, fetchProducts } from "./menu-api";
import { toCategory, toDish } from "./to-dish";

// Reads go through these functions only. Errors are not caught here: the page decides what to show.

export const getMenu = cache(async (): Promise<Menu> => {
  const [categories, products] = await Promise.all([fetchCategories(), fetchProducts()]);
  const ids = new Set(categories.map((c) => c.id));
  // GET /products also returns the products of hidden categories.
  return { categories: categories.map(toCategory), dishes: products.filter((p) => ids.has(p.categoryId)).map(toDish) };
});

export async function getDishesByIds(ids: readonly string[]): Promise<readonly Dish[]> {
  return (await getMenu()).dishes.filter((dish) => ids.includes(dish.id));
}
