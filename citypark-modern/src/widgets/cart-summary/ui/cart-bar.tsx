"use client";

import type { Dish } from "@/entities/dish";
import { formatPrice, plural } from "@/shared/lib";
import { AppLink } from "@/shared/ui";
import { usePricedCart } from "./use-priced-cart";

/** Sticky bar at the bottom of the menu page below 1024 px: number of dishes, total, link to the checkout. */
export function CartBar({ dishes }: { dishes: readonly Dish[] }) {
  const { count, subtotal } = usePricedCart(dishes);
  if (count === 0) return null;

  return (
    <nav
      aria-label="Корзина"
      className="sticky bottom-0 z-10 bg-linear-to-b from-transparent to-bg to-35% px-3 pt-2.5 pb-3 lg:hidden"
    >
      <AppLink
        href="/checkout"
        className="flex min-h-[58px] items-center justify-between rounded-pill bg-inverse-bg pr-2 pl-[22px] font-semibold text-inverse-text"
      >
        <span>
          Корзина · {count} {plural(count, ["блюдо", "блюда", "блюд"])}
        </span>
        <span className="rounded-pill bg-accent px-4 py-2.5 text-on-accent">{formatPrice(subtotal)} ↗</span>
      </AppLink>
    </nav>
  );
}
