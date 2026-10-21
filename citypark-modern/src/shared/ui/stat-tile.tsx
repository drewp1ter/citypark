import type { ReactNode } from "react";
import { cn } from "@/shared/lib";

type StatTileProps = {
  /** Big number: "12", "24/7", "0 ₽". */
  value: string;
  caption: ReactNode;
  /** Lime tile instead of the white one. */
  accent?: boolean;
  className?: string;
};

/** Big number + muted caption (home and hotel pages). Render inside a <ul>. */
export function StatTile({ value, caption, accent = false, className }: StatTileProps) {
  return (
    <li
      className={cn(
        "rounded-sm px-4 py-3.5 md:rounded-md md:px-6 md:py-[22px]",
        accent ? "bg-accent text-on-accent" : "border border-border bg-surface",
        className,
      )}
    >
      <span className="block font-display text-[22px] font-medium md:text-[28px] md:tracking-[-0.02em]">{value}</span>
      <span className={cn("text-sm md:text-base", !accent && "text-text-muted")}>{caption}</span>
    </li>
  );
}
