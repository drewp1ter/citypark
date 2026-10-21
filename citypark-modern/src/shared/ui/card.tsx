import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/shared/lib";

export type CardVariant = "surface" | "inverse" | "accent";

const variants: Record<CardVariant, string> = {
  surface: "border border-border bg-surface text-text",
  // Dark card on the light page (featured room, "Всё меню доставки"); a lighter gray in the dark theme.
  inverse: "bg-inverse-bg text-inverse-text",
  accent: "bg-accent text-on-accent",
};

/** Class names of a card, for cards that are links (use with AppLink). Add the radius and padding yourself. */
export function cardClass(variant: CardVariant = "surface", className?: string): string {
  return cn("relative flex flex-col overflow-hidden", variants[variant], className);
}

type CardProps = ComponentPropsWithoutRef<"div"> & {
  as?: "div" | "article" | "section" | "aside";
  variant?: CardVariant;
};

export function Card({ as: Tag = "div", variant = "surface", className, ...props }: CardProps) {
  return <Tag className={cardClass(variant, className)} {...props} />;
}
