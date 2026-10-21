"use client";

import Image from "next/image";
import { AddToCartButton } from "@/features/add-to-cart";
import type { PricedLine } from "@/entities/dish";
import { formatPrice } from "@/shared/lib";

/** One cart line with a «− N +» stepper; shared by the mobile CartSheet and the desktop CartPanel. */
export function CartLine({ item }: { item: PricedLine }) {
  return (
    <li className="flex items-center gap-2.5 border-b border-inverse-border py-2.5">
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
  );
}
