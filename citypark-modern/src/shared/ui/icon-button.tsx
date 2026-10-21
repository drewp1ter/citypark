import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/shared/lib";
import { AppLink } from "./app-link";

export type IconButtonVariant = "outline" | "surface" | "inverse" | "accent";

const variants: Record<IconButtonVariant, string> = {
  outline: "border border-border text-text",
  surface: "border border-border bg-surface text-text",
  inverse: "bg-inverse-bg text-inverse-text",
  accent: "bg-accent text-on-accent",
};

type OwnProps = {
  variant?: IconButtonVariant;
  /** Icon buttons have no text, so the label is required. */
  "aria-label": string;
};
type IconButtonAsLink = OwnProps & { href: string } & Omit<ComponentPropsWithoutRef<"a">, "href" | "aria-label">;
type IconButtonAsButton = OwnProps & { href?: undefined } & Omit<ComponentPropsWithoutRef<"button">, "aria-label">;

/** 44×44 circle (phone, cart, "+", "↗", theme toggle). Link when `href` is given, otherwise a <button>. */
export function IconButton(props: IconButtonAsLink | IconButtonAsButton) {
  const { variant = "outline", className, ...rest } = props;
  const classes = cn(
    "inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-full disabled:cursor-not-allowed disabled:opacity-50",
    variants[variant],
    className,
  );

  if (rest.href !== undefined) {
    return <AppLink {...(rest as IconButtonAsLink)} className={classes} />;
  }
  return <button type="button" {...(rest as IconButtonAsButton)} className={classes} />;
}
