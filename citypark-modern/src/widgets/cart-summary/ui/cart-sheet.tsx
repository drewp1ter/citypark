"use client";

import Image from "next/image";
import type { MouseEvent, RefObject } from "react";
import { AddToCartButton } from "@/features/add-to-cart";
import { quoteDelivery, type Dish } from "@/entities/dish";
import { site } from "@/shared/config";
import { formatPrice, plural } from "@/shared/lib";
import { AppLink, Button, CloseIcon, IconButton } from "@/shared/ui";
import { usePricedCart } from "./use-priced-cart";

type CartSheetProps = {
  dishes: readonly Dish[];
  dialog: RefObject<HTMLDialogElement | null>;
  onClose: () => void;
};

/**
 * The cart as a bottom sheet (below 1024 px), opened from the CartBar.
 * It is a native modal <dialog>, like MobileMenu: Esc, focus trap and an inert page come from the browser.
 */
export function CartSheet({ dishes, dialog, onClose }: CartSheetProps) {
  const { items, count, subtotal } = usePricedCart(dishes);
  const quote = quoteDelivery({ method: "delivery", zone: "city", subtotal });

  const closeOnBackdropOrLink = (event: MouseEvent<HTMLDialogElement>) => {
    const target = event.target as HTMLElement;
    if (target === dialog.current || target.closest("a")) dialog.current?.close();
  };

  return (
    <dialog
      ref={dialog}
      aria-labelledby="cart-sheet-title"
      onClick={closeOnBackdropOrLink}
      onClose={onClose}
      className="fixed inset-x-0 top-auto bottom-0 m-0 max-h-[85dvh] w-full max-w-none overflow-y-auto rounded-t-xl bg-inverse-bg p-5 pb-6 text-inverse-text backdrop:bg-scrim lg:hidden"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 id="cart-sheet-title" className="heading text-2xl">
          Корзина
        </h2>
        <span className="ml-auto text-sm text-inverse-muted">
          {count} {plural(count, ["блюдо", "блюда", "блюд"])}
        </span>
        <IconButton variant="inverse" aria-label="Закрыть корзину" onClick={() => dialog.current?.close()}>
          <CloseIcon />
        </IconButton>
      </div>

      <ul className="mt-2">
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
            <span className="min-w-0 flex-1 text-[15px]">
              {item.dish.name}
              {item.dish.variants.length > 1 && <span className="text-inverse-muted"> · {item.variant.label}</span>}
              <br />
              <strong>{formatPrice(item.total)}</strong>
            </span>
            <AddToCartButton dish={item.dish} variantId={item.variant.id} />
          </li>
        ))}
      </ul>

      <div className="mt-3.5 flex justify-between text-[15px]">
        <span className="text-inverse-muted">Доставка</span>
        <span className="font-semibold text-accent">{quote.label}</span>
      </div>
      <div className="mt-3.5 flex items-baseline justify-between">
        <span>Итого</span>
        <span className="font-display text-[30px] font-medium">{formatPrice(subtotal)}</span>
      </div>
      <Button href="/checkout" className="mt-3.5 min-h-[54px] w-full">
        Оформить заказ
      </Button>
      <p className="mt-3.5 text-center text-[13px] text-inverse-muted">
        или{" "}
        <AppLink href={site.phones.restaurant.href} className="tap-area text-inverse-text">
          {site.phones.restaurant.display}
        </AppLink>
      </p>
    </dialog>
  );
}
