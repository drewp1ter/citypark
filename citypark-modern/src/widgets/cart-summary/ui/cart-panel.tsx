"use client";

import Image from "next/image";
import { quoteDelivery, type Dish } from "@/entities/dish";
import { site } from "@/shared/config";
import { formatPrice, plural } from "@/shared/lib";
import { AppLink, Button } from "@/shared/ui";
import { usePricedCart } from "./use-priced-cart";

/** Dark cart column next to the menu (from 1024 px). Below that the CartBar is used instead. */
export function CartPanel({ dishes }: { dishes: readonly Dish[] }) {
  const { items, count, subtotal, ready } = usePricedCart(dishes);
  const quote = quoteDelivery({ method: "delivery", zone: "city", subtotal });
  const empty = ready && items.length === 0;

  return (
    <aside
      id="cart"
      aria-label="Корзина"
      className="hidden w-[340px] shrink-0 flex-col gap-3.5 rounded-xl bg-inverse-bg p-6 text-inverse-text lg:sticky lg:top-[160px] lg:flex"
    >
      <div className="flex items-center justify-between">
        <h2 className="heading text-2xl">Корзина</h2>
        {count > 0 && (
          <span className="text-sm text-inverse-muted">
            {count} {plural(count, ["блюдо", "блюда", "блюд"])}
          </span>
        )}
      </div>

      {empty ? (
        <p className="text-[15px] text-inverse-muted">Пока пусто. Добавьте блюда кнопкой «+» на карточках.</p>
      ) : (
        <>
          <ul>
            {items.map((item) => (
              <li
                key={`${item.dish.id}:${item.variant.id}`}
                className="flex items-center gap-2.5 border-b border-inverse-border py-2.5"
              >
                <Image
                  src={item.dish.photo.src}
                  alt=""
                  width={48}
                  height={48}
                  className="size-12 shrink-0 rounded-2xs object-cover"
                />
                <span className="flex-1 text-[15px]">
                  {item.dish.name}
                  {item.dish.variants.length > 1 && <span className="text-inverse-muted"> · {item.variant.label}</span>}
                  <br />
                  <span className="text-[13px] text-inverse-muted">× {item.quantity}</span>
                </span>
                <strong>{formatPrice(item.total)}</strong>
              </li>
            ))}
          </ul>
          <div className="flex justify-between text-[15px]">
            <span className="text-inverse-muted">Доставка</span>
            <span className="font-semibold text-accent">{quote.label}</span>
          </div>
          <div className="flex items-baseline justify-between">
            <span>Итого</span>
            <span className="font-display text-[30px] font-medium">{formatPrice(subtotal)}</span>
          </div>
          <Button href="/checkout" className="min-h-[54px]">
            Оформить заказ
          </Button>
        </>
      )}

      <p className="text-center text-[13px] text-inverse-muted">
        {empty ? "Заказ по телефону: " : "или "}
        <AppLink href={site.phones.restaurant.href} className="tap-area text-inverse-text">
          {site.phones.restaurant.display}
        </AppLink>
      </p>
    </aside>
  );
}
