"use client";

import { useRef, useState, type ReactNode } from "react";
import type { Dish, MenuCategory } from "@/entities/dish";
import { ClearButton, SearchIcon } from "@/shared/ui";
import { CategoryTabs } from "./category-tabs";
import { DishCard } from "./dish-card";

type MenuCatalogProps = {
  categories: readonly MenuCategory[];
  /** All dishes; the catalog picks the ones to show. */
  dishes: readonly Dish[];
  /** Selected category (from the URL). The first one when missing or unknown. */
  categoryId?: string;
  /** The cart column, shown to the right of the dishes from 1024 px. */
  aside: ReactNode;
};

/**
 * Search, category tabs and the dish grid. The tabs are links (/menu?category=…), so the server knows the
 * category and the address can be shared. The search runs in the browser and looks through the whole menu.
 */
export function MenuCatalog({ categories, dishes, categoryId, aside }: MenuCatalogProps) {
  const [query, setQuery] = useState("");
  const searchRef = useRef<HTMLInputElement>(null);
  const category = categories.find((c) => c.id === categoryId) ?? categories[0];

  const needle = query.trim().toLocaleLowerCase("ru");
  const visible = needle
    ? dishes.filter((dish) => `${dish.name} ${dish.composition ?? ""}`.toLocaleLowerCase("ru").includes(needle))
    : dishes.filter((dish) => dish.categoryId === category.id);
  const title = needle ? `Найдено: ${visible.length}` : (category.title ?? category.name);

  return (
    <div className="mt-1 grid md:mt-6 md:grid-cols-[240px_minmax(0,1fr)] md:gap-x-2">
      <div className="bg-bg py-2 md:sticky md:top-[84px] md:z-[5]">
        <label className="flex min-h-12 items-center gap-2 rounded-pill border border-border bg-surface pl-4 pr-4 has-[button]:pr-1">
          <SearchIcon />
          <input
            ref={searchRef}
            type="search"
            placeholder="Поиск блюда"
            aria-label="Поиск по меню"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="min-h-10 w-full border-0 bg-transparent"
          />
          {query !== "" && (
            <ClearButton
              onClick={() => {
                setQuery("");
                searchRef.current?.focus();
              }}
            />
          )}
        </label>
      </div>

      <CategoryTabs categories={categories} selectedId={category.id} />

      <div className="flex items-start gap-3 pt-3 md:col-span-2 md:pt-4">
        <div className="min-w-0 flex-1">
          <h2 className="mb-3 font-display text-[22px] font-medium md:mb-4 md:text-[30px] md:tracking-[-0.02em]">
            {title}
          </h2>
          {visible.length > 0 ? (
            <div className="grid grid-cols-2 gap-2 md:grid-cols-3 md:gap-3">
              {visible.map((dish, index) => (
                <DishCard key={dish.id} dish={dish} eager={index < 3} />
              ))}
            </div>
          ) : (
            <p className="text-text-muted" role="status">
              {needle
                ? `Ничего не нашли по запросу «${query.trim()}». Попробуйте другое слово.`
                : "В этом разделе скоро появятся блюда."}
            </p>
          )}
        </div>
        {aside}
      </div>
    </div>
  );
}
