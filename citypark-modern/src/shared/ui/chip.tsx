import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/shared/lib";
import { AppLink, type LinkExtras } from "./app-link";

type OwnProps = { selected?: boolean };
type ChipAsLink = OwnProps & LinkExtras & { href: string } & Omit<ComponentPropsWithoutRef<"a">, "href">;
type ChipAsButton = OwnProps & { href?: undefined } & ComponentPropsWithoutRef<"button">;

/** Pill filter / category tab. Selected = inverse colours, others = outline. Link when `href` is given. */
export function Chip(props: ChipAsLink | ChipAsButton) {
  const { selected = false, className, ...rest } = props;
  const classes = cn(
    "inline-flex min-h-11 shrink-0 cursor-pointer items-center rounded-pill px-[18px]",
    selected ? "bg-inverse-bg font-semibold text-inverse-text" : "border border-border-strong font-medium text-text",
    className,
  );

  if (rest.href !== undefined) {
    return <AppLink {...(rest as ChipAsLink)} aria-current={selected ? "true" : undefined} className={classes} />;
  }
  return <button type="button" {...(rest as ChipAsButton)} aria-pressed={selected} className={classes} />;
}
