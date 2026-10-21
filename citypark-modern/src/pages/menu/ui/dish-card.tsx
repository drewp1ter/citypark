"use client";

import Image from "next/image";
import { useState } from "react";
import type { Dish } from "@/entities/dish";
import { AddToCartButton } from "@/features/add-to-cart";
import { cn, formatPrice } from "@/shared/lib";
import { Card } from "@/shared/ui";

/** "500 г / 1 кг" switch of a dish that comes in more than one size. */
function SizeSwitch({ dish, value, onChange }: { dish: Dish; value: string; onChange: (id: string) => void }) {
  return (
    <div role="group" aria-label="Размер" className="inline-flex self-start rounded-pill bg-bg p-[3px]">
      {dish.variants.map((variant) => (
        <button
          key={variant.id}
          type="button"
          aria-pressed={variant.id === value}
          onClick={() => onChange(variant.id)}
          className={cn(
            "min-h-11 cursor-pointer rounded-pill px-3 text-[13px]",
            variant.id === value ? "bg-inverse-bg font-semibold text-inverse-text" : "text-text",
          )}
        >
          {variant.label}
        </button>
      ))}
    </div>
  );
}

/**
 * Menu card. Compact on mobile (2 columns, the price is inside the add button), roomier from 768 px
 * (photo, composition, price and a round "+").
 */
export function DishCard({ dish, eager = false }: { dish: Dish; eager?: boolean }) {
  const [variantId, setVariantId] = useState(dish.variants[0].id);
  const variant = dish.variants.find((v) => v.id === variantId) ?? dish.variants[0];
  const hasSizes = dish.variants.length > 1;
  const price = variant.price === null ? "—" : formatPrice(variant.price);

  return (
    <Card as="article" className="rounded-md p-1.5 md:rounded-lg md:p-2">
      <div className="relative h-[130px] overflow-hidden rounded-sm md:h-[180px]">
        <Image
          src={dish.photo.src}
          alt={dish.photo.alt}
          fill
          loading={eager ? "eager" : undefined}
          fetchPriority={eager ? "high" : undefined}
          sizes="(min-width: 1024px) 300px, (min-width: 768px) 33vw, 50vw"
          className="object-cover"
        />
      </div>

      <div className="flex flex-1 flex-col px-1.5 pt-2.5 pb-1 md:gap-1 md:px-2.5 md:pt-3 md:pb-1.5">
        <h3 className="text-[15px] leading-[1.3] font-bold md:text-[17px] md:leading-normal">{dish.name}</h3>

        {hasSizes ? (
          <div className="mt-1.5 flex flex-1 flex-col gap-1.5">
            <SizeSwitch dish={dish} value={variant.id} onChange={setVariantId} />
          </div>
        ) : (
          <p className="mt-0.5 mb-2 flex-1 text-[13px] text-text-muted md:mt-0 md:mb-0 md:text-sm">
            {dish.composition && <span className="hidden md:inline">{dish.composition} · </span>}
            {variant.weight}
          </p>
        )}

        {/* Desktop: price on the left, round button on the right */}
        <div className="mt-2.5 hidden items-center justify-between md:flex">
          <span className="font-display text-[19px] font-medium">{price}</span>
          <AddToCartButton dish={dish} variantId={variant.id} />
        </div>
        {/* Mobile: one pill with the price */}
        <div className="mt-1.5 md:hidden">
          <AddToCartButton dish={dish} variantId={variant.id} layout="pill" />
        </div>
      </div>
    </Card>
  );
}
