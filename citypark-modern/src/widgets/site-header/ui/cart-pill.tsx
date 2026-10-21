"use client";

import { useMemo } from "react";
import { useCartLines } from "@/entities/cart";
import { priceLines, type Dish } from "@/entities/dish";
import { formatPrice } from "@/shared/lib";
import { AppLink } from "@/shared/ui";

/** "Корзина [1 250 ₽]" button of the menu page header; it scrolls to the cart column. */
export function CartPill({ dishes }: { dishes: readonly Dish[] }) {
  const lines = useCartLines();
  const { subtotal, count } = useMemo(() => priceLines(lines, dishes), [lines, dishes]);

  return (
    <AppLink
      href="#cart"
      className="inline-flex min-h-11 items-center gap-2.5 rounded-pill bg-inverse-bg pr-2 pl-5 font-semibold text-inverse-text"
    >
      Корзина
      {count > 0 && (
        <span className="rounded-pill bg-accent px-2.5 py-1 text-sm text-on-accent">{formatPrice(subtotal)}</span>
      )}
    </AppLink>
  );
}
